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
	"context"
	"fmt"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/do"
	"github.com/spf13/cobra"
)

// Keys generates the serverless 'keys' subtree for addition to the doctl command
func Keys() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "key",
			Short: "Manage access keys for functions namespaces",
			Long: `Access keys provide secure authentication for serverless operations without using your main DigitalOcean token.

These commands allow you to create, list, and delete namespace-specific access keys.
Keys operate on the currently connected namespace by default, but can target any namespace using the --namespace flag.`,
			Aliases: []string{"keys"},
		},
	}

	create := CmdBuilder(cmd, RunAccessKeyCreate, "create", "Creates a new access key",
		`Creates a new access key for the specified namespace. The secret is displayed only once upon creation.

Examples:
  doctl serverless key create --name "my-laptop-key"
  doctl serverless key create --name "ci-cd-key" --namespace fn-abc123`,
		Writer)
	AddStringFlag(create, "name", "n", "", "name for the access key", requiredOpt())
	AddStringFlag(create, "namespace", "", "", "target namespace (uses connected namespace if not specified)")

	list := CmdBuilder(cmd, RunAccessKeyList, "list", "Lists access keys",
		`Lists all access keys for the specified namespace with their metadata.

Examples:
  doctl serverless key list
  doctl serverless key list --namespace fn-abc123`,
		Writer, aliasOpt("ls"), displayerType(&displayers.AccessKeys{}))
	AddStringFlag(list, "namespace", "", "", "target namespace (uses connected namespace if not specified)")

	delete := CmdBuilder(cmd, RunAccessKeyDelete, "delete <access-key>", "Deletes an access key",
		`Permanently deletes an existing access key. This action cannot be undone.

Examples:
  doctl serverless key delete <access-key>
  doctl serverless key delete <access-key> --force`,
		Writer, aliasOpt("rm"))
	AddStringFlag(delete, "namespace", "", "", "target namespace (uses connected namespace if not specified)")
	AddBoolFlag(delete, "force", "f", false, "skip confirmation prompt")

	return cmd
}

// RunAccessKeyCreate handles the access key create command
func RunAccessKeyCreate(c *CmdConfig) error {
	name, _ := c.Doit.GetString(c.NS, "name")
	namespace, _ := c.Doit.GetString(c.NS, "namespace")

	// Resolve target namespace
	targetNamespace, err := resolveTargetNamespace(c, namespace)
	if err != nil {
		return err
	}

	// Create the access key
	ss := c.Serverless()
	ctx := context.TODO()

	accessKey, err := ss.CreateNamespaceAccessKey(ctx, targetNamespace, name)
	if err != nil {
		return err
	}

	// Display with security warning
	fmt.Fprintf(c.Out, "Notice: The secret key for \"%s\" is shown below.\n", name)
	fmt.Fprintf(c.Out, "Please save this secret. You will not be able to see it again.\n\n")

	// Display table with full secret (using ForCreate to show complete secret)
	displayKeys := &displayers.AccessKeys{AccessKeys: []do.AccessKey{accessKey}}
	return c.Display(displayKeys.ForCreate())
}

// RunAccessKeyList handles the access key list command
func RunAccessKeyList(c *CmdConfig) error {
	if len(c.Args) > 0 {
		return doctl.NewTooManyArgsErr(c.NS)
	}
	namespace, _ := c.Doit.GetString(c.NS, "namespace")

	// Resolve target namespace
	targetNamespace, err := resolveTargetNamespace(c, namespace)
	if err != nil {
		return err
	}

	// List access keys
	ss := c.Serverless()
	ctx := context.TODO()

	keys, err := ss.ListNamespaceAccessKeys(ctx, targetNamespace)
	if err != nil {
		return err
	}

	return c.Display(&displayers.AccessKeys{AccessKeys: keys})
}

// RunAccessKeyDelete handles the access key delete command
func RunAccessKeyDelete(c *CmdConfig) error {
	err := ensureOneArg(c)
	if err != nil {
		return err
	}

	keyID := c.Args[0]
	namespace, _ := c.Doit.GetString(c.NS, "namespace")
	force, _ := c.Doit.GetBool(c.NS, "force")

	// Resolve target namespace
	targetNamespace, err := resolveTargetNamespace(c, namespace)
	if err != nil {
		return err
	}

	// Confirmation prompt unless --force
	if !force {
		fmt.Fprintf(c.Out, "Warning: Deleting this key is a permanent action.\n")
		if err := AskForConfirm(fmt.Sprintf("delete key %s", keyID)); err != nil {
			return err
		}
	}

	// Delete the key
	ss := c.Serverless()
	ctx := context.TODO()

	err = ss.DeleteNamespaceAccessKey(ctx, targetNamespace, keyID)
	if err != nil {
		return err
	}

	fmt.Fprintf(c.Out, "Key %s has been deleted.\n", keyID)
	return nil
}

// resolveTargetNamespace determines which namespace to operate on
// If explicitNamespace is provided, use it; otherwise use the currently connected namespace
func resolveTargetNamespace(c *CmdConfig, explicitNamespace string) (string, error) {
	ss := c.Serverless()

	if explicitNamespace != "" {
		// VALIDATE NAMESPACE EXISTS
		_, err := ss.GetNamespace(context.TODO(), explicitNamespace)
		if err != nil {
			return "", fmt.Errorf("namespace '%s' not found or not accessible", explicitNamespace)
		}
		return explicitNamespace, nil
	}

	// Use connected namespace
	if err := ss.CheckServerlessStatus(); err != nil {
		return "", err
	}
	creds, err := ss.ReadCredentials()
	if err != nil {
		return "", fmt.Errorf("not connected to any namespace. Use --namespace flag or run 'doctl serverless connect' first")
	}

	if creds.Namespace == "" {
		return "", fmt.Errorf("not connected to any namespace. Use --namespace flag or run 'doctl serverless connect' first")
	}

	return creds.Namespace, nil
}
