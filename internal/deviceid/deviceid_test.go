package deviceid

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGet_DisabledByEnv(t *testing.T) {
	t.Setenv(EnvDisable, "1")
	reset()
	assert.Equal(t, "", Get(), "Get must return empty when DOCTL_DISABLE_DEVICE_ID is set")
}

func TestGet_DisabledByEnv_VariantsAreCaseInsensitive(t *testing.T) {
	for _, v := range []string{"1", "true", "TRUE", "Yes", "  yes  "} {
		t.Run(v, func(t *testing.T) {
			t.Setenv(EnvDisable, v)
			reset()
			assert.Equal(t, "", Get())
		})
	}
}

func TestGet_NotDisabledByEnv_FalsyValuesIgnored(t *testing.T) {
	// "0", "false", "" must not be treated as opt-out. We can't assert what
	// Get returns (it depends on the host), but we can assert isDisabled
	// behavior directly, which is the only env-driven branch in Get.
	for _, v := range []string{"", "0", "false", "no", "off", "anything-else"} {
		t.Run(v, func(t *testing.T) {
			t.Setenv(EnvDisable, v)
			assert.False(t, isDisabled(), "value %q must not trip opt-out", v)
		})
	}
}

func TestGet_Cached(t *testing.T) {
	// Two calls must yield the same value; sync.Once guarantees read() is
	// invoked at most once. We rely on Get being deterministic for the
	// lifetime of the process.
	t.Setenv(EnvDisable, "")
	reset()
	first := Get()
	second := Get()
	assert.Equal(t, first, second)
}
