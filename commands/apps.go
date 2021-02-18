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
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"
)

// Apps creates the apps command.
func Apps() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "apps",
			Aliases: []string{"app", "a"},
			Short:   "Display commands for working with apps",
			Long:    "The subcommands of `doctl app` manage your App Platform apps.",
		},
	}

	create := CmdBuilder(
		cmd,
		RunAppsCreate,
		"create",
		"Create an app",
		`Create an app with the given app spec.`,
		Writer,
		aliasOpt("c"),
		displayerType(&displayers.Apps{}),
	)
	AddStringFlag(create, doctl.ArgAppSpec, "", "", "Path to an app spec in JSON or YAML format. For more information about app specs, see https://www.digitalocean.com/docs/app-platform/concepts/app-spec", requiredOpt())

	CmdBuilder(
		cmd,
		RunAppsGet,
		"get <app id>",
		"Get an app",
		`Get an app with the provided id.

Only basic information is included with the text output format. For complete app details including its app spec, use the JSON format.`,
		Writer,
		aliasOpt("g"),
		displayerType(&displayers.Apps{}),
	)

	CmdBuilder(
		cmd,
		RunAppsList,
		"list",
		"List all apps",
		`List all apps.

Only basic information is included with the text output format. For complete app details including the app specs, use the JSON format.`,
		Writer,
		aliasOpt("ls"),
		displayerType(&displayers.Apps{}),
	)

	update := CmdBuilder(
		cmd,
		RunAppsUpdate,
		"update <app id>",
		"Update an app",
		`Update the specified app with the given app spec. For more information about app specs, see https://www.digitalocean.com/docs/app-platform/concepts/app-spec`,
		Writer,
		aliasOpt("u"),
		displayerType(&displayers.Apps{}),
	)
	AddStringFlag(update, doctl.ArgAppSpec, "", "", "Path to an app spec in JSON or YAML format.", requiredOpt())

	deleteApp := CmdBuilder(
		cmd,
		RunAppsDelete,
		"delete <app id>",
		"Deletes an app",
		`Deletes an app with the provided id.

This permanently deletes the app and all its associated deployments.`,
		Writer,
		aliasOpt("d"),
	)
	AddBoolFlag(deleteApp, doctl.ArgForce, doctl.ArgShortForce, false, "Delete the App without a confirmation prompt")

	deploymentCreate := CmdBuilder(
		cmd,
		RunAppsCreateDeployment,
		"create-deployment <app id>",
		"Create a deployment",
		`Create a deployment for an app.

Creating an app deployment will pull the latest changes from your repository and schedule a new deployment for your app.`,
		Writer,
		aliasOpt("cd"),
		displayerType(&displayers.Deployments{}),
	)
	AddBoolFlag(deploymentCreate, doctl.ArgAppForceRebuild, "", false, "Force a re-build even if a previous build is eligible for reuse")

	CmdBuilder(
		cmd,
		RunAppsGetDeployment,
		"get-deployment <app id> <deployment id>",
		"Get a deployment",
		`Get a deployment for an app.

Only basic information is included with the text output format. For complete app details including its app specs, use the JSON format.`,
		Writer,
		aliasOpt("gd"),
		displayerType(&displayers.Deployments{}),
	)

	CmdBuilder(
		cmd,
		RunAppsListDeployments,
		"list-deployments <app id>",
		"List all deployments",
		`List all deployments for an app.

Only basic information is included with the text output format. For complete app details including the app specs, use the JSON format.`,
		Writer,
		aliasOpt("lsd"),
		displayerType(&displayers.Deployments{}),
	)

	logs := CmdBuilder(
		cmd,
		RunAppsGetLogs,
		"logs <app id> <component name (defaults to all components)>",
		"Get logs",
		`Get component logs for a deployment of an app.

Three types of logs are supported and can be configured with --`+doctl.ArgAppLogType+`:
- build
- deploy
- run `,
		Writer,
		aliasOpt("l"),
	)
	AddStringFlag(logs, doctl.ArgAppDeployment, "", "", "The deployment ID. Defaults to current deployment.")
	AddStringFlag(logs, doctl.ArgAppLogType, "", strings.ToLower(string(godo.AppLogTypeRun)), "The type of logs.")
	AddBoolFlag(logs, doctl.ArgAppLogFollow, "f", false, "Follow logs as they are emitted.")

	CmdBuilder(
		cmd,
		RunAppsListRegions,
		"list-regions",
		"List App Platform regions",
		`List all regions supported by App Platform including details about their current availability.`,
		Writer,
		displayerType(&displayers.AppRegions{}),
	)

	cmd.AddCommand(appsSpec())
	cmd.AddCommand(appsTier())

	return cmd
}

