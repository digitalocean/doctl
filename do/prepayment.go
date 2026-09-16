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

package do

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/digitalocean/godo"
)

// prepaymentConfigPath is billing's public prepay route. It returns the
// account's prepayment config and its current gate status together, so one
// request answers every question the CLI asks about prepay.
const prepaymentConfigPath = "v2/customers/my/prepayment_config"

// ErrNoBillingPermission is returned when the API token lacks the billing:read
// scope that both prepay routes require. It is a distinct, user-actionable
// state rather than a failure: the caller can still be told to ask a Team
// Owner or Biller, which is the only thing they could act on anyway.
var ErrNoBillingPermission = errors.New("token lacks the billing:read scope required to read the prepayment balance")

// PrepaymentStatus is the account's current prepay gate state. Money values
// are decimal-dollar strings ("25.50"), matching billing's wire format; they
// are deliberately not parsed into floats here.
type PrepaymentStatus struct {
	Balance string `json:"balance"`
	// IsAutoPrepayEnabled is absent rather than false on the wire when
	// disabled, because proto3 JSON omits default values. The Go zero value
	// is the correct reading of that absence.
	IsAutoPrepayEnabled bool   `json:"is_auto_prepay_enabled"`
	Blocked             bool   `json:"blocked"`
	Eligible            bool   `json:"eligible"`
	MonthToDateBalance  string `json:"month_to_date_balance"`
}

// PrepaymentConfig is the account's auto top-off configuration. CreatedAt is
// unset for an account that has never configured prepay — billing returns a
// zero-valued config in that case rather than omitting the object.
type PrepaymentConfig struct {
	SpendLimit          string     `json:"spend_limit"`
	IsAutoPrepayEnabled bool       `json:"is_auto_prepay_enabled"`
	PrepayAmount        string     `json:"prepay_amount"`
	PrepayThreshold     string     `json:"prepay_threshold"`
	CreatedAt           *time.Time `json:"created_at,omitempty"`
	UpdatedAt           *time.Time `json:"updated_at,omitempty"`
}

// PrepaymentService wraps billing's public prepay endpoints.
type PrepaymentService interface {
	// Get returns the account's prepayment config and gate status. It returns
	// ErrNoBillingPermission when the token cannot read billing.
	//
	// Unlike most do/* methods it takes a context rather than using the
	// package-wide context.TODO(), because one caller reads the balance while
	// rendering a 402 and has to be able to give up: a slow billing API must
	// not stall an error the user has already earned.
	Get(ctx context.Context) (*PrepaymentConfig, *PrepaymentStatus, error)
}

// prepaymentConfigResponse is the GetPrepaymentConfigPublic envelope.
type prepaymentConfigResponse struct {
	Config *PrepaymentConfig `json:"config"`
	Status *PrepaymentStatus `json:"status"`
}

type prepaymentService struct {
	client *godo.Client
}

var _ PrepaymentService = &prepaymentService{}

// NewPrepaymentService builds a PrepaymentService instance.
func NewPrepaymentService(godoClient *godo.Client) PrepaymentService {
	return &prepaymentService{client: godoClient}
}

func (s *prepaymentService) Get(ctx context.Context) (*PrepaymentConfig, *PrepaymentStatus, error) {
	req, err := s.client.NewRequest(ctx, http.MethodGet, prepaymentConfigPath, nil)
	if err != nil {
		return nil, nil, err
	}

	root := new(prepaymentConfigResponse)
	if _, err := s.client.Do(ctx, req, root); err != nil {
		var er *godo.ErrorResponse
		if errors.As(err, &er) && er.Response != nil && er.Response.StatusCode == http.StatusForbidden {
			return nil, nil, ErrNoBillingPermission
		}
		return nil, nil, err
	}

	return root.Config, root.Status, nil
}
