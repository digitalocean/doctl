package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"time"

	"github.com/mrjones/oauth"
)

func Usage() {
	fmt.Println("Usage:")
	fmt.Print("go run examples/twitter/twitter.go")
	fmt.Print("  --consumerkey <consumerkey>")
	fmt.Println("  --consumersecret <consumersecret>")
	fmt.Println("")
	fmt.Println("In order to get your consumerkey and consumersecret, you must register an 'app' at twitter.com:")
	fmt.Println("https://dev.twitter.com/apps/new")
}

func main() {
	var consumerKey *string = flag.String(
		"consumerkey",
		"",
		"Consumer Key from Twitter. See: https://dev.twitter.com/apps/new")

	var consumerSecret *string = flag.String(
		"consumersecret",
		"",
		"Consumer Secret from Twitter. See: https://dev.twitter.com/apps/new")

	var postUpdate *bool = flag.Bool(
		"postupdate",
		false,
		"If true, post a status update to the timeline")

	flag.Parse()

	if len(*consumerKey) == 0 || len(*consumerSecret) == 0 {
		fmt.Println("You must set the --consumerkey and --consumersecret flags.")
		fmt.Println("---")
		Usage()
		os.Exit(1)
	}

	c := oauth.NewConsumer(
		*consumerKey,
		*consumerSecret,
		oauth.ServiceProvider{
			RequestTokenUrl:   "https://api.twitter.com/oauth/request_token",
			AuthorizeTokenUrl: "https://api.twitter.com/oauth/authorize",
			AccessTokenUrl:    "https://api.twitter.com/oauth/access_token",
		})

	c.Debug(true)

	requestToken, u, err := c.GetRequestTokenAndUrl("oob")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("(1) Go to: " + u)
	fmt.Println("(2) Grant access, you should get back a verification code.")
	fmt.Println("(3) Enter that verification code here: ")

	verificationCode := ""
	fmt.Scanln(&verificationCode)

	accessToken, err := c.AuthorizeToken(requestToken, verificationCode)
	if err != nil {
		log.Fatal(err)
	}

	client, err := c.MakeHttpClient(accessToken)
	if err != nil {
		log.Fatal(err)
	}

	response, err := client.Get(
		"https://api.twitter.com/1.1/statuses/home_timeline.json?count=1")
	if err != nil {
		log.Fatal(err)
	}
	defer response.Body.Close()

	bits, err := ioutil.ReadAll(response.Body)
	fmt.Println("The newest item in your home timeline is: " + string(bits))

	if *postUpdate {
		status := fmt.Sprintf("Test post via the API using Go (http://golang.org/) at %s", time.Now().String())

		response, err = client.PostForm(
			"https://api.twitter.com/1.1/statuses/update.json",
			url.Values{"status": []string{status}})

		if err != nil {
			log.Fatal(err)
		}

		log.Printf("%v\n", response)
	}
}
