/*
Copyright 2026 The Doctl Authors All rights reserved.
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

	"github.com/digitalocean/doctl/do"
)

// A SecurityScan is the displayer for showing the results of a single CSPM
// scan.
type SecurityScan struct {
	Scan do.Scan
}

var _ Displayable = &SecurityScan{}

func (s *SecurityScan) JSON(out io.Writer) error {
	return writeJSON(s.Scan, out)
}

func (s *SecurityScan) Cols() []string {
	return []string{"Rule ID", "Name", "Affected Resources", "Found At", "Severity"}
}

func (s *SecurityScan) ColMap() map[string]string {
	return map[string]string{
		"Rule ID":            "Rule ID",
		"Name":               "Name",
		"Affected Resources": "Affected Resources",
		"Found At":           "Found At",
		"Severity":           "Severity",
	}
}

func (s *SecurityScan) KV() []map[string]any {
	out := make([]map[string]any, 0, len(s.Scan.Findings))

	for _, finding := range s.Scan.Findings {
		o := map[string]any{
			"Rule ID":            finding.RuleUUID,
			"Name":               finding.Name,
			"Affected Resources": finding.AffectedResourcesCount,
			"Found At":           finding.FoundAt,
			"Severity":           finding.Severity,
		}
		out = append(out, o)
	}

	return out
}

// SecurityScans is the displayer for showing the results of multiple CSPM
// scans.
type SecurityScans struct {
	Scans do.Scans
}

var _ Displayable = &SecurityScans{}

func (s *SecurityScans) JSON(out io.Writer) error {
	return writeJSON(s.Scans, out)
}

func (s *SecurityScans) Cols() []string {
	return []string{"ID", "Status", "Created At"}
}

func (s *SecurityScans) ColMap() map[string]string {
	return map[string]string{
		"ID":         "ID",
		"Status":     "Status",
		"Created At": "Created At",
	}
}

func (s *SecurityScans) KV() []map[string]any {
	out := make([]map[string]any, 0, len(s.Scans))

	for _, scan := range s.Scans {
		o := map[string]any{
			"ID":         scan.ID,
			"Status":     scan.Status,
			"Created At": scan.CreatedAt,
		}
		out = append(out, o)
	}

	return out
}

type SecurityAffectedResource struct {
	AffectedResources do.AffectedResources
}

var _ Displayable = &SecurityAffectedResource{}

func (s *SecurityAffectedResource) JSON(out io.Writer) error {
	return writeJSON(s.AffectedResources, out)
}

func (s *SecurityAffectedResource) Cols() []string {
	return []string{"URN", "Name", "Type"}
}

func (s *SecurityAffectedResource) ColMap() map[string]string {
	return map[string]string{
		"URN":  "URN",
		"Name": "Name",
		"Type": "Type",
	}
}

func (s *SecurityAffectedResource) KV() []map[string]any {
	out := make([]map[string]any, 0, len(s.AffectedResources))

	for _, resource := range s.AffectedResources {
		o := map[string]any{
			"URN":  resource.URN,
			"Name": resource.Name,
			"Type": resource.Type,
		}
		out = append(out, o)
	}

	return out
}
