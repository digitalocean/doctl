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

type KnowledgeBase struct {
	KnowledgeBases do.KnowledgeBases
}

var _ Displayable = &KnowledgeBase{}

func (v *KnowledgeBase) JSON(out io.Writer) error {
	return writeJSON(v.KnowledgeBases, out)
}

func (v *KnowledgeBase) ColMap() map[string]string {
	return map[string]string{
		"AddedToAgentAt":     "Added To Agent At",
		"CreatedAt":          "Created At",
		"DatabaseId":         "Database Id",
		"EmbeddingModelUuid": "Embedding Model Uuid",
		"IsPublic":           "Is Public",
		"LastIndexingJob":    "Last Indexing Job",
		"Name":               "Name",
		"Region":             "Region",
		"ProjectId":          "Project Id",
		"Tags":               "Tags",
		"UpdatedAt":          "Updated At",
		"UserId":             "User Id",
		"UUID":               "UUID",
	}
}

func (v *KnowledgeBase) Cols() []string {
	return []string{
		"AddedToAgentAt",
		"CreatedAt",
		"DatabaseId",
		"IsPublic",
		"EmbeddingModelUuid",
		"LastIndexingJob",
		"Name",
		"Region",
		"ProjectId",
		"Tags",
		"UpdatedAt",
		"UserId",
		"UUID",
	}
}

func (v *KnowledgeBase) KV() []map[string]any {
	if v == nil || v.KnowledgeBases == nil {
		return []map[string]any{}
	}
	out := make([]map[string]any, 0, len(v.KnowledgeBases))

	for _, kb := range v.KnowledgeBases {
		o := map[string]any{
			"AddedToAgentAt":     kb.AddedToAgentAt,
			"CreatedAt":          kb.CreatedAt,
			"DatabaseId":         kb.DatabaseId,
			"EmbeddingModelUuid": kb.EmbeddingModelUuid,
			"IsPublic":           kb.IsPublic,
			"LastIndexingJob":    kb.LastIndexingJob,
			"Name":               kb.Name,
			"Region":             kb.Region,
			"ProjectId":          kb.ProjectId,
			"Tags":               kb.Tags,
			"UpdatedAt":          kb.UpdatedAt,
			"UserId":             kb.UserId,
			"UUID":               kb.Uuid,
		}
		out = append(out, o)
	}

	return out
}

type KnowledgeBaseDataSource struct {
	KnowledgeBaseDataSources do.KnowledgeBaseDataSources
}

var _ Displayable = &KnowledgeBaseDataSource{}

func (v *KnowledgeBaseDataSource) JSON(out io.Writer) error {
	return writeJSON(v.KnowledgeBaseDataSources, out)
}

func (v *KnowledgeBaseDataSource) ColMap() map[string]string {
	return map[string]string{
		"CreatedAt":            "Created At",
		"FileUploadDataSource": "File Upload Datasource",
		"LastIndexingJob":      "Last Indexing Job",
		"SpacesDataSource":     "Spaces Datasource",
		"UpdatedAt":            "Updated At",
		"UUID":                 "UUID",
		"WebCrawlerDataSource": "Web Crawler Datasource",
	}
}

func (v *KnowledgeBaseDataSource) Cols() []string {
	return []string{
		"CreatedAt",
		"FileUploadDataSource",
		"LastIndexingJob",
		"SpacesDataSource",
		"UpdatedAt",
		"UUID",
		"WebCrawlerDataSource",
	}
}

func (v *KnowledgeBaseDataSource) KV() []map[string]any {
	if v == nil || v.KnowledgeBaseDataSources == nil {
		return []map[string]any{}
	}
	out := make([]map[string]any, 0, len(v.KnowledgeBaseDataSources))

	for _, kb := range v.KnowledgeBaseDataSources {
		o := map[string]any{
			"CreatedAt":            kb.CreatedAt,
			"FileUploadDataSource": kb.FileUploadDataSource,
			"LastIndexingJob":      kb.LastIndexingJob,
			"SpacesDataSource":     kb.SpacesDataSource,
			"UpdatedAt":            kb.UpdatedAt,
			"UUID":                 kb.Uuid,
			"WebCrawlerDataSource": kb.WebCrawlerDataSource,
		}
		out = append(out, o)
	}

	return out
}
