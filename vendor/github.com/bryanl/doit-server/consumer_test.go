package doitserver_test

import (
	"testing"

	"github.com/bryanl/doit-server"
)

func TestConsumers(t *testing.T) {
	consumers := doitserver.NewConsumers()
	_ = consumers.Get("abc")

	if got, want := consumers.Len(), 1; got != want {
		t.Fatalf("Len() = %d; want = %d", got, want)
	}

	consumers.Remove("abc")

	if got, want := consumers.Len(), 0; got != want {
		t.Fatalf("Len() = %d; want = %d", got, want)
	}
}
