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
	"os"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"
)

// DedicatedInferenceCmd creates the dedicated-inference command and its subcommands.
func DedicatedInferenceCmd() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "dedicated-inference",
			Aliases: []string{"di", "dedicated-inferences"},
			Short:   "Display commands for managing dedicated inference endpoints",
			Long:    "The subcommands of `doctl dedicated-inference` manage your dedicated inference endpoints.",
			GroupID: manageResourcesGroup,
		},
	}

	cmdCreate := CmdBuilder(
		cmd,
		RunDedicatedInferenceCreate,
		"create",
		"Create a dedicated inference endpoint",
		`Creates a dedicated inference endpoint on your account using a spec file in JSON or YAML format.
Use the `+"`"+`--spec`+"`"+` flag to provide the path to the spec file.
Optionally provide a Hugging Face access token using `+"`"+`--hugging-face-token`+"`"+`.`,
		Writer,
		aliasOpt("c"),
		displayerType(&displayers.DedicatedInference{}),
	)
	AddStringFlag(cmdCreate, doctl.ArgDedicatedInferenceSpec, "", "", `Path to a dedicated inference spec in JSON or YAML format. Set to "-" to read from stdin.`, requiredOpt())
	AddStringFlag(cmdCreate, doctl.ArgDedicatedInferenceHuggingFaceToken, "", "", "Hugging Face token for accessing gated models (optional)")
	cmdCreate.Example = `The following example creates a dedicated inference endpoint using a spec file: doctl dedicated-inference create --spec spec.yaml --hugging-face-token "hf_mytoken"

For more information, see https://docs.digitalocean.com/reference/api/digitalocean/#tag/Dedicated-Inference/operation/dedicatedInferences_create`

	cmdGet := CmdBuilder(
		cmd,
		RunDedicatedInferenceGet,
		"get <dedicated-inference-id>",
		"Retrieve a dedicated inference endpoint",
		`Retrieves details about a dedicated inference endpoint, including its ID, name, region, status, VPC, endpoints, and deployment specs.`,
		Writer,
		aliasOpt("g"),
		displayerType(&displayers.DedicatedInference{}),
	)
	cmdGet.Example = `The following example retrieves a dedicated inference endpoint: doctl dedicated-inference get 12345678-1234-1234-1234-123456789012`

	cmdDelete := CmdBuilder(
		cmd,
		RunDedicatedInferenceDelete,
		"delete <dedicated-inference-id>",
		"Delete a dedicated inference endpoint",
		`Deletes a dedicated inference endpoint by its ID. All associated resources will be destroyed.`,
		Writer,
		aliasOpt("d", "rm"),
	)
	AddBoolFlag(cmdDelete, doctl.ArgForce, doctl.ArgShortForce, false, "Delete the dedicated inference endpoint without a confirmation prompt")
	cmdDelete.Example = `The following example deletes a dedicated inference endpoint: doctl dedicated-inference delete 12345678-1234-1234-1234-123456789012`

	cmdUpdate := CmdBuilder(
		cmd,
		RunDedicatedInferenceUpdate,
		"update <dedicated-inference-id>",
		"Update a dedicated inference endpoint",
		`Updates a dedicated inference endpoint using a spec file in JSON or YAML format.
Use the `+"`"+`--spec`+"`"+` flag to provide the path to the spec file.
Optionally provide a Hugging Face access token using `+"`"+`--hugging-face-token`+"`"+`.`,
		Writer,
		aliasOpt("u"),
		displayerType(&displayers.DedicatedInference{}),
	)
	AddStringFlag(cmdUpdate, doctl.ArgDedicatedInferenceSpec, "", "", `Path to a dedicated inference spec in JSON or YAML format. Set to "-" to read from stdin.`, requiredOpt())
	AddStringFlag(cmdUpdate, doctl.ArgDedicatedInferenceHuggingFaceToken, "", "", "Hugging Face token for accessing gated models (optional)")
	cmdUpdate.Example = `The following example updates a dedicated inference endpoint using a spec file: doctl dedicated-inference update 12345678-1234-1234-1234-123456789012 --spec spec.yaml

For more information, see https://docs.digitalocean.com/reference/api/digitalocean/#tag/Dedicated-Inference/operation/dedicatedInferences_update`

	cmdList := CmdBuilder(
		cmd,
		RunDedicatedInferenceList,
		"list",
		"List all dedicated inference endpoints",
		`Lists all dedicated inference endpoints on your account, including their IDs, names, regions, statuses, and endpoints.
Optionally use `+"`"+`--region`+"`"+` to filter by region or `+"`"+`--name`+"`"+` to filter by name.`,
		Writer,
		aliasOpt("ls"),
		displayerType(&displayers.DedicatedInferenceList{}),
	)
	AddStringFlag(cmdList, doctl.ArgDedicatedInferenceRegion, "", "", "Filter by region (optional)")
	AddStringFlag(cmdList, doctl.ArgDedicatedInferenceName, "", "", "Filter by name (optional)")
	cmdList.Example = `The following example lists all dedicated inference endpoints: doctl dedicated-inference list

The following example filters by region: doctl dedicated-inference list --region nyc2

The following example filters by name: doctl dedicated-inference list --name my-endpoint`

	cmdListAccelerators := CmdBuilder(
		cmd,
		RunDedicatedInferenceListAccelerators,
		"list-accelerators <dedicated-inference-id>",
		"List accelerators for a dedicated inference endpoint",
		`Lists the accelerators provisioned for a dedicated inference endpoint, including their IDs, names, slugs, and statuses.
Optionally use `+"`"+`--slug`+"`"+` to filter by accelerator slug.`,
		Writer,
		aliasOpt("la"),
		displayerType(&displayers.DedicatedInferenceAccelerator{}),
	)
	AddStringFlag(cmdListAccelerators, doctl.ArgDedicatedInferenceAcceleratorSlug, "", "", "Filter accelerators by slug (optional)")
	cmdListAccelerators.Example = `The following example lists accelerators for a dedicated inference endpoint: doctl dedicated-inference list-accelerators 12345678-1234-1234-1234-123456789012

The following example filters by slug: doctl dedicated-inference list-accelerators 12345678-1234-1234-1234-123456789012 --slug gpu-mi300x1-192gb`

	cmdCreateToken := CmdBuilder(
		cmd,
		RunDedicatedInferenceCreateToken,
		"create-token <dedicated-inference-id>",
		"Create an auth token for a dedicated inference endpoint",
		`Creates a new authentication token for a dedicated inference endpoint.
Use the `+"`"+`--token-name`+"`"+` flag to specify the name of the token.`,
		Writer,
		aliasOpt("ct"),
		displayerType(&displayers.DedicatedInferenceTokenDisplayer{}),
	)
	AddStringFlag(cmdCreateToken, doctl.ArgDedicatedInferenceTokenName, "", "", "Name for the auth token", requiredOpt())
	cmdCreateToken.Example = `The following example creates an auth token for a dedicated inference endpoint: doctl dedicated-inference create-token 12345678-1234-1234-1234-123456789012 --token-name my-token`

	cmdListTokens := CmdBuilder(
		cmd,
		RunDedicatedInferenceListTokens,
		"list-tokens <dedicated-inference-id>",
		"List auth tokens for a dedicated inference endpoint",
		`Lists all authentication tokens for a dedicated inference endpoint, including their IDs, names, and creation timestamps.
Note: token values are not returned when listing tokens.`,
		Writer,
		aliasOpt("lt"),
		displayerType(&displayers.DedicatedInferenceTokenDisplayer{}),
	)
	cmdListTokens.Example = `The following example lists auth tokens for a dedicated inference endpoint: doctl dedicated-inference list-tokens 12345678-1234-1234-1234-123456789012`

	cmdRevokeToken := CmdBuilder(
		cmd,
		RunDedicatedInferenceRevokeToken,
		"revoke-token <dedicated-inference-id> <token-id>",
		"Revoke an auth token for a dedicated inference endpoint",
		`Revokes (deletes) an authentication token for a dedicated inference endpoint.
Provide the dedicated inference ID and the token ID as arguments.`,
		Writer,
		aliasOpt("rt"),
	)
	AddBoolFlag(cmdRevokeToken, doctl.ArgForce, doctl.ArgShortForce, false, "Revoke the token without a confirmation prompt")
	cmdRevokeToken.Example = `The following example revokes an auth token: doctl dedicated-inference revoke-token 12345678-1234-1234-1234-123456789012 tok-abc123`

	cmdGetSizes := CmdBuilder(
		cmd,
		RunDedicatedInferenceGetSizes,
		"get-sizes",
		"List available dedicated inference GPU sizes and pricing",
		`Returns the available GPU sizes for dedicated inference endpoints, including pricing, region availability, CPU, memory, GPU, and disk details.`,
		Writer,
		aliasOpt("gs"),
		displayerType(&displayers.DedicatedInferenceSizeDisplayer{}),
	)
	cmdGetSizes.Example = `The following example lists available dedicated inference sizes: doctl dedicated-inference get-sizes`

	cmdGetGPUModelConfig := CmdBuilder(
		cmd,
		RunDedicatedInferenceGetGPUModelConfig,
		"get-gpu-model-config",
		"List supported GPU model configurations",
		`Returns the supported GPU model configurations for dedicated inference endpoints, including model slugs, names, compatible GPU slugs, and whether models are gated.`,
		Writer,
		aliasOpt("ggmc"),
		displayerType(&displayers.DedicatedInferenceGPUModelConfigDisplayer{}),
	)
	cmdGetGPUModelConfig.Example = `The following example lists GPU model configurations: doctl dedicated-inference get-gpu-model-config`

	return cmd
}

