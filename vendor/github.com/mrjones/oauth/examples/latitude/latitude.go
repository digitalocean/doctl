// THIS NO LONGER WORKS!!
// Latitude is using OAuth 2.0 now.
package main

import (
	"flag"
	"fmt"
	"github.com/mrjones/oauth"
	"io/ioutil"
	"log"
)

const (
	CURRENT_LOCATION_URL = "https://www.googleapis.com/latitude/v1/currentLocation"
)

func Usage() {
	fmt.Println("Usage:")
	fmt.Print("go run examples/latitude/latitude.go")
	fmt.Print("  --consumerkey <consumerkey>")
	fmt.Print("  --consumersecret <consumersecret>")
	fmt.Println("  --apikey <apikey>")
	fmt.Println("")
}

func main() {
	var consumerKey *string = flag.String("consumerkey", "", "")
	var consumerSecret *string = flag.String("consumersecret", "", "")
	var apiKey *string = flag.String("apikey", "", "")
	flag.Parse()

	c := oauth.NewConsumer(
		*consumerKey,
		*consumerSecret,
		oauth.ServiceProvider{
			RequestTokenUrl:   "https://www.google.com/accounts/OAuthGetRequestToken",
			AuthorizeTokenUrl: "https://www.google.com/latitude/apps/OAuthAuthorizeToken",
			AccessTokenUrl:    "https://www.google.com/accounts/OAuthGetAccessToken",
		})

	c.AdditionalParams["scope"] = "https://www.googleapis.com/auth/latitude"
	requestToken, url, err := c.GetRequestTokenAndUrl("oob")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("(1) Go to: " + url + "&domain=mrjon.es&granularity=best&location=all")
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
		fmt.Sprintf("%s?key=%s", CURRENT_LOCATION_URL, *apiKey))
	if err != nil {
		log.Fatal(err)
	}

	defer response.Body.Close()

	bits, err := ioutil.ReadAll(response.Body)
	fmt.Println("Your latest location: " + string(bits))
}
