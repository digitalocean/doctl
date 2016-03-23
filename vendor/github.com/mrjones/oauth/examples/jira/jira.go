// Note: I haven't had a chance to test that this works.  This example was
// graciously provided by https://github.com/zimmski
//
// You should be able to set your consumer key, and provide a public key in
// the Jira admin, as documented in:
// https://www.prodpad.com/2013/05/tech-tutorial-oauth-in-jira/
//
// To generate a public/private key pair, do something like:
// $ openssl genrsa -out private_key.pem 4096
// $ openssl rsa -pubout -in private_key.pem -out public_key.pem
// Upload the public key to Jira, and reference the private key via
// the --privatekeyfile flag.
package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/mrjones/oauth"
)

func Usage() {
	fmt.Println("Usage:")
	fmt.Print("go run examples/jira/jira.go")
	fmt.Print("  --consumerkey <consumerkey>")
	fmt.Print("  --privatekeyfile <privatekeyfile>")
	fmt.Print("  --jiraurl <jiraurl>")
	fmt.Println("")
}

func main() {
	var consumerKey *string = flag.String(
		"consumerkey",
		"",
		"Consumer Key from service provider.")

	var privateKeyFile *string = flag.String(
		"privatekeyfile",
		"",
		"File name of a PEM encoded private key.")

	var jiraUrl *string = flag.String(
		"jiraurl",
		"",
		"Base URL of the Jira service.")

	flag.Parse()

	if len(*consumerKey) == 0 || len(*privateKeyFile) == 0 || len(*jiraUrl) == 0 {
		fmt.Println("You must set the --consumerkey, --privatekeyfile and --jiraurl flags.")
		fmt.Println("---")
		Usage()
		os.Exit(1)
	}

	privateKeyFileContents, err := ioutil.ReadFile(*privateKeyFile)
	if err != nil {
		log.Fatal(err)
	}

	block, _ := pem.Decode([]byte(privateKeyFileContents))
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		log.Fatal(err)
	}

	c := oauth.NewRSAConsumer(
		*consumerKey,
		privateKey,
		oauth.ServiceProvider{
			RequestTokenUrl:   *jiraUrl + "/plugins/servlet/oauth/request-token",
			AuthorizeTokenUrl: *jiraUrl + "/plugins/servlet/oauth/authorize",
			AccessTokenUrl:    *jiraUrl + "/plugins/servlet/oauth/access-token",
			HttpMethod:        "POST",
		})

	c.Debug(true)

	c.HttpClient = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

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

	response, err := client.Get(*jiraUrl + "/rest/api/2/issue/BULK-1")
	if err != nil {
		log.Fatal(err)
	}
	defer response.Body.Close()

	bits, err := ioutil.ReadAll(response.Body)
	fmt.Println("Data: " + string(bits))

}
