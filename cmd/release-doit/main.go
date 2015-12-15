package main

import (
	"flag"
	"log"

	"github.com/bryanl/doit"
	"github.com/bryanl/doit/install"
)

var (
	ver    = flag.String("ver", doit.DoitVersion.String(), "doit version")
	path   = flag.String("path", "", "upload path")
	user   = flag.String("user", "", "bintray user")
	apikey = flag.String("apikey", "", "bintray apikey")
)

func main() {
	flag.Parse()

	if *path == "" {
		log.Fatal("path is required")
	}

	ui := install.UserInfo{
		User:   *user,
		Apikey: *apikey,
	}

	err := install.Upload(ui, *ver, *path)
	if err != nil {
		log.Fatalf("upload failed: %v", err)
	}
}
