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
	"bytes"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

const (
	completionLong = "`" + `doctl completion` + "`" + ` helps you configure your terminal's shell so that doctl commands autocomplete when you press the TAB key.

Supported shells:

- bash
- zsh
- fish`
	bashLong = `
Use ` + "`" + `doctl completion bash` + "`" + ` to configure your bash shell so that doctl commands autocomplete when you press the TAB key.

To review the configuration, run ` + "`" + `doctl completion bash` + "`" + `.

To enable the configuration, add the following line to your .profile or .bashrc.

	source <(doctl completion bash)

Note:

- macOS users must install the ` + "`" + `bash-completion` + "`" + ` framework to use completion features, which can be done with homebrew:

		brew install bash-completion

	Once installed, edit your .profile or .bashrc file and add the following line:

		source $(brew --prefix)/etc/bash_completion
`
	zshLong = `
Use ` + "`" + `doctl completion zsh` + "`" + ` to configure your zsh shell so that doctl commands autocomplete when you press the TAB key.

To review the configuration, run ` + "`" + `doctl completion zsh` + "`" + `.

To enable the configuration, add the following line to your .profile or .bashrc.

	source  <(doctl completion zsh)

Note:

- zsh completions requires zsh 5.2 or newer.
`
	fishLong = `
Use ` + "`" + `doctl completion fish` + "`" + ` to configure your zsh shell so that doctl commands autocomplete when you press the TAB key.

To review the configuration, run ` + "`" + `doctl completion fish` + "`" + `.

To enable the configuration, add the following line to your .profile or .bashrc.

	source  <(doctl completion fish)
`

	doctlLicense = `# Copyright 2018 The Doctl Authors All rights reserved.
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#   http://www.apache.org/licenses/LICENSE-2.0
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
`
)

// Completion creates the completion command
func Completion() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "completion",
			Short: "Modify your shell so doctl commands autocomplete with TAB",
			Long:  completionLong,
		},
	}

	cmdBuilderWithInit(cmd, RunCompletionBash, "bash", "Generate completion code for bash", bashLong, Writer, false)
	cmdBuilderWithInit(cmd, RunCompletionFish, "fish", "Generate completion code for fish", fishLong, Writer, false)
	cmdBuilderWithInit(cmd, RunCompletionZsh, "zsh", "Generate completion code for zsh", zshLong, Writer, false)

	return cmd
}

// RunCompletionBash outputs completion code for bash.
func RunCompletionBash(c *CmdConfig) error {
	var buf bytes.Buffer

	_, err := buf.Write([]byte(doctlLicense))
	if err != nil {
		return fmt.Errorf("Error while generating bash completion: %v", err)
	}

	err = DoitCmd.GenBashCompletion(&buf)
	if err != nil {
		return fmt.Errorf("Error while generating bash completion: %v", err)
	}

	// remove the command "completion" from auto-completion
	code := buf.String()
	code = strings.Replace(code, `commands+=("completion")`, "", -1)

	fmt.Print(code)
	return nil
}

