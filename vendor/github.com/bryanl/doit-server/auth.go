package doitserver

import (
	"net/http"

	"github.com/markbates/goth/gothic"
)

// UnknownClientError signifies that the client is unknown.
type UnknownClientError struct{}

func (err *UnknownClientError) Error() string {
	return "unknown client"
}

type Auth struct {
	key string
}

var _ http.Handler = &Auth{}

func NewAuth(key string) *Auth {
	return &Auth{
		key: key,
	}
}

func (a *Auth) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	cs := r.URL.Query().Get("cs")
	cliAuth := r.URL.Query().Get("cliauth")

	if encodeID(id, a.key) != cs {
		err := &UnknownClientError{}
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	session, err := gothic.Store.Get(r, "doit-server")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if cliAuth != "" {
		session.Values["cli-auth"] = "yes"
	}

	session.Values["current-auth"] = id
	err = session.Save(r, w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	gothic.BeginAuthHandler(w, r)
}
