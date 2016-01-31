// Check Trello OAuth Detail At: https://trello.com/docs/gettingstarted/oauth.html
package main

import (
	"flag"
	"fmt"
	"github.com/mrjones/oauth"
	"io/ioutil"
	"log"
	"net/http"
)

var (
	tokens map[string]*oauth.RequestToken
	c      *oauth.Consumer
)

func main() {

	tokens = make(map[string]*oauth.RequestToken)

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
		port *int = flag.Int(
			"port",
			8888,
			"Port to listen on.",
		)
	)

	flag.Parse()

	c = oauth.NewConsumer(
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

	http.HandleFunc("/", RedirectUserToTrello)
	http.HandleFunc("/maketoken", GetTrelloToken)
	u := fmt.Sprintf(":%d", *port)
	fmt.Printf("Listening on '%s'\n", u)
	http.ListenAndServe(u, nil)

}

func RedirectUserToTrello(w http.ResponseWriter, r *http.Request) {

	tokenUrl := fmt.Sprintf("http://%s/maketoken", r.Host)
	token, requestUrl, err := c.GetRequestTokenAndUrl(tokenUrl)
	if err != nil {
		log.Fatal(err)
	}
	tokens[token.Token] = token
	http.Redirect(w, r, requestUrl, http.StatusTemporaryRedirect)
}

func GetTrelloToken(w http.ResponseWriter, r *http.Request) {

	values := r.URL.Query()
	verificationCode := values.Get("oauth_verifier")
	tokenKey := values.Get("oauth_token")

	accessToken, err := c.AuthorizeToken(tokens[tokenKey], verificationCode)
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
	fmt.Fprintf(w, "My profiles at Trello are:\n"+string(bits))
}
