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
	cmdCreate.Example = `The following example creates a dedicated inference endpoint using a spec file: doctl dedicated-inference create --spec spec.yaml --hugging-face-token "hf_mytoken"`

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
