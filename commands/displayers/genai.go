package displayers

import (
	"io"

	"github.com/digitalocean/doctl/do"
)

type KnowledgeBase struct {
	KnowledgeBases do.KnowledgeBases
}

var _ Displayable = &KnowledgeBase{}

func (v *KnowledgeBase) JSON(out io.Writer) error {
	return writeJSON(v.KnowledgeBases, out)
}

func (v *KnowledgeBase) ColMap() map[string]string {
	return map[string]string{
		// Add appropriate column mappings here, for example:
		"AddedToAgentAt":     "AddedToAgentAt",
		"CreatedAt":          "CreatedAt",
		"DatabaseId":         "DatabaseId",
		"EmbeddingModelUuid": "EmbeddingModelUuid",
		"IsPublic":           "IsPublic",
		"LastIndexingJob":    "LastIndexingJob",
		"Name":               "Name",
		"Region":             "Region",
		"ProjectId":          "ProjectId",
		"Tags":               "Tags",
		"UpdatedAt":          "UpdatedAt",
		"UserId":             "UserId",
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
		return nil
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
			"Region":             kb.Name,
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
		"BucketName":           "BucketName",
		"CreatedAt":            "CreatedAt",
		"FileUploadDataSource": "FileUploadDataSource",
		"LastIndexingJob":      "LastIndexingJob",
		"ItemPath":             "ItemPath",
		"Region":               "Region",
		"SpacesDataSource":     "SpacesDataSource",
		"UpdatedAt":            "UpdatedAt",
		"UUID":                 "UUID",
		"WebCrawlerDataSource": "WebCrawlerDataSource",
	}
}
func (v *KnowledgeBaseDataSource) Cols() []string {
	return []string{
		// Add appropriate column names here, for example:
		"BucketName",
		"CreatedAt",
		"FileUploadDataSource",
		"ItemPath",
		"LastIndexingJob",
		"Region",
		"SpacesDataSource",
		"UpdatedAt",
		"UUID",
		"WebCrawlerDataSource",
	}
}

func (v *KnowledgeBaseDataSource) KV() []map[string]any {
	out := make([]map[string]any, 0, len(v.KnowledgeBaseDataSources))

	for _, kb := range v.KnowledgeBaseDataSources {
		o := map[string]any{
			"BucketName":           kb.BucketName,
			"CreatedAt":            kb.CreatedAt,
			"FileUploadDataSource": kb.FileUploadDataSource,
			"ItemPath":             kb.ItemPath,
			"LastIndexingJob":      kb.LastIndexingJob,
			"Region":               kb.Region,
			"SpacesDataSource":     kb.SpacesDataSource,
			"UpdatedAt":            kb.UpdatedAt,
			"UUID":                 kb.UUID,
			"WebCrawlerDataSource": kb.WebCrawlerDataSource,
		}
		out = append(out, o)
	}

	return out
}

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

func (a *Agent) KV() []map[string]any {
	out := make([]map[string]any, 0, len(a.Agents))
	for _, agent := range a.Agents {
		out = append(out, map[string]any{
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
