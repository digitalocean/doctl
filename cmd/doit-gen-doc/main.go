package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/bryanl/doit/commands"
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
	outputDir = flag.String("outputDir", "./", "output directory")
)

func main() {
	flag.Parse()

	log.SetPrefix("doit: ")
	cmd := commands.Init()
	cmd.DisableAutoGenTag = false

	if _, err := os.Stat(*outputDir); os.IsNotExist(err) {
		log.Fatalf("output directory %q does not exist", *outputDir)
	}

	filePrepender := func(filename string) string {
		now := time.Now().Format(time.RFC3339)
		name := filepath.Base(filename)
		base := strings.TrimSuffix(name, path.Ext(name))
		url := "/commands/" + strings.ToLower(base) + "/"
		return fmt.Sprintf(fmTemplate, now, strings.Replace(base, "_", " ", -1), base, url)
	}

	linkHandler := func(name string) string {
		base := strings.TrimSuffix(name, path.Ext(name))
		return "/commands/" + strings.ToLower(base) + "/"
	}

	doc.GenMarkdownTreeCustom(cmd, *outputDir, filePrepender, linkHandler)
}
