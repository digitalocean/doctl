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
	outputDir = flag.String("outputDir", "", "output directory")
)

func main() {
	flag.Parse()

	log.SetPrefix("doit: ")
	cmd := commands.Init()
	cmd.DisableAutoGenTag = true

	if _, err := os.Stat(*outputDir); os.IsNotExist(err) {
		log.Fatalf("output directory %q does not exist", *outputDir)
	}

	err := genTree(cmd, *outputDir, filePrepender, linkHandler)
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

	var title string
	if strings.HasPrefix(base, "doctl_compute") {
		title = strings.TrimPrefix(base, "doctl_compute")
	} else {

	}

	fm := frontMatter{
		"date":  now,
		"title": strings.Replace(title, "_", " ", -1),
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
	base := strings.TrimSuffix(name, path.Ext(name))
	return "/commands/" + strings.ToLower(base) + "/"
}

func genTree(
	cmd *commands.Command,
	dir string,
	filePrepender func(string, string) string,
	linkHandler func(string) string,
) error {
	for _, c := range cmd.ChildCommands() {
		if !c.IsAvailableCommand() || c.IsHelpCommand() {
			continue
		}
		if err := genTree(c, dir, filePrepender, linkHandler); err != nil {
			return err
		}
	}

	for _, cat := range cmd.DocCategories {
		basename := strings.Replace(cmd.CommandPath(), " ", "_", -1) + ".md"
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
