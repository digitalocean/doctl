package displayers

import (
	"io"

	"github.com/digitalocean/doctl/do"
)

type Agent struct {
	Agents do.Agents
}
type ApiKeyInfo struct {
	ApiKeyInfo do.ApiKeys
}

var _ Displayable = &Agent{}
var _ Displayable = &ApiKeyInfo{}

func (v *Agent) JSON(out io.Writer) error {
	return writeJSON(v.Agents, out)
}
func (v *ApiKeyInfo) JSON(out io.Writer) error {
	return writeJSON(v.ApiKeyInfo, out)
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
		"Id":         "ID",
		"Name":       "Name",
		"Region":     "Region",
		"Project-id": "Project ID",
		"Model-id":   "Model ID",
		"CreatedAt":  "Created At",
		"UserId":     "User ID",
	}
}

func (a *Agent) KV() []map[string]any {
	if a == nil || a.Agents == nil {
		return []map[string]any{}
	}
	out := make([]map[string]any, 0, len(a.Agents))
	for _, agent := range a.Agents {
		modelID := ""
		if agent.Model != nil {
			modelID = agent.Model.Uuid
		}
		out = append(out, map[string]any{
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

func (a *ApiKeyInfo) Cols() []string {
	return []string{
		"Id",
		"Name",
		"CreatedBy",
		"SecretKey",
		"DeletedAt",
		"CreatedAt",
	}
}

func (a *ApiKeyInfo) ColMap() map[string]string {
	return map[string]string{
		"Id":        "ID",
		"Name":      "Name",
		"SecretKey": "Secret Key",
		"CreatedBy": "Created By",
		"DeletedAt": "Deleted At",
		"CreatedAt": "Created At",
	}
}

func (a *ApiKeyInfo) KV() []map[string]any {
	if a == nil || a.ApiKeyInfo == nil {
		return []map[string]any{}
	}
	out := make([]map[string]any, 0, len(a.ApiKeyInfo))
	for _, apikey := range a.ApiKeyInfo {

		out = append(out, map[string]any{
			"Id":        apikey.Uuid,
			"Name":      apikey.Name,
			"SecretKey": apikey.SecretKey,
			"CreatedBy": apikey.CreatedBy,
			"CreatedAt": apikey.CreatedAt,
			"DeletedAt": apikey.DeletedAt,
		})
	}
	return out
}
