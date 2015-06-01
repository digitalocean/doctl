package account

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"reflect"
	"testing"

	"github.com/bryanl/docli/docli"
	"github.com/codegangsta/cli"
	"github.com/digitalocean/godo"
)

type testCS struct {
	client *godo.Client
}

var testAccount = &godo.Account{
	DropletLimit:  10,
	Email:         "user@example.com",
	UUID:          "1234",
	EmailVerified: true,
}

func (cs *testCS) NewClient(_ string) *godo.Client {
	return cs.client
}

func TestAccountAction(t *testing.T) {
	var b bytes.Buffer
	app := cli.NewApp()
	app.Writer = bufio.NewWriter(&b)

	globalSet := flag.NewFlagSet("global test", 0)
	globalSet.String("token", "token", "token")

	set := flag.NewFlagSet("local test", 0)
	c := cli.NewContext(app, set, globalSet)

	accountDidGet := false

	client := &godo.Client{
		Account: &docli.AccountServiceMock{
			GetFn: func() (*godo.Account, *godo.Response, error) {
				accountDidGet = true
				return testAccount, nil, nil
			},
		},
	}

	cs := &testCS{client}

	docli.WithinTest(cs, func() {
		Action(c)
		if !accountDidGet {
			t.Errorf("Action() did not run")
		}
	})
}

func TestAccountGet(t *testing.T) {
	client := &godo.Client{
		Account: &docli.AccountServiceMock{
			GetFn: func() (*godo.Account, *godo.Response, error) {
				return testAccount, nil, nil
			},
		},
	}

	var b bytes.Buffer
	w := bufio.NewWriter(&b)

	AccountGet(client, w)
	w.Flush()

	var ar godo.Account
	err := json.Unmarshal(b.Bytes(), &ar)
	if err != nil {
		t.Fatalf("AccountGet() can't unmarshal: %v", err)
	}

	if got, expected := ar, *testAccount; !reflect.DeepEqual(got, expected) {
		t.Errorf("AccountGet() = %#v; expected %#v", got, expected)
	}
}

func TestAccountGet_APIError(t *testing.T) {
	client := &godo.Client{
		Account: &docli.AccountServiceMock{
			GetFn: func() (*godo.Account, *godo.Response, error) {
				return nil, nil, fmt.Errorf("an error")
			},
		},
	}

	var b bytes.Buffer
	w := bufio.NewWriter(&b)

	err := AccountGet(client, w)
	w.Flush()

	if err == nil {
		t.Errorf("AccountGet expected error")
	}

}
