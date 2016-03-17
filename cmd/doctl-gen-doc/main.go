package main

import (
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

const fmTemplate = `---
date: %s
title: "%s"
slug: %s
url: %s
---
`

var (
	outputDir = flag.String("outputDir", "", "output directory")
)

func main() {
	flag.Parse()

	log.SetPrefix("doit: ")
	cmd := commands.Init()
	cmd.DisableAutoGenTag = false

	if _, err := os.Stat(*outputDir); os.IsNotExist(err) {
		log.Fatalf("output directory %q does not exist", *outputDir)
	}

	mainPath := filepath.Join(*outputDir, "command-main")
	genTree(cmd, mainPath, filePrepender("main"), linkHandler, "compute")

	computeCmd := childByName(cmd, "compute")
	if computeCmd == nil {
		log.Fatalf("could not find compute command")
	}

	computePath := filepath.Join(*outputDir, "command-compute")
	if err := os.MkdirAll(computePath, 0755); err != nil {
		log.Fatalf("could not create %s: %v", computePath, err)
	}

	genTree(computeCmd, computePath, filePrepender("compute"), linkHandler)
}

func filePrepender(section string) func(string) string {
	return func(filename string) string {
		now := time.Now().Format(time.RFC3339)
		name := filepath.Base(filename)
		base := strings.TrimSuffix(name, path.Ext(name))
		log.Printf("base: %s\n", base)
		url := fmt.Sprintf("/commands/%s/%s/", section, strings.ToLower(base))

		var title string
		if strings.HasPrefix(base, "doctl_compute") {
			title = strings.TrimPrefix(base, "doctl_compute")
		} else {

		}

		// title := strings.Replace()
		return fmt.Sprintf(fmTemplate, now, strings.Replace(title, "_", " ", -1), base, url)
	}
}

func linkHandler(name string) string {
	base := strings.TrimSuffix(name, path.Ext(name))
	return "/commands/" + strings.ToLower(base) + "/"
}

func genTree(cmd *cobra.Command, dir string, filePrepender, linkHandler func(string) string, skip ...string) error {
	for _, c := range cmd.Commands() {
		if !c.IsAvailableCommand() || c.IsHelpCommand() || contains(c.Name(), skip) {
			continue
		}
		if err := genTree(c, dir, filePrepender, linkHandler); err != nil {
			return err
		}
	}

	basename := strings.Replace(cmd.CommandPath(), " ", "_", -1) + ".md"
	filename := filepath.Join(dir, basename)
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := io.WriteString(f, filePrepender(filename)); err != nil {
		return err
	}
	if err := doc.GenMarkdownCustom(cmd, f, linkHandler); err != nil {
		return err
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
