/*
Copyright 2018 The Doctl Authors All rights reserved.
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
	"errors"

	"github.com/spf13/cobra"

	"github.com/digitalocean/doctl/commands/displayers"
)

// authClient creates the commands for inspecting the OAuth application doctl
// registered for itself.
func authClient() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "client",
			Short: "Display the OAuth application doctl registered for this machine",
			Long: `The subcommands of ` + "`" + `doctl auth client` + "`" + ` describe the OAuth application doctl registered with DigitalOcean the first time you ran ` + "`" + `doctl auth login` + "`" + `.

doctl registers one application per authorization server and reuses it for every authentication context, so this is machine-wide rather than per account.`,
		},
	}

	cmdClientShow := cmdBuilderWithInit(cmd, RunAuthClientShow, "show", "Display the registered OAuth application", `Display the OAuth application doctl registered for itself, including its client ID and the redirect URIs it accepts.

The credential that manages the registration is never printed.

If no application has been registered yet, run `+"`"+`doctl auth login`+"`"+` to create one.`, Writer, false, aliasOpt("get"), displayerType(&displayers.OAuthClient{}))
	cmdClientShow.Example = `The following example displays the registered application as JSON: doctl auth client show --output json`

	return cmd
}

// RunAuthClientShow displays the stored dynamic client registration.
func RunAuthClientShow(c *CmdConfig) error {
	client := loadOAuthClientState()
	if client == nil {
		return errors.New("doctl has not registered an OAuth application yet. Run `doctl auth login` to register one")
	}

	return c.Display(&displayers.OAuthClient{
		ClientID:      client.ClientID,
		Issuer:        client.Issuer,
		RedirectURIs:  client.RedirectURIs,
		RegisteredAt:  client.RegisteredAt,
		ManagementURL: client.RegistrationClientURI,
	})
}
