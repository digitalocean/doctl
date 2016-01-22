package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_extractDropletIPs(t *testing.T) {
	ips := extractDropletIPs(&testDroplet)
	assert.Equal(t, testDroplet.Networks.V4[0].IPAddress, ips["public"])
	assert.Equal(t, testDroplet.Networks.V4[1].IPAddress, ips["private"])
}
