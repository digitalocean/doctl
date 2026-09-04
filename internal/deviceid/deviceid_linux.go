//go:build linux

package deviceid

import "os"

// Note: this is the systemd install UUID, not the SMBIOS hardware UUID
// (which is root-only on Linux at /sys/class/dmi/id/product_uuid).
var machineIDPaths = []string{
	"/etc/machine-id",
	"/var/lib/dbus/machine-id",
}

func read() string {
	for _, p := range machineIDPaths {
		b, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		if len(b) > 0 {
			return string(b)
		}
	}
	return ""
}
