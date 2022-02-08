package do

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
)

// SandboxService is an interface for interacting with the sandbox plugin.
type SandboxService interface {
	Cmd(string, []string) (*exec.Cmd, error)
	Exec(*exec.Cmd) (SandboxOutput, error)
}

type sandboxService struct {
	sandboxJs  string
	sandboxDir string
	node       string
}

var _ SandboxService = &sandboxService{}

// SandboxOutput contains the output returned from calls to the sandbox plugin.
type SandboxOutput = struct {
	Table     []map[string]interface{} `json:"table,omitempty"`
	Captured  []string                 `json:"captured,omitempty"`
	Formatted []string                 `json:"formatted,omitempty"`
	Entity    interface{}              `json:"entity,omitempty"`
	Error     string                   `json:"error,omitempty"`
}

// NewSandboxService returns a configure SandboxService.
func NewSandboxService(sandboxJs string, sandboxDir string, node string) SandboxService {
	return &sandboxService{
		sandboxJs:  sandboxJs,
		sandboxDir: sandboxDir,
		node:       node,
	}
}

// Cmd builds an *exec.Cmd for calling into the sandbox plugin.
func (n *sandboxService) Cmd(command string, args []string) (*exec.Cmd, error) {
	args = append([]string{n.sandboxJs, command}, args...)
	cmd := exec.Command(n.node, args...)
	cmd.Env = append(os.Environ(), "NIMBELLA_DIR="+n.sandboxDir)

	return cmd, nil
}

// Exec executes an *exec.Cmd and captures its output in a SandboxOutput.
func (n *sandboxService) Exec(cmd *exec.Cmd) (SandboxOutput, error) {
	output, err := cmd.Output()
	if err != nil {
		// Ignore "errors" that are just non-zero exit.  The
		// sandbox uses this as a secondary indicator but the output
		// is still trustworthy (and includes error information inline)
		if _, ok := err.(*exec.ExitError); !ok {
			// Real error of some sort
			return SandboxOutput{}, err
		}
	}
	var result SandboxOutput
	err = json.Unmarshal(output, &result)
	if err != nil {
		return SandboxOutput{}, err
	}
	// Result is sound JSON but if it has an Error field the rest is not trustworthy
	if len(result.Error) > 0 {
		return SandboxOutput{}, errors.New(result.Error)
	}
	// Result is both sound and error free
	return result, nil
}
