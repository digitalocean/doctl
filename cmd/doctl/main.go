package main

import (
	"log"

	"github.com/bryanl/doit/commands"
)

func main() {
	log.SetPrefix("doit: ")
	cmd := commands.Init()
	cmd.Execute()
}
