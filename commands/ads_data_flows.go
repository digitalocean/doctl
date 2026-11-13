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
	"errors"
	"fmt"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"
)

// adsMaxDataFlowPages bounds how many pages `ads data-flow list` follows, so
// a server that keeps returning a page token cannot loop forever.
const adsMaxDataFlowPages = 1000

func adsDataFlow() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "data-flow",
			Aliases: []string{"data-flows", "df"},
			Short:   "Display commands for ADS data flows",
			Long: `The subcommands of ` + "`" + `doctl ads data-flow` + "`" + ` manage the data flows that load data into a workspace: create and update them, control their lifecycle, and attach function-chain archives.

Every data flow command needs the workspace's ` + "`" + `--region` + "`" + ` and ` + "`" + `--workspace` + "`" + ` UUID.`,
		},
	}

	create := CmdBuilder(cmd, RunADSDataFlowCreate, "create", "Create a data flow",
		"Creates a data flow in a workspace from a manifest. If the response includes an upload location, use `--format UUID,State,UploadURL` or `--output json` to see the presigned upload URL.", Writer,
		aliasOpt("c"),
		displayerType(&displayers.ADSDataFlowChange{}))
	addADSRegionFlag(create)
	addADSWorkspaceFlag(create)
	AddStringFlag(create, doctl.ArgADSName, "", "", "Name of the data flow")
	AddStringFlag(create, doctl.ArgADSDataFlowManifestFile, "", "", "Path to the data flow manifest, a JSON or YAML object. Set to - to read from stdin")
	AddStringFlag(create, doctl.ArgADSDataFlowImageRef, "", "", "Container image reference for the data flow")
	AddIntFlag(create, doctl.ArgADSDataFlowDesiredReplicas, "", 0, "Number of replicas to run")
	AddIntFlag(create, doctl.ArgADSDataFlowMaxReplicas, "", 0, "Maximum number of replicas")
	create.Example = "The following example creates a data flow from a manifest file: doctl ads data-flow create --region nyc3 --workspace f81d4fae-7dec-11d0-a765-00a0c91e6bf6 --name orders --manifest-file orders.yaml"

	list := CmdBuilder(cmd, RunADSDataFlowList, "list", "List data flows",
		"Lists the data flows in a workspace.", Writer,
		aliasOpt("ls"),
		displayerType(&displayers.ADSDataFlow{}))
	addADSRegionFlag(list)
	addADSWorkspaceFlag(list)

	get := CmdBuilder(cmd, RunADSDataFlowGet, "get <data-flow-id>", "Get a data flow",
		"Retrieves a data flow. Use `--output json` to see its full spec.", Writer,
		aliasOpt("g"),
		displayerType(&displayers.ADSDataFlow{}))
	get.Args = cobra.ExactArgs(1)
	addADSRegionFlag(get)
	addADSWorkspaceFlag(get)

	update := CmdBuilder(cmd, RunADSDataFlowUpdate, "update <data-flow-id>", "Update a data flow",
		"Updates a data flow's replica settings, secret references, or manifest. Set at least one of the flags below; unset flags leave the current value unchanged.", Writer,
		aliasOpt("u"),
		displayerType(&displayers.ADSDataFlowChange{}))
	update.Args = cobra.ExactArgs(1)
	addADSRegionFlag(update)
	addADSWorkspaceFlag(update)
	AddIntFlag(update, doctl.ArgADSDataFlowDesiredReplicas, "", 0, "Number of replicas to run")
	AddIntFlag(update, doctl.ArgADSDataFlowMaxReplicas, "", 0, "Maximum number of replicas")
	AddStringSliceFlag(update, doctl.ArgADSDataFlowSecretRefs, "", []string{}, "Comma-separated keys of workspace secrets the data flow uses. Replaces the current list")
	AddBoolFlag(update, doctl.ArgADSDataFlowClearSecretRefs, "", false, "Remove every secret reference from the data flow")
	AddStringFlag(update, doctl.ArgADSDataFlowManifestFile, "", "", "Path to a replacement manifest, a JSON or YAML object. Set to - to read from stdin")

	deleteCmd := CmdBuilder(cmd, RunADSDataFlowDelete, "delete <data-flow-id>", "Delete a data flow",
		"Starts deleting a data flow. This cannot be undone.", Writer,
		aliasOpt("d", "rm"))
	deleteCmd.Args = cobra.ExactArgs(1)
	addADSRegionFlag(deleteCmd)
	addADSWorkspaceFlag(deleteCmd)
	AddBoolFlag(deleteCmd, doctl.ArgForce, doctl.ArgShortForce, false, "Delete the data flow without a confirmation prompt")

	for _, action := range []struct {
		use, short, long string
		run              CmdRunner
	}{
		{"pause <data-flow-id>", "Pause a data flow", "Pauses an active data flow.", RunADSDataFlowPause},
		{"resume <data-flow-id>", "Resume a data flow", "Resumes a paused data flow.", RunADSDataFlowResume},
		{"restart <data-flow-id>", "Restart a data flow", "Restarts a data flow.", RunADSDataFlowRestart},
	} {
		c := CmdBuilder(cmd, action.run, action.use, action.short, action.long, Writer)
		c.Args = cobra.ExactArgs(1)
		addADSRegionFlag(c)
		addADSWorkspaceFlag(c)
	}

	rewind := CmdBuilder(cmd, RunADSDataFlowRewind, "rewind <data-flow-id>", "Rewind a data flow",
		"Asks a data flow to reprocess data from an earlier time.\n\nThe API currently reports rewind as unsupported, so this command returns that error until the server supports it.", Writer)
	rewind.Args = cobra.ExactArgs(1)
	addADSRegionFlag(rewind)
	addADSWorkspaceFlag(rewind)
	AddStringFlag(rewind, doctl.ArgADSDataFlowRewindTo, "", "", "Time to rewind to, in RFC 3339 format, for example `2026-11-01T00:00:00Z`")

	uploadURL := CmdBuilder(cmd, RunADSDataFlowUploadURL, "upload-url", "Create an upload URL for a function-chain archive",
		"Creates a temporary presigned URL for uploading a function-chain archive. Upload the archive to the URL with an HTTP PUT, then pass the returned object key to `doctl ads data-flow attach`.", Writer,
		displayerType(&displayers.ADSFchainUploadURL{}))
	addADSRegionFlag(uploadURL)
	addADSWorkspaceFlag(uploadURL)
	AddStringFlag(uploadURL, doctl.ArgADSFchainSHA256, "", "", "Hex-encoded SHA-256 checksum of the archive")

	attach := CmdBuilder(cmd, RunADSDataFlowAttach, "attach <data-flow-id>", "Attach a function-chain archive to a data flow",
		"Attaches a function-chain archive, uploaded with the URL from `doctl ads data-flow upload-url`, to a data flow.", Writer)
	attach.Args = cobra.ExactArgs(1)
	addADSRegionFlag(attach)
	addADSWorkspaceFlag(attach)
	AddStringFlag(attach, doctl.ArgADSFchainObjectKey, "", "", "Object key returned by `doctl ads data-flow upload-url`", requiredOpt())

	return cmd
}

