package main

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/digitalocean/doctl/Godeps/_workspace/src/github.com/docker/docker/pkg/term"
	"github.com/digitalocean/doctl/Godeps/_workspace/src/golang.org/x/crypto/ssh"

	"github.com/digitalocean/doctl/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/digitalocean/doctl/Godeps/_workspace/src/github.com/digitalocean/godo"

	"github.com/digitalocean/doctl/Godeps/_workspace/src/golang.org/x/oauth2"
)

// SSHCommand allows users to ssh into their droplets
var SSHCommand = cli.Command{
	Name:   "ssh",
	Usage:  "<name> SSH into droplet.",
	Action: connect,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "key, k",
			Usage: "Path to SSH key",
		},
	},
}

func authMethods(ctx *cli.Context) (methods []ssh.AuthMethod, err error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Password: ")
	text, _ := reader.ReadString('\n')
	methods = append(methods, ssh.Password(text))

	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}

	keyPath := ctx.String("key")
	if keyPath == "" {
		keyPath = filepath.Join(usr.HomeDir, ".ssh", "id_rsa")
	}

	key, err := ioutil.ReadFile(keyPath)
	if err != nil {
		return
	}

	privateKey, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return
	}
	methods = append(methods, ssh.PublicKeys(privateKey))

	return
}

func connect(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		log.Fatal("Error: Must provide name of droplet.")
	}

	name := ctx.Args().First()

	tokenSource := &TokenSource{
		AccessToken: APIKey,
	}
	oauthClient := oauth2.NewClient(oauth2.NoContext, tokenSource)
	client := godo.NewClient(oauthClient)

	droplet, err := FindDropletByName(client, name)
	if err != nil {
		log.Fatal(err)
	}

	var ip string
	for _, n := range droplet.Networks.V4 {
		if n.Type == "public" {
			ip = n.IPAddress
		}
	}
	endpoint := fmt.Sprintf("%s:%d", ip, 22)

	methods, err := authMethods(ctx)
	if err != nil {
		log.Fatal(err)
	}

	user := "root"
	if strings.Contains(droplet.Image.Slug, "coreos") {
		user = "core"
	}

	c := &ssh.ClientConfig{
		User: user,
		Auth: methods,
	}

	conn, err := ssh.Dial("tcp", endpoint, c)
	if err != nil {
		log.Fatal("Unable to connect.", err.Error())
	}

	session, err := conn.NewSession()
	if err != nil {
		log.Fatal(err.Error())
	}

	session.Stdout = os.Stdout
	session.Stderr = os.Stderr
	session.Stdin = os.Stdin

	defer session.Close()

	modes := ssh.TerminalModes{
		ssh.ECHO: 1,
	}
	fd := os.Stdin.Fd()
	var (
		termWidth, termHeight int
	)

	if term.IsTerminal(fd) {
		oldState, err := term.MakeRaw(fd)
		if err != nil {
			log.Fatal(err)
		}

		defer term.RestoreTerminal(fd, oldState)

		winsize, err := term.GetWinsize(fd)
		if err != nil {
			termWidth = 80
			termHeight = 24
		} else {
			termWidth = int(winsize.Width)
			termHeight = int(winsize.Height)
		}
	}

	if err := session.RequestPty("xterm", termWidth, termHeight, modes); err != nil {
		session.Close()
		log.Fatalf("request for pseudo terminal failed: %s", err)
	}
	if err == nil {
		err = session.Shell()
	}
	if err != nil {
		log.Fatal(err)
	}

	err = session.Wait()
	if err != nil && err != io.EOF {
		// Ignore the error if it's an ExitError with an empty message,
		// this occurs when you do CTRL+c and then run exit cmd which isn't an
		// actual error.
		waitMsg, ok := err.(*ssh.ExitError)
		if ok && waitMsg.Msg() == "" {
			return
		}

		log.Fatal(err)
	}

}
