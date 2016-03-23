package doitserver

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

type AuthMonitor struct {
	consumers *Consumers
	key       string
	upgrader  websocket.Upgrader
}

var _ http.Handler = &AuthMonitor{}

func NewAuthMonitor(consumers *Consumers, key string) *AuthMonitor {
	return &AuthMonitor{
		consumers: consumers,
		key:       key,
		upgrader:  websocket.Upgrader{},
	}
}

func (am *AuthMonitor) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := am.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer conn.Close()

	var m map[string]interface{}
	err = conn.ReadJSON(&m)
	if err != nil {
		log.Print("read id", err)
		return
	}

	id := m["id"].(string)
	cs := m["cs"].(string)

	if eid := encodeID(id, am.key); eid != cs {
		log.Printf("encoded id %q (%s) does not match checksum %q", id, eid, cs)
		return
	}

	c := am.consumers.Get(id)

	for {
		ar := <-c

		if ar.ID != id {
			continue
		}

		msg := TokenResponse{
			ID: ar.ID,
		}
		if ar.Err == "" {
			msg.AccessToken = ar.AccessToken
		} else {
			msg.Err = ar.Err
			msg.Message = ar.Message
		}

		if err := conn.WriteJSON(msg); err != nil {
			log.Print("could not write response:", err)
		}

		am.consumers.Remove(id)

		break
	}
}