var errADSManifestNotObject = errors.New("manifest file must contain a single JSON or YAML object")

// readADSManifest reads a data flow manifest from path, or from stdin when
// path is "-", and returns it as JSON. JSON input is sent as written.
func readADSManifest(path string) (json.RawMessage, error) {
	data, err := readADSFile(path)
	if err != nil {
		return nil, err
	}
	data = bytes.TrimSpace(data)
	if json.Valid(data) {
		if len(data) == 0 || data[0] != '{' {
			return nil, errADSManifestNotObject
		}
		return json.RawMessage(data), nil
	}
	converted, err := yaml.YAMLToJSON(data)
	if err != nil {
		return nil, fmt.Errorf("parsing manifest file: %w", err)
	}
	if len(converted) == 0 || converted[0] != '{' {
		return nil, errADSManifestNotObject
	}
	return json.RawMessage(converted), nil
}

// adsReplicas reads --desired-replicas and --max-replicas.
func adsReplicas(c *CmdConfig) (desired, max int32, err error) {
	d, err := c.Doit.GetInt(c.NS, doctl.ArgADSDataFlowDesiredReplicas)
	if err != nil {
		return 0, 0, err
	}
	m, err := c.Doit.GetInt(c.NS, doctl.ArgADSDataFlowMaxReplicas)
	if err != nil {
		return 0, 0, err
	}
	if d < 0 || m < 0 {
		return 0, 0, fmt.Errorf("--%s and --%s must not be negative", doctl.ArgADSDataFlowDesiredReplicas, doctl.ArgADSDataFlowMaxReplicas)
	}
	if m > 0 && d > m {
		return 0, 0, fmt.Errorf("--%s must not exceed --%s", doctl.ArgADSDataFlowDesiredReplicas, doctl.ArgADSDataFlowMaxReplicas)
	}
	return int32(d), int32(m), nil
}