func RunCompletionFish(c *CmdConfig) error {
	var buf bytes.Buffer

	fishCompletion := `
# Completions for the global flags
complete -c doctl -s h -l help         -d 'Show help'
complete -c doctl -s t -l access-token -d "API V2 access token"
complete -c doctl -s u -l api-url      -d "Overide default API endpoint"
complete -c doctl -s c -l config       -d "Specify a custom config file"
complete -c doctl -l context           -d "Specify a custom authentication context name"
complete -c doctl -s o -l output       -d "Desired output format [text|json] (default \"text\")"
complete -c doctl -l trace             -d "Show a log of network activity"
complete -c doctl -s v -l verbose      -d "Enable verbose output"

# Completions for the 'doctl' root command
complete -c doctl -n '__fish_use_subcommand' -a account    -d "Retrieve account details"
complete -c doctl -n '__fish_use_subcommand' -a auth       -d "Authenticating doctl with an account"
complete -c doctl -n '__fish_use_subcommand' -a balance    -d "Retrieving your account balance"
complete -c doctl -n '__fish_use_subcommand' -a completion -d "Autocomplete doctl commands"
complete -c doctl -n '__fish_use_subcommand' -a compute    -d "Manage infrastructure"
complete -c doctl -n '__fish_use_subcommand' -a databases  -d "Manage databases"
complete -c doctl -n '__fish_use_subcommand' -a help       -d "Show helps"
complete -c doctl -n '__fish_use_subcommand' -a invoice    -d "Retrieving invoices for your account"
complete -c doctl -n '__fish_use_subcommand' -a kubernetes -d "Manage Kubernetes clusters and configurations"
complete -c doctl -n '__fish_use_subcommand' -a projects   -d "Manage projects and assign resources to them"
complete -c doctl -n '__fish_use_subcommand' -a version    -d "Show the current version"

# Completions for the 'doctl account' command
complete -c doctl -n '__fish_seen_subcommand_from account' -a get       -d "Account profile details"
complete -c doctl -n '__fish_seen_subcommand_from account' -a ratelimit -d "API usage and the remaining quota"

# Completions for the 'doctl account get' command
complete -c doctl -n '__fish_seen_subcommand_from account get' -l format    -d "Columns for output"
complete -c doctl -n '__fish_seen_subcommand_from account get' -l no-header -d "Return raw data with no headers"

# COmpletions for the 'doctl account ratelimit' command
complete -c doctl -n '__fish_seen_subcommand_from account ratelimit' -l format    -d "Columns for output"
complete -c doctl -n '__fish_seen_subcommand_from account ratelimit' -l no-header -d "Return raw data with no headers"

# Completions for the 'doctl auth' command
complete -c doctl -n '__fish_seen_subcommand_from auth' -a init   -d "Initialize doctl"
complete -c doctl -n '__fish_seen_subcommand_from auth' -a list   -d "List available authentication context"
complete -c doctl -n '__fish_seen_subcommand_from auth' -a switch -d "Switches between authentication contexts"

# COmpletions for the 'doctl auth list' command
complete -c doctl -n '__fish_seen_subcommand_from auth list' -l format -d "Columns for output"

# Completions for the 'doctl balance' command
complete -c doctl -n '__fish_seen_subcommand_from balance' -a get -d "Account balance"

# COmpletions for the 'doctl balance get' command
complete -c doctl -n '__fish_seen_subcommand_from balance get' -l format    -d "Columns for output"
complete -c doctl -n '__fish_seen_subcommand_from balance get' -l no-header -d "Return raw data with no headers"

# Completions for the 'doctl completion' command
complete -c doctl -n '__fish_seen_subcommand_from completion' -a bash  -d "Generate completion code for bash"
complete -c doctl -n '__fish_seen_subcommand_from completion' -a fish  -d "Generate completion code for fish"
complete -c doctl -n '__fish_seen_subcommand_from completion' -a zsh   -d "Generate completion code for zsh"

# Completions for the 'doctl compute' command
complete -c doctl -n '__fish_seen_subcommand_from compute' -a action             -d "Display commands for retrieving resource action history"
complete -c doctl -n '__fish_seen_subcommand_from compute' -a cdn                -d "Display commands that manage CDNs"
complete -c doctl -n '__fish_seen_subcommand_from compute' -a certificate        -d "Display commands that manage SSL certificates and private keys"
complete -c doctl -n '__fish_seen_subcommand_from compute' -a domain             -d "Display commands that manage domains"
complete -c doctl -n '__fish_seen_subcommand_from compute' -a droplet            -d "Manage virtual machines (Droplets)"
complete -c doctl -n '__fish_seen_subcommand_from compute' -a droplet-action     -d "Display Droplet action commands"
complete -c doctl -n '__fish_seen_subcommand_from compute' -a firewall           -d "Display commands to manage cloud firewalls"
complete -c doctl -n '__fish_seen_subcommand_from compute' -a floating-ip        -d "Display commands to manage floating IP addresses"
complete -c doctl -n '__fish_seen_subcommand_from compute' -a floating-ip-action -d "Display commands to associate floating IP addresses with Droplets"
complete -c doctl -n '__fish_seen_subcommand_from compute' -a image              -d "Display commands to manage images"
complete -c doctl -n '__fish_seen_subcommand_from compute' -a image-action       -d "Display commands to perform actions on images"
complete -c doctl -n '__fish_seen_subcommand_from compute' -a load-balancer      -d "Display commands to manage load balancers"
complete -c doctl -n '__fish_seen_subcommand_from compute' -a region             -d "Display commands to list datacenter regions"
complete -c doctl -n '__fish_seen_subcommand_from compute' -a size               -d "List available Droplet sizes"
complete -c doctl -n '__fish_seen_subcommand_from compute' -a snapshot           -d "Access and manage snapshots"
complete -c doctl -n '__fish_seen_subcommand_from compute' -a ssh                -d "Access a Droplet using SSH"
complete -c doctl -n '__fish_seen_subcommand_from compute' -a ssh-key            -d "Display commands to manage SSH keys on your account"
complete -c doctl -n '__fish_seen_subcommand_from compute' -a tag                -d "Display commands to manage tags"
complete -c doctl -n '__fish_seen_subcommand_from compute' -a volume             -d "Display commands to manage block storage volumes"
complete -c doctl -n '__fish_seen_subcommand_from compute' -a volume-action      -d "Display commands to perform actions on a volume"

# Completions for the 'doctl compute action' command
complete -c doctl -n '__fish_seen_subcommand_from compute action' -a get  -d "Retrieve details about a specific action"
complete -c doctl -n '__fish_seen_subcommand_from compute action' -a list -d "Retrieve a  list of all recent actions taken on your resources"
complete -c doctl -n '__fish_seen_subcommand_from compute action' -a wait -d "Block thread until an action completes"

# Completions for the 'doctl compute action get' command
complete -c doctl -n '__fish_seen_subcommand_from compute action get' -l format    -d "Columns for output"
complete -c doctl -n '__fish_seen_subcommand_from compute action get' -l no-header -d "Return raw data with no headers"

# Completions for the 'doctl compute action list' command
complete -c doctl -n '__fish_seen_subcommand_from compute action list' -l action-type   -d "Action type"
complete -c doctl -n '__fish_seen_subcommand_from compute action list' -l after         -d "Action completed after in RFC3339 format"
complete -c doctl -n '__fish_seen_subcommand_from compute action list' -l before        -d "Action completed before in RFC3339 format"
complete -c doctl -n '__fish_seen_subcommand_from compute action list' -l format        -d "Columns for output"
complete -c doctl -n '__fish_seen_subcommand_from compute action list' -l no-header     -d "Return raw data with no headers"
complete -c doctl -n '__fish_seen_subcommand_from compute action list' -l region        -d "Action region"
complete -c doctl -n '__fish_seen_subcommand_from compute action list' -l resource-type -d "Action resource type"
complete -c doctl -n '__fish_seen_subcommand_from compute action list' -l status        -d "Action status"

# Completions for the 'doctl compute action wait' command
complete -c doctl -n '__fish_seen_subcommand_from compute action wait' -l no-header    -d "Return raw data with no headers"
complete -c doctl -n '__fish_seen_subcommand_from compute action wait' -l poll-timeout -d "Re-poll time in seconds"

# Completions for the 'doctl databases' command
complete -c doctl -n '__fish_seen_subcommand_from databases' -a backups            -d "List database cluster backups"
complete -c doctl -n '__fish_seen_subcommand_from databases' -a connection         -d "Retrieve connection details for a database cluster"
complete -c doctl -n '__fish_seen_subcommand_from databases' -a create             -d "Create a database cluster"
complete -c doctl -n '__fish_seen_subcommand_from databases' -a db                 -d "Display commands for managing individual databases within a cluster"
complete -c doctl -n '__fish_seen_subcommand_from databases' -a delete             -d "Delete a database cluster"
complete -c doctl -n '__fish_seen_subcommand_from databases' -a get                -d "Get details for a database cluster"
complete -c doctl -n '__fish_seen_subcommand_from databases' -a list               -d "List your database clusters"
complete -c doctl -n '__fish_seen_subcommand_from databases' -a maintenance-window -d "Display commands for scheduling automatic maintenance on your database cluster"
complete -c doctl -n '__fish_seen_subcommand_from databases' -a migrate            -d "Migrate a database cluster to a new region"
complete -c doctl -n '__fish_seen_subcommand_from databases' -a pool               -d "Display commands for managing connection pools"
complete -c doctl -n '__fish_seen_subcommand_from databases' -a replica            -d "Display commands to manage read-only database replicas"
complete -c doctl -n '__fish_seen_subcommand_from databases' -a resize             -d "Resize a database cluster"
complete -c doctl -n '__fish_seen_subcommand_from databases' -a sql-mode           -d "Display commands to configure a MySQL database cluster's SQL modes"
complete -c doctl -n '__fish_seen_subcommand_from databases' -a user               -d "Display commands for managing database users"

# Completions for the 'doctl databases backups' command
complete -c doctl -n '__fish_seen_subcommand_from databases backups' -l format    -d "Columns for output"
complete -c doctl -n '__fish_seen_subcommand_from databases backups' -l no-header -d "Return raw data with no headers"

# Completions for the 'doctl databases connection' command
complete -c doctl -n '__fish_seen_subcommand_from databases connection' -l format    -d "Columns for output"
complete -c doctl -n '__fish_seen_subcommand_from databases connection' -l no-header -d "Return raw data with no headers"

# Completions for the 'doctl databases create' command
complete -c doctl -n '__fish_seen_subcommand_from databases create' -l engine               -d "Database engine to be used for the cluster"
complete -c doctl -n '__fish_seen_subcommand_from databases create' -l num-nodes            -d "Number of nodes in the database cluster"
complete -c doctl -n '__fish_seen_subcommand_from databases create' -l private-network-uuid -d "UUID to use for private network connection"
complete -c doctl -n '__fish_seen_subcommand_from databases create' -l region               -d "Region where the database cluster will be created"
complete -c doctl -n '__fish_seen_subcommand_from databases create' -l size                 -d "Size of the nodes in the database cluster"
complete -c doctl -n '__fish_seen_subcommand_from databases create' -l version              -d "Database engine version"

# Completions for the 'doctl databases db' command
complete -c doctl -n '__fish_seen_subcommand_from databases db' -a create -d "Create a database within a cluster"
complete -c doctl -n '__fish_seen_subcommand_from databases db' -a delete -d "Delete the specified database from the cluster"
complete -c doctl -n '__fish_seen_subcommand_from databases db' -a get    -d "Retrieve the name of a database within a cluster"
complete -c doctl -n '__fish_seen_subcommand_from databases db' -a list   -d "Retrieve a list of databases within a cluster"

# Completions for the 'doctl invoice' command
complete -c doctl -n '__fish_seen_subcommand_from invoice' -a csv     -d "Download a CSV file of an invoice"
complete -c doctl -n '__fish_seen_subcommand_from invoice' -a get     -d "Retrieve a list of all the items on an invoice"
complete -c doctl -n '__fish_seen_subcommand_from invoice' -a list    -d "List all of the invoices for your account"
complete -c doctl -n '__fish_seen_subcommand_from invoice' -a pdf     -d "Download a PDF file of an invoice"
complete -c doctl -n '__fish_seen_subcommand_from invoice' -a summary -d "Get a summary of an invoice"

# Completions for the 'doctl invoice get' command
complete -c doctl -n '__fish_seen_subcommand_from invoice get' -l format    -d "Columns for output"
complete -c doctl -n '__fish_seen_subcommand_from invoice get' -l no-header -d "Return raw data with no headers"

# Completions for the 'doctl invoice list' command
complete -c doctl -n '__fish_seen_subcommand_from invoice list' -l format    -d "Columns for output"
complete -c doctl -n '__fish_seen_subcommand_from invoice list' -l no-header -d "Return raw data with no headers"

# Completions for the 'doctl invoice summary' command
complete -c doctl -n '__fish_seen_subcommand_from invoice summary' -l format    -d "Columns for output"
complete -c doctl -n '__fish_seen_subcommand_from invoice summary' -l no-header -d "Return raw data with no headers"

# Completions for the 'doctl kubernetes' command
complete -c doctl -n '__fish_seen_subcommand_from kubernetes' -a cluster -d "Display commands for managing Kubernetes clusters"
complete -c doctl -n '__fish_seen_subcommand_from kubernetes' -a options -d "List possible option values for use inside Kubernetes commands"

# Completions for the 'doctl kubernetes cluster' command
complete -c doctl -n '__fish_seen_subcommand_from kubernetes cluster' -a create       -d "Create a Kubernetes cluster"
complete -c doctl -n '__fish_seen_subcommand_from kubernetes cluster' -a delete       -d "Delete Kubernetes clusters"
complete -c doctl -n '__fish_seen_subcommand_from kubernetes cluster' -a get          -d "Retrieve details about a Kubernetes cluster"
complete -c doctl -n '__fish_seen_subcommand_from kubernetes cluster' -a get-upgrades -d "Retrieve a list of available Kubernetes version upgrades"
complete -c doctl -n '__fish_seen_subcommand_from kubernetes cluster' -a kubeconfig   -d "Display commands for managing your local kubeconfig"
complete -c doctl -n '__fish_seen_subcommand_from kubernetes cluster' -a list         -d "Retrieve the list of Kubernetes clusters for your account"
complete -c doctl -n '__fish_seen_subcommand_from kubernetes cluster' -a node-pool    -d "Display commands for managing node pools"
complete -c doctl -n '__fish_seen_subcommand_from kubernetes cluster' -a update       -d "Update a Kubernetes cluster's configuration"
complete -c doctl -n '__fish_seen_subcommand_from kubernetes cluster' -a upgrade      -d "Upgrades a cluster to a new Kubernetes version"

# Completions for the 'doctl kubernetes cluster create' command
complete -c doctl -n '__fish_seen_subcommand_from kubernetes cluster create' -l auto-upgrade       -d "Boolean specifying whether to enable auto-upgrade for the cluster"
complete -c doctl -n '__fish_seen_subcommand_from kubernetes cluster create' -l count              -d "Number of nodes in the default node pool"
complete -c doctl -n '__fish_seen_subcommand_from kubernetes cluster create' -l maintenance-window -d "Sets the beginning of the four hour maintenance window"
complete -c doctl -n '__fish_seen_subcommand_from kubernetes cluster create' -l node-pool          -d "Comma-separated list of node pools"
complete -c doctl -n '__fish_seen_subcommand_from kubernetes cluster create' -l size               -d "Machine size to use when creating nodes in the default node pool"
complete -c doctl -n '__fish_seen_subcommand_from kubernetes cluster create' -l tag                -d "Comma-separated list of tags to apply to the cluster"
complete -c doctl -n '__fish_seen_subcommand_from kubernetes cluster create' -l update-kubeconfig  -d "Boolean that specifies whether to add a configuration context"
complete -c doctl -n '__fish_seen_subcommand_from kubernetes cluster create' -l version            -d "Kubernetes version"
complete -c doctl -n '__fish_seen_subcommand_from kubernetes cluster create' -l wait               -d "Boolean that specifies whether to wait for cluster creation"

# Completions for the 'doctl kubernetes cluster delete' command
complete -c doctl -n '__fish_seen_subcommand_from kubernetes cluster delete' -l force              -d "Boolean indicating whether to delete the cluster without a confirmation prompt"
complete -c doctl -n '__fish_seen_subcommand_from kubernetes cluster delete' -l update-kuberconfig -d "Boolean indicating whether to remove the deleted cluster from your kubeconfig"

# Completions for the 'doctl kubernetes cluster get' command
complete -c doctl -n '__fish_seen_subcommand_from kubernetes cluster get' -l format    -d "Columns for output"
complete -c doctl -n '__fish_seen_subcommand_from kubernetes cluster get' -l no-header -d "Return raw data with no headers"

# Completions for the 'doctl kubernetes cluster kubeconfig' command
complete -c doctl -n '__fish_seen_subcommand_from kubernetes cluster kubeconfig' -a remove -d "Remove a cluster's credentials from your local kubeconfig"
complete -c doctl -n '__fish_seen_subcommand_from kubernetes cluster kubeconfig' -a save   -d "Save a cluster's credentials to your local kubeconfig"
complete -c doctl -n '__fish_seen_subcommand_from kubernetes cluster kubeconfig' -a show   -d "Show a Kubernetes cluster's kubeconfig YAML"

# Completions for the 'doctl kubernetes cluster kubeconfig save' command
complete -c doctl -n '__fish_seen_subcommand_from kubernetes cluster kubeconfig save' -a set-current-context  -d "Boolean indicating whether to set the current kubectl context"

# Completions for the 'doctl kubernetes cluster list' command
complete -c doctl -n '__fish_seen_subcommand_from kubernetes cluster list' -l format    -d "Columns for output"
complete -c doctl -n '__fish_seen_subcommand_from kubernetes cluster list' -l no-header -d "Return raw data with no headers"

# Completions for the 'doctl kubernetes cluster node-pool' command
complete -c doctl -n '__fish_seen_subcommand_from kubernetes cluster node-pool' -a create       -d "Create a new node pool for a cluster"
complete -c doctl -n '__fish_seen_subcommand_from kubernetes cluster node-pool' -a delete       -d "Delete a node pool"
complete -c doctl -n '__fish_seen_subcommand_from kubernetes cluster node-pool' -a delete-node  -d "Delete a node"
complete -c doctl -n '__fish_seen_subcommand_from kubernetes cluster node-pool' -a get          -d "Retrieve information about a cluster's node pool"
complete -c doctl -n '__fish_seen_subcommand_from kubernetes cluster node-pool' -a list         -d "List a cluster's node pools"
complete -c doctl -n '__fish_seen_subcommand_from kubernetes cluster node-pool' -a replace-node -d "Replace node with a new one"
complete -c doctl -n '__fish_seen_subcommand_from kubernetes cluster node-pool' -a update       -d "Update an existing node pool in a cluster"

# Completions for the 'doctl kubernetes cluster node-pool create' command
complete -c doctl -n '__fish_seen_subcommand_from kubernetes cluster node-pool create' -l auto-scale -d "Boolean indicating whether to enable auto-scaling on the node pool"
complete -c doctl -n '__fish_seen_subcommand_from kubernetes cluster node-pool create' -rl count     -d "Size of (number of nodes in) the node pool"
complete -c doctl -n '__fish_seen_subcommand_from kubernetes cluster node-pool create' -l label      -d "Label in key=value notation to apply to the node pool"
complete -c doctl -n '__fish_seen_subcommand_from kubernetes cluster node-pool create' -l max-nodes  -d "Maximum number of nodes in the node pool when autoscaling is enabled"
complete -c doctl -n '__fish_seen_subcommand_from kubernetes cluster node-pool create' -l min-nodes  -d "Minimum number of nodes in the node pool when autoscaling is enabled"
complete -c doctl -n '__fish_seen_subcommand_from kubernetes cluster node-pool create' -l name       -d "Name of the node pool"
complete -c doctl -n '__fish_seen_subcommand_from kubernetes cluster node-pool create' -rl size      -d "Size of the nodes in the node pool"
complete -c doctl -n '__fish_seen_subcommand_from kubernetes cluster node-pool create' -l tag        -d "Tag to apply to the node pool"

# Completions for the 'doctl kubernetes cluster node-pool delete' command
complete -c doctl -n '__fish_seen_subcommand_from kubernetes cluster node-pool delete' -l force -d "Delete node pool without confirmation prompt"

# Completions for the 'doctl kubernetes cluster node-pool delete-node' command
complete -c doctl -n '__fish_seen_subcommand_from kubernetes cluster node-pool delete-node' -l force      -d "Delete the node without a confirmation prompt"
complete -c doctl -n '__fish_seen_subcommand_from kubernetes cluster node-pool delete-node' -l skip-drain -d "Skip draining the node before deletion"

# Completions for the 'doctl kubernetes cluster node-pool get' command
complete -c doctl -n '__fish_seen_subcommand_from kubernetes cluster node-pool get' -l format    -d "Columns for output"
complete -c doctl -n '__fish_seen_subcommand_from kubernetes cluster node-pool get' -l no-header -d "Return raw data with no headers"

# Completions for the 'doctl kubernetes cluster node-pool list' command
complete -c doctl -n '__fish_seen_subcommand_from kubernetes cluster node-pool list' -l format    -d "Columns for output"
complete -c doctl -n '__fish_seen_subcommand_from kubernetes cluster node-pool list' -l no-header -d "Return raw data with no headers"

# Completions for the 'doctl kubernetes cluster node-pool replace-node' command
complete -c doctl -n '__fish_seen_subcommand_from kubernetes cluster node-pool replace-node' -l force      -d "Delete the node without a confirmation prompt"
complete -c doctl -n '__fish_seen_subcommand_from kubernetes cluster node-pool replace-node' -l skip-drain -d "Skip draining the node before deletion"

# Completions for the 'doctl kubernetes cluster node-pool update' command
complete -c doctl -n '__fish_seen_subcommand_from kubernetes cluster node-pool update' -l auto-scale -d "Boolean indicating whether to enable auto-scaling on the node pool"
complete -c doctl -n '__fish_seen_subcommand_from kubernetes cluster node-pool update' -l count      -d "Size of (number of nodes in) the node pool"
complete -c doctl -n '__fish_seen_subcommand_from kubernetes cluster node-pool update' -l label      -d "Label in key=value notation to apply to the node pool"
complete -c doctl -n '__fish_seen_subcommand_from kubernetes cluster node-pool update' -l max-nodes  -d "Maximum number of nodes in the node pool when autoscaling is enabled"
complete -c doctl -n '__fish_seen_subcommand_from kubernetes cluster node-pool update' -l min-nodes  -d "Minimum number of nodes in the node pool when autoscaling is enabled"
complete -c doctl -n '__fish_seen_subcommand_from kubernetes cluster node-pool update' -l name       -d "Name of the node pool"
complete -c doctl -n '__fish_seen_subcommand_from kubernetes cluster node-pool update' -l size       -d "Size of the nodes in the node pool"
complete -c doctl -n '__fish_seen_subcommand_from kubernetes cluster node-pool update' -l tag        -d "Tag to apply to the node pool"

# Completions for the 'doctl kubernetes cluster update' command
complete -c doctl -n '__fish_seen_subcommand_from kubernetes cluster update' -l auto-upgrade           -d "Boolean specifying whether to enable auto-upgrade for the cluster"
complete -c doctl -n '__fish_seen_subcommand_from kubernetes cluster update' -l cluster-name           -d "Specifies a new cluster name"
complete -c doctl -n '__fish_seen_subcommand_from kubernetes cluster update' -l maintenance-window     -d "Sets the beginning of the four hour maintenance window for the cluster"
complete -c doctl -n '__fish_seen_subcommand_from kubernetes cluster update' -l set-current-context    -d "Boolean specifying whether to set the current kubectl context"
complete -c doctl -n '__fish_seen_subcommand_from kubernetes cluster update' -l tag                    -d "A comma-separated list of tags to apply to the cluster"
complete -c doctl -n '__fish_seen_subcommand_from kubernetes cluster update' -l update-kubeconfig      -d "Boolean specifying whether to update the cluster"

# Completions for the 'doctl kubernetes cluster upgrade' command
complete -c doctl -n '__fish_seen_subcommand_from kubernetes cluster upgrade' -l version -d "Desired Kubernetes version"

# Completions for the 'doctl kubernetes options' command
complete -c doctl -n '__fish_seen_subcommand_from kubernetes options' -a regions  -d "List regions that support DigitalOcean Kubernetes clusters"
complete -c doctl -n '__fish_seen_subcommand_from kubernetes options' -a sizes    -d "List machine sizes that can be used in a DigitalOcean Kubernetes cluster"
complete -c doctl -n '__fish_seen_subcommand_from kubernetes options' -a versions -d "List Kubernetes versions that can be used with DigitalOcean clusters"

# Completions for the 'doctl projects' command
complete -c doctl -n '__fish_seen_subcommand_from projects' -a create    -d "Create a new project"
complete -c doctl -n '__fish_seen_subcommand_from projects' -a delete    -d "Delete the specified project"
complete -c doctl -n '__fish_seen_subcommand_from projects' -a get       -d "Retrieve details for a specific project"
complete -c doctl -n '__fish_seen_subcommand_from projects' -a list      -d "List existing projects"
complete -c doctl -n '__fish_seen_subcommand_from projects' -a resources -d "Manage resources assigned to a project"
complete -c doctl -n '__fish_seen_subcommand_from projects' -a update    -d "Update an existing project"

# Completions for the 'doctl projects create' command
complete -c doctl -n '__fish_seen_subcommand_from projects create' -l description -d "Dscription of the project"
complete -c doctl -n '__fish_seen_subcommand_from projects create' -l environment -d "Environment in which your project resides."
complete -c doctl -n '__fish_seen_subcommand_from projects create' -l format      -d "Columns for output in a comma-separated list."
complete -c doctl -n '__fish_seen_subcommand_from projects create' -rl name       -d "Name for the project"
complete -c doctl -n '__fish_seen_subcommand_from projects create' -rl no-header  -d "Return raw data with no headers"
complete -c doctl -n '__fish_seen_subcommand_from projects create' -rl purpose    -d "Project's purpose"

# Completions for the 'doctl projects delete' command
complete -c doctl -n '__fish_seen_subcommand_from doctl projects delete' -l force -d "Delete the project without confirmation"

# Completions for the 'doctl projects get' command
complete -c doctl -n '__fish_seen_subcommand_from projects get' -l format    -d "Columns for output"
complete -c doctl -n '__fish_seen_subcommand_from projects get' -l no-header -d "Return raw data with no headers"

# Completions for the 'doctl projects list' command
complete -c doctl -n '__fish_seen_subcommand_from projects list' -l format    -d "Columns for output"
complete -c doctl -n '__fish_seen_subcommand_from projects list' -l no-header -d "Return raw data with no headers"

# Completions for the 'doctl projects resources' command
complete -c doctl -n '__fish_seen_subcommand_from projects resources' -a assign -d "Assign one or more resources to a project"
complete -c doctl -n '__fish_seen_subcommand_from projects resources' -a get    -d "Retrieve a resource by its URN"
complete -c doctl -n '__fish_seen_subcommand_from projects resources' -a list   -d "List resources assigned to a project"

# Completions for the 'doctl projects resources assign' command
complete -c doctl -n '__fish_seen_subcommand_from projects resources assign' -l resource -d "URNs specifying resources to assign to the project"

# Completions for the 'doctl projects resources get' command
complete -c doctl -n '__fish_seen_subcommand_from projects resources get' -l format    -d "Columns for output"
complete -c doctl -n '__fish_seen_subcommand_from projects resources get' -l no-header -d "Return raw data with no headers"

# Completions for the 'doctl projects resources list' command
complete -c doctl -n '__fish_seen_subcommand_from projects resources list' -l format    -d "Columns for output"
complete -c doctl -n '__fish_seen_subcommand_from projects resources list' -l no-header -d "Return raw data with no headers"

# Completions for the 'doctl projects update' command
complete -c doctl -n '__fish_seen_subcommand_from projects update' -l description -d "Description of the project"
complete -c doctl -n '__fish_seen_subcommand_from projects update' -l environment -d "Environment in which your project resides."
complete -c doctl -n '__fish_seen_subcommand_from projects update' -l format      -d "Columns for output in a comma-separated list."
complete -c doctl -n '__fish_seen_subcommand_from projects update' -rl name       -d "Name for the project"
complete -c doctl -n '__fish_seen_subcommand_from projects update' -l no-header   -d "Return raw data with no headers"
complete -c doctl -n '__fish_seen_subcommand_from projects update' -rl purpose    -d "Project's purpose"
`
	_, err := buf.Write([]byte(fishCompletion))
	if err != nil {
		return fmt.Errorf("Error while generating fish completion: %v", err)
	}

	code := buf.String()
	fmt.Print(code)
	return nil
}

