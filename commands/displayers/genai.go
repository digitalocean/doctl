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

type AgentVersion struct {
	AgentVersions do.AgentVersions
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

var _ Displayable = &ApiKeyInfo{}

func (v *ApiKeyInfo) JSON(out io.Writer) error {
	return writeJSON(v.ApiKeyInfo, out)
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

var _ Displayable = &AgentVersion{}

func (v *AgentVersion) JSON(out io.Writer) error {
	return writeJSON(v.AgentVersions, out)
}

func (a *AgentVersion) Cols() []string {
	return []string{
		"AgentUuid",
		"AttachedChildAgents",
		"AttachedFunctions",
		"AttachedGuardrails",
		"AttachedKnowledgeBases",
		"CanRollback",
		"CreatedAt",
		"CreatedByEmail",
		"CurrentlyApplied",
		"Description",
		"ID",
		"Instruction",
		"K",
		"MaxTokens",
		"ModelName",
		"Name",
		"ProvideCitations",
		"RetrievalMethod",
		"Tags",
		"Temperature",
		"TopP",
		"TriggerAction",
		"VersionHash",
	}
}

func (a *AgentVersion) ColMap() map[string]string {
	return map[string]string{
		"AgentUuid":              "Agent UUID",
		"AttachedChildAgents":    "Attached Child Agents",
		"AttachedFunctions":      "Attached Functions",
		"AttachedGuardrails":     "Attached Guardrails",
		"AttachedKnowledgeBases": "Attached Knowledge Bases",
		"CanRollback":            "Can Rollback",
		"CreatedAt":              "Created At",
		"CreatedByEmail":         "Created By Email",
		"CurrentlyApplied":       "Currently Applied",
		"Description":            "Description",
		"ID":                     "ID",
		"Instruction":            "Instruction",
		"K":                      "K",
		"MaxTokens":              "Max Tokens",
		"ModelName":              "Model Name",
		"Name":                   "Name",
		"ProvideCitations":       "Provide Citations",
		"RetrievalMethod":        "Retrieval Method",
		"Tags":                   "Tags",
		"Temperature":            "Temperature",
		"TopP":                   "Top P",
		"TriggerAction":          "Trigger Action",
		"VersionHash":            "Version Hash",
	}
}

func (a *AgentVersion) KV() []map[string]any {
	if a == nil || a.AgentVersions == nil {
		return []map[string]any{}
	}
	out := make([]map[string]any, 0, len(a.AgentVersions))
	for _, v := range a.AgentVersions {

		out = append(out, map[string]any{
			"AgentUuid":              v.AgentUuid,
			"AttachedChildAgents":    v.AttachedChildAgents,
			"AttachedFunctions":      v.AttachedFunctions,
			"AttachedGuardrails":     v.AttachedGuardrails,
			"AttachedKnowledgeBases": v.AttachedKnowledgeBases,
			"CanRollback":            v.CanRollback,
			"CreatedAt":              v.CreatedAt,
			"CreatedByEmail":         v.CreatedByEmail,
			"CurrentlyApplied":       v.CurrentlyApplied,
			"Description":            v.Description,
			"ID":                     v.ID,
			"Instruction":            v.Instruction,
			"K":                      v.K,
			"MaxTokens":              v.MaxTokens,
			"ModelName":              v.ModelName,
			"Name":                   v.Name,
			"ProvideCitations":       v.ProvideCitations,
			"RetrievalMethod":        v.RetrievalMethod,
			"Tags":                   v.Tags,
			"Temperature":            v.Temperature,
			"TopP":                   v.TopP,
			"TriggerAction":          v.TriggerAction,
			"VersionHash":            v.VersionHash,
		})
	}
	return out
}
