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
	"time"

	"github.com/digitalocean/doctl/do"
)

type PrepaymentConfig struct {
	*do.PrepaymentConfigResponse
}

var _ Displayable = &PrepaymentConfig{}

func (p *PrepaymentConfig) JSON(out io.Writer) error {
	return writeJSON(p.PrepaymentConfigResponse, out)
}

func (p *PrepaymentConfig) Cols() []string {
	return []string{
		"SpendLimit",
		"IsAutoPrepayEnabled",
		"PrepayAmount",
		"PrepayThreshold",
		"Balance",
		"Blocked",
		"Eligible",
		"MonthToDateBalance",
		"CreatedAt",
		"UpdatedAt",
	}
}

func (p *PrepaymentConfig) ColMap() map[string]string {
	return map[string]string{
		"SpendLimit":          "Spend Limit",
		"IsAutoPrepayEnabled": "Auto Prepay Enabled",
		"PrepayAmount":        "Prepay Amount",
		"PrepayThreshold":     "Prepay Threshold",
		"Balance":             "Balance",
		"Blocked":             "Blocked",
		"Eligible":            "Eligible",
		"MonthToDateBalance":  "Month-to-date Balance",
		"CreatedAt":           "Created At",
		"UpdatedAt":           "Updated At",
	}
}

func (p *PrepaymentConfig) KV() []map[string]any {
	x := map[string]any{}
	if p.Config != nil {
		x["SpendLimit"] = p.Config.SpendLimit
		x["IsAutoPrepayEnabled"] = p.Config.IsAutoPrepayEnabled
		x["PrepayAmount"] = p.Config.PrepayAmount
		x["PrepayThreshold"] = p.Config.PrepayThreshold
		x["CreatedAt"] = p.Config.CreatedAt.Format(time.RFC3339)
		x["UpdatedAt"] = p.Config.UpdatedAt.Format(time.RFC3339)
	}
	if p.Status != nil {
		x["Balance"] = p.Status.Balance
		x["Blocked"] = p.Status.Blocked
		x["Eligible"] = p.Status.Eligible
		x["MonthToDateBalance"] = p.Status.MonthToDateBalance
		if _, ok := x["IsAutoPrepayEnabled"]; !ok {
			x["IsAutoPrepayEnabled"] = p.Status.IsAutoPrepayEnabled
		}
	}
	return []map[string]any{x}
}

type PrepaymentStatus struct {
	*do.PrepaymentStatusResponse
}

var _ Displayable = &PrepaymentStatus{}

func (p *PrepaymentStatus) JSON(out io.Writer) error {
	return writeJSON(p.PrepaymentStatusResponse, out)
}

func (p *PrepaymentStatus) Cols() []string {
	return []string{
		"Balance",
		"IsAutoPrepayEnabled",
		"Blocked",
		"Eligible",
		"MonthToDateBalance",
	}
}

func (p *PrepaymentStatus) ColMap() map[string]string {
	return map[string]string{
		"Balance":             "Balance",
		"IsAutoPrepayEnabled": "Auto Prepay Enabled",
		"Blocked":             "Blocked",
		"Eligible":            "Eligible",
		"MonthToDateBalance":  "Month-to-date Balance",
	}
}

func (p *PrepaymentStatus) KV() []map[string]any {
	x := map[string]any{}
	if p.Status != nil {
		x["Balance"] = p.Status.Balance
		x["IsAutoPrepayEnabled"] = p.Status.IsAutoPrepayEnabled
		x["Blocked"] = p.Status.Blocked
		x["Eligible"] = p.Status.Eligible
		x["MonthToDateBalance"] = p.Status.MonthToDateBalance
	}
	return []map[string]any{x}
}
