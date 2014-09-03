package godo

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAction_List(t *testing.T) {
	setup()
	defer teardown()

	assert := assert.New(t)

	mux.HandleFunc("/v2/actions", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"actions": [{"id":1},{"id":2}]}`)
		testMethod(t, r, "GET")
	})

	actions, _, err := client.Actions.List()
	assert.NoError(err)
	expected := []Action{{ID: 1}, {ID: 2}}
	assert.Equal(expected, actions)
}

func TestAction_Get(t *testing.T) {
	setup()
	defer teardown()

	assert := assert.New(t)

	mux.HandleFunc("/v2/actions/12345", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"action": {"id":12345}}`)
		testMethod(t, r, "GET")
	})

	action, _, err := client.Actions.Get(12345)
	assert.NoError(err)
	assert.Equal(12345, action.ID)
}

func TestAction_String(t *testing.T) {
	assert := assert.New(t)
	pt, err := time.Parse(time.RFC3339, "2014-05-08T20:36:47Z")
	assert.NoError(err)

	startedAt := &Timestamp{
		Time: pt,
	}
	action := &Action{
		ID:        1,
		Status:    "in-progress",
		Type:      "transfer",
		StartedAt: startedAt,
	}

	stringified := action.String()
	expected := `godo.Action{ID:1, Status:"in-progress", Type:"transfer", ` +
		`StartedAt:godo.Timestamp{2014-05-08 20:36:47 +0000 UTC}, ` +
		`ResourceID:0, ResourceType:""}`
	if expected != stringified {
		t.Errorf("Action.Stringify returned %+v, expected %+v", stringified, expected)
	}
}
