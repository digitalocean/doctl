//go:build windows

package deviceid

import (
	"context"
	"os/exec"
	"regexp"
	"time"
)

var uuidLine = regexp.MustCompile(`[0-9A-Fa-f]{8}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{12}`)

// read tries wmic (deprecated, removed from Win 11 24H2 fresh installs)
// then falls back to PowerShell's Get-CimInstance.
func read() string {
	if id := readViaWmic(); id != "" {
		return id
	}
	return readViaPowerShell()
}

func readViaWmic() string {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	out, err := exec.CommandContext(ctx, "wmic", "csproduct", "get", "UUID").Output()
	if err != nil {
		return ""
	}
	return string(uuidLine.Find(out))
}

// 5 s ceiling to absorb powershell.exe cold-start (~1–2 s).
func readViaPowerShell() string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	out, err := exec.CommandContext(ctx, "powershell",
		"-NoProfile",
		"-NonInteractive",
		"-Command",
		"(Get-CimInstance -ClassName Win32_ComputerSystemProduct).UUID",
	).Output()
	if err != nil {
		return ""
	}
	return string(uuidLine.Find(out))
}
