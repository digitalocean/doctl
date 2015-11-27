package doitserver

import (
	"encoding/json"
	"net/http"
)

type TokenGenerator struct {
	key string
}

var _ http.Handler = &TokenGenerator{}

func NewTokenGenerator(key string) *TokenGenerator {
	return &TokenGenerator{key: key}
}

func (tg *TokenGenerator) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	ac := AuthCredentials{
		ID: id,
		CS: encodeID(id, tg.key),
	}

	_ = json.NewEncoder(w).Encode(ac)
}
