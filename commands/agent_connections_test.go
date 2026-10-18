/*
Copyright 2026 The Doctl Authors All rights reserved.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package commands

import (
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

// Every connections subcommand resolves --provider at runtime, so each must
// register the flag. get/revoke once relied on it without registering it,
// which only a real command-tree assertion (not a direct Doit.Set) catches.
func TestAgentConnectionsProviderFlagRegistered(t *testing.T) {
	root := AgentConnections()
	for _, name := range []string{"list", "create", "get", "revoke"} {
		var sub *cobra.Command
		for _, c := range root.Commands() {
			if c.Name() == name {
				sub = c
				break
			}
		}
		if !assert.NotNilf(t, sub, "subcommand %q not found", name) {
			continue
		}
		f := sub.Flags().Lookup(doctl.ArgAgentConnProvider)
		if assert.NotNilf(t, f, "%q is missing the --%s flag", name, doctl.ArgAgentConnProvider) {
			assert.Equalf(t, "github", f.DefValue, "%q --%s default", name, doctl.ArgAgentConnProvider)
		}
	}
}

// The API accepts --status revoked, but tenant list reads always omit revoked
// connections, even when that is the filter (AR-2546; see the status field of
// ListConnectionsRequest in tool-registry.proto). So revoked must not be
// advertised as a filter value, and the help must say why it is missing:
// otherwise the empty result reads as "no revoked connections" (MARSOHS-2175).
func TestAgentConnectionsListStatusFlagUsage(t *testing.T) {
	var list *cobra.Command
	for _, c := range AgentConnections().Commands() {
		if c.Name() == "list" {
			list = c
			break
		}
	}
	if !assert.NotNil(t, list, "list subcommand not found") {
		return
	}
	f := list.Flags().Lookup(doctl.ArgAgentStatus)
	if !assert.NotNilf(t, f, "list is missing the --%s flag", doctl.ArgAgentStatus) {
		return
	}

	m := regexp.MustCompile(`\(([a-z|]+)\)`).FindStringSubmatch(f.Usage)
	if assert.Lenf(t, m, 2, "--%s usage should list its values as (a|b|c): %q", doctl.ArgAgentStatus, f.Usage) {
		assert.ElementsMatch(t, []string{"active", "pending", "expired"}, strings.Split(m[1], "|"),
			"advertised --%s values", doctl.ArgAgentStatus)
	}
	assert.Containsf(t, strings.ToLower(f.Usage), "revoked",
		"--%s usage should explain that revoked connections are never listed", doctl.ArgAgentStatus)
}

func TestRunAgentsConnectionsList(t *testing.T) {
	t.Run("forwards actor/status/paging filters", func(t *testing.T) {
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			tm.hostedAgents.EXPECT().
				ListConnections("github", &godo.HostedAgentConnectionListOptions{
					ActorID: "alice",
					Status:  "active",
					Page:    2,
					PerPage: 10,
				}).
				Return(&godo.HostedAgentConnectionsListResponse{
					Connections: []godo.HostedAgentConnection{
						{ID: "conn_1", Provider: "github", ActorID: "alice", Status: "active"},
					},
					Pagination: godo.HostedAgentConnectionPagination{Page: 2, PerPage: 10, Total: 1},
				}, nil)

			config.Doit.Set(config.NS, doctl.ArgAgentConnProvider, "github")
			config.Doit.Set(config.NS, doctl.ArgAgentConnActor, "alice")
			config.Doit.Set(config.NS, doctl.ArgAgentStatus, "active")
			config.Doit.Set(config.NS, doctl.ArgAgentConnPage, 2)
			config.Doit.Set(config.NS, doctl.ArgAgentConnPerPage, 10)
			assert.NoError(t, RunAgentsConnectionsList(config))
		})
	})

	t.Run("501 becomes a not-enabled hint", func(t *testing.T) {
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			tm.hostedAgents.EXPECT().
				ListConnections("github", gomock.Any()).
				Return(nil, &godo.ErrorResponse{
					Response: &http.Response{StatusCode: http.StatusNotImplemented},
					Message:  "oauth connections are not enabled",
				})

			config.Doit.Set(config.NS, doctl.ArgAgentConnProvider, "github")
			err := RunAgentsConnectionsList(config)
			assert.ErrorContains(t, err, "not enabled")
		})
	})
}

func TestRunAgentsConnectionsCreate(t *testing.T) {
	orig := agentsAuthPollInterval
	agentsAuthPollInterval = time.Millisecond
	defer func() { agentsAuthPollInterval = orig }()

	t.Run("already active exits without polling", func(t *testing.T) {
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			tm.hostedAgents.EXPECT().
				CreateConnection("github", &godo.HostedAgentConnectionCreateRequest{ActorID: "alice"}).
				Return(&godo.HostedAgentConnectionEnvelope{
					Connection: godo.HostedAgentConnection{ID: "conn_1", Provider: "github", ActorID: "alice", Status: "active"},
				}, nil)

			config.Doit.Set(config.NS, doctl.ArgAgentConnProvider, "github")
			config.Doit.Set(config.NS, doctl.ArgAgentConnActor, "alice")
			assert.NoError(t, RunAgentsConnectionsCreate(config))
		})
	})

	t.Run("pending connection polls until active", func(t *testing.T) {
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			tm.hostedAgents.EXPECT().
				CreateConnection("github", &godo.HostedAgentConnectionCreateRequest{
					ActorID: "alice",
					Scopes:  []string{"repo", "read:org"},
				}).
				Return(&godo.HostedAgentConnectionEnvelope{
					Connection: godo.HostedAgentConnection{ID: "conn_1", Provider: "github", ActorID: "alice", Status: "pending"},
					Authorization: &godo.HostedAgentConnectionAuthorization{
						Status:     "pending",
						ConnectURL: "https://example.com/connect",
					},
				}, nil)
			gomock.InOrder(
				tm.hostedAgents.EXPECT().
					GetConnection("github", "conn_1").
					Return(&godo.HostedAgentConnectionEnvelope{
						Connection: godo.HostedAgentConnection{ID: "conn_1", Provider: "github", ActorID: "alice", Status: "pending"},
					}, nil),
				tm.hostedAgents.EXPECT().
					GetConnection("github", "conn_1").
					Return(&godo.HostedAgentConnectionEnvelope{
						Connection: godo.HostedAgentConnection{ID: "conn_1", Provider: "github", ActorID: "alice", Status: "active"},
					}, nil),
			)

			config.Doit.Set(config.NS, doctl.ArgAgentConnProvider, "github")
			config.Doit.Set(config.NS, doctl.ArgAgentConnActor, "alice")
			config.Doit.Set(config.NS, doctl.ArgAgentConnScopes, []string{"repo", "read:org"})
			config.Doit.Set(config.NS, doctl.ArgAgentAuthNoBrowser, true)
			assert.NoError(t, RunAgentsConnectionsCreate(config))
		})
	})

	t.Run("requires an actor", func(t *testing.T) {
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			config.Doit.Set(config.NS, doctl.ArgAgentConnProvider, "github")
			assert.Error(t, RunAgentsConnectionsCreate(config))
		})
	})
}

func TestRunAgentsConnectionsGet(t *testing.T) {
	t.Run("reads one connection", func(t *testing.T) {
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			tm.hostedAgents.EXPECT().
				GetConnection("github", "conn_1").
				Return(&godo.HostedAgentConnectionEnvelope{
					Connection: godo.HostedAgentConnection{ID: "conn_1", Provider: "github", ActorID: "alice", Status: "active"},
				}, nil)

			config.Doit.Set(config.NS, doctl.ArgAgentConnProvider, "github")
			config.Args = []string{"conn_1"}
			assert.NoError(t, RunAgentsConnectionsGet(config))
		})
	})

	t.Run("requires a connection id", func(t *testing.T) {
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			config.Doit.Set(config.NS, doctl.ArgAgentConnProvider, "github")
			assert.Error(t, RunAgentsConnectionsGet(config))
		})
	})
}

func TestRunAgentsConnectionsRevoke(t *testing.T) {
	t.Run("revokes by id", func(t *testing.T) {
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			tm.hostedAgents.EXPECT().
				DeleteConnection("github", "conn_1").
				Return(&godo.HostedAgentConnectionEnvelope{
					Connection: godo.HostedAgentConnection{ID: "conn_1", Provider: "github", ActorID: "alice", Status: "expired"},
				}, nil)

			config.Doit.Set(config.NS, doctl.ArgAgentConnProvider, "github")
			config.Args = []string{"conn_1"}
			assert.NoError(t, RunAgentsConnectionsRevoke(config))
		})
	})

	t.Run("requires a connection id", func(t *testing.T) {
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			config.Doit.Set(config.NS, doctl.ArgAgentConnProvider, "github")
			assert.Error(t, RunAgentsConnectionsRevoke(config))
		})
	})
}