// RunAppsCreate creates an app.
func RunAppsCreate(c *CmdConfig) error {
	specPath, err := c.Doit.GetString(c.NS, doctl.ArgAppSpec)
	if err != nil {
		return err
	}

	specFile, err := os.Open(specPath) // guardrails-disable-line
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("Failed to open app spec: %s does not exist", specPath)
		}
		return fmt.Errorf("Failed to open app spec: %w", err)
	}
	defer specFile.Close()

	specBytes, err := ioutil.ReadAll(specFile)
	if err != nil {
		return fmt.Errorf("Failed to read app spec: %w", err)
	}

	appSpec, err := parseAppSpec(specBytes)
	if err != nil {
		return err
	}

	app, err := c.Apps().Create(&godo.AppCreateRequest{Spec: appSpec})
	if err != nil {
		return err
	}
	notice("App created")

	return c.Display(displayers.Apps{app})
}

// RunAppsGet gets an app.
func RunAppsGet(c *CmdConfig) error {
	if len(c.Args) < 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	id := c.Args[0]

	app, err := c.Apps().Get(id)
	if err != nil {
		return err
	}

	return c.Display(displayers.Apps{app})
}

// RunAppsList lists all apps.
func RunAppsList(c *CmdConfig) error {
	apps, err := c.Apps().List()
	if err != nil {
		return err
	}

	return c.Display(displayers.Apps(apps))
}

// RunAppsUpdate updates an app.
func RunAppsUpdate(c *CmdConfig) error {
	if len(c.Args) < 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	id := c.Args[0]

	specPath, err := c.Doit.GetString(c.NS, doctl.ArgAppSpec)
	if err != nil {
		return err
	}

	specFile, err := os.Open(specPath) // guardrails-disable-line
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("Failed to open app spec: %s does not exist", specPath)
		}
		return fmt.Errorf("Failed to open app spec: %w", err)
	}
	defer specFile.Close()

	specBytes, err := ioutil.ReadAll(specFile)
	if err != nil {
		return fmt.Errorf("Failed to read app spec: %w", err)
	}

	appSpec, err := parseAppSpec(specBytes)
	if err != nil {
		return err
	}

	app, err := c.Apps().Update(id, &godo.AppUpdateRequest{Spec: appSpec})
	if err != nil {
		return err
	}
	notice("App updated")

	return c.Display(displayers.Apps{app})
}

// RunAppsDelete deletes an app.
func RunAppsDelete(c *CmdConfig) error {
	if len(c.Args) < 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	id := c.Args[0]

	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}

	if !force && AskForConfirmDelete("App", 1) != nil {
		return fmt.Errorf("Operation aborted.")
	}

	err = c.Apps().Delete(id)
	if err != nil {
		return err
	}
	notice("App deleted")

	return nil
}

// RunAppsCreateDeployment creates a deployment for an app.
func RunAppsCreateDeployment(c *CmdConfig) error {
	if len(c.Args) < 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	appID := c.Args[0]
	forceRebuild, err := c.Doit.GetBool(c.NS, doctl.ArgAppForceRebuild)
	if err != nil {
		return err
	}

	deployment, err := c.Apps().CreateDeployment(appID, forceRebuild)
	if err != nil {
		return err
	}
	notice("Deployment created")

	return c.Display(displayers.Deployments{deployment})
}

// RunAppsGetDeployment gets a deployment for an app.
func RunAppsGetDeployment(c *CmdConfig) error {
	if len(c.Args) < 2 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	appID := c.Args[0]
	deploymentID := c.Args[1]

	deployment, err := c.Apps().GetDeployment(appID, deploymentID)
	if err != nil {
		return err
	}

	return c.Display(displayers.Deployments{deployment})
}

// RunAppsListDeployments lists deployments for an app.
func RunAppsListDeployments(c *CmdConfig) error {
	if len(c.Args) < 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	appID := c.Args[0]

	deployments, err := c.Apps().ListDeployments(appID)
	if err != nil {
		return err
	}

	return c.Display(displayers.Deployments(deployments))
}

