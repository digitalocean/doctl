package commands

import (
	"testing"

	"github.com/bryanl/doit/do"
	"github.com/stretchr/testify/assert"
)

func Test_extractDropletIPs(t *testing.T) {
	ips := extractDropletIPs(&do.Droplet{Droplet: &testDroplet})
	assert.Equal(t, testDroplet.Networks.V4[0].IPAddress, ips["public"])
	assert.Equal(t, testDroplet.Networks.V4[1].IPAddress, ips["private"])
}
