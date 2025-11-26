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
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/doctl/internal/apps"
	"github.com/digitalocean/doctl/pkg/terminal"
	"github.com/digitalocean/godo"
	"github.com/google/uuid"
	multierror "github.com/hashicorp/go-multierror"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
	"sigs.k8s.io/yaml"
)

// Apps creates the apps command.
func Apps() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "apps",
			Aliases: []string{"app", "a"},
			Short:   "Displays commands for working with apps",
			Long:    "The subcommands of `doctl app` manage your App Platform apps. For documentation on app specs, see the [app spec reference](https://www.digitalocean.com/docs/app-platform/concepts/app-spec).",
			GroupID: manageResourcesGroup,
		},
	}

	cmd.AddCommand(AppsDev())

	create := CmdBuilder(
		cmd,
		RunAppsCreate,
		"create",
		"Create an app",
		`Create an app with the given app spec. To create an app, App Platform must have access to your repository service. Click one of the following links to provide access for your preferred service: [GitHub](https://cloud.digitalocean.com/apps/gitlab/install), [GitLab](https://cloud.digitalocean.com/apps/gitlab/install), [BitBucket](https://cloud.digitalocean.com/apps/bitbucket/install)`,
		Writer,
		aliasOpt("c"),
		displayerType(&displayers.Apps{}),
	)
	AddStringFlag(create, doctl.ArgAppSpec, "", "", `Path to an app spec in JSON or YAML format. Set to "-" to read from stdin.`, requiredOpt())
	AddBoolFlag(create, doctl.ArgCommandWait, "", false,
		"Boolean that specifies whether to wait for an app to complete before returning control to the terminal")
	AddBoolFlag(create, doctl.ArgCommandUpsert, "", false, `A boolean value that creates or updates an app’s configuration with the attached app spec. This does not pull changes from the app’s container registry or source repository. Instead, App Platform uses the image from the app’s most recent deployment. To additionally pull the latest changes from the app’s source, set the `+"`"+`--update-sources`+"`"+` flag.`)
	AddBoolFlag(create, doctl.ArgCommandUpdateSources, "", false, "Boolean that specifies whether, on update, the app should also update its source code")
	AddStringFlag(create, doctl.ArgProjectID, "", "", "The ID of the project to assign the created app and resources to. If not provided, the default project will be used.")
	create.Example = `The following example creates an app in a project named ` + "`" + `example-project` + "`" + ` using an app spec located in a directory called ` + "`" + `/src/your-app.yaml` + "`" + `. Additionally, the command returns the new app's ID, ingress information, and creation date: doctl apps create --spec src/your-app.yaml --format ID,DefaultIngress,Created`

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

	list := CmdBuilder(
		cmd,
		RunAppsList,
		"list",
		"Lists all apps",
		`Lists all apps associated with your account, including their ID, spec name, creation date, and other information.

Only basic information is included with the text output format. For complete app details including an updated app spec, use the `+"`"+`--output`+"`"+` global flag and specify the JSON format.`,
		Writer,
		aliasOpt("ls"),
		displayerType(&displayers.Apps{}),
	)
	AddBoolFlag(list, doctl.ArgAppWithProjects, "", false, "Boolean that specifies whether project ids should be fetched along with listed apps")
	list.Example = `The following lists all apps in your account, but returns just their ID and creation date: doctl apps list --format ID,Created`

	update := CmdBuilder(
		cmd,
		RunAppsUpdate,
		"update <app id>",
		"Updates an app",
		`Updates an existing app with the attached app spec. By default, this does not retrieve the latest image from the app’s container registry or changes source repository. To deploy an app with changes from its source repository and app spec configuration, use the `+"`"+`--update-sources`+"`"+` flag. For more information about app specs, see the [app spec reference](https://www.digitalocean.com/docs/app-platform/concepts/app-spec)`,
		Writer,
		aliasOpt("u"),
		displayerType(&displayers.Apps{}),
	)
	AddStringFlag(update, doctl.ArgAppSpec, "", "", `Path to an app spec in JSON or YAML format. Set to "-" to read from stdin.`, requiredOpt())
	AddBoolFlag(update, doctl.ArgCommandUpdateSources, "", false, "Boolean that specifies whether the app should also update its source code")
	AddBoolFlag(update, doctl.ArgCommandWait, "", false,
		"Boolean that specifies whether to wait for an app to complete updating before allowing further terminal input. This can be helpful for scripting.")
	update.Example = `The following example updates an app with the ID ` + "`" + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + "`" + ` using an app spec located in a directory called ` + "`" + `/src/your-app.yaml` + "`" + `. Additionally, the command returns the updated app's ID, ingress information, and creation date: doctl apps update f81d4fae-7dec-11d0-a765-00a0c91e6bf6 --spec src/your-app.yaml --format ID,DefaultIngress,Created`

	deleteApp := CmdBuilder(
		cmd,
		RunAppsDelete,
		"delete <app id>",
		"Deletes an app",
		`Deletes the specified app.

This permanently deletes the app and all of its associated deployments.`,
		Writer,
		aliasOpt("d", "rm"),
	)
	AddBoolFlag(deleteApp, doctl.ArgForce, doctl.ArgShortForce, false, "Delete the App without a confirmation prompt")
	deleteApp.Example = `The following example deletes an app with the ID ` + "`" + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + "`" + `: doctl apps delete f81d4fae-7dec-11d0-a765-00a0c91e6bf6`

	restartApp := CmdBuilder(
		cmd,
		RunAppsRestart,
		"restart <app id>",
		"Restarts an app",
		`Restarts the specified app or some of its components.`,
		Writer,
		aliasOpt("r"),
		displayerType(&displayers.Deployments{}),
	)
	AddStringSliceFlag(restartApp, doctl.ArgAppComponents, "", nil, "The components to restart. If not provided, all components are restarted.")
	AddBoolFlag(restartApp, doctl.ArgCommandWait, "", false,
		"Boolean that specifies whether to wait for the restart to complete before allowing further terminal input. This can be helpful for scripting.")
	restartApp.Example = `The following example restarts an app with the ID ` + "`" + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + "`" + `. Additionally, the command returns the app's ID and status: doctl apps restart f81d4fae-7dec-11d0-a765-00a0c91e6bf6 --format ID,Status`

	deploymentCreate := CmdBuilder(
		cmd,
		RunAppsCreateDeployment,
		"create-deployment <app id>",
		"Creates a deployment",
		`Creates an app using the provided app spec. To redeploy an existing app using its latest image or source code changes, use the --update-sources flag. To update an existing app’s spec configuration without pulling its latest changes or image, use the `+"`"+`--upsert`+"`"+` flag or `+"`"+`doctl apps update`+"`"+` command.
`,
		Writer,
		aliasOpt("cd"),
		displayerType(&displayers.Deployments{}),
	)
	AddBoolFlag(deploymentCreate, doctl.ArgAppForceRebuild, "", false, "Force a re-build even if a previous build is eligible for reuse.")
	AddBoolFlag(deploymentCreate, doctl.ArgCommandWait, "", false,
		"Boolean that specifies whether to wait for the deployment to complete before allowing further terminal input. This can be helpful for scripting.")
	deploymentCreate.Example = `The following example creates a deployment for an app with the ID ` + "`" + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + "`" + `. Additionally, the command returns the app's ID and status: doctl apps create-deployment f81d4fae-7dec-11d0-a765-00a0c91e6bf6 --format ID,Status`

	getDeployment := CmdBuilder(
		cmd,
		RunAppsGetDeployment,
		"get-deployment <app id> <deployment id>",
		"Get a deployment",
		`Gets information about a specific deployment for the given app, including when the app updated and what triggered the deployment (Cause).

Only basic information is included with the text output format. For complete app details including an updated app spec, use the `+"`"+`--output`+"`"+` global flag and specify the JSON format.`,
		Writer,
		aliasOpt("gd"),
		displayerType(&displayers.Deployments{}),
	)
	getDeployment.Example = `The following example gets information about a deployment with the ID ` + "`" + `418b7972-fc67-41ea-ab4b-6f9477c4f7d8` + "`" + ` for an app with the ID ` + "`" + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + "`" + `. Additionally, the command returns the deployment's ID, status, and cause: doctl apps get-deployment f81d4fae-7dec-11d0-a765-00a0c91e6bf6 418b7972-fc67-41ea-ab4b-6f9477c4f7d8 --format ID,Status,Cause`

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

	listJobInvocations := CmdBuilder(
		cmd,
		RunAppsListJobInvocations,
		"list-job-invocations <app id>",
		"List all job invocations",
		`List all job invocations for an app.

Only basic information is included with the text output format. For complete app details including the app specs, use the JSON format.`,
		Writer,
		aliasOpt("lsji"),
		displayerType(&displayers.JobInvocations{}),
	)

	AddStringSliceFlag(listJobInvocations, doctl.ArgAppDeployment, "", []string{}, "The deployment ID to filter job invocations for. If not provided, all job invocations for given app are returned.")
	AddStringSliceFlag(listJobInvocations, doctl.ArgAppJobName, "", []string{}, "The job name to filter job invocations for. If not provided, all job invocations for given app are returned.")

	listJobInvocations.Example = `The following example retrieves the list of job invocations for the app with the ID ` + "`" + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + "`" + ` and the deployment ID ` + "`" + `418b7972-fc67-41ea-ab4b-6f9477c4f7d8` + "`" + ` and the job ` + "`" + `cron` + "`" + `: doctl apps list-job-invocations f81d4fae-7dec-11d0-a765-00a0c91e6bf6 --job-name cron --deployment 418b7972-fc67-41ea-ab4b-6f9477c4f7d8`

	CmdBuilder(
		cmd,
		RunAppsGetJobInvocation,
		"get-job-invocation <app id> <job invocation id>",
		"Get a job invocation",
		`Gets information about a specific job invocation for the given app, including when the job was created.

Only basic information is included with the text output format. For complete app details including an updated app spec, use the `+"`"+`--output`+"`"+` global flag and specify the JSON format.`,
		Writer,
		aliasOpt("gji"),
		displayerType(&displayers.JobInvocations{}),
	)

	CmdBuilder(
		cmd,
		RunAppsCancelJobInvocation,
		"cancel-job-invocation <app id> <job invocation id>",
		"Cancel a job invocation",
		`Cancels a specific job invocation for the given app.

Only basic information is included with the text output format. For complete app details including an updated app spec, use the `+"`"+`--output`+"`"+` global flag and specify the JSON format.`,
		Writer,
		aliasOpt("cji"),
		displayerType(&displayers.JobInvocations{}),
	)

	logs := CmdBuilder(
		cmd,
		RunAppsGetLogs,
		"logs <app name or id> <component name (defaults to all components)>",
		"Retrieves logs",
		`Retrieves component logs for a deployment or a job invocation of an app.

Three types of logs are supported and can be specified with the --`+doctl.ArgAppLogType+` flag:
- build
- deploy
- run
- run_restarted

To retrieve job invocation logs, pass the job invocation ID with the --`+doctl.ArgAppJobInvocation+` flag.

For more information about logs, see [How to View Logs](https://www.digitalocean.com/docs/app-platform/how-to/view-logs/).
`,
		Writer,
		aliasOpt("l"),
	)
	AddStringFlag(logs, doctl.ArgAppDeployment, "", "", "Retrieves logs for a specific deployment ID. Defaults to current deployment.")
	AddStringFlag(logs, doctl.ArgAppJobInvocation, "", "", "Retrieves logs for a specific job invocation ID.")
	AddStringFlag(logs, doctl.ArgAppLogType, "", strings.ToLower(string(godo.AppLogTypeRun)), "Retrieves logs for a specific log type. Defaults to run logs.")
	AddBoolFlag(logs, doctl.ArgAppLogFollow, "f", false, "Returns logs as they are emitted by the app.")
	AddIntFlag(logs, doctl.ArgAppLogTail, "", -1, "Specifies the number of lines to show from the end of the log.")
	AddBoolFlag(logs, doctl.ArgNoPrefix, "", false, "Removes the prefix from logs. Useful for JSON structured logs")

	logs.Example = `The following example retrieves the build logs for the app with the ID ` + "`" + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + "`" + ` and the component ` + "`" + `web` + "`" + `: doctl apps logs f81d4fae-7dec-11d0-a765-00a0c91e6bf6 web --type build`

	console := CmdBuilder(
		cmd,
		RunAppsConsole,
		"console <app id> <component name> <instance name (optional)>",
		"Starts a console session",
		`Instantiates a console session for a component of an app. Note: avoid creating scripts or making changes that need to persist on these instances, as they are ephemeral and may be terminated at any time`,
		Writer,
		aliasOpt("cs"),
	)
	AddStringFlag(console, doctl.ArgAppDeployment, "", "", "Starts a console session for a specific deployment ID. Defaults to current deployment.")
	AddStringFlag(console, doctl.ArgAppInstanceName, "", "", "Starts a console session for a specific instance name. Optional, defaults to the first available instance. For apps with multiple instances, you can specify the instance name to start the console session for that particular instance.")

	console.Example = `The following example initiates a console session for the app with the ID ` + "`" + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + "`" + ` and the component ` + "`" + `web` + "`" + `: doctl apps console f81d4fae-7dec-11d0-a765-00a0c91e6bf6 web. To initiate a console session to a specific instance, append the instance id: doctl apps console f81d4fae-7dec-11d0-a765-00a0c91e6bf6 web sample-golang-5d9f95556c-5f58g`

	appInstances := CmdBuilder(
		cmd,
		RunGetAppInstances,
		"list-instances <app id>",
		"Get app instances",
		`Returns an app's currently running ephemeral compute instances.`,
		Writer,
		aliasOpt("i"),
	)

	appInstances.Example = `The following examples retrieves the currently running, ephemeral compute instances for the app with the ID ` + "`" + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + "`" + `: doctl apps list-instances f81d4fae-7dec-11d0-a765-00a0c91e6bf6`

	listRegions := CmdBuilder(
		cmd,
		RunAppsListRegions,
		"list-regions",
		"Lists available App Platform regions",
		`Lists all regions supported by App Platform, including details about their current availability.`,
		Writer,
		displayerType(&displayers.AppRegions{}),
	)
	listRegions.Example = `The following example lists all regions supported by App Platform, including details about their current availability: doctl apps list-regions --format DataCenters,Disabled,Reason`

	propose := CmdBuilder(
		cmd,
		RunAppsPropose,
		"propose",
		"Proposes an app spec",
		`Reviews and validates an app specification for a new or existing app. The request returns some information about the proposed app, including app cost and upgrade cost. If an existing app ID is specified, the app spec is treated as a proposed update to the existing app.

Only basic information is included with the text output format. For complete app details including an updated app spec, use the `+"`"+`--output`+"`"+` global flag and specify the JSON format.`,
		Writer,
		aliasOpt("p"),
		displayerType(&displayers.Apps{}),
	)
	AddStringFlag(propose, doctl.ArgAppSpec, "", "", "Path to an app spec in JSON or YAML format. For more information about app specs, see the [app spec reference](https://www.digitalocean.com/docs/app-platform/concepts/app-spec)", requiredOpt())
	AddStringFlag(propose, doctl.ArgApp, "", "", "An optional existing app ID. If specified, App Platform treats the spec as a proposed update to the existing app.")
	propose.Example = `The following example proposes an app spec from the file directory ` + "`" + `src/your-app.yaml` + "`" + ` for a new app: doctl apps propose --spec src/your-app.yaml`

	listAlerts := CmdBuilder(
		cmd,
		RunAppListAlerts,
		"list-alerts <app id>",
		"Lists alerts on an app",
		`Lists all alerts associated to an app and its component, such as deployment failures and domain failures.`,
		Writer,
		aliasOpt("la"),
		displayerType(&displayers.AppAlerts{}),
	)
	listAlerts.Example = `The following example lists all alerts associated to an app with the ID ` + "`" + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + "`" + ` and uses the ` + "`" + `--format` + "`" + ` flag to specifically return the alert ID, trigger, and rule: doctl apps list-alerts f81d4fae-7dec-11d0-a765-00a0c91e6bf6 --format ID,Trigger,Spec.Rule`

	updateAlertDestinations := CmdBuilder(
		cmd,
		RunAppUpdateAlertDestinations,
		"update-alert-destinations <app id> <alert id>",
		"Updates alert destinations",
		`Updates alert destinations`,
		Writer,
		aliasOpt("uad"),
		displayerType(&displayers.AppAlerts{}),
	)
	updateAlertDestinations.Example = `The following example updates the alert destinations for an app with the ID ` + "`" + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + "`" + ` and the alert ID ` + "`" + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + "`" + `: doctl apps update-alert-destinations f81d4fae-7dec-11d0-a765-00a0c91e6bf6 f81d4fae-7dec-11d0-a765-00a0c91e6bf6 --alert-destinations src/your-alert-destinations.yaml`
	AddStringFlag(updateAlertDestinations, doctl.ArgAppAlertDestinations, "", "", "Path to an alert destinations file in JSON or YAML format.")

	listBuildpacks := CmdBuilder(
		cmd,
		RunAppListBuildpacks,
		"list-buildpacks",
		"Lists buildpacks",
		`Lists all buildpacks available on App Platform`,
		Writer,
		displayerType(&displayers.Buildpacks{}),
	)
	listBuildpacks.Example = `The following example lists all buildpacks available on App Platform and uses the ` + "`" + `--format` + "`" + ` flag to specifically return the buildpack ID and version: doctl apps list-buildpacks --format ID,Version`

	upgradeBuildpack := CmdBuilder(
		cmd,
		RunAppUpgradeBuildpack,
		"upgrade-buildpack <app id>",
		"Upgrades app's buildpack",
		`Upgrades an app's buildpack. For more information about buildpacks, see the [buildpack reference](https://docs.digitalocean.com/products/app-platform/reference/buildpacks/)`,
		Writer,
		displayerType(&displayers.Deployments{}),
	)
	AddStringFlag(upgradeBuildpack,
		doctl.ArgBuildpack, "", "", "The ID of the buildpack to upgrade to. Use the list-buildpacks command to list available buildpacks.", requiredOpt())
	AddIntFlag(upgradeBuildpack,
		doctl.ArgMajorVersion, "", 0, "The major version to upgrade to. If empty, the buildpack upgrades to the latest available version.")
	AddBoolFlag(upgradeBuildpack,
		doctl.ArgTriggerDeployment, "", true, "Specifies whether to trigger a new deployment to apply the upgrade.")
	upgradeBuildpack.Example = `The following example upgrades an app's buildpack with the ID ` + "`" + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + "`" + ` to the latest available version: doctl apps upgrade-buildpack f81d4fae-7dec-11d0-a765-00a0c91e6bf6 --buildpack f81d4fae-7dec-11d0-a765-00a0c91e6bf6`

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

	appSpec, err := apps.ReadAppSpec(os.Stdin, specPath)
	if err != nil {
		return err
	}

	upsert, err := c.Doit.GetBool(c.NS, doctl.ArgCommandUpsert)
	if err != nil {
		return err
	}

	updateSources, err := c.Doit.GetBool(c.NS, doctl.ArgCommandUpdateSources)
	if err != nil {
		return err
	}

	projectID, err := c.Doit.GetString(c.NS, doctl.ArgProjectID)
	if err != nil {
		return err
	}

	app, err := c.Apps().Create(&godo.AppCreateRequest{Spec: appSpec, ProjectID: projectID})
	if err != nil {
		if gerr, ok := err.(*godo.ErrorResponse); ok && gerr.Response.StatusCode == 409 && upsert {
			notice("App already exists, updating")

			apps, err := c.Apps().List(false)
			if err != nil {
				return err
			}

			id, err := getIDByName(apps, appSpec.Name)
			if err != nil {
				return err
			}

			app, err = c.Apps().Update(id, &godo.AppUpdateRequest{Spec: appSpec, UpdateAllSourceVersions: updateSources})
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	wait, err := c.Doit.GetBool(c.NS, doctl.ArgCommandWait)
	if err != nil {
		return err
	}

	var errs error

	if wait {
		apps := c.Apps()
		notice("App creation is in progress, waiting for app to be running")
		err := waitForActiveDeployment(apps, app.ID, app.GetPendingDeployment().GetID())
		if err != nil {
			errs = multierror.Append(errs, fmt.Errorf("app deployment couldn't enter `running` state: %v", err))
			if err := c.Display(displayers.Apps{app}); err != nil {
				errs = multierror.Append(errs, err)
			}
			return errs
		}
		app, _ = c.Apps().Get(app.ID)
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
	withProjects, err := c.Doit.GetBool(c.NS, doctl.ArgAppWithProjects)
	if err != nil {
		return err
	}

	apps, err := c.Apps().List(withProjects)
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

	updateSources, err := c.Doit.GetBool(c.NS, doctl.ArgCommandUpdateSources)
	if err != nil {
		return err
	}

	appSpec, err := apps.ReadAppSpec(os.Stdin, specPath)
	if err != nil {
		return err
	}

	app, err := c.Apps().Update(id, &godo.AppUpdateRequest{Spec: appSpec, UpdateAllSourceVersions: updateSources})
	if err != nil {
		return err
	}

	wait, err := c.Doit.GetBool(c.NS, doctl.ArgCommandWait)
	if err != nil {
		return err
	}

	var errs error

	if wait {
		apps := c.Apps()
		notice("App update is in progress, waiting for app to be running")
		err := waitForActiveDeployment(apps, app.ID, app.GetPendingDeployment().GetID())
		if err != nil {
			errs = multierror.Append(errs, fmt.Errorf("app deployment couldn't enter `running` state: %v", err))
			if err := c.Display(displayers.Apps{app}); err != nil {
				errs = multierror.Append(errs, err)
			}
			return errs
		}
		app, _ = c.Apps().Get(app.ID)
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
		return errOperationAborted
	}

	err = c.Apps().Delete(id)
	if err != nil {
		return err
	}
	notice("App deleted")

	return nil
}

// RunAppsRestart restarts an app.
func RunAppsRestart(c *CmdConfig) error {
	if len(c.Args) < 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	appID := c.Args[0]
	components, err := c.Doit.GetStringSlice(c.NS, doctl.ArgAppComponents)
	if err != nil {
		return err
	}

	wait, err := c.Doit.GetBool(c.NS, doctl.ArgCommandWait)
	if err != nil {
		return err
	}

	deployment, err := c.Apps().Restart(appID, components)
	if err != nil {
		return err
	}

	var errs error

	if wait {
		apps := c.Apps()
		notice("Restart is in progress, waiting for the restart to complete")
		err := waitForActiveDeployment(apps, appID, deployment.ID)
		if err != nil {
			errs = multierror.Append(errs, fmt.Errorf("app deployment couldn't enter `running` state: %v", err))
			if err := c.Display(displayers.Deployments{deployment}); err != nil {
				errs = multierror.Append(errs, err)
			}
			return errs
		}
		deployment, _ = c.Apps().GetDeployment(appID, deployment.ID)
	}

	notice("Restarted")

	return c.Display(displayers.Deployments{deployment})
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

	wait, err := c.Doit.GetBool(c.NS, doctl.ArgCommandWait)
	if err != nil {
		return err
	}

	deployment, err := c.Apps().CreateDeployment(appID, forceRebuild)
	if err != nil {
		return err
	}

	var errs error

	if wait {
		apps := c.Apps()
		notice("App deployment is in progress, waiting for deployment to be running")
		err := waitForActiveDeployment(apps, appID, deployment.ID)
		if err != nil {
			errs = multierror.Append(errs, fmt.Errorf("app deployment couldn't enter `running` state: %v", err))
			if err := c.Display(displayers.Deployments{deployment}); err != nil {
				errs = multierror.Append(errs, err)
			}
			return errs
		}
		deployment, _ = c.Apps().GetDeployment(appID, deployment.ID)
	}

	notice("Deployment created")

	return c.Display(displayers.Deployments{deployment})
}

func waitForActiveDeployment(apps do.AppsService, appID string, deploymentID string) error {
	const maxAttempts = 180
	attempts := 0
	printNewLineSet := false

	for i := 0; i < maxAttempts; i++ {
		if attempts != 0 {
			fmt.Fprint(os.Stderr, ".")
			if !printNewLineSet {
				printNewLineSet = true
				defer fmt.Fprintln(os.Stderr)
			}
		}

		deployment, err := apps.GetDeployment(appID, deploymentID)
		if err != nil {
			return err
		}

		allSuccessful := deployment.Progress.SuccessSteps == deployment.Progress.TotalSteps
		if allSuccessful {
			return nil
		}

		if deployment.Progress.ErrorSteps > 0 {
			return fmt.Errorf("error deploying app (%s) (deployment ID: %s):\n%s", appID, deployment.ID, godo.Stringify(deployment.Progress))
		}
		attempts++
		time.Sleep(10 * time.Second)
	}
	return fmt.Errorf("timeout waiting to app (%s) deployment", appID)
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

	jobInvocationID, err := c.Doit.GetString(c.NS, doctl.ArgAppJobInvocation)
	if err != nil {
		return err
	}

	_, err = uuid.Parse(appID)
	if err != nil || deploymentID == "" {
		app, err := c.Apps().Find(appID)
		if err != nil {
			return err
		}

		appID = app.ID

		if deploymentID == "" {
			if app.ActiveDeployment != nil {
				deploymentID = app.ActiveDeployment.ID
			} else if app.InProgressDeployment != nil {
				deploymentID = app.InProgressDeployment.ID
			} else {
				return fmt.Errorf("unable to retrieve logs; no deployment found for app %s", appID)
			}
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
	case strings.ToLower(string(godo.AppLogTypeRunRestarted)):
		logType = godo.AppLogTypeRunRestarted
	default:
		// if jobInvocationID is provided, we can skip the logType validation as logType will be set to JOB_INVOCATION
		if jobInvocationID == "" {
			return fmt.Errorf("Invalid log type %s", logTypeStr)
		}
	}
	logFollow, err := c.Doit.GetBool(c.NS, doctl.ArgAppLogFollow)
	if err != nil {
		return err
	}
	logTail, err := c.Doit.GetInt(c.NS, doctl.ArgAppLogTail)
	if err != nil {
		return err
	}

	noPrefixFlag, err := c.Doit.GetBool(c.NS, doctl.ArgNoPrefix)
	if err != nil {
		return err
	}

	var logs *godo.AppLogs
	if jobInvocationID != "" {
		if component == "" {
			return fmt.Errorf("component name is required when job invocation id is provided")
		}

		opts := &godo.GetJobInvocationLogsOptions{
			JobName:   component,
			Follow:    logFollow,
			TailLines: logTail,
		}

		logs, err = c.Apps().GetJobInvocationLogs(appID, jobInvocationID, opts)
	} else {
		logs, err = c.Apps().GetLogs(appID, deploymentID, component, logType, logFollow, logTail)
	}

	if err != nil {
		return err
	}

	if logs.LiveURL != "" {
		url, err := url.Parse(logs.LiveURL)
		if err != nil {
			return err
		}

		schemaFunc := func(message []byte) (io.Reader, error) {
			data := struct {
				Data string `json:"data"`
			}{}
			err = json.Unmarshal(message, &data)
			if err != nil {
				return nil, err
			}
			r := strings.NewReader(data.Data)

			if noPrefixFlag {
				content, err := io.ReadAll(r)
				if err != nil {
					return nil, err
				}
				logParts := strings.SplitN(string(content), " ", 3)
				if len(logParts) > 2 {
					jsonLog := logParts[2]
					return strings.NewReader(jsonLog), nil
				}
			}

			return r, nil
		}

		token := url.Query().Get("token")
		switch url.Scheme {
		case "http":
			url.Scheme = "ws"
		default:
			url.Scheme = "wss"
		}

		listener := c.Doit.Listen(url, token, schemaFunc, c.Out, nil)
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
		defer cancel()
		err = listener.Listen(ctx)
		if err != nil {
			return err
		}
	} else if len(logs.HistoricURLs) > 0 {
		resp, err := http.Get(logs.HistoricURLs[0])
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			logLine := scanner.Text()
			if noPrefixFlag {
				logParts := strings.SplitN(logLine, " ", 3)
				if len(logParts) > 2 {
					logLine = logParts[2]
				}
			}
			fmt.Fprintln(c.Out, logLine)
		}
		if err := scanner.Err(); err != nil {
			return err
		}
	} else {
		warn("No logs found for app component")
	}

	return nil
}

// RunAppsConsole initiates a console session for an app.
func RunAppsConsole(c *CmdConfig) error {
	if len(c.Args) < 2 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	appID := c.Args[0]
	componentName := c.Args[1]
	var instanceName string
	if len(c.Args) >= 3 {
		instanceName = c.Args[2]
	}

	deploymentID, err := c.Doit.GetString(c.NS, doctl.ArgAppDeployment)
	if err != nil {
		return err
	}

	opts := &godo.AppGetExecOptions{
		DeploymentID: deploymentID,
		InstanceName: instanceName,
	}
	execResp, err := c.Apps().GetExecWithOpts(appID, componentName, opts)

	if err != nil {
		return err
	}
	url, err := url.Parse(execResp.URL)
	if err != nil {
		return err
	}
	token := url.Query().Get("token")

	schemaFunc := func(message []byte) (io.Reader, error) {
		data := struct {
			Data string `json:"data"`
		}{}
		err = json.Unmarshal(message, &data)
		if err != nil {
			return nil, err
		}
		r := strings.NewReader(data.Data)
		return r, nil
	}

	inputCh := make(chan []byte)

	listener := c.Doit.Listen(url, token, schemaFunc, c.Out, inputCh)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	grp, ctx := errgroup.WithContext(ctx)

	term := c.Doit.Terminal()
	stdinCh := make(chan string)
	restoreTerminal, err := term.ReadRawStdin(ctx, stdinCh)
	if err != nil {
		return err
	}
	defer restoreTerminal()

	resizeEvents := make(chan terminal.TerminalSize)
	grp.Go(func() error {
		return term.MonitorResizeEvents(ctx, resizeEvents)
	})

	grp.Go(func() error {
		keepaliveTicker := time.NewTicker(30 * time.Second)
		defer keepaliveTicker.Stop()
		type stdinOp struct {
			Op   string `json:"op"`
			Data string `json:"data"`
		}
		type resizeOp struct {
			Op     string `json:"op"`
			Width  int    `json:"width"`
			Height int    `json:"height"`
		}
		for {
			select {
			case <-ctx.Done():
				return nil
			case in := <-stdinCh:
				b, err := json.Marshal(stdinOp{Op: "stdin", Data: in})
				if err != nil {
					return fmt.Errorf("error encoding stdin: %v", err)
				}
				inputCh <- b
			case <-keepaliveTicker.C:
				b, err := json.Marshal(stdinOp{Op: "stdin", Data: ""})
				if err != nil {
					return fmt.Errorf("error encoding keepalive event: %v", err)
				}
				inputCh <- b
			case ev := <-resizeEvents:
				b, err := json.Marshal(resizeOp{Op: "resize", Width: ev.Width, Height: ev.Height})
				if err != nil {
					return fmt.Errorf("error encoding resize event: %v", err)
				}
				inputCh <- b
			}
		}
	})

	grp.Go(func() error {
		err = listener.Listen(ctx)
		if err != nil {
			return err
		}
		cancel() // cancel the context to stop the other goroutines
		return nil
	})

	if err := grp.Wait(); err != nil {
		return err
	}

	return nil
}

// RunAppsPropose proposes an app spec
func RunAppsPropose(c *CmdConfig) error {
	appID, err := c.Doit.GetString(c.NS, doctl.ArgApp)
	if err != nil {
		return err
	}

	specPath, err := c.Doit.GetString(c.NS, doctl.ArgAppSpec)
	if err != nil {
		return err
	}

	appSpec, err := apps.ReadAppSpec(os.Stdin, specPath)
	if err != nil {
		return err
	}

	res, err := c.Apps().Propose(&godo.AppProposeRequest{
		Spec:  appSpec,
		AppID: appID,
	})

	if err != nil {
		// most likely an invalid app spec. The error message would start with "error validating app spec"
		return err
	}

	return c.Display(displayers.AppProposeResponse{Res: res})
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
	AddStringFlag(getCmd, doctl.ArgFormat, "", "yaml", `the format to output the spec in; either "yaml" or "json"`)

	validateCmd := cmdBuilderWithInit(cmd, RunAppsSpecValidate, "validate <spec file>", "Validate an application spec", `Use this command to check whether a given app spec (YAML or JSON) is valid.

You may pass - as the filename to read from stdin.`, Writer, false)
	AddBoolFlag(validateCmd, doctl.ArgSchemaOnly, "", false, "Only validate the spec schema and not the correctness of the spec.")

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
// doesn't require auth & connection to the API with doctl.ArgSchemaOnly flag
func RunAppsSpecValidate(c *CmdConfig) error {
	if len(c.Args) < 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	specPath := c.Args[0]
	appSpec, err := apps.ReadAppSpec(os.Stdin, specPath)
	if err != nil {
		return err
	}

	schemaOnly, err := c.Doit.GetBool(c.NS, doctl.ArgSchemaOnly)
	if err != nil {
		return err
	}

	// validate schema only (offline)
	if schemaOnly {
		ymlSpec, err := yaml.Marshal(appSpec)
		if err != nil {
			return fmt.Errorf("marshaling the spec as yaml: %v", err)
		}
		_, err = c.Out.Write(ymlSpec)
		return err
	}

	// validate the spec against the API
	if err := c.initServices(c); err != nil {
		return err
	}
	res, err := c.Apps().Propose(&godo.AppProposeRequest{
		Spec: appSpec,
	})
	if err != nil {
		// most likely an invalid app spec. The error message would start with "error validating app spec"
		return err
	}

	ymlSpec, err := yaml.Marshal(res.Spec)
	if err != nil {
		return fmt.Errorf("marshaling the spec as yaml: %v", err)
	}
	_, err = c.Out.Write(ymlSpec)
	return err
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

	tierDeprecationMsg := "This command is deprecated and will be removed in a future release. Use `doctl apps tier instance-size <get|list>` instead.\n\n"
	CmdBuilder(cmd, RunAppsTierList, "list", "List all app tiers", tierDeprecationMsg+`Use this command to list all the available app tiers.`, Writer, aliasOpt("ls"), hiddenCmd())
	CmdBuilder(cmd, RunAppsTierGet, "get <tier slug>", "Retrieve an app tier", tierDeprecationMsg+`Use this command to retrieve information about a specific app tier.`, Writer, hiddenCmd())

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

	CmdBuilder(cmd, RunAppsTierInstanceSizeList, "list", "List all app instance sizes", `Use this command to list all the available app instance sizes.`, Writer, aliasOpt("ls"))
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

// RunAppListAlerts gets configured alerts on an app
func RunAppListAlerts(c *CmdConfig) error {
	if len(c.Args) < 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	appID := c.Args[0]

	alerts, err := c.Apps().ListAlerts(appID)
	if err != nil {
		return err
	}
	return c.Display(displayers.AppAlerts(alerts))
}

func RunAppUpdateAlertDestinations(c *CmdConfig) error {
	if len(c.Args) < 2 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	appID := c.Args[0]
	alertID := c.Args[1]

	alertDestinationsPath, err := c.Doit.GetString(c.NS, doctl.ArgAppAlertDestinations)
	if err != nil {
		return err
	}
	update, err := readAppAlertDestination(os.Stdin, alertDestinationsPath)
	if err != nil {
		return err
	}

	alert, err := c.Apps().UpdateAlertDestinations(appID, alertID, update)
	if err != nil {
		return err
	}
	return c.Display(displayers.AppAlerts([]*godo.AppAlert{alert}))
}

func readAppAlertDestination(stdin io.Reader, path string) (*godo.AlertDestinationUpdateRequest, error) {
	var alertDestinations io.Reader
	if path == "-" {
		alertDestinations = stdin
	} else {
		alertDestinationsFile, err := os.Open(path) // guardrails-disable-line
		if err != nil {
			if os.IsNotExist(err) {
				return nil, fmt.Errorf("opening app alert destinations: %s does not exist", path)
			}
			return nil, fmt.Errorf("opening app alert destinations: %w", err)
		}
		defer alertDestinationsFile.Close()
		alertDestinations = alertDestinationsFile
	}

	byt, err := io.ReadAll(alertDestinations)
	if err != nil {
		return nil, fmt.Errorf("reading app alert destinations: %w", err)
	}

	s, err := parseAppAlert(byt)
	if err != nil {
		return nil, fmt.Errorf("parsing app alert destinations: %w", err)
	}

	return s, nil
}

func parseAppAlert(destinations []byte) (*godo.AlertDestinationUpdateRequest, error) {
	jsonAlertDestinations, err := yaml.YAMLToJSON(destinations)
	if err != nil {
		return nil, err
	}

	dec := json.NewDecoder(bytes.NewReader(jsonAlertDestinations))
	dec.DisallowUnknownFields()

	var alertDestinations godo.AlertDestinationUpdateRequest
	if err := dec.Decode(&alertDestinations); err != nil {
		return nil, err
	}

	return &alertDestinations, nil
}

// RunAppListBuildpacks lists buildpacks
func RunAppListBuildpacks(c *CmdConfig) error {
	bps, err := c.Apps().ListBuildpacks()
	if err != nil {
		return err
	}
	return c.Display(displayers.Buildpacks(bps))
}

// RunAppUpgradeBuildpack upgrades a buildpack for an app
func RunAppUpgradeBuildpack(c *CmdConfig) error {
	if len(c.Args) < 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	appID := c.Args[0]
	buildpack, err := c.Doit.GetString(c.NS, doctl.ArgBuildpack)
	if err != nil {
		return err
	}
	majorVersion, err := c.Doit.GetInt(c.NS, doctl.ArgMajorVersion)
	if err != nil {
		return err
	}
	triggerDeployment, err := c.Doit.GetBool(c.NS, doctl.ArgTriggerDeployment)
	if err != nil {
		return err
	}

	components, dep, err := c.Apps().UpgradeBuildpack(appID, godo.UpgradeBuildpackOptions{
		BuildpackID:       buildpack,
		MajorVersion:      int32(majorVersion),
		TriggerDeployment: triggerDeployment,
	})
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "upgraded buildpack %s. %d components were affected: %v.\n", buildpack, len(components), components)

	if dep != nil {
		fmt.Fprint(os.Stderr, "triggered a new deployment to apply the upgrade:\n\n")
		return c.Display(displayers.Deployments([]*godo.Deployment{dep}))
	}

	return nil
}

func getIDByName(apps []*godo.App, name string) (string, error) {
	for _, app := range apps {
		if app.Spec.Name == name {
			return app.ID, nil
		}
	}

	return "", fmt.Errorf("app not found")
}

// RunGetAppInstances gets currently running ephemeral compute instances for an app
func RunGetAppInstances(c *CmdConfig) error {
	if len(c.Args) < 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	appID := c.Args[0]

	opts := &godo.GetAppInstancesOpts{}
	instances, err := c.Apps().GetAppInstances(appID, opts)
	if err != nil {
		return err
	}

	return c.Display(displayers.AppInstances(instances))
}

// RunAppsListJobInvocations lists job invocations for a given app
func RunAppsListJobInvocations(c *CmdConfig) error {
	if len(c.Args) < 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	appID := c.Args[0]

	deploymentID, err := c.Doit.GetString(c.NS, doctl.ArgAppDeployment)
	if err != nil {
		return err
	}

	jobName, err := c.Doit.GetString(c.NS, doctl.ArgAppJobName)
	if err != nil {
		return err
	}

	opts := &godo.ListJobInvocationsOptions{
		DeploymentID: deploymentID,
		JobNames:     []string{jobName},
	}

	invocations, err := c.Apps().ListJobInvocations(appID, opts)
	if err != nil {
		return err
	}

	return c.Display(displayers.JobInvocations(invocations))
}

// RunAppsGetJobInvocation gets a job invocation for an app.
func RunAppsGetJobInvocation(c *CmdConfig) error {
	if len(c.Args) < 2 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	appID := c.Args[0]
	jobInvocationID := c.Args[1]

	opts := &godo.GetJobInvocationOptions{}

	jobName, err := c.Doit.GetString(c.NS, doctl.ArgAppJobName)
	if err != nil {
		return err
	}

	if jobName != "" {
		opts.JobName = jobName
	}

	jobInvocation, err := c.Apps().GetJobInvocation(appID, jobInvocationID, opts)
	if err != nil {
		return err
	}

	return c.Display(displayers.JobInvocations{jobInvocation})
}

// RunAppsGetJobInvocation gets a job invocation for an app.
func RunAppsCancelJobInvocation(c *CmdConfig) error {
	if len(c.Args) < 2 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	appID := c.Args[0]
	jobInvocationID := c.Args[1]

	opts := &godo.CancelJobInvocationOptions{}

	jobName, err := c.Doit.GetString(c.NS, doctl.ArgAppJobName)
	if err != nil {
		return err
	}

	if jobName != "" {
		opts.JobName = jobName
	}

	jobInvocation, err := c.Apps().CancelJobInvocation(appID, jobInvocationID, opts)
	if err != nil {
		return err
	}

	return c.Display(displayers.JobInvocations{jobInvocation})
}