// readDedicatedInferenceSpec reads and parses a dedicated inference spec from a file path or stdin.
func readDedicatedInferenceSpec(stdin io.Reader, path string) (*godo.DedicatedInferenceSpecRequest, error) {
	var specReader io.Reader
	if path == "-" && stdin != nil {
		specReader = stdin
	} else {
		f, err := os.Open(path)
		if err != nil {
			if os.IsNotExist(err) {
				return nil, fmt.Errorf("opening spec: %s does not exist", path)
			}
			return nil, fmt.Errorf("opening spec: %w", err)
		}
		defer f.Close()
		specReader = f
	}

	byt, err := io.ReadAll(specReader)
	if err != nil {
		return nil, fmt.Errorf("reading spec: %w", err)
	}

	jsonSpec, err := yaml.YAMLToJSON(byt)
	if err != nil {
		return nil, fmt.Errorf("parsing spec: %w", err)
	}

	dec := json.NewDecoder(bytes.NewReader(jsonSpec))
	dec.DisallowUnknownFields()

	var spec godo.DedicatedInferenceSpecRequest
	if err := dec.Decode(&spec); err != nil {
		return nil, fmt.Errorf("parsing spec: %w", err)
	}

	return &spec, nil
}

// RunDedicatedInferenceCreate creates a new dedicated inference endpoint.
func RunDedicatedInferenceCreate(c *CmdConfig) error {
	specPath, err := c.Doit.GetString(c.NS, doctl.ArgDedicatedInferenceSpec)
	if err != nil {
		return err
	}

	spec, err := readDedicatedInferenceSpec(os.Stdin, specPath)
	if err != nil {
		return err
	}

	req := &godo.DedicatedInferenceCreateRequest{
		Spec: spec,
	}

	hfToken, _ := c.Doit.GetString(c.NS, doctl.ArgDedicatedInferenceHuggingFaceToken)
	if hfToken != "" {
		req.Secrets = &godo.DedicatedInferenceSecrets{
			HuggingFaceToken: hfToken,
		}
	}

	endpoint, _, err := c.DedicatedInferences().Create(req)
	if err != nil {
		return err
	}
	return c.Display(&displayers.DedicatedInference{DedicatedInferences: do.DedicatedInferences{*endpoint}})
}

