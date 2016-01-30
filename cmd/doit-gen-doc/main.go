package main

import (
	"log"

	"github.com/bryanl/doit/commands"
	"github.com/spf13/cobra/doc"
)

func main() {
	log.SetPrefix("doit: ")
	cmd := commands.Init()
	cmd.DisableAutoGenTag = false

	doc.GenMarkdownTree(cmd, "./")
}
