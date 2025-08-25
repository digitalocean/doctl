/*
Copyright 2024 The Doctl Authors All rights reserved.
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

type BYOIPPrefix struct {
	BYOIPPrefixes do.BYOIPPrefixes
}

var _ Displayable = &BYOIPPrefix{}

func (bp *BYOIPPrefix) JSON(out io.Writer) error {
	return writeJSON(bp.BYOIPPrefixes, out)
}

func (bp *BYOIPPrefix) Cols() []string {
	return []string{
		"Prefix", "Region", "Status", "UUID", "Advertised", "FailureReason", "Validations",
	}
}

func (bp *BYOIPPrefix) ColMap() map[string]string {
	return map[string]string{
		"Prefix": "Prefix", "Region": "Region", "Status": "Status", "UUID": "UUID",
		"Advertised": "Advertised", "FailureReason": "Failure Reason", "Validations": "Validations",
	}
}

func (bp *BYOIPPrefix) KV() []map[string]any {
	out := make([]map[string]any, 0, len(bp.BYOIPPrefixes))

	for _, f := range bp.BYOIPPrefixes {

		o := map[string]any{
			"Prefix":        f.BYOIPPrefix.Prefix,
			"Region":        f.BYOIPPrefix.Region,
			"Status":        f.BYOIPPrefix.Status,
			"UUID":          f.BYOIPPrefix.UUID,
			"Advertised":    f.BYOIPPrefix.Advertised,
			"FailureReason": f.BYOIPPrefix.FailureReason,
			"Validations":   f.BYOIPPrefix.Validations,
		}

		out = append(out, o)
	}

	return out
}

type BYOIPPrefixResource struct {
	BYOIPPrefixResource do.BYOIPPrefixResources
}

var _ Displayable = &BYOIPPrefixResource{}

func (bpr *BYOIPPrefixResource) JSON(out io.Writer) error {
	return writeJSON(bpr.BYOIPPrefixResource, out)
}

func (bpr *BYOIPPrefixResource) Cols() []string {
	return []string{
		"ID", "IP", "Region", "Resource", "AssignedAt",
	}
}

func (bpr *BYOIPPrefixResource) ColMap() map[string]string {
	return map[string]string{
		"ID": "ID", "IP": "IP", "Region": "Region", "Resource": "Resource",
		"AssignedAt": "Assigned At",
	}
}

func (bpr *BYOIPPrefixResource) KV() []map[string]any {
	out := make([]map[string]any, 0, len(bpr.BYOIPPrefixResource))

	for _, f := range bpr.BYOIPPrefixResource {

		o := map[string]any{
			"ID":         f.BYOIPPrefixResource.ID,
			"IP":         f.BYOIPPrefixResource.BYOIP,
			"Region":     f.BYOIPPrefixResource.Region,
			"Resource":   f.BYOIPPrefixResource.Resource,
			"AssignedAt": f.BYOIPPrefixResource.AssignedAt,
		}

		out = append(out, o)
	}

	return out
}

type BYOIPPrefixCreate struct {
	do.BYOIPPrefixCreate
}

var _ Displayable = &BYOIPPrefixCreate{}

func (bp *BYOIPPrefixCreate) JSON(out io.Writer) error {
	return writeJSON(bp.BYOIPPrefixCreateResp, out)
}

func (bp *BYOIPPrefixCreate) Cols() []string {
	return []string{
		"UUID", "Region", "Status",
	}
}

func (bp *BYOIPPrefixCreate) ColMap() map[string]string {
	return map[string]string{
		"Region": "Region", "Status": "Status", "UUID": "UUID",
	}
}

func (bp *BYOIPPrefixCreate) KV() []map[string]any {

	out := map[string]any{
		"Region": bp.Region,
		"Status": bp.Status,
		"UUID":   bp.UUID,
	}

	return []map[string]any{out}
}
