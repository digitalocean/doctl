
/*
Copyright 2016 The Doctl Authors All rights reserved.
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

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/bryanl/doit/commands"
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

var (
	outputDir  = flag.String("outputDir", "", "output directory")
	pageLookup = map[string]string{}
)

func main() {
	flag.Parse()

	log.SetPrefix("doit: ")
	cmd := commands.Init()
	cmd.DisableAutoGenTag = true

	if _, err := os.Stat(*outputDir); os.IsNotExist(err) {
		log.Fatalf("output directory %q does not exist", *outputDir)
	}

	err := genTree(cmd, *outputDir, filePrepender)
	if err != nil {
		log.Fatalf("generate documentation tree: %v", err)
	}
}

func filePrepender(section, filename string) string {
	now := time.Now().Format(time.RFC3339)
	name := filepath.Base(filename)
	base := strings.TrimSuffix(name, path.Ext(name))
	log.Printf("base: %s\n", base)
	url := fmt.Sprintf("/commands/%s/%s/", section, strings.ToLower(base))

	pageLookup[strings.ToLower(base)] = url

	fm := frontMatter{
		"date":  now,
		"title": strings.Replace(base, "_", " ", -1),
		"slug":  base,
		"url":   url,
	}

	b, err := json.MarshalIndent(fm, "", "  ")
	if err != nil {
		log.Fatalf("unable to generate front matter: %v", err)
	}

	return string(b) + "\n\n"
}

func linkHandler(name string) string {
	base := strings.ToLower(strings.TrimSuffix(name, path.Ext(name)))
	url, ok := pageLookup[base]
	if !ok {
		log.Printf("*** no match for %s", base)
	}
	return url
}

func genTree(
	cmd *commands.Command,
	dir string,
	filePrepender func(string, string) string,
) error {
	for _, c := range cmd.ChildCommands() {
		if !c.IsAvailableCommand() || c.IsHelpCommand() {
			continue
		}
		if err := genTree(c, dir, filePrepender); err != nil {
			return err
		}
	}

	for _, cat := range cmd.DocCategories {
		var basename string
		if cmd.IsIndex {
			basename = "index.md"
		} else {
			basename = strings.Replace(cmd.CommandPath(), " ", "_", -1) + ".md"
		}

		baseDir := filepath.Join(dir, cat)
		if err := os.MkdirAll(baseDir, 0755); err != nil {
			return fmt.Errorf("create directory %q: %v", baseDir, err)
		}

		filename := filepath.Join(baseDir, basename)
		f, err := os.Create(filename)
		if err != nil {
			return err
		}
		defer f.Close()

		if _, err := io.WriteString(f, filePrepender(cat, filename)); err != nil {
			return err
		}
		if err := doc.GenMarkdownCustom(cmd.Command, f, linkHandler); err != nil {
			return err
		}

	}

	return nil
}

func contains(s string, matches []string) bool {
	for _, m := range matches {
		if s == m {
			return true
		}
	}

	return false
}

func childByName(cmd *cobra.Command, name string) *cobra.Command {
	for _, c := range cmd.Commands() {
		if c.Name() == name {
			return c
		}
	}
	return nil
}

type frontMatter map[string]interface{}