// RunAppsGetLogs gets app logs for a given component.
func RunAppsGetLogs(c *CmdConfig) error {
	if len(c.Args) < 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	appID := c.Args[0]
	var component string
	if len(c.Args) >= 2 {
		component = c.Args[1]
	}

	deploymentID, err := c.Doit.GetString(c.NS, doctl.ArgAppDeployment)
	if err != nil {
		return err
	}
	if deploymentID == "" {
		app, err := c.Apps().Get(appID)
		if err != nil {
			return err
		}
		if app.ActiveDeployment != nil {
			deploymentID = app.ActiveDeployment.ID
		} else if app.InProgressDeployment != nil {
			deploymentID = app.InProgressDeployment.ID
		} else {
			return fmt.Errorf("unable to retrieve logs; no deployment found for app %s", appID)
		}
	}

	logTypeStr, err := c.Doit.GetString(c.NS, doctl.ArgAppLogType)
	if err != nil {
		return err
	}
	var logType godo.AppLogType
	switch logTypeStr {
	case strings.ToLower(string(godo.AppLogTypeBuild)):
		logType = godo.AppLogTypeBuild
	case strings.ToLower(string(godo.AppLogTypeDeploy)):
		logType = godo.AppLogTypeDeploy
	case strings.ToLower(string(godo.AppLogTypeRun)):
		logType = godo.AppLogTypeRun
	default:
		return fmt.Errorf("Invalid log type %s", logTypeStr)
	}
	logFollow, err := c.Doit.GetBool(c.NS, doctl.ArgAppLogFollow)
	if err != nil {
		return err
	}

	logs, err := c.Apps().GetLogs(appID, deploymentID, component, logType, logFollow)
	if err != nil {
		return err
	}

	if logs.LiveURL != "" {
		resp, err := http.Get(logs.LiveURL)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		io.Copy(c.Out, resp.Body)
	} else if len(logs.HistoricURLs) > 0 {
		resp, err := http.Get(logs.HistoricURLs[0])
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		io.Copy(c.Out, resp.Body)
	} else {
		warn("No logs found for app component")
	}

	return nil
}

func parseAppSpec(spec []byte) (*godo.AppSpec, error) {
	jsonSpec, err := yaml.YAMLToJSON(spec)
	if err != nil {
		return nil, err
	}

	dec := json.NewDecoder(bytes.NewReader(jsonSpec))
	dec.DisallowUnknownFields()

	var appSpec godo.AppSpec
	if err := dec.Decode(&appSpec); err != nil {
		return nil, fmt.Errorf("Failed to parse app spec: %v", err)
	}

	return &appSpec, nil
}

func appsSpec() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "spec",
			Short: "Display commands for working with app specs",
			Long:  "The subcommands of `doctl app spec` manage your app specs.",
		},
	}

	getCmd := CmdBuilder(cmd, RunAppsSpecGet, "get <app id>", "Retrieve an application's spec", `Use this command to retrieve the latest spec of an app.

Optionally, pass a deployment ID to get the spec of that specific deployment.`, Writer)
	AddStringFlag(getCmd, doctl.ArgAppDeployment, "", "", "optional: a deployment ID")
	AddStringFlag(getCmd, doctl.ArgFormat, "", "yaml", `the format to output the spec as; either "yaml" or "json"`)

	CmdBuilder(cmd, RunAppsSpecValidate(os.Stdin), "validate <spec file>", "Validate an application spec", `Use this command to check whether a given app spec (YAML or JSON) is valid.

You may pass - as the filename to read from stdin.`, Writer)

	return cmd
}

// RunAppsSpecGet gets the spec for an app
func RunAppsSpecGet(c *CmdConfig) error {
	if len(c.Args) < 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	appID := c.Args[0]
	deploymentID, err := c.Doit.GetString(c.NS, doctl.ArgAppDeployment)
	if err != nil {
		return err
	}

	format, err := c.Doit.GetString(c.NS, doctl.ArgFormat)
	if err != nil {
		return err
	}

	var spec *godo.AppSpec
	if deploymentID == "" {
		app, err := c.Apps().Get(appID)
		if err != nil {
			return err
		}
		spec = app.Spec
	} else {
		deployment, err := c.Apps().GetDeployment(appID, deploymentID)
		if err != nil {
			return err
		}
		spec = deployment.Spec
	}

	switch format {
	case "json":
		e := json.NewEncoder(c.Out)
		e.SetIndent("", "  ")
		return e.Encode(spec)
	case "yaml":
		yaml, err := yaml.Marshal(spec)
		if err != nil {
			return fmt.Errorf("marshaling the spec as yaml: %v", err)
		}
		_, err = c.Out.Write(yaml)
		return err
	default:
		return fmt.Errorf("invalid spec format %q, must be one of: json, yaml", format)
	}
}

