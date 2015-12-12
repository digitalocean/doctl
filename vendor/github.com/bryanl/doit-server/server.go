package doitserver

import (
	"net/http"

	"github.com/gorilla/pat"
)

type AuthCredentials struct {
	ID string `json:"id"`
	CS string `json:"cs"`
}

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	ID          string `json:"id"`
	Err         string `json:"err"`
	Message     string `json:"msg"`
}

type Server struct {
	Consumers *Consumers
	Mux       http.Handler

	key string
}

func NewServer(key string) *Server {
	cc := NewConsumers()
	p := pat.New()

	a := NewAuth(key)
	ac := NewAuthCallback(cc, key)
	t := NewTokenGenerator(key)
	am := NewAuthMonitor(cc, key)

	p.Add("GET", "/auth/{provider}/callback", ac)
	p.Add("GET", "/auth/{provider}", a)
	p.Add("GET", "/token", t)
	p.Add("GET", "/status", am)

	return &Server{
		Consumers: cc,
		Mux:       p,
		key:       key,
	}
}