// RunADSDataFlowCreate creates a data flow.
func RunADSDataFlowCreate(c *CmdConfig) error {
	region, workspace, err := adsRegionWorkspace(c)
	if err != nil {
		return err
	}
	name, err := c.Doit.GetString(c.NS, doctl.ArgADSName)
	if err != nil {
		return err
	}
	imageRef, err := c.Doit.GetString(c.NS, doctl.ArgADSDataFlowImageRef)
	if err != nil {
		return err
	}
	desired, max, err := adsReplicas(c)
	if err != nil {
		return err
	}
	manifestPath, err := c.Doit.GetString(c.NS, doctl.ArgADSDataFlowManifestFile)
	if err != nil {
		return err
	}
	var manifest json.RawMessage
	if manifestPath != "" {
		if manifest, err = readADSManifest(manifestPath); err != nil {
			return err
		}
	}

	resp, err := c.ADS().CreateDataFlow(&godo.ADSDataFlowCreateRequest{
		Region:          region,
		Workspace:       workspace,
		Name:            name,
		DesiredReplicas: desired,
		MaxReplicas:     max,
		ImageRef:        imageRef,
		Manifest:        manifest,
	})
	if err != nil {
		return err
	}
	return c.Display(&displayers.ADSDataFlowChange{UUID: resp.UUID, State: resp.State, Upload: resp.Upload})
}

// RunADSDataFlowList lists every data flow in a workspace, following page tokens.
func RunADSDataFlowList(c *CmdConfig) error {
	region, workspace, err := adsRegionWorkspace(c)
	if err != nil {
		return err
	}

	var flows []godo.ADSDataFlow
	req := &godo.ADSDataFlowListRequest{Region: region, Workspace: workspace}
	for page := 0; ; page++ {
		if page == adsMaxDataFlowPages {
			return fmt.Errorf("listing data flows: stopped after %d pages", adsMaxDataFlowPages)
		}
		resp, err := c.ADS().ListDataFlows(req)
		if err != nil {
			return err
		}
		flows = append(flows, resp.DataFlows...)
		if resp.NextPageToken == "" || resp.NextPageToken == req.PageToken {
			break
		}
		req = &godo.ADSDataFlowListRequest{Region: region, Workspace: workspace, PageToken: resp.NextPageToken}
	}
	return c.Display(&displayers.ADSDataFlow{DataFlows: flows})
}

// RunADSDataFlowGet retrieves a data flow.
func RunADSDataFlowGet(c *CmdConfig) error {
	if len(c.Args) != 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	region, workspace, err := adsRegionWorkspace(c)
	if err != nil {
		return err
	}
	flow, err := c.ADS().GetDataFlow(region, workspace, c.Args[0])
	if err != nil {
		return err
	}
	return c.Display(&displayers.ADSDataFlow{DataFlows: []godo.ADSDataFlow{*flow}})
}

