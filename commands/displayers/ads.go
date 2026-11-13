/*
Copyright 2018 The Doctl Authors All rights reserved.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
	http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package displayers

import (
	"fmt"
	"io"
	"strings"

	"github.com/digitalocean/godo"
)

// ADSPlan displays the regions and query engine sizes available to ADS workspaces.
type ADSPlan struct {
	Plans []godo.ADSPlan
}

var _ Displayable = &ADSPlan{}

func (d *ADSPlan) Cols() []string {
	return []string{"Region", "AvailableForCreate", "QueryEngineSizes"}
}

func (d *ADSPlan) ColMap() map[string]string {
	return map[string]string{
		"Region":             "Region",
		"AvailableForCreate": "Available For Create",
		"QueryEngineSizes":   "Query Engine Sizes",
	}
}

func (d *ADSPlan) KV() []map[string]any {
	rows := make([]map[string]any, 0, len(d.Plans))
	for _, plan := range d.Plans {
		sizes := make([]string, 0, len(plan.QueryEngineSizes))
		for _, size := range plan.QueryEngineSizes {
			sizes = append(sizes, size.Size)
		}
		rows = append(rows, map[string]any{
			"Region":             plan.RegionSlug,
			"AvailableForCreate": plan.AvailableForCreate,
			"QueryEngineSizes":   strings.Join(sizes, ","),
		})
	}
	return rows
}

func (d *ADSPlan) JSON(out io.Writer) error {
	return writeJSON(d.Plans, out)
}

// ADSWorkspace displays ADS workspaces.
type ADSWorkspace struct {
	Workspaces []godo.ADSWorkspace
}

var _ Displayable = &ADSWorkspace{}

func (d *ADSWorkspace) Cols() []string {
	return []string{"UUID", "Name", "Region", "Status", "QueryInstanceSize", "QueryEndpoint"}
}

func (d *ADSWorkspace) ColMap() map[string]string {
	return map[string]string{
		"UUID":              "UUID",
		"Name":              "Name",
		"Region":            "Region",
		"Status":            "Status",
		"QueryInstanceSize": "Query Instance Size",
		"QueryEndpoint":     "Query Endpoint",
		"OwnerUUID":         "Owner UUID",
	}
}

func (d *ADSWorkspace) KV() []map[string]any {
	rows := make([]map[string]any, 0, len(d.Workspaces))
	for _, ws := range d.Workspaces {
		rows = append(rows, map[string]any{
			"UUID":              ws.UUID,
			"Name":              ws.Name,
			"Region":            ws.Region,
			"Status":            ws.Status,
			"QueryInstanceSize": ws.QueryInstanceSize,
			"QueryEndpoint":     ws.QueryEndpoint,
			"OwnerUUID":         ws.OwnerUUID,
		})
	}
	return rows
}

func (d *ADSWorkspace) JSON(out io.Writer) error {
	return writeJSON(d.Workspaces, out)
}

// ADSQueryCredentials displays the query user and password of an ADS
// workspace. The password is a secret; callers print a warning alongside it.
type ADSQueryCredentials struct {
	Credentials godo.ADSQueryCredentials
}

var _ Displayable = &ADSQueryCredentials{}

func (d *ADSQueryCredentials) Cols() []string {
	return []string{"User", "Password"}
}

func (d *ADSQueryCredentials) ColMap() map[string]string {
	return map[string]string{
		"User":     "User",
		"Password": "Password",
	}
}

func (d *ADSQueryCredentials) KV() []map[string]any {
	return []map[string]any{{
		"User":     d.Credentials.User,
		"Password": d.Credentials.Password,
	}}
}

func (d *ADSQueryCredentials) JSON(out io.Writer) error {
	return writeJSON(d.Credentials, out)
}

// ADSTokenCreated displays a newly created ADS token, including the access
// token that the API returns only once.
type ADSTokenCreated struct {
	Token godo.ADSTokenCreateResponse
}

var _ Displayable = &ADSTokenCreated{}

func (d *ADSTokenCreated) Cols() []string {
	return []string{"ID", "Name", "AccessToken", "ExpiresAt"}
}

func (d *ADSTokenCreated) ColMap() map[string]string {
	return map[string]string{
		"ID":          "ID",
		"Name":        "Name",
		"AccessToken": "Access Token",
		"ExpiresAt":   "Expires At",
	}
}

func (d *ADSTokenCreated) KV() []map[string]any {
	return []map[string]any{{
		"ID":          d.Token.ID,
		"Name":        d.Token.Name,
		"AccessToken": d.Token.AccessToken,
		"ExpiresAt":   d.Token.ExpiresAt,
	}}
}

func (d *ADSTokenCreated) JSON(out io.Writer) error {
	return writeJSON(d.Token, out)
}

// ADSToken displays ADS token summaries. Summaries never carry the access token.
type ADSToken struct {
	Tokens []godo.ADSTokenSummary
}

var _ Displayable = &ADSToken{}

func (d *ADSToken) Cols() []string {
	return []string{"ID", "Name", "Purpose", "WorkspaceUUIDs", "CreatedAt", "ExpiresAt"}
}

func (d *ADSToken) ColMap() map[string]string {
	return map[string]string{
		"ID":             "ID",
		"Name":           "Name",
		"Purpose":        "Purpose",
		"WorkspaceUUIDs": "Workspace UUIDs",
		"QueryHosts":     "Query Hosts",
		"CreatedAt":      "Created At",
		"ExpiresAt":      "Expires At",
	}
}

func (d *ADSToken) KV() []map[string]any {
	rows := make([]map[string]any, 0, len(d.Tokens))
	for _, token := range d.Tokens {
		rows = append(rows, map[string]any{
			"ID":             token.ID,
			"Name":           token.Name,
			"Purpose":        adsTokenPurpose(token.Purpose),
			"WorkspaceUUIDs": strings.Join(token.WorkspaceUUIDs, ","),
			"QueryHosts":     strings.Join(token.QueryHosts, ","),
			"CreatedAt":      token.CreatedAt,
			"ExpiresAt":      token.ExpiresAt,
		})
	}
	return rows
}

func (d *ADSToken) JSON(out io.Writer) error {
	return writeJSON(d.Tokens, out)
}

func adsTokenPurpose(purpose godo.ADSTokenPurpose) string {
	if purpose == godo.ADSTokenPurposeQuery {
		return "query"
	}
	return string(purpose)
}

// ADSSecretWrite displays the metadata of a secret version written by
// `ads secret put`. It never carries the values that were written.
type ADSSecretWrite struct {
	Key      string
	Metadata *godo.ADSSecretMetadata
}

var _ Displayable = &ADSSecretWrite{}

func (d *ADSSecretWrite) Cols() []string {
	return []string{"Key", "Version", "CreatedTime"}
}

func (d *ADSSecretWrite) ColMap() map[string]string {
	return map[string]string{
		"Key":         "Key",
		"Version":     "Version",
		"CreatedTime": "Created Time",
	}
}

func (d *ADSSecretWrite) KV() []map[string]any {
	row := map[string]any{"Key": d.Key, "Version": "", "CreatedTime": ""}
	if d.Metadata != nil {
		row["Version"] = d.Metadata.Version
		row["CreatedTime"] = d.Metadata.CreatedTime
	}
	return []map[string]any{row}
}

func (d *ADSSecretWrite) JSON(out io.Writer) error {
	return writeJSON(struct {
		Key      string                  `json:"key"`
		Metadata *godo.ADSSecretMetadata `json:"metadata,omitempty"`
	}{d.Key, d.Metadata}, out)
}

// ADSSecret displays an ADS workspace secret's metadata and property names.
// Property values are never displayed, in table or JSON output.
type ADSSecret struct {
	Key      string
	Keys     []string
	Metadata *godo.ADSSecretMetadata
}

var _ Displayable = &ADSSecret{}

func (d *ADSSecret) Cols() []string {
	return []string{"Key", "Properties", "Version", "CreatedTime", "Destroyed"}
}

func (d *ADSSecret) ColMap() map[string]string {
	return map[string]string{
		"Key":          "Key",
		"Properties":   "Properties",
		"Version":      "Version",
		"CreatedTime":  "Created Time",
		"DeletionTime": "Deletion Time",
		"Destroyed":    "Destroyed",
	}
}

func (d *ADSSecret) KV() []map[string]any {
	row := map[string]any{
		"Key":          d.Key,
		"Properties":   strings.Join(d.Keys, ","),
		"Version":      "",
		"CreatedTime":  "",
		"DeletionTime": "",
		"Destroyed":    false,
	}
	if d.Metadata != nil {
		row["Version"] = d.Metadata.Version
		row["CreatedTime"] = d.Metadata.CreatedTime
		row["DeletionTime"] = d.Metadata.DeletionTime
		row["Destroyed"] = d.Metadata.Destroyed
	}
	return []map[string]any{row}
}

func (d *ADSSecret) JSON(out io.Writer) error {
	return writeJSON(struct {
		Key        string                  `json:"key"`
		Properties []string                `json:"properties"`
		Metadata   *godo.ADSSecretMetadata `json:"metadata,omitempty"`
	}{d.Key, d.Keys, d.Metadata}, out)
}

// ADSSecretKey displays the secret key names of an ADS workspace.
type ADSSecretKey struct {
	Keys []string
}

var _ Displayable = &ADSSecretKey{}

func (d *ADSSecretKey) Cols() []string {
	return []string{"Key"}
}

func (d *ADSSecretKey) ColMap() map[string]string {
	return map[string]string{"Key": "Key"}
}

func (d *ADSSecretKey) KV() []map[string]any {
	rows := make([]map[string]any, 0, len(d.Keys))
	for _, key := range d.Keys {
		rows = append(rows, map[string]any{"Key": key})
	}
	return rows
}

func (d *ADSSecretKey) JSON(out io.Writer) error {
	return writeJSON(d.Keys, out)
}

// ADSDataFlow displays ADS data flows.
type ADSDataFlow struct {
	DataFlows []godo.ADSDataFlow
}

var _ Displayable = &ADSDataFlow{}

func (d *ADSDataFlow) Cols() []string {
	return []string{"UUID", "Name", "State", "DesiredState", "DesiredReplicas", "MaxReplicas", "UpdatedAt"}
}

func (d *ADSDataFlow) ColMap() map[string]string {
	return map[string]string{
		"UUID":            "UUID",
		"Name":            "Name",
		"State":           "State",
		"DesiredState":    "Desired State",
		"DesiredReplicas": "Desired Replicas",
		"MaxReplicas":     "Max Replicas",
		"Workspace":       "Workspace",
		"ImageRef":        "Image Ref",
		"FchainID":        "Function Chain ID",
		"LastError":       "Last Error",
		"CreatedAt":       "Created At",
		"UpdatedAt":       "Updated At",
	}
}

func (d *ADSDataFlow) KV() []map[string]any {
	rows := make([]map[string]any, 0, len(d.DataFlows))
	for _, flow := range d.DataFlows {
		rows = append(rows, map[string]any{
			"UUID":            flow.UUID,
			"Name":            flow.Name,
			"State":           flow.State,
			"DesiredState":    flow.DesiredState,
			"DesiredReplicas": flow.DesiredReplicas,
			"MaxReplicas":     flow.MaxReplicas,
			"Workspace":       flow.Workspace,
			"ImageRef":        flow.ImageRef,
			"FchainID":        flow.FchainID,
			"LastError":       flow.LastError,
			"CreatedAt":       flow.CreatedAt,
			"UpdatedAt":       flow.UpdatedAt,
		})
	}
	return rows
}

func (d *ADSDataFlow) JSON(out io.Writer) error {
	return writeJSON(d.DataFlows, out)
}

// ADSDataFlowChange displays the result of creating or updating an ADS data
// flow. The presigned upload URL is available with --format or --output json.
type ADSDataFlowChange struct {
	UUID   string
	State  string
	Upload *godo.ADSTarballUpload
}

var _ Displayable = &ADSDataFlowChange{}

func (d *ADSDataFlowChange) Cols() []string {
	return []string{"UUID", "State", "UploadObjectKey", "UploadExpiresAt"}
}

func (d *ADSDataFlowChange) ColMap() map[string]string {
	return map[string]string{
		"UUID":            "UUID",
		"State":           "State",
		"UploadURL":       "Upload URL",
		"UploadObjectKey": "Upload Object Key",
		"UploadExpiresAt": "Upload Expires At",
	}
}

func (d *ADSDataFlowChange) KV() []map[string]any {
	row := map[string]any{
		"UUID":            d.UUID,
		"State":           d.State,
		"UploadURL":       "",
		"UploadObjectKey": "",
		"UploadExpiresAt": "",
	}
	if d.Upload != nil {
		row["UploadURL"] = d.Upload.URL
		row["UploadObjectKey"] = d.Upload.ObjectKey
		row["UploadExpiresAt"] = d.Upload.ExpiresAt
	}
	return []map[string]any{row}
}

func (d *ADSDataFlowChange) JSON(out io.Writer) error {
	return writeJSON(struct {
		UUID   string                 `json:"uuid,omitempty"`
		State  string                 `json:"state,omitempty"`
		Upload *godo.ADSTarballUpload `json:"upload,omitempty"`
	}{d.UUID, d.State, d.Upload}, out)
}

// ADSFchainUploadURL displays a presigned upload URL for a function-chain archive.
type ADSFchainUploadURL struct {
	Upload godo.ADSFchainUploadURLResponse
}

var _ Displayable = &ADSFchainUploadURL{}

func (d *ADSFchainUploadURL) Cols() []string {
	return []string{"ObjectKey", "ExpiresAt", "UploadURL"}
}

func (d *ADSFchainUploadURL) ColMap() map[string]string {
	return map[string]string{
		"ObjectKey": "Object Key",
		"ExpiresAt": "Expires At",
		"UploadURL": "Upload URL",
	}
}

func (d *ADSFchainUploadURL) KV() []map[string]any {
	return []map[string]any{{
		"ObjectKey": d.Upload.ObjectKey,
		"ExpiresAt": d.Upload.ExpiresAt,
		"UploadURL": d.Upload.UploadURL,
	}}
}

func (d *ADSFchainUploadURL) JSON(out io.Writer) error {
	return writeJSON(d.Upload, out)
}

// ADSMetrics displays the rows of an ADS workspace metrics query. Columns
// come from the response, so they differ by query.
type ADSMetrics struct {
	Metrics godo.ADSWorkspaceMetricsResponse
}

var _ Displayable = &ADSMetrics{}

func (d *ADSMetrics) Cols() []string {
	return d.Metrics.Columns
}

func (d *ADSMetrics) ColMap() map[string]string {
	cols := make(map[string]string, len(d.Metrics.Columns))
	for _, col := range d.Metrics.Columns {
		cols[col] = col
	}
	return cols
}

func (d *ADSMetrics) KV() []map[string]any {
	rows := make([]map[string]any, 0, len(d.Metrics.Rows))
	for _, row := range d.Metrics.Rows {
		kv := make(map[string]any, len(d.Metrics.Columns))
		for _, col := range d.Metrics.Columns {
			if v, ok := row[col]; ok && v != nil {
				kv[col] = fmt.Sprint(v)
			} else {
				kv[col] = ""
			}
		}
		rows = append(rows, kv)
	}
	return rows
}

func (d *ADSMetrics) JSON(out io.Writer) error {
	return writeJSON(d.Metrics, out)
}
