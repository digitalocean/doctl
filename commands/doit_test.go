package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDoitCommand(t *testing.T) {
	addCommands()
	assert.NotNil(t, DoitCmd)
	assertCommandNames(t, DoitCmd, "account", "action", "auth", "domain", "droplet", "droplet-action", "floating-ip", "floating-ip-action", "image", "region", "size", "ssh", "ssh-key", "version")
}

func Test_extractDropletIPs(t *testing.T) {
	ips := extractDropletIPs(&testDroplet)
	assert.Equal(t, testDroplet.Networks.V4[0].IPAddress, ips["public"])
	assert.Equal(t, testDroplet.Networks.V4[1].IPAddress, ips["private"])
}
