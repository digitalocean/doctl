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
	"io"
	"strings"

	"github.com/digitalocean/doctl/do"
)

type VectorDBs struct {
	VectorDBs do.VectorDBs
	Short     bool
}

var _ Displayable = &VectorDBs{}

func (v *VectorDBs) JSON(out io.Writer) error {
	return writeJSON(v.VectorDBs, out)
}

func (v *VectorDBs) Cols() []string {
	if v.Short {
		return []string{
			"ID",
			"Name",
			"Region",
			"Status",
			"Size",
		}
	}

	return []string{
		"ID",
		"Name",
		"Region",
		"Status",
		"Size",
		"HTTPEndpoint",
		"GRPCEndpoint",
		"Tags",
		"Created",
	}
}

func (v *VectorDBs) ColMap() map[string]string {
	return map[string]string{
		"ID":           "ID",
		"Name":         "Name",
		"Region":       "Region",
		"Status":       "Status",
		"Size":         "Size",
		"HTTPEndpoint": "HTTP Endpoint",
		"GRPCEndpoint": "gRPC Endpoint",
		"Tags":         "Tags",
		"Created":      "Created At",
	}
}

func (v *VectorDBs) KV() []map[string]any {
	out := make([]map[string]any, 0, len(v.VectorDBs))

	for _, db := range v.VectorDBs {
		o := map[string]any{
			"ID":      db.ID,
			"Name":    db.Name,
			"Region":  db.Region,
			"Status":  db.Status,
			"Size":    db.Size,
			"Tags":    strings.Join(db.Tags, ","),
			"Created": db.CreatedAt,
		}
		if db.Endpoints != nil {
			o["HTTPEndpoint"] = db.Endpoints.HTTP
			o["GRPCEndpoint"] = db.Endpoints.GRPC
		}
		out = append(out, o)
	}

	return out
}

type VectorDBBackups struct {
	VectorDBBackups do.VectorDBBackups
}

var _ Displayable = &VectorDBBackups{}

func (vb *VectorDBBackups) JSON(out io.Writer) error {
	return writeJSON(vb.VectorDBBackups, out)
}

func (vb *VectorDBBackups) Cols() []string {
	return []string{
		"BackupID",
		"Status",
		"Started",
		"Completed",
	}
}

func (vb *VectorDBBackups) ColMap() map[string]string {
	return map[string]string{
		"BackupID":  "Backup ID",
		"Status":    "Status",
		"Started":   "Started At",
		"Completed": "Completed At",
	}
}

func (vb *VectorDBBackups) KV() []map[string]any {
	out := make([]map[string]any, 0, len(vb.VectorDBBackups))

	for _, b := range vb.VectorDBBackups {
		o := map[string]any{
			"BackupID":  b.BackupID,
			"Status":    b.Status,
			"Started":   b.StartedAt,
			"Completed": b.CompletedAt,
		}
		out = append(out, o)
	}

	return out
}

type VectorDBAdminCredentials struct {
	VectorDBAdminCredentials do.VectorDBAdminCredentials
}

var _ Displayable = &VectorDBAdminCredentials{}

func (vc *VectorDBAdminCredentials) JSON(out io.Writer) error {
	return writeJSON(vc.VectorDBAdminCredentials, out)
}

func (vc *VectorDBAdminCredentials) Cols() []string {
	return []string{
		"UserID",
		"APIToken",
	}
}

func (vc *VectorDBAdminCredentials) ColMap() map[string]string {
	return map[string]string{
		"UserID":   "User ID",
		"APIToken": "API Token",
	}
}

func (vc *VectorDBAdminCredentials) KV() []map[string]any {
	c := vc.VectorDBAdminCredentials
	return []map[string]any{
		{
			"UserID":   c.UserID,
			"APIToken": c.APIToken,
		},
	}
}

type VectorDBRestoreBackupResponse struct {
	VectorDBRestoreBackupResponse do.VectorDBRestoreBackupResponse
}

var _ Displayable = &VectorDBRestoreBackupResponse{}

func (vr *VectorDBRestoreBackupResponse) JSON(out io.Writer) error {
	return writeJSON(vr.VectorDBRestoreBackupResponse, out)
}

func (vr *VectorDBRestoreBackupResponse) Cols() []string {
	return []string{
		"BackupID",
		"Status",
	}
}

func (vr *VectorDBRestoreBackupResponse) ColMap() map[string]string {
	return map[string]string{
		"BackupID": "Backup ID",
		"Status":   "Status",
	}
}

func (vr *VectorDBRestoreBackupResponse) KV() []map[string]any {
	r := vr.VectorDBRestoreBackupResponse
	return []map[string]any{
		{
			"BackupID": r.BackupID,
			"Status":   r.Status,
		},
	}
}

type VectorDBRestoreStatus struct {
	VectorDBRestoreStatus do.VectorDBRestoreStatus
}

var _ Displayable = &VectorDBRestoreStatus{}

func (vs *VectorDBRestoreStatus) JSON(out io.Writer) error {
	return writeJSON(vs.VectorDBRestoreStatus, out)
}

func (vs *VectorDBRestoreStatus) Cols() []string {
	return []string{
		"BackupID",
		"Status",
		"Error",
	}
}

func (vs *VectorDBRestoreStatus) ColMap() map[string]string {
	return map[string]string{
		"BackupID": "Backup ID",
		"Status":   "Status",
		"Error":    "Error",
	}
}

func (vs *VectorDBRestoreStatus) KV() []map[string]any {
	s := vs.VectorDBRestoreStatus
	return []map[string]any{
		{
			"BackupID": s.BackupID,
			"Status":   s.Status,
			"Error":    s.Error,
		},
	}
}
