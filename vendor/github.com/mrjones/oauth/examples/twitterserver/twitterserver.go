// Similar to the twitter example, but using an HTTP server instead of
// the command line.  Illustrates the need to preserve requestTokens across
// requests.  This does it very dumbly (using a global variable).
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/mrjones/oauth"
)

var tokens map[string]*oauth.RequestToken
var c *oauth.Consumer

func main() {
	tokens = make(map[string]*oauth.RequestToken)

	var consumerKey *string = flag.String(
		"consumerkey",
		"",
		"Consumer Key from Twitter. See: https://dev.twitter.com/apps/new")

	var consumerSecret *string = flag.String(
		"consumersecret",
		"",
		"Consumer Secret from Twitter. See: https://dev.twitter.com/apps/new")

	var port *int = flag.Int(
		"port",
		8888,
		"Port to listen on.")

	flag.Parse()

	c = oauth.NewConsumer(
		*consumerKey,
		*consumerSecret,
		oauth.ServiceProvider{
			RequestTokenUrl:   "https://api.twitter.com/oauth/request_token",
			AuthorizeTokenUrl: "https://api.twitter.com/oauth/authorize",
			AccessTokenUrl:    "https://api.twitter.com/oauth/access_token",
		},
	)
	c.Debug(true)

	http.HandleFunc("/", RedirectUserToTwitter)
	http.HandleFunc("/maketoken", GetTwitterToken)
	u := fmt.Sprintf(":%d", *port)
	fmt.Printf("Listening on '%s'\n", u)
	http.ListenAndServe(u, nil)
}

func RedirectUserToTwitter(w http.ResponseWriter, r *http.Request) {
	tokenUrl := fmt.Sprintf("http://%s/maketoken", r.Host)
	token, requestUrl, err := c.GetRequestTokenAndUrl(tokenUrl)
	if err != nil {
		log.Fatal(err)
	}
	// Make sure to save the token, we'll need it for AuthorizeToken()
	tokens[token.Token] = token
	http.Redirect(w, r, requestUrl, http.StatusTemporaryRedirect)
}

func GetTwitterToken(w http.ResponseWriter, r *http.Request) {
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

	response, err := client.Get(
		"https://api.twitter.com/1.1/statuses/home_timeline.json?count=1")
	if err != nil {
		log.Fatal(err)
	}
	defer response.Body.Close()

	bits, err := ioutil.ReadAll(response.Body)
	fmt.Fprintf(w, "The newest item in your home timeline is: "+string(bits))
}
