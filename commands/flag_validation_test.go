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
	"bytes"
	"strings"
	"testing"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/internal/ui"
	"github.com/muesli/termenv"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestParent() *Command {
	return &Command{Command: &cobra.Command{Use: "test"}}
}

func TestValidateCommandFlags_AggregatesMissingRequired(t *testing.T) {
	cmd := CmdBuilder(newTestParent(), func(c *CmdConfig) error { return nil }, "create", "short", "long", &bytes.Buffer{})
	AddStringFlag(cmd, "size", "", "", "size desc",
		requiredOpt(),
		flagPurpose("Droplet size"),
		flagHint("run doctl compute size list"),
	)
	AddStringFlag(cmd, "image", "", "", "image desc",
		requiredOpt(),
		flagPurpose("Droplet image"),
		flagHint("run doctl compute image list"),
	)

	err := validateCommandFlags(cmd.Command)
	require.Error(t, err)

	fv, ok := err.(*FlagValidationError)
	require.True(t, ok)
	assert.Len(t, fv.Issues, 2)

	msg := err.Error()
	assert.Contains(t, msg, "--size")
	assert.Contains(t, msg, "--image")
	assert.Contains(t, msg, "Droplet size")
	assert.Contains(t, msg, "run doctl compute size list")
	assert.Contains(t, msg, "Droplet image")
	assert.Contains(t, msg, "run doctl compute image list")
	assert.Contains(t, msg, "Missing 2 required flags")
	assert.Contains(t, msg, "for usage")
}

func TestValidateCommandFlags_EmptyRequiredCountsAsMissing(t *testing.T) {
	cmd := CmdBuilder(newTestParent(), func(c *CmdConfig) error { return nil }, "create", "short", "long", &bytes.Buffer{})
	AddStringFlag(cmd, "size", "", "", "size desc", requiredOpt(), flagPurpose("Droplet size"))

	require.NoError(t, cmd.Flags().Set("size", "   "))

	err := validateCommandFlags(cmd.Command)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "provided but empty")
}

func TestValidateCommandFlags_AcceptsNonEmptyDefault(t *testing.T) {
	cmd := CmdBuilder(newTestParent(), func(c *CmdConfig) error { return nil }, "create", "short", "long", &bytes.Buffer{})
	AddStringFlag(cmd, "subscription-tier", "", "basic", "tier", requiredOpt())

	assert.NoError(t, validateCommandFlags(cmd.Command))
}

func TestValidateCommandFlags_AcceptsConfigValueWhenDefaultEmpty(t *testing.T) {
	cmd := CmdBuilder(newTestParent(), func(c *CmdConfig) error { return nil }, "create", "short", "long", &bytes.Buffer{})
	AddStringFlag(cmd, "region", "", "", "region", requiredOpt())

	key := "test.create.region"
	viper.Set(key, "nyc1")
	t.Cleanup(func() { viper.Set(key, nil) })

	assert.NoError(t, validateCommandFlags(cmd.Command))
}

func TestValidateCommandFlags_AcceptsConfigStringSlice(t *testing.T) {
	cmd := CmdBuilder(newTestParent(), func(c *CmdConfig) error { return nil }, "apply", "short", "long", &bytes.Buffer{})
	AddStringSliceFlag(cmd, "resource", "", nil, "resources", requiredOpt())

	key := "test.apply.resource"
	viper.Set(key, []string{"do:droplet:12345"})
	t.Cleanup(func() { viper.Set(key, nil) })

	assert.NoError(t, validateCommandFlags(cmd.Command))
}

func TestValidateCommandFlags_ZeroIntRequiredIsMissing(t *testing.T) {
	cmd := CmdBuilder(newTestParent(), func(c *CmdConfig) error { return nil }, "resize", "short", "long", &bytes.Buffer{})
	AddIntFlag(cmd, "size", "", 0, "size in GiB", requiredOpt())

	err := validateCommandFlags(cmd.Command)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--size")
}

func TestValidateCommandFlags_MutuallyExclusive(t *testing.T) {
	cmd := CmdBuilder(newTestParent(), func(c *CmdConfig) error { return nil }, "create", "short", "long", &bytes.Buffer{})
	AddStringFlag(cmd, doctl.ArgUserData, "", "", "inline",
		flagPurpose("inline user data"),
		flagHint("use one of --user-data or --user-data-file"),
	)
	AddStringFlag(cmd, doctl.ArgUserDataFile, "", "", "file",
		flagPurpose("user data file"),
	)
	cmd.MarkFlagsMutuallyExclusive(doctl.ArgUserData, doctl.ArgUserDataFile)

	require.NoError(t, cmd.Flags().Set(doctl.ArgUserData, "echo hi"))
	require.NoError(t, cmd.Flags().Set(doctl.ArgUserDataFile, "/tmp/x"))

	err := validateCommandFlags(cmd.Command)
	require.Error(t, err)
	msg := err.Error()
	assert.Contains(t, msg, "--user-data")
	assert.Contains(t, msg, "--user-data-file")
	assert.Contains(t, msg, "cannot be combined")
}

func TestValidateCommandFlags_RequiredTogether(t *testing.T) {
	cmd := CmdBuilder(newTestParent(), func(c *CmdConfig) error { return nil }, "create", "short", "long", &bytes.Buffer{})
	AddStringFlag(cmd, "alpha", "", "", "alpha")
	AddStringFlag(cmd, "beta", "", "", "beta")
	AddIntFlag(cmd, "gamma", "", 0, "gamma")
	cmd.MarkFlagsRequiredTogether("alpha", "beta", "gamma")

	require.NoError(t, cmd.Flags().Set("alpha", "one"))

	err := validateCommandFlags(cmd.Command)
	require.Error(t, err)
	msg := err.Error()
	assert.Contains(t, msg, "beta")
	assert.Contains(t, msg, "gamma")
	assert.Contains(t, msg, "must be set together")
}

