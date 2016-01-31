package doitserver

import (
	"fmt"
	"html/template"
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

	var isCLIAuth bool
	id := session.Values["current-auth"].(string)
	if _, ok := session.Values["cli-auth"]; ok {
		isCLIAuth = true
	}

	user, err := gothic.CompleteUserAuth(w, r)

	if !isCLIAuth {
		c := ac.consumers.Get(id)

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
		return
	}

	if err != nil {
		fmt.Fprintln(w, "Unable to retrieve access token")
		delete(session.Values, "cli-auth")
		_ = session.Save(r, w)
		return
	}

	delete(session.Values, "cli-auth")
	_ = session.Save(r, w)

	t, _ := template.New("cliTemplate").Parse(cliTemplate)
	t.Execute(w, user)
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

var cliTemplate = `
<!doctype html>
<html lang="en">
<body>
<p>
Please copy the access token, switch back to doit cli and it paste the following token:
</p>

<p>
<strong>{{ .AccessToken }}</strong>
</p>
</body>
</html>`
