/*
Copyright 2026 The Doctl Authors All rights reserved.
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

// Command manifestkeys regenerates agents_manifest_keys_tree.go from the
// schema copy. It runs from the commands directory via `go generate`.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/digitalocean/doctl/commands/internal/manifestkeys"
)

func main() {
	schema := flag.String("schema", "testdata/agent.flat.schema.json", "path to the flat agent manifest schema copy")
	out := flag.String("out", "agents_manifest_keys_tree.go", "path of the generated Go file")
	flag.Parse()

	raw, err := os.ReadFile(*schema)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	src, err := manifestkeys.Generate(raw)
	if err != nil {
		fmt.Fprintln(os.Stderr, "manifestkeys:", err)
		os.Exit(1)
	}
	if err := os.WriteFile(*out, src, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
