package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVersionCommand(t *testing.T) {
	cmd := Version()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd)
}

func TestVersion(t *testing.T) {
	cases := []struct {
		v version
		s string
	}{
		// version with no label
		{
			v: version{Major: 0, Minor: 1, Patch: 2, Name: "Version"},
			s: `doit version 0.1.2 "Version"`,
		},
		// version with label
		{
			v: version{Major: 0, Minor: 1, Patch: 2, Name: "Version", Label: "dev"},
			s: `doit version 0.1.2-dev "Version"`,
		},
		// version with label and build
		{
			v: version{Major: 0, Minor: 1, Patch: 2, Name: "Version", Label: "dev", Build: "12345"},
			s: "doit version 0.1.2-dev \"Version\"\nGit commit hash: 12345",
		},
	}

	for _, c := range cases {
		if got, want := c.v.String(), c.s; got != want {
			t.Errorf("Version String for %#v = %q; want = %q", c.v, got, want)
		}
	}
}
