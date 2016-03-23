package doit

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func TestVersion(t *testing.T) {
	slr1 := &stubLatestRelease{version: "0.1.0"}
	slr2 := &stubLatestRelease{version: "1.0.0"}

	cases := []struct {
		v   Version
		s   string
		ver string
		slr LatestVersioner
	}{
		// version with no label
		{
			v:   Version{Major: 0, Minor: 1, Patch: 2},
			s:   `doctl version 0.1.2`,
			ver: "0.1.2",
			slr: slr1,
		},
		// version with label
		{
			v:   Version{Major: 0, Minor: 1, Patch: 2, Label: "dev"},
			s:   `doctl version 0.1.2-dev`,
			ver: "0.1.2-dev",
			slr: slr1,
		},
		// version with label and build
		{
			v:   Version{Major: 0, Minor: 1, Patch: 2, Label: "dev", Build: "12345"},
			s:   "doctl version 0.1.2-dev\nGit commit hash: 12345",
			ver: "0.1.2-dev",
			slr: slr1,
		},
		// version with no label and higher released version
		{
			v:   Version{Major: 0, Minor: 1, Patch: 2},
			s:   "doctl version 0.1.2\n\"1.0.0\" is a newer release than \"0.1.2\"",
			ver: `0.1.2`,
			slr: slr2,
		},
	}

	for _, c := range cases {
		if got, want := c.v.String(), c.ver; got != want {
			t.Errorf("version string for %#v = %q; want = %q", c.v, got, want)
		}
		if got, want := c.v.Complete(c.slr), c.s; got != want {
			t.Errorf("complete version string for %#v = %q; want = %q", c.v, got, want)
		}
	}
}

type stubLatestRelease struct {
	version string
}

func (slr stubLatestRelease) LatestVersion() (string, error) {
	return slr.version, nil
}
