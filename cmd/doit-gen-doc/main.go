package main

import (
	"flag"
	"log"
	"os"

	"github.com/bryanl/doit/commands"
	"github.com/spf13/cobra/doc"
)

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

	doc.GenMarkdownTree(cmd, *outputDir)
}
