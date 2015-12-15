package doit

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func TestVersion(t *testing.T) {
	cases := []struct {
		v   Version
		s   string
		ver string
	}{
		// version with no label
		{
			v:   Version{Major: 0, Minor: 1, Patch: 2, Name: "Version"},
			s:   `doit version 0.1.2 "Version"`,
			ver: "0.1.2",
		},
		// version with label
		{
			v:   Version{Major: 0, Minor: 1, Patch: 2, Name: "Version", Label: "dev"},
			s:   `doit version 0.1.2-dev "Version"`,
			ver: "0.1.2-dev",
		},
		// version with label and build
		{
			v:   Version{Major: 0, Minor: 1, Patch: 2, Name: "Version", Label: "dev", Build: "12345"},
			s:   "doit version 0.1.2-dev \"Version\"\nGit commit hash: 12345",
			ver: "0.1.2-dev",
		},
	}

	for _, c := range cases {
		if got, want := c.v.String(), c.ver; got != want {
			t.Errorf("version string for %#v = %q; want = %q", c.v, got, want)
		}
		if got, want := c.v.Complete(), c.s; got != want {
			t.Errorf("complete version string for %#v = %q; want = %q", c.v, got, want)
		}
	}
}
