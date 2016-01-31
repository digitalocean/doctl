package doitserver

import "testing"

func Test_encodeID(t *testing.T) {
	expected := "c3ab8ff13720e8ad9047dd39466b3c8974e592c2fa383d4a3960714caef0c4f2"
	if got, want := encodeID("foo", "bar"), expected; got != want {
		t.Fatalf("encodeID() = %q; want = %q", got, want)
	}
}
