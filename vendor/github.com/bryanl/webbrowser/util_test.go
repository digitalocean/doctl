package webbrowser

import (
	"strings"
	"testing"
)

func Test_getEnv(t *testing.T) {
	cases := []struct {
		k   string
		in  []string
		out string
	}{
		{k: "foo", in: []string{"foo=bar"}, out: "bar"},
		{k: "bar", in: []string{"foo=bar"}, out: ""},
		{k: "in", in: []string{"foo=bar", "in=out"}, out: "out"},
	}

	for _, c := range cases {
		if got, want := getEnv(c.k, c.in), c.out; got != want {
			t.Errorf("getEnv(%q, %q) = %q; want = %q", c.k, strings.Join(c.in, ","), got, want)
		}
	}
}