// RunDedicatedInferenceGet retrieves a dedicated inference endpoint by ID.
func RunDedicatedInferenceGet(c *CmdConfig) error {
	if len(c.Args) < 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	id := c.Args[0]

	endpoint, err := c.DedicatedInferences().Get(id)
	if err != nil {
		return err
	}
	return c.Display(&displayers.DedicatedInference{DedicatedInferences: do.DedicatedInferences{*endpoint}})
}

// RunDedicatedInferenceList lists all dedicated inference endpoints.
func RunDedicatedInferenceList(c *CmdConfig) error {
	region, _ := c.Doit.GetString(c.NS, doctl.ArgDedicatedInferenceRegion)
	name, _ := c.Doit.GetString(c.NS, doctl.ArgDedicatedInferenceName)

	list, err := c.DedicatedInferences().List(region, name)
	if err != nil {
		return err
	}
	return c.Display(&displayers.DedicatedInferenceList{DedicatedInferenceListItems: list})
}

// RunDedicatedInferenceListAccelerators lists accelerators for a dedicated inference endpoint.
func RunDedicatedInferenceListAccelerators(c *CmdConfig) error {
	if len(c.Args) < 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	diID := c.Args[0]

	slug, _ := c.Doit.GetString(c.NS, doctl.ArgDedicatedInferenceAcceleratorSlug)

	accelerators, err := c.DedicatedInferences().ListAccelerators(diID, slug)
	if err != nil {
		return err
	}
	return c.Display(&displayers.DedicatedInferenceAccelerator{DedicatedInferenceAcceleratorInfos: accelerators})
}

