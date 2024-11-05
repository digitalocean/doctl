package droplets

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/digitalocean/godo"
	"sigs.k8s.io/yaml"
)

func ReadDropletBackupPolicy(stdin io.Reader, path string) (*godo.DropletBackupPolicyRequest, error) {
	var policy io.Reader
	if path == "-" && stdin != nil {
		policy = stdin
	} else {
		specFile, err := os.Open(path) // guardrails-disable-line
		if err != nil {
			if os.IsNotExist(err) {
				return nil, fmt.Errorf("opening droplet backup policy: %s does not exist", path)
			}
			return nil, fmt.Errorf("opening droplet backup policy: %w", err)
		}
		defer specFile.Close()
		policy = specFile
	}

	byt, err := io.ReadAll(policy)
	if err != nil {
		return nil, fmt.Errorf("reading droplet backup policy: %w", err)
	}

	s, err := ParseDropletBackupPolicy(byt)
	if err != nil {
		return nil, fmt.Errorf("parsing droplet backup policy: %w", err)
	}

	return s, nil
}

func ParseDropletBackupPolicy(spec []byte) (*godo.DropletBackupPolicyRequest, error) {
	jsonSpec, err := yaml.YAMLToJSON(spec)
	if err != nil {
		return nil, err
	}

	dec := json.NewDecoder(bytes.NewReader(jsonSpec))
	dec.DisallowUnknownFields()

	var policy godo.DropletBackupPolicyRequest
	if err := dec.Decode(&policy); err != nil {
		return nil, err
	}

	return &policy, nil
}
