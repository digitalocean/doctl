package displayers

import (
	"io"
	// "strings"

	"github.com/digitalocean/doctl/do"
)

type Agent struct {
	Agents do.Agents
}

var _ Displayable = &Agent{}

func (a *Agent) JSON(out io.Writer) error {
	return writeJSON(a.Agents, out)
}

func (a *Agent) Cols() []string {
	return []string{
		"Name",
		"Region",
		//"Description",
		//"Instruction",
		"Project-id",
		//"Model-id",
		"CreatedAt",
		"UserId",
	}
}

func (a *Agent) ColMap() map[string]string {
	return map[string]string{
		"Name":   "Name",
		"Region": "Region",
		//"Description": "Description",
		//"Instruction": "Instruction",
		"Project-id": "ProjectID",
		//"Model-id":   "ModelID",
		"CreatedAt": "CreatedAt",
		"UserId":    "UserId",
	}
}

func (a *Agent) KV() []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(a.Agents))
	for _, agent := range a.Agents {
		out = append(out, map[string]interface{}{
			"Name":   agent.Name,
			"Region": agent.Region,
			//"Description": agent.Description,
			//"Instruction": agent.Instruction,
			"Project-id": agent.ProjectId,
			//"Model-id":   agent.Model,
			"CreatedAt": agent.CreatedAt,
			"UserId":    agent.UserId,
		})
	}
	return out
}
