package config

import (
	"testing"
	"time"

	"github.com/digitalocean/doctl"
	"github.com/stretchr/testify/assert"
)

func TestDoctlConfigSource(t *testing.T) {
	doctlConfig := doctl.NewTestConfig()
	doctlConfig.Set("", "interactive", true)
	doctlConfig.Set("last-seen-ago", "bufo", time.Minute)

	// doctl's IsSet implementation pretends namespaces don't exist
	t.Run("no namespace", func(t *testing.T) {
		cs := DoctlConfigSource(doctlConfig, "")
		assert.True(t, cs.IsSet("interactive"))
		assert.False(t, cs.IsSet("last-seen-ago.bufo")) // namespace not considered by IsSet
		assert.True(t, cs.IsSet("bufo"))

		assert.Equal(t, true, cs.GetBool("interactive"))
		assert.Equal(t, time.Minute, cs.GetDuration("last-seen-ago.bufo"))
		assert.Equal(t, time.Duration(0), cs.GetDuration("bufo")) // does not exist
	})

	t.Run("yes namespace", func(t *testing.T) {
		cs := DoctlConfigSource(doctlConfig, "last-seen-ago")
		assert.True(t, cs.IsSet("interactive"))
		assert.False(t, cs.IsSet("last-seen-ago.bufo")) // namespace not considered by IsSet
		assert.True(t, cs.IsSet("bufo"))

		assert.Equal(t, false, cs.GetBool("interactive"))                       // does not exist
		assert.Equal(t, time.Duration(0), cs.GetDuration("last-seen-ago.bufo")) // does not exist
		assert.Equal(t, time.Minute, cs.GetDuration("bufo"))
	})
}
