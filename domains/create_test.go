package domains

import "testing"

func TestCreateRequestIsValid(t *testing.T) {
	cases := []struct {
		n         string
		name      string
		ipAddress string
		output    bool
	}{
		{
			"valid params",
			"example.com",
			"127.0.0.1",
			true,
		},
		{
			"missing name",
			"",
			"127.0.0.1",
			false,
		},
		{
			"missing ip address",
			"example.com",
			"",
			false,
		},
		{
			"empty cr",
			"",
			"",
			false,
		},
	}

	for _, c := range cases {
		cr := CreateRequest{
			Name:      c.name,
			IPAddress: c.ipAddress,
		}

		if got, expected := cr.IsValid(), c.output; got != expected {
			t.Errorf("case %q = %v; expected %v", c.n, got, expected)
		}
	}
}
