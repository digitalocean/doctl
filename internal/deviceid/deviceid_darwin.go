//go:build darwin

package deviceid

import (
	"context"
	"os/exec"
	"regexp"
	"time"
)

var ioregPath = "/usr/sbin/ioreg"

// ioreg output line: `"IOPlatformUUID" = "<uuid>"`
var ioPlatformUUID = regexp.MustCompile(`"IOPlatformUUID"\s*=\s*"([^"]+)"`)

func read() string {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	out, err := exec.CommandContext(ctx, ioregPath, "-d2", "-c", "IOPlatformExpertDevice").Output()
	if err != nil {
		return ""
	}
	m := ioPlatformUUID.FindSubmatch(out)
	if len(m) < 2 {
		return ""
	}
	return string(m[1])
}
