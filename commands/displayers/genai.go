package displayers

import (
	"io"

	"github.com/digitalocean/doctl/do"
)

type Agent struct {
	Agents do.Agents
}

var _ Displayable = &Agent{}

func (v *Agent) JSON(out io.Writer) error {
	return writeJSON(v.Agents, out)
}

func (a *Agent) Cols() []string {
	return []string{
		"Id",
		"Name",
		"Region",
		"Project-id",
		"Model-id",
		"CreatedAt",
		"UserId",
	}
}

func (a *Agent) ColMap() map[string]string {
	return map[string]string{
		"Id":         "Id",
		"Name":       "Name",
		"Region":     "Region",
		"Project-id": "ProjectID",
		"Model-id":   "ModelID",
		"CreatedAt":  "CreatedAt",
		"UserId":     "UserId",
	}
}

func (a *Agent) KV() []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(a.Agents))
	for _, agent := range a.Agents {
		modelID := ""
		if agent.Model != nil {
			modelID = agent.Model.Uuid
		}
		// tags := ""
		// if len(agent.Tags) > 0 {
		// 	tags = strings.Join(agent.Tags, ",")
		// }
		out = append(out, map[string]interface{}{
			"Id":         agent.Uuid,
			"Name":       agent.Name,
			"Region":     agent.Region,
			"Project-id": agent.ProjectId,
			"Model-id":   modelID,
			"CreatedAt":  agent.CreatedAt,
			"UserId":     agent.UserId,
		})
	}
	return out
}
