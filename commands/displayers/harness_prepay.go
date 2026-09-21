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
	"fmt"
	"io"
	"strings"

	"github.com/digitalocean/doctl/do"
)

// HarnessPrepayBalance renders the team's prepayment balance and gate state.
//
// Config may be nil when the token could not read billing; the table then
// reports the gate state as unknown rather than implying a healthy zero.
type HarnessPrepayBalance struct {
	Config *do.PrepaymentConfig
	Status *do.PrepaymentStatus

	// NoPermission records that the token lacks billing:read. It is a
	// user-actionable state, not an error, so it is displayed rather than
	// returned.
	NoPermission bool
}

var _ Displayable = &HarnessPrepayBalance{}

func (b *HarnessPrepayBalance) JSON(out io.Writer) error {
	// Mirror the API envelope so scripts can read the same shape doctl saw.
	return writeJSON(struct {
		Config *do.PrepaymentConfig `json:"config,omitempty"`
		Status *do.PrepaymentStatus `json:"status,omitempty"`
	}{Config: b.Config, Status: b.Status}, out)
}

func (b *HarnessPrepayBalance) Cols() []string {
	return []string{"Balance", "MonthToDateBalance", "AutoTopOff", "Status"}
}

func (b *HarnessPrepayBalance) ColMap() map[string]string {
	return map[string]string{
		"Balance":            "Balance",
		"MonthToDateBalance": "Month-to-date Balance",
		"AutoTopOff":         "Auto Top-off",
		"Status":             "Status",
	}
}

func (b *HarnessPrepayBalance) KV() []map[string]any {
	return []map[string]any{{
		"Balance":            b.balance(),
		"MonthToDateBalance": b.monthToDate(),
		"AutoTopOff":         b.autoTopOff(),
		"Status":             b.status(),
	}}
}

func (b *HarnessPrepayBalance) balance() string {
	if b.Status == nil {
		return "unknown"
	}
	return prepayMoney(b.Status.Balance)
}

func (b *HarnessPrepayBalance) monthToDate() string {
	if b.Status == nil {
		return "unknown"
	}
	return prepayMoney(b.Status.MonthToDateBalance)
}

// autoTopOff spells out both halves of the enrollment — when it fires and what
// it charges — so the user can tell whether it will clear a block on its own.
func (b *HarnessPrepayBalance) autoTopOff() string {
	if b.Status == nil && b.Config == nil {
		return "unknown"
	}

	enabled := (b.Status != nil && b.Status.IsAutoPrepayEnabled) ||
		(b.Config != nil && b.Config.IsAutoPrepayEnabled)
	if !enabled {
		return "off"
	}
	if b.Config == nil {
		return "on"
	}
	return fmt.Sprintf("on (at %s, charging %s)",
		prepayMoney(b.Config.PrepayThreshold), prepayMoney(b.Config.PrepayAmount))
}

func (b *HarnessPrepayBalance) status() string {
	if b.NoPermission {
		return "unknown (token lacks billing:read)"
	}
	if b.Status == nil {
		return "unknown"
	}
	if b.Status.Blocked {
		return "Blocked — add funds"
	}
	if !b.Status.Eligible {
		return "Not on prepay"
	}
	return "OK"
}

// prepayMoney formats billing's decimal-dollar string for display.
func prepayMoney(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "unknown"
	}
	if strings.HasPrefix(s, "-") {
		return "-$" + strings.TrimPrefix(s, "-")
	}
	return "$" + s
}
