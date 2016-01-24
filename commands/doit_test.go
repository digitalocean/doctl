package commands

import (
	"testing"

	"github.com/bryanl/doit/do"
	"github.com/stretchr/testify/assert"
)

func Test_extractDropletIPs(t *testing.T) {
	d := do.Droplet{Droplet: &testDroplet}
	ips := d.IPs()
	assert.Equal(t, testDroplet.Networks.V4[0].IPAddress, ips[do.InterfacePublic])
	assert.Equal(t, testDroplet.Networks.V4[1].IPAddress, ips[do.InterfacePrivate])
}
