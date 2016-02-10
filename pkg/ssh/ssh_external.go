package ssh

import (
	"os"
	"os/exec"
	"strconv"
)

func runExternalSSH(r *Runner) error {
	args := []string{}
	if r.KeyPath != "" {
		args = append(args, "-i", r.KeyPath)
	}

	sshHost := r.Host
	if r.User != "" {
		sshHost = r.User + "@" + sshHost
	}

	if r.Port > 0 {
		args = append(args, "-p", strconv.Itoa(r.Port))
	}

	args = append(args, sshHost)

	cmd := exec.Command("ssh", args...)

	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin

	return cmd.Run()
}