// RunAppsSpecValidate validates an app spec file
func RunAppsSpecValidate(stdin io.Reader) func(c *CmdConfig) error {
	return func(c *CmdConfig) error {
		if len(c.Args) < 1 {
			return doctl.NewMissingArgsErr(c.NS)
		}

		specPath := c.Args[0]
		var spec io.Reader
		if specPath == "-" {
			spec = stdin
		} else {
			specFile, err := os.Open(specPath) // guardrails-disable-line
			if err != nil {
				if os.IsNotExist(err) {
					return fmt.Errorf("Failed to open app spec: %s does not exist", specPath)
				}
				return fmt.Errorf("Failed to open app spec: %w", err)
			}
			defer specFile.Close()
			spec = specFile
		}

		specBytes, err := ioutil.ReadAll(spec)
		if err != nil {
			return fmt.Errorf("Failed to read app spec: %w", err)
		}

		_, err = parseAppSpec(specBytes)
		if err != nil {
			return err
		}

		c.Out.Write([]byte("The spec is valid.\n"))
		return nil
	}
}

// RunAppsListRegions lists all app platform regions.
func RunAppsListRegions(c *CmdConfig) error {
	regions, err := c.Apps().ListRegions()
	if err != nil {
		return err
	}

	return c.Display(displayers.AppRegions(regions))
}

func appsTier() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "tier",
			Short: "Display commands for working with app tiers",
			Long:  "The subcommands of `doctl app tier` retrieve information about app tiers.",
		},
	}

	CmdBuilder(cmd, RunAppsTierList, "list", "List all app tiers", `Use this command to list all the available app tiers.`, Writer)
	CmdBuilder(cmd, RunAppsTierGet, "get <tier slug>", "Retrieve an app tier", `Use this command to retrieve information about a specific app tier.`, Writer)

	cmd.AddCommand(appsTierInstanceSize())

	return cmd
}

// RunAppsTierList lists all app tiers.
func RunAppsTierList(c *CmdConfig) error {
	tiers, err := c.Apps().ListTiers()
	if err != nil {
		return err
	}

	return c.Display(displayers.AppTiers(tiers))
}

// RunAppsTierGet gets an app tier.
func RunAppsTierGet(c *CmdConfig) error {
	if len(c.Args) < 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	slug := c.Args[0]

	tier, err := c.Apps().GetTier(slug)
	if err != nil {
		return err
	}

	return c.Display(displayers.AppTiers([]*godo.AppTier{tier}))
}

func appsTierInstanceSize() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "instance-size",
			Short: "Display commands for working with app instance sizes",
			Long:  "The subcommands of `doctl app tier instance-size` retrieve information about app instance sizes.",
		},
	}

	CmdBuilder(cmd, RunAppsTierInstanceSizeList, "list", "List all app instance sizes", `Use this command to list all the available app instance sizes.`, Writer)
	CmdBuilder(cmd, RunAppsTierInstanceSizeGet, "get <instance size slug>", "Retrieve an app instance size", `Use this command to retrieve information about a specific app instance size.`, Writer)

	return cmd
}

// RunAppsTierInstanceSizeList lists all app tiers.
func RunAppsTierInstanceSizeList(c *CmdConfig) error {
	instanceSizes, err := c.Apps().ListInstanceSizes()
	if err != nil {
		return err
	}

	return c.Display(displayers.AppInstanceSizes(instanceSizes))
}

// RunAppsTierInstanceSizeGet gets an app tier.
func RunAppsTierInstanceSizeGet(c *CmdConfig) error {
	if len(c.Args) < 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	slug := c.Args[0]

	instanceSize, err := c.Apps().GetInstanceSize(slug)
	if err != nil {
		return err
	}

	return c.Display(displayers.AppInstanceSizes([]*godo.AppInstanceSize{instanceSize}))
}