// RunCompletionZsh outputs completion code for zsh shell.
func RunCompletionZsh(c *CmdConfig) error {
	var buf bytes.Buffer

	// zshHead is the header required to declare zsh completion
	zshHead := "#compdef doctl\n"

	// zshInit represents intialization code needed to convert bash completion
	// code to zsh completion.
	zshInit := `
__doctl_bash_source() {
	alias shopt=':'
	alias _expand=_bash_expand
	alias _complete=_bash_comp
	emulate -L sh
	setopt kshglob noshglob braceexpand

	source "$@"
}

__doctl_type() {
	# -t is not supported by zsh
	if [ "$1" == "-t" ]; then
		shift

		# fake Bash 4 to disable "complete -o nospace". Instead
		# "compopt +-o nospace" is used in the code to toggle trailing
		# spaces. We don't support that, but leave trailing spaces on
		# all the time
		if [ "$1" = "__doctl_compopt" ]; then
			echo builtin
			return 0
		fi
	fi
	type "$@"
}

__doctl_compgen() {
	local completions w
	completions=( $(compgen "$@") ) || return $?

	# filter by given word as prefix
	while [[ "$1" = -* && "$1" != -- ]]; do
		shift
		shift
	done
	if [[ "$1" == -- ]]; then
		shift
	fi
	for w in "${completions[@]}"; do
		if [[ "${w}" = "$1"* ]]; then
			echo "${w}"
		fi
	done
}

__doctl_compopt() {
	true # don't do anything. Not supported by bashcompinit in zsh
}

__doctl_ltrim_colon_completions()
{
	if [[ "$1" == *:* && "$COMP_WORDBREAKS" == *:* ]]; then
		# Remove colon-word prefix from COMPREPLY items
		local colon_word=${1%${1##*:}}
		local i=${#COMPREPLY[*]}
		while [[ $((--i)) -ge 0 ]]; do
			COMPREPLY[$i]=${COMPREPLY[$i]#"$colon_word"}
		done
	fi
}

__doctl_get_comp_words_by_ref() {
	cur="${COMP_WORDS[COMP_CWORD]}"
	prev="${COMP_WORDS[${COMP_CWORD}-1]}"
	words=("${COMP_WORDS[@]}")
	cword=("${COMP_CWORD[@]}")
}

__doctl_filedir() {
	local RET OLD_IFS w qw

	__debug "_filedir $@ cur=$cur"
	if [[ "$1" = \~* ]]; then
		# somehow does not work. Maybe, zsh does not call this at all
		eval echo "$1"
		return 0
	fi

	OLD_IFS="$IFS"
	IFS=$'\n'
	if [ "$1" = "-d" ]; then
		shift
		RET=( $(compgen -d) )
	else
		RET=( $(compgen -f) )
	fi
	IFS="$OLD_IFS"

	IFS="," __debug "RET=${RET[@]} len=${#RET[@]}"

	for w in ${RET[@]}; do
		if [[ ! "${w}" = "${cur}"* ]]; then
			continue
		fi
		if eval "[[ \"\${w}\" = *.$1 || -d \"\${w}\" ]]"; then
			qw="$(__doctl_quote "${w}")"
			if [ -d "${w}" ]; then
				COMPREPLY+=("${qw}/")
			else
				COMPREPLY+=("${qw}")
			fi
		fi
	done
}

__doctl_quote() {
    if [[ $1 == \'* || $1 == \"* ]]; then
        # Leave out first character
        printf %q "${1:1}"
    else
    	printf %q "$1"
    fi
}

autoload -U +X bashcompinit && bashcompinit

# use word boundary patterns for BSD or GNU sed
LWORD='[[:<:]]'
RWORD='[[:>:]]'
SEDHELP=$(sed --help 2>&1)
if echo ${SEDHELP} | grep -q GNU || echo ${SEDHELP} | grep -q BusyBox; then
	LWORD='\<'
	RWORD='\>'
fi

__doctl_convert_bash_to_zsh() {
	sed \
	-e 's/declare -F/whence -w/' \
	-e 's/_get_comp_words_by_ref "\$@"/_get_comp_words_by_ref "\$*"/' \
	-e 's/local \([a-zA-Z0-9_]*\)=/local \1; \1=/' \
	-e 's/flags+=("\(--.*\)=")/flags+=("\1"); two_word_flags+=("\1")/' \
	-e 's/must_have_one_flag+=("\(--.*\)=")/must_have_one_flag+=("\1")/' \
	-e "s/${LWORD}_filedir${RWORD}/__doctl_filedir/g" \
	-e "s/${LWORD}_get_comp_words_by_ref${RWORD}/__doctl_get_comp_words_by_ref/g" \
	-e "s/${LWORD}__ltrim_colon_completions${RWORD}/__doctl_ltrim_colon_completions/g" \
	-e "s/${LWORD}compgen${RWORD}/__doctl_compgen/g" \
	-e "s/${LWORD}compopt${RWORD}/__doctl_compopt/g" \
	-e "s/${LWORD}declare${RWORD}/builtin declare/g" \
	-e 's/aliashash\["\(.\{1,\}\)"\]/aliashash[\1]/g' \
	-e "s/\\\$(type${RWORD}/\$(__doctl_type/g" \
	<<'BASH_COMPLETION_EOF'
`

	// zshFinalize is code going to the end of completion file
	// that calls conversion bash to zsh.
	zshFinalize := `
BASH_COMPLETION_EOF
}

__doctl_bash_source <(__doctl_convert_bash_to_zsh)
_complete doctl 2>/dev/null
`

	_, err := buf.Write([]byte(zshHead))
	if err != nil {
		return fmt.Errorf("Error while generating zsh completion: %v", err)
	}

	_, err = buf.Write([]byte(doctlLicense))
	if err != nil {
		return fmt.Errorf("Error while generating zsh completion: %v", err)
	}

	_, err = buf.Write([]byte(zshInit))
	if err != nil {
		return fmt.Errorf("Error while generating zsh completion: %v", err)
	}

	err = DoitCmd.GenBashCompletion(&buf)
	if err != nil {
		return fmt.Errorf("error wheil generating zsh completion: %v", err)
	}

	_, err = buf.Write([]byte(zshFinalize))
	if err != nil {
		return fmt.Errorf("Error while generating zsh completion: %v", err)
	}

	// remove the command "completion" from auto-completion
	code := buf.String()
	code = strings.Replace(code, `commands+=("completion")`, "", -1)

	fmt.Print(code)
	return nil
}
