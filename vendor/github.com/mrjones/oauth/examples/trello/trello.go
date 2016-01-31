// Check Trello OAuth Detail At: https://trello.com/docs/gettingstarted/oauth.html
package main

import (
	"flag"
	"fmt"
	"github.com/mrjones/oauth"
	"io/ioutil"
	"log"
	"os"
)

func Usage() {

	fmt.Println("Usage:")
	fmt.Print("go run examples/trello/trello.go")
	fmt.Print(" --consumerkey <consumerkey>")
	fmt.Println(" --consumersecret <consumersecret>")
	fmt.Println("In order to get your consumerkey and consumersecret, you must register an 'app' at https://trello.com:")
	fmt.Println("https://trello.com/1/appKey/generate")
}

func main() {

	var (
		consumerKey *string = flag.String(
			"consumerkey",
			"",
			"Consumer Key from Trello. See: https://trello.com/1/appKey/generate",
		)
		consumerSecret *string = flag.String(
			"consumersecret",
			"",
			"Consumer Secret from Trello. See: https://trello.com/1/appKey/generate",
		)
	)

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
			RequestTokenUrl:   "https://trello.com/1/OAuthGetRequestToken",
			AuthorizeTokenUrl: "https://trello.com/1/OAuthAuthorizeToken",
			AccessTokenUrl:    "https://trello.com/1/OAuthGetAccessToken",
		},
	)
	// App Name
	c.AdditionalAuthorizationUrlParams["name"] = "Trello OAuth"
	// Token Expiration - Default 30 days
	c.AdditionalAuthorizationUrlParams["expiration"] = "never"
	// Authorization Scope
	c.AdditionalAuthorizationUrlParams["scope"] = "read"

	c.Debug(true)

	requestToken, u, err := c.GetRequestTokenAndUrl("")
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

	response, err := client.Get("https://trello.com/1/members/me")
	if err != nil {
		log.Fatal(err)
	}
	defer response.Body.Close()

	bits, err := ioutil.ReadAll(response.Body)
	fmt.Println("My profiles at Trello are:\n" + string(bits))
}
