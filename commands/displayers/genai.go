package displayers

import (
	"fmt"
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

// IndexingJobDataSource displayer
type IndexingJobDataSource struct {
	IndexingJobDataSources do.IndexingJobDataSources
}

var _ Displayable = &IndexingJobDataSource{}

func (v *IndexingJobDataSource) JSON(out io.Writer) error {
	return writeJSON(v.IndexingJobDataSources, out)
}

func (v *IndexingJobDataSource) ColMap() map[string]string {
	return map[string]string{
		"DataSourceUuid":    "Data Source UUID",
		"Status":            "Status",
		"StartedAt":         "Started At",
		"CompletedAt":       "Completed At",
		"IndexedItemCount":  "Indexed Items",
		"FailedItemCount":   "Failed Items",
		"SkippedItemCount":  "Skipped Items",
		"RemovedItemCount":  "Removed Items",
		"IndexedFileCount":  "Indexed Files",
		"TotalFileCount":    "Total Files",
		"TotalBytes":        "Total Bytes",
		"TotalBytesIndexed": "Total Bytes Indexed",
		"ErrorMsg":          "Error Message",
		"ErrorDetails":      "Error Details",
	}
}

func (v *IndexingJobDataSource) Cols() []string {
	return []string{
		"DataSourceUuid",
		"Status",
		"StartedAt",
		"CompletedAt",
		"IndexedItemCount",
		"FailedItemCount",
		"SkippedItemCount",
		"IndexedFileCount",
		"TotalFileCount",
	}
}

func (v *IndexingJobDataSource) KV() []map[string]any {
	if v == nil || v.IndexingJobDataSources == nil {
		return []map[string]any{}
	}
	out := make([]map[string]any, 0, len(v.IndexingJobDataSources))
	for _, ds := range v.IndexingJobDataSources {
		startedAt := ""
		if ds.StartedAt != nil {
			startedAt = ds.StartedAt.String()
		}
		completedAt := ""
		if ds.CompletedAt != nil {
			completedAt = ds.CompletedAt.String()
		}

		out = append(out, map[string]any{
			"DataSourceUuid":    ds.DataSourceUuid,
			"Status":            ds.Status,
			"StartedAt":         startedAt,
			"CompletedAt":       completedAt,
			"IndexedItemCount":  ds.IndexedItemCount,
			"FailedItemCount":   ds.FailedItemCount,
			"SkippedItemCount":  ds.SkippedItemCount,
			"RemovedItemCount":  ds.RemovedItemCount,
			"IndexedFileCount":  ds.IndexedFileCount,
			"TotalFileCount":    ds.TotalFileCount,
			"TotalBytes":        ds.TotalBytes,
			"TotalBytesIndexed": ds.TotalBytesIndexed,
			"ErrorMsg":          ds.ErrorMsg,
			"ErrorDetails":      ds.ErrorDetails,
		})
	}
	return out
}

// IndexingJob displayer
type IndexingJob struct {
	IndexingJobs do.IndexingJobs
}

var _ Displayable = &IndexingJob{}

func (v *IndexingJob) JSON(out io.Writer) error {
	return writeJSON(v.IndexingJobs, out)
}

func (v *IndexingJob) ColMap() map[string]string {
	return map[string]string{
		"CompletedDatasources": "Completed Datasources",
		"CreatedAt":            "Created At",
		"DataSourceUuids":      "Data Source UUIDs",
		"FinishedAt":           "Finished At",
		"KnowledgeBaseUuid":    "Knowledge Base UUID",
		"Phase":                "Phase",
		"StartedAt":            "Started At",
		"Status":               "Status",
		"Tokens":               "Tokens",
		"TotalDatasources":     "Total Datasources",
		"TotalItemsFailed":     "Total Items Failed",
		"TotalItemsIndexed":    "Total Items Indexed",
		"TotalItemsSkipped":    "Total Items Skipped",
		"UpdatedAt":            "Updated At",
		"UUID":                 "UUID",
	}
}

func (v *IndexingJob) Cols() []string {
	return []string{
		"UUID",
		"KnowledgeBaseUuid",
		"Phase",
		"Status",
		"CompletedDatasources",
		"TotalDatasources",
		"Tokens",
		"TotalItemsIndexed",
		"TotalItemsFailed",
		"TotalItemsSkipped",
		"CreatedAt",
		"StartedAt",
		"FinishedAt",
		"UpdatedAt",
	}
}

func (v *IndexingJob) KV() []map[string]any {
	if v == nil || v.IndexingJobs == nil {
		return []map[string]any{}
	}
	out := make([]map[string]any, 0, len(v.IndexingJobs))

	for _, job := range v.IndexingJobs {
		o := map[string]any{
			"CompletedDatasources": job.CompletedDatasources,
			"CreatedAt":            job.CreatedAt,
			"DataSourceUuids":      job.DataSourceUuids,
			"FinishedAt":           job.FinishedAt,
			"KnowledgeBaseUuid":    job.KnowledgeBaseUuid,
			"Phase":                job.Phase,
			"StartedAt":            job.StartedAt,
			"Status":               job.Status,
			"Tokens":               job.Tokens,
			"TotalDatasources":     job.TotalDatasources,
			"TotalItemsFailed":     job.TotalItemsFailed,
			"TotalItemsIndexed":    job.TotalItemsIndexed,
			"TotalItemsSkipped":    job.TotalItemsSkipped,
			"UpdatedAt":            job.UpdatedAt,
			"UUID":                 job.Uuid,
		}
		out = append(out, o)
	}

	return out
}

type FunctionRoute struct {
	Agent do.Agent
}

var _ Displayable = &FunctionRoute{}

func (f *FunctionRoute) JSON(out io.Writer) error {
	return writeJSON(f.Agent.Functions, out)
}

func (f *FunctionRoute) Cols() []string {
	return []string{
		"Uuid",
		"Name",
		"Description",
		"FaasName",
		"FaasNamespace",
		"CreatedAt",
		"UpdatedAt",
	}
}

func (f *FunctionRoute) ColMap() map[string]string {
	return map[string]string{
		"Uuid":          "UUID",
		"Name":          "Name",
		"Description":   "Description",
		"FaasName":      "FaaS Name",
		"FaasNamespace": "FaaS Namespace",
		"CreatedAt":     "Created At",
		"UpdatedAt":     "Updated At",
	}
}

func (f *FunctionRoute) KV() []map[string]any {
	if f.Agent.Functions == nil {
		return []map[string]any{}
	}

	out := make([]map[string]any, 0, len(f.Agent.Functions))
	for _, fn := range f.Agent.Functions {
		out = append(out, map[string]any{
			"Uuid":          fn.Uuid,
			"Name":          fn.Name,
			"Description":   fn.Description,
			"FaasName":      fn.FaasName,
			"FaasNamespace": fn.FaasNamespace,
			"CreatedAt":     fn.CreatedAt,
			"UpdatedAt":     fn.UpdatedAt,
		})
	}
	return out
}

type AgentRoute struct {
	AgentRouteResponses []do.AgentRouteResponse
}

var _ Displayable = &AgentRoute{}

func (a *AgentRoute) JSON(out io.Writer) error {
	return writeJSON(a.AgentRouteResponses, out)
}

func (a *AgentRoute) Cols() []string {
	return []string{
		"Id",
		"ParentAgentId",
		"ChildAgentId",
		"Rollback",
	}
}

func (a *AgentRoute) ColMap() map[string]string {
	return map[string]string{
		"Id":            "Id",
		"ParentAgentId": "Parent Agent Id",
		"ChildAgentId":  "Child Agent Id",
		"Rollback":      "Rollback",
	}
}

func (a *AgentRoute) KV() []map[string]any {
	if a == nil || a.AgentRouteResponses == nil {
		return []map[string]any{}
	}
	out := make([]map[string]any, 0, len(a.AgentRouteResponses))

	for _, response := range a.AgentRouteResponses {
		o := map[string]any{
			"Id":            response.UUID,
			"ParentAgentId": response.ParentAgentUuid,
			"ChildAgentId":  response.ChildAgentUuid,
			"Rollback":      response.Rollback,
		}
		out = append(out, o)
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
		"CanRollback",
		"CreatedAt",
		"CreatedByEmail",
		"CurrentlyApplied",
		"ID",
		"ModelName",
		"Name",
		"VersionHash",
	}
}

func (a *AgentVersion) ColMap() map[string]string {
	return map[string]string{
		"AgentUuid":        "Agent UUID",
		"CanRollback":      "Can Rollback",
		"CreatedAt":        "Created At",
		"CreatedByEmail":   "Created By Email",
		"CurrentlyApplied": "Currently Applied",
		"ID":               "ID",
		"ModelName":        "Model Name",
		"Name":             "Name",
		"VersionHash":      "Version Hash",
	}
}

func (a *AgentVersion) KV() []map[string]any {
	if a == nil || a.AgentVersions == nil {
		return []map[string]any{}
	}
	out := make([]map[string]any, 0, len(a.AgentVersions))
	for _, v := range a.AgentVersions {

		out = append(out, map[string]any{
			"AgentUuid":        v.AgentUuid,
			"CanRollback":      v.CanRollback,
			"CreatedAt":        v.CreatedAt,
			"CreatedByEmail":   v.CreatedByEmail,
			"CurrentlyApplied": v.CurrentlyApplied,
			"ID":               v.ID,
			"ModelName":        v.ModelName,
			"Name":             v.Name,
			"VersionHash":      v.VersionHash,
		})
	}
	return out
}

type OpenAiApiKey struct {
	OpenAiApiKeys do.OpenAiApiKeys
}

var _ Displayable = &OpenAiApiKey{}

func (o *OpenAiApiKey) JSON(out io.Writer) error {
	return writeJSON(o.OpenAiApiKeys, out)
}

func (o *OpenAiApiKey) Cols() []string {
	return []string{
		"Name",
		"UUID",
		"CreatedAt",
		"CreatedBy",
		"UpdatedAt",
		"DeletedAt",
	}
}

func (o *OpenAiApiKey) ColMap() map[string]string {
	return map[string]string{
		"Name":      "Name",
		"UUID":      "UUID",
		"CreatedAt": "Created At",
		"CreatedBy": "Created By",
		"UpdatedAt": "Updated At",
		"DeletedAt": "Deleted At",
	}
}

func (o *OpenAiApiKey) KV() []map[string]any {
	if o == nil || o.OpenAiApiKeys == nil {
		return []map[string]any{}
	}
	out := make([]map[string]any, 0, len(o.OpenAiApiKeys))
	for _, key := range o.OpenAiApiKeys {
		out = append(out, map[string]any{
			"Name":      key.Name,
			"UUID":      key.Uuid,
			"CreatedAt": key.CreatedAt,
			"CreatedBy": key.CreatedBy,
			"UpdatedAt": key.UpdatedAt,
			"DeletedAt": key.DeletedAt,
		})
	}
	return out
}

type Model struct {
	Models []do.Model
}

var _ Displayable = &Model{}

func (m *Model) JSON(out io.Writer) error {
	return writeJSON(m.Models, out)
}
func (m *Model) Cols() []string {
	return []string{
		"Id",
		"Name",
		"Agreement",
		"CreatedAt",
		"UpdatedAt",
		"isFoundational",
		"ParentId",
		"UploadComplete",
		"URL",
		"Version",
	}
}

func (m *Model) ColMap() map[string]string {
	return map[string]string{
		"Id":             "ID",
		"Name":           "Name",
		"Agreement":      "Agreement",
		"CreatedAt":      "Created At",
		"UpdatedAt":      "Updated At",
		"isFoundational": "Is Foundational",
		"ParentId":       "Parent ID",
		"UploadComplete": "Upload Complete",
		"URL":            "URL",
		"Version":        "Version",
	}
}

func (m *Model) KV() []map[string]any {
	if m == nil {
		return []map[string]any{}
	}
	out := make([]map[string]any, 0, len(m.Models))
	for _, model := range m.Models {
		// Format Agreement field
		agreementName := ""
		if model.Agreement != nil {
			agreementName = model.Agreement.Name
		}

		// Format Version field
		versionString := ""
		if model.Version != nil {
			versionString = fmt.Sprintf("%d.%d.%d", model.Version.Major, model.Version.Minor, model.Version.Patch)
		}

		out = append(out, map[string]any{
			"Id":             model.Uuid,
			"Name":           model.Name,
			"Agreement":      agreementName,
			"CreatedAt":      model.CreatedAt,
			"UpdatedAt":      model.UpdatedAt,
			"isFoundational": model.IsFoundational,
			"ParentId":       model.ParentUuid,
			"UploadComplete": model.UploadComplete,
			"URL":            model.Url,
			"Version":        versionString,
		})
	}
	return out
}

type DatacenterRegion struct {
	DatacenterRegions do.DatacenterRegions
}

var _ Displayable = &DatacenterRegion{}

func (d *DatacenterRegion) JSON(out io.Writer) error {
	return writeJSON(d.DatacenterRegions, out)
}

func (d *DatacenterRegion) Cols() []string {
	return []string{
		"InferenceURL",
		"Region",
		"ServesBatch",
		"ServesInference",
		"StreamInferenceUrl",
	}
}

func (d *DatacenterRegion) ColMap() map[string]string {
	return map[string]string{
		"InferenceURL":       "Inference URL",
		"Region":             "Region",
		"ServesBatch":        "Serves Batch",
		"ServesInference":    "Serves Inference",
		"StreamInferenceUrl": "Stream Inference URL",
	}
}

func (d *DatacenterRegion) KV() []map[string]any {
	if d == nil || d.DatacenterRegions == nil {
		return []map[string]any{}
	}
	out := make([]map[string]any, 0, len(d.DatacenterRegions))
	for _, region := range d.DatacenterRegions {
		out = append(out, map[string]any{
			"InferenceURL":       region.InferenceUrl,
			"Region":             region.Region,
			"ServesBatch":        region.ServesBatch,
			"ServesInference":    region.ServesInference,
			"StreamInferenceUrl": region.StreamInferenceUrl,
		})
	}
	return out
}
