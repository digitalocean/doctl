package apps

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/digitalocean/godo"
	"sigs.k8s.io/yaml"
)

func ReadAppSpec(stdin io.Reader, path string) (*godo.AppSpec, error) {
	var spec io.Reader
	if path == "-" && stdin != nil {
		spec = stdin
	} else {
		specFile, err := os.Open(path) // guardrails-disable-line
		if err != nil {
			if os.IsNotExist(err) {
				return nil, fmt.Errorf("opening app spec: %s does not exist", path)
			}
			return nil, fmt.Errorf("opening app spec: %w", err)
		}
		defer specFile.Close()
		spec = specFile
	}

	byt, err := ioutil.ReadAll(spec)
	if err != nil {
		return nil, fmt.Errorf("reading app spec: %w", err)
	}

	s, err := ParseAppSpec(byt)
	if err != nil {
		return nil, fmt.Errorf("parsing app spec: %w", err)
	}

	return s, nil
}

func ParseAppSpec(spec []byte) (*godo.AppSpec, error) {
	jsonSpec, err := yaml.YAMLToJSON(spec)
	if err != nil {
		return nil, err
	}

	dec := json.NewDecoder(bytes.NewReader(jsonSpec))
	dec.DisallowUnknownFields()

	var appSpec godo.AppSpec
	if err := dec.Decode(&appSpec); err != nil {
		return nil, err
	}

	return &appSpec, nil
}