// RunADSDataFlowUpdate updates a data flow.
func RunADSDataFlowUpdate(c *CmdConfig) error {
	if len(c.Args) != 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	region, workspace, err := adsRegionWorkspace(c)
	if err != nil {
		return err
	}
	desired, max, err := adsReplicas(c)
	if err != nil {
		return err
	}
	secretRefs, err := c.Doit.GetStringSlice(c.NS, doctl.ArgADSDataFlowSecretRefs)
	if err != nil {
		return err
	}
	clearRefs, err := c.Doit.GetBool(c.NS, doctl.ArgADSDataFlowClearSecretRefs)
	if err != nil {
		return err
	}
	if clearRefs && len(secretRefs) > 0 {
		return fmt.Errorf("--%s cannot be combined with --%s", doctl.ArgADSDataFlowClearSecretRefs, doctl.ArgADSDataFlowSecretRefs)
	}
	manifestPath, err := c.Doit.GetString(c.NS, doctl.ArgADSDataFlowManifestFile)
	if err != nil {
		return err
	}
	var manifest json.RawMessage
	if manifestPath != "" {
		if manifest, err = readADSManifest(manifestPath); err != nil {
			return err
		}
	}
	if desired == 0 && max == 0 && len(secretRefs) == 0 && !clearRefs && manifest == nil {
		return fmt.Errorf("nothing to update: set --%s, --%s, --%s, --%s, or --%s",
			doctl.ArgADSDataFlowDesiredReplicas, doctl.ArgADSDataFlowMaxReplicas,
			doctl.ArgADSDataFlowSecretRefs, doctl.ArgADSDataFlowClearSecretRefs, doctl.ArgADSDataFlowManifestFile)
	}

	resp, err := c.ADS().UpdateDataFlow(&godo.ADSDataFlowUpdateRequest{
		Region:          region,
		ID:              c.Args[0],
		Workspace:       workspace,
		DesiredReplicas: desired,
		MaxReplicas:     max,
		SecretRefs:      secretRefs,
		ClearSecretRefs: clearRefs,
		Manifest:        manifest,
	})
	if err != nil {
		return err
	}
	return c.Display(&displayers.ADSDataFlowChange{UUID: resp.UUID, State: resp.State, Upload: resp.Upload})
}

// RunADSDataFlowDelete deletes a data flow.
func RunADSDataFlowDelete(c *CmdConfig) error {
	if len(c.Args) != 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	region, workspace, err := adsRegionWorkspace(c)
	if err != nil {
		return err
	}
	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}
	if err := confirmDelete(force, "ADS data flow", 1); err != nil {
		return err
	}
	return c.ADS().DeleteDataFlow(region, workspace, c.Args[0])
}

// RunADSDataFlowPause pauses a data flow.
func RunADSDataFlowPause(c *CmdConfig) error {
	if len(c.Args) != 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	region, workspace, err := adsRegionWorkspace(c)
	if err != nil {
		return err
	}
	return c.ADS().PauseDataFlow(region, workspace, c.Args[0])
}

// RunADSDataFlowResume resumes a data flow.
func RunADSDataFlowResume(c *CmdConfig) error {
	if len(c.Args) != 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	region, workspace, err := adsRegionWorkspace(c)
	if err != nil {
		return err
	}
	return c.ADS().ResumeDataFlow(region, workspace, c.Args[0])
}

// RunADSDataFlowRestart restarts a data flow.
func RunADSDataFlowRestart(c *CmdConfig) error {
	if len(c.Args) != 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	region, workspace, err := adsRegionWorkspace(c)
	if err != nil {
		return err
	}
	return c.ADS().RestartDataFlow(region, workspace, c.Args[0])
}

// RunADSDataFlowRewind asks a data flow to rewind. The server currently
// rejects rewind as unsupported; that error is returned as is.
func RunADSDataFlowRewind(c *CmdConfig) error {
	if len(c.Args) != 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	region, workspace, err := adsRegionWorkspace(c)
	if err != nil {
		return err
	}
	rewindTo, err := c.Doit.GetString(c.NS, doctl.ArgADSDataFlowRewindTo)
	if err != nil {
		return err
	}
	return c.ADS().RewindDataFlow(&godo.ADSDataFlowRewindRequest{
		Region:    region,
		ID:        c.Args[0],
		Workspace: workspace,
		RewindTo:  rewindTo,
	})
}

// RunADSDataFlowUploadURL creates a presigned upload URL for a function-chain archive.
func RunADSDataFlowUploadURL(c *CmdConfig) error {
	region, workspace, err := adsRegionWorkspace(c)
	if err != nil {
		return err
	}
	sha, err := c.Doit.GetString(c.NS, doctl.ArgADSFchainSHA256)
	if err != nil {
		return err
	}
	resp, err := c.ADS().CreateFchainUploadURL(&godo.ADSFchainUploadURLRequest{
		Region:    region,
		Workspace: workspace,
		SHA256:    sha,
	})
	if err != nil {
		return err
	}
	return c.Display(&displayers.ADSFchainUploadURL{Upload: *resp})
}

