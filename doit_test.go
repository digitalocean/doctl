/*
Copyright 2016 The Doctl Authors All rights reserved.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package doctl

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func TestUserAgent(t *testing.T) {
	dv := DoitVersion
	defer func() {
		DoitVersion = dv
	}()

	DoitVersion = Version{Major: 0, Minor: 1, Patch: 2}

	assert.Equal(t, "doctl/0.1.2", userAgent())
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
			s:   "doctl version 0.1.2\nrelease 1.0.0 is available, check it out! ",
			ver: `0.1.2`,
			slr: slr2,
		},
		// version with dev label and released version
		{
			v:   Version{Major: 1, Minor: 0, Patch: 0, Label: "dev"},
			s:   "doctl version 1.0.0-dev\nrelease 1.0.0 is available, check it out! ",
			ver: `1.0.0-dev`,
			slr: slr2,
		},
		// version with release label and released version available
		{
			v:   Version{Major: 1, Minor: 0, Patch: 0, Label: "release"},
			s:   "doctl version 1.0.0-release",
			ver: `1.0.0-release`,
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
