package sshkeys

import "testing"

func TestSSHGetArguments(t *testing.T) {
	cases := []struct {
		name        string
		id          int
		fingerprint string
		fail        bool
	}{
		{
			"id specified",
			1,
			"",
			false,
		},
		{
			"fingerprint specified",
			0,
			"fingerprint",
			false,
		},
		{
			"both specified",
			1,
			"fingerprint",
			true,
		},
		{
			"neither specified",
			0,
			"",
			true,
		},
	}

	for _, c := range cases {
		err := IsValidGetArgs(c.id, c.fingerprint)

		if c.fail && err == nil {
			t.Errorf("case %q expected error, but not was returned", c.name)
		} else if !c.fail && err != nil {
			t.Errorf("case %q unexpected error: %v", c.name, err)
		}
	}
}
