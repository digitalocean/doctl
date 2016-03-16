package main

import (
	"log"

	"github.com/bryanl/doit/commands"
)

func main() {
	log.SetPrefix("doctl: ")
	cmd := commands.Init()
	cmd.Execute()
}
