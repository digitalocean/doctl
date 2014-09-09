package main

import (
	"encoding/json"
	"fmt"
	"os"

	"gopkg.in/yaml.v1"
)

func WriteOutput(data interface{}) {
	var output []byte
	var err error

	switch OutputFormat {
	case "json":
		output, err = json.Marshal(data)
		if err != nil {
			fmt.Printf("JSON Encoding Error: %s", err)
			os.Exit(1)
		}

	case "yaml":
		output, err = yaml.Marshal(data)
		if err != nil {
			fmt.Printf("YAML Encoding Error: %s", err)
			os.Exit(1)
		}
	}
	fmt.Printf("%s", string(output))
}