// RunADSDataFlowAttach attaches an uploaded function-chain archive to a data flow.
func RunADSDataFlowAttach(c *CmdConfig) error {
	if len(c.Args) != 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	region, workspace, err := adsRegionWorkspace(c)
	if err != nil {
		return err
	}
	objectKey, err := c.Doit.GetString(c.NS, doctl.ArgADSFchainObjectKey)
	if err != nil {
		return err
	}
	if objectKey == "" {
		return doctl.NewMissingArgsErr(fmt.Sprintf("%s.%s", c.NS, doctl.ArgADSFchainObjectKey))
	}
	return c.ADS().AttachDataFlowFchain(&godo.ADSAttachDataFlowFchainRequest{
		Region:    region,
		ID:        c.Args[0],
		Workspace: workspace,
		ObjectKey: objectKey,
	})
}

func adsMetrics() *Command {
	cmd := CmdBuilder(nil, RunADSMetrics, "metrics", "Query data flow metrics for a workspace",
		"Queries data flow metrics for a workspace. The columns in the output depend on the metrics and grouping you ask for; use `--output json` for the raw rows.", Writer,
		displayerType(&displayers.ADSMetrics{}))
	addADSRegionFlag(cmd)
	addADSWorkspaceFlag(cmd)
	AddStringSliceFlag(cmd, doctl.ArgADSMetricNames, "", []string{}, "Comma-separated metric names to query")
	AddStringFlag(cmd, doctl.ArgADSMetricDataFlow, "", "", "Only query metrics for this data flow ID")
	AddStringFlag(cmd, doctl.ArgADSMetricFrom, "", "", "Start of the query window, in RFC 3339 format")
	AddStringFlag(cmd, doctl.ArgADSMetricTo, "", "", "End of the query window, in RFC 3339 format")
	AddStringSliceFlag(cmd, doctl.ArgADSMetricGroupBy, "", []string{}, "Comma-separated labels to group results by")
	AddIntFlag(cmd, doctl.ArgADSMetricLimit, "", 0, "Maximum number of rows to return")
	AddStringFlag(cmd, doctl.ArgADSMetricType, "", "", "Metric type to query")
	cmd.Example = "The following example queries one data flow's metrics for a day: doctl ads metrics --region nyc3 --workspace f81d4fae-7dec-11d0-a765-00a0c91e6bf6 --data-flow 3f2c9a1e --from 2026-11-01T00:00:00Z --to 2026-11-02T00:00:00Z"
	return cmd
}

// RunADSMetrics queries data flow metrics for a workspace.
func RunADSMetrics(c *CmdConfig) error {
	region, workspace, err := adsRegionWorkspace(c)
	if err != nil {
		return err
	}
	names, err := c.Doit.GetStringSlice(c.NS, doctl.ArgADSMetricNames)
	if err != nil {
		return err
	}
	dataFlow, err := c.Doit.GetString(c.NS, doctl.ArgADSMetricDataFlow)
	if err != nil {
		return err
	}
	from, err := c.Doit.GetString(c.NS, doctl.ArgADSMetricFrom)
	if err != nil {
		return err
	}
	to, err := c.Doit.GetString(c.NS, doctl.ArgADSMetricTo)
	if err != nil {
		return err
	}
	groupBy, err := c.Doit.GetStringSlice(c.NS, doctl.ArgADSMetricGroupBy)
	if err != nil {
		return err
	}
	limit, err := c.Doit.GetInt(c.NS, doctl.ArgADSMetricLimit)
	if err != nil {
		return err
	}
	if limit < 0 {
		return fmt.Errorf("--%s must not be negative", doctl.ArgADSMetricLimit)
	}
	metricType, err := c.Doit.GetString(c.NS, doctl.ArgADSMetricType)
	if err != nil {
		return err
	}

	resp, err := c.ADS().GetWorkspaceMetrics(&godo.ADSWorkspaceMetricsRequest{
		Region:      region,
		Workspace:   workspace,
		MetricNames: names,
		DataFlowID:  dataFlow,
		From:        from,
		To:          to,
		GroupBy:     groupBy,
		Limit:       int32(limit),
		MetricType:  metricType,
	})
	if err != nil {
		return err
	}
	return c.Display(&displayers.ADSMetrics{Metrics: *resp})
}
