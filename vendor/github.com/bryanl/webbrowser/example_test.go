package webbrowser_test

import (
	"log"

	"github.com/bryanl/webbrowser"
)

func ExampleOpen() {
	err := webbrowser.Open("http://digitalocean.com", webbrowser.NewTab, true)
	if err != nil {
		log.Println("open failed:", err)
	}
}
