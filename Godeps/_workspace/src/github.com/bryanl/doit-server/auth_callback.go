package doitserver

import (
	"fmt"
	"net/http"

	"github.com/markbates/goth/gothic"
)

type AuthCallback struct {
	consumers *Consumers
	key       string
}

var _ http.Handler = &AuthCallback{}

func NewAuthCallback(consumers *Consumers, key string) *AuthCallback {
	return &AuthCallback{
		consumers: consumers,
		key:       key,
	}
}

func (ac *AuthCallback) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	session, err := gothic.Store.Get(r, "doit-server")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	id := session.Values["current-auth"].(string)
	c := ac.consumers.Get(id)

	user, err := gothic.CompleteUserAuth(w, r)
	if err != nil {
		c <- Consumer{
			ID:      id,
			Err:     err.Error(),
			Message: "unble to complete authorization",
		}

		return
	}

	c <- Consumer{
		ID:          id,
		AccessToken: user.AccessToken,
	}

	fmt.Fprintf(w, updateTemplate)
}

var updateTemplate = `
<!doctype html>
<html lang="en">
<body>
<script>
window.location = "https://github.com/bryanl/doit/blob/master/README.md";
</script>
</body>
</html>`
