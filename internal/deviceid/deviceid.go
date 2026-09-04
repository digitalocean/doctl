// Package deviceid resolves the hardware UUID of the host running doctl for
// the hosted-agents API. Get returns "" on any failure; doctl never fails on
// a missing UUID. The lookup is cached per process.
package deviceid

import (
	"os"
	"strings"
	"sync"
)

// EnvDisable disables the lookup when set to "1", "true", or "yes".
const EnvDisable = "DOCTL_DISABLE_DEVICE_ID"

var (
	once   sync.Once
	cached string
)

// Get returns the host hardware UUID, or "" if unavailable / opted out.
func Get() string {
	once.Do(func() {
		if isDisabled() {
			return
		}
		cached = strings.TrimSpace(read())
	})
	return cached
}

// reset clears the cache. Test hook only.
func reset() {
	once = sync.Once{}
	cached = ""
}

func isDisabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(EnvDisable))) {
	case "1", "true", "yes":
		return true
	}
	return false
}