func TestValidateCommandFlags_UsesFlagUsageAsFallback(t *testing.T) {
	cmd := CmdBuilder(newTestParent(), func(c *CmdConfig) error { return nil }, "create", "short", "long", &bytes.Buffer{})
	AddStringFlag(cmd, "size", "", "", "The size of the nodes. Use the `doctl kubernetes options sizes` command for a list of possible values.", requiredOpt())

	err := validateCommandFlags(cmd.Command)
	require.Error(t, err)
	msg := err.Error()
	assert.Contains(t, msg, "--size")
	assert.Contains(t, msg, "The size of the nodes.")
	assert.Contains(t, msg, "run doctl kubernetes options sizes")
}

func TestValidateCommandFlags_OKWhenComplete(t *testing.T) {
	cmd := CmdBuilder(newTestParent(), func(c *CmdConfig) error { return nil }, "create", "short", "long", &bytes.Buffer{})
	AddStringFlag(cmd, "size", "", "", "size desc", requiredOpt())
	AddStringFlag(cmd, "image", "", "", "image desc", requiredOpt())
	require.NoError(t, cmd.Flags().Set("size", "s-1vcpu-1gb"))
	require.NoError(t, cmd.Flags().Set("image", "ubuntu-22-04-x64"))

	assert.NoError(t, validateCommandFlags(cmd.Command))
}

func TestDropletCreate_PreRunAggregatesSizeAndImage(t *testing.T) {
	cmd := Droplet()
	create := findSubCommand(cmd, "create")
	require.NotNil(t, create)
	require.NotNil(t, create.PreRunE)

	err := create.PreRunE(create.Command, []string{"example"})
	require.Error(t, err)
	fv, ok := err.(*FlagValidationError)
	require.True(t, ok, "got %T: %v", err, err)

	flags := map[string]FlagIssue{}
	for _, issue := range fv.Issues {
		flags[issue.Flag] = issue
	}
	require.Contains(t, flags, doctl.ArgSizeSlug)
	require.Contains(t, flags, doctl.ArgImage)
	assert.True(t, strings.Contains(flags[doctl.ArgSizeSlug].Hint, "size list"))
	assert.True(t, strings.Contains(flags[doctl.ArgImage].Hint, "image list"))
}

func TestDropletCreate_BackupPolicyPartialFlagsAllowed(t *testing.T) {
	cmd := Droplet()
	create := findSubCommand(cmd, "create")
	require.NotNil(t, create)
	require.NotNil(t, create.PreRunE)

	require.NoError(t, create.Flags().Set(doctl.ArgSizeSlug, "s-1vcpu-1gb"))
	require.NoError(t, create.Flags().Set(doctl.ArgImage, "ubuntu-22-04-x64"))

	// Daily plan does not require weekday; hour may be omitted (defaults to 0).
	require.NoError(t, create.Flags().Set(doctl.ArgDropletBackupPolicyPlan, "daily"))
	assert.NoError(t, create.PreRunE(create.Command, []string{"example"}))

	// Weekly plan with weekday but without hour should also pass (hour defaults to 0).
	require.NoError(t, create.Flags().Set(doctl.ArgDropletBackupPolicyPlan, "weekly"))
	require.NoError(t, create.Flags().Set(doctl.ArgDropletBackupPolicyWeekday, "SUN"))
	assert.NoError(t, create.PreRunE(create.Command, []string{"example"}))
}

func TestRegistryCreate_DefaultSubscriptionTierPassesPreRun(t *testing.T) {
	cmd := Registry()
	create := findSubCommand(cmd, "create")
	require.NotNil(t, create)
	require.NotNil(t, create.PreRunE)
	assert.NoError(t, create.PreRunE(create.Command, []string{"foo"}))
}

func TestKubernetesClusterCreate_DefaultRegionPassesPreRun(t *testing.T) {
	cmd := Kubernetes()
	cluster := findSubCommand(cmd, "cluster")
	require.NotNil(t, cluster)
	create := findSubCommand(cluster, "create")
	require.NotNil(t, create)
	require.NotNil(t, create.PreRunE)

	// Region defaults to nyc1; other required flags still missing is OK for this check —
	// we only assert region is not reported when unset on the CLI.
	err := create.PreRunE(create.Command, []string{"my-cluster"})
	if err == nil {
		return
	}
	fv, ok := err.(*FlagValidationError)
	require.True(t, ok, "got %T: %v", err, err)
	for _, issue := range fv.Issues {
		assert.NotEqual(t, doctl.ArgRegionSlug, issue.Flag, "default region should satisfy required --region")
	}
}

func TestFlagValidationDisplayIncludesDesignRed(t *testing.T) {
	err := &FlagValidationError{
		Command: "doctl compute droplet create",
		Issues: []FlagIssue{
			{Flag: "size", Problem: "is required but was not set", Purpose: "Droplet size", Hint: "run doctl compute size list"},
		},
	}
	env := ui.Detect(ioDiscard{}, ioDiscard{}, ui.WithProfile(termenv.TrueColor), ui.WithASCII(false))
	out := err.format(ui.NewStyle(env))
	if !strings.Contains(out, "38;2;215;70;35") {
		t.Fatalf("expected design red in display output; got %q", out)
	}
}

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) { return len(p), nil }

func findSubCommand(parent *Command, name string) *Command {
	for _, child := range parent.ChildCommands() {
		if child.Name() == name {
			return child
		}
	}
	return nil
}
