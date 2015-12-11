package main

import (
	"log"

	"github.com/bryanl/doit/commands"
)

func main() {
	log.SetPrefix("doit: ")
	commands.Execute()
}