// RunDedicatedInferenceUpdate updates an existing dedicated inference endpoint.
func RunDedicatedInferenceUpdate(c *CmdConfig) error {
	if len(c.Args) < 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	id := c.Args[0]

	specPath, err := c.Doit.GetString(c.NS, doctl.ArgDedicatedInferenceSpec)
	if err != nil {
		return err
	}

	spec, err := readDedicatedInferenceSpec(os.Stdin, specPath)
	if err != nil {
		return err
	}

	req := &godo.DedicatedInferenceUpdateRequest{
		Spec: spec,
	}

	hfToken, _ := c.Doit.GetString(c.NS, doctl.ArgDedicatedInferenceHuggingFaceToken)
	if hfToken != "" {
		req.Secrets = &godo.DedicatedInferenceSecrets{
			HuggingFaceToken: hfToken,
		}
	}

	endpoint, err := c.DedicatedInferences().Update(id, req)
	if err != nil {
		return err
	}
	return c.Display(&displayers.DedicatedInference{DedicatedInferences: do.DedicatedInferences{*endpoint}})
}

// RunDedicatedInferenceCreateToken creates a new auth token for a dedicated inference endpoint.
func RunDedicatedInferenceCreateToken(c *CmdConfig) error {
	if len(c.Args) < 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	diID := c.Args[0]

	tokenName, err := c.Doit.GetString(c.NS, doctl.ArgDedicatedInferenceTokenName)
	if err != nil {
		return err
	}

	req := &godo.DedicatedInferenceTokenCreateRequest{
		Name: tokenName,
	}

	token, err := c.DedicatedInferences().CreateToken(diID, req)
	if err != nil {
		return err
	}
	return c.Display(&displayers.DedicatedInferenceTokenDisplayer{DedicatedInferenceTokens: []do.DedicatedInferenceToken{*token}})
}

// RunDedicatedInferenceListTokens lists all auth tokens for a dedicated inference endpoint.
func RunDedicatedInferenceListTokens(c *CmdConfig) error {
	if len(c.Args) < 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	diID := c.Args[0]

	tokens, err := c.DedicatedInferences().ListTokens(diID)
	if err != nil {
		return err
	}

	displayTokens := make([]do.DedicatedInferenceToken, len(tokens))
	for i := range tokens {
		displayTokens[i] = tokens[i]
	}
	return c.Display(&displayers.DedicatedInferenceTokenDisplayer{DedicatedInferenceTokens: displayTokens})
}

// RunDedicatedInferenceRevokeToken revokes an auth token for a dedicated inference endpoint.
func RunDedicatedInferenceRevokeToken(c *CmdConfig) error {
	if len(c.Args) < 2 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	diID := c.Args[0]
	tokenID := c.Args[1]

	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}

	if force || AskForConfirmDelete("dedicated inference token", 1) == nil {
		return c.DedicatedInferences().RevokeToken(diID, tokenID)
	}

	return errOperationAborted
}

// RunDedicatedInferenceGetSizes returns available dedicated inference sizes and pricing.
func RunDedicatedInferenceGetSizes(c *CmdConfig) error {
	_, sizes, err := c.DedicatedInferences().GetSizes()
	if err != nil {
		return err
	}
	return c.Display(&displayers.DedicatedInferenceSizeDisplayer{DedicatedInferenceSizes: sizes})
}

// RunDedicatedInferenceGetGPUModelConfig returns supported GPU model configurations.
func RunDedicatedInferenceGetGPUModelConfig(c *CmdConfig) error {
	configs, err := c.DedicatedInferences().GetGPUModelConfig()
	if err != nil {
		return err
	}
	return c.Display(&displayers.DedicatedInferenceGPUModelConfigDisplayer{DedicatedInferenceGPUModelConfigs: configs})
}

// RunDedicatedInferenceDelete deletes a dedicated inference endpoint by ID.
func RunDedicatedInferenceDelete(c *CmdConfig) error {
	if len(c.Args) < 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}

	if force || AskForConfirmDelete("dedicated inference endpoint", 1) == nil {
		id := c.Args[0]
		return c.DedicatedInferences().Delete(id)
	}

	return errOperationAborted
}
