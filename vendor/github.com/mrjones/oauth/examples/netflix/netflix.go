// NOTE: Netflix shut down its API in 2014.
//       This code no longer works.
//
// go run examples/netflix/netflix.go --consumerkey <key> --consumersecret <secret> --appname <appname>
package main

import (
	"encoding/xml"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/mrjones/oauth"
)

func main() {
	var consumerKey *string = flag.String(
		"consumerkey",
		"",
		"Consumer Key from NetFlix. See: http://developer.netflix.com/apps/mykeys")

	var consumerSecret *string = flag.String(
		"consumersecret",
		"",
		"Consumer Key from NetFlix. See: http://developer.netflix.com/apps/mykeys")

	var appName *string = flag.String(
		"appname",
		"",
		"Application name registered with NetFlix.")

	var debug *bool = flag.Bool(
		"debug",
		false,
		"If true, print debugging information")

	flag.Parse()

	if len(*consumerKey) == 0 || len(*consumerSecret) == 0 || len(*appName) == 0 {
		fmt.Println("You must set the --consumerkey, --consumersecret and --appname flags.")
		os.Exit(1)
	}

	c := oauth.NewConsumer(
		*consumerKey,
		*consumerSecret,
		oauth.ServiceProvider{
			RequestTokenUrl:   "http://api-public.netflix.com/oauth/request_token",
			AuthorizeTokenUrl: "https://api-user.netflix.com/oauth/login",
			AccessTokenUrl:    "http://api-public.netflix.com/oauth/access_token",
		})

	// See #4 here:
	// http://josephsmarr.com/2008/10/01/using-netflixs-new-api-a-step-by-step-guide/
	c.AdditionalAuthorizationUrlParams = map[string]string{
		"application_name":   *appName,
		"oauth_consumer_key": *consumerKey,
	}

	c.Debug(*debug)

	requestToken, url, err := c.GetRequestTokenAndUrl("oob")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("(1) Go to: " + url)
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
		"http://api-public.netflix.com/users/current")
	if err != nil {
		log.Fatal(err)
	}
	defer response.Body.Close()

	bits, err := ioutil.ReadAll(response.Body)
	profileXml := Resource{}
	xml.Unmarshal(bits, &profileXml)

	if len(profileXml.Link.Href) == 0 {
		fmt.Println("ERROR: Couldn't parse subscriber-id from: ", string(bits))
		return
	}

	recsUrl := fmt.Sprintf("%s/recommendations?max_results=1&start_index=0",
		profileXml.Link.Href)
	response, err = client.Get(recsUrl)
	if err != nil {
		log.Fatal(err)
	}
	defer response.Body.Close()

	bits, err = ioutil.ReadAll(response.Body)
	fmt.Println("NetFlix recommends: " + string(bits))
}

type Resource struct {
	Link Link `xml:"link"`
}

type Link struct {
	Href string `xml:"href,attr"`
}
