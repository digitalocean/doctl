package doit

import (
	"github.com/digitalocean/godo"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
)

var (
	Bail func(err error, msg string)
)

func GetClient() *godo.Client {
	token := viper.GetString("token")
	tokenSource := &TokenSource{AccessToken: token}
	oauthClient := oauth2.NewClient(oauth2.NoContext, tokenSource)
	return godo.NewClient(oauthClient)
}
