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

package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestRunAgentsBalance(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		cfg := &do.PrepaymentConfig{
			SpendLimit:      "100.00",
			PrepayAmount:    "25.00",
			PrepayThreshold: "5.00",
		}
		status := &do.PrepaymentStatus{
			Balance:            "12.50",
			Eligible:           true,
			MonthToDateBalance: "3.25",
		}
		tm.prepayment.EXPECT().Get(gomock.Any()).Return(cfg, status, nil)

		require.NoError(t, RunAgentsBalance(config))
	})
}

func balanceHTTPError(code int, msg string) error {
	return &godo.ErrorResponse{
		Response:  &http.Response{StatusCode: code},
		Message:   msg,
		RequestID: "req-123",
	}
}

// forbiddenBalanceErr is what the do layer returns for a 403.
func forbiddenBalanceErr() error {
	return fmt.Errorf("%w: %w", do.ErrNoBillingPermission,
		balanceHTTPError(http.StatusForbidden, "You are not authorized to perform this operation"))
}

const (
	balMsgInvalid  = "Your token does not work: it is invalid, expired, or revoked. Run `doctl auth init` or `doctl auth login`."
	balMsgNoScope  = "Your token works but does not include the billing:read scope. Create a token with billing:read, or ask a Team Owner or Biller."
	balMsgRole     = "Your token has billing:read, but your team role or account state does not allow reading billing. Ask a Team Owner or Biller."
	balMsgTeam     = "Your token was refused for this team. Check that your auth context points at the right team."
	balMsgUnsure   = "Cannot read the balance: the token may lack the billing:read scope. Server said: Permission denied."
	balMsgUnsure41 = "Cannot read the balance: the token may lack the billing:read scope. Server said: You are not authorized to perform this operation."
)

// Every way the balance can fail to read is a command failure that names its
// cause, so scripts see a non-zero exit instead of an "unknown" table.
func TestRunAgentsBalance_ExplainsUnreadableBalance(t *testing.T) {
	setAPIURL(t, "")
	denied := balanceHTTPError(http.StatusUnauthorized, "Permission denied")
	tests := []struct {
		name        string
		balanceErr  error
		tokenInfo   *do.OAuthTokenInfo
		tokenErr    error
		wantMessage string
	}{
		{"token info says 401", denied, nil, balanceHTTPError(http.StatusUnauthorized, "Unable to authenticate you"), balMsgInvalid},
		{"token info says 401 after a 403", forbiddenBalanceErr(), nil, balanceHTTPError(http.StatusUnauthorized, "Unable to authenticate you"), balMsgInvalid},
		{"scopes lack billing read on 403", forbiddenBalanceErr(), &do.OAuthTokenInfo{Scopes: []string{"droplet:read"}}, nil, balMsgNoScope},
		{"scopes lack billing read on 401", denied, &do.OAuthTokenInfo{Scopes: []string{"droplet:read"}}, nil, balMsgNoScope},
		{"scopes cover billing read on 403", forbiddenBalanceErr(), &do.OAuthTokenInfo{Scopes: []string{"billing:read"}}, nil, balMsgRole},
		{"full read scope covers billing on 403", forbiddenBalanceErr(), &do.OAuthTokenInfo{Scopes: []string{"read"}}, nil, balMsgRole},
		{"scopes cover billing read on 401", denied, &do.OAuthTokenInfo{Scopes: []string{"billing:*"}}, nil, balMsgTeam},
		{"token info unreachable on 401", denied, nil, errors.New("dial tcp: connection refused"), balMsgUnsure},
		{"token info server error on 403", forbiddenBalanceErr(), nil, balanceHTTPError(http.StatusBadGateway, "bad gateway"), balMsgUnsure41},
		{"empty scopes are unknown", denied, &do.OAuthTokenInfo{}, nil, balMsgUnsure},
		{"missing token info is unknown", denied, nil, nil, balMsgUnsure},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				tm.prepayment.EXPECT().Get(gomock.Any()).Return(nil, nil, tt.balanceErr)
				tm.oauth.EXPECT().TokenInfoContext(gomock.Any(), "").Return(tt.tokenInfo, tt.tokenErr)

				err := RunAgentsBalance(config)
				assert.EqualError(t, err, tt.wantMessage)

				// The API error stays reachable, so checkErr keeps the status
				// and request ID, and no generic --help next step is added.
				var gerr *godo.ErrorResponse
				require.True(t, errors.As(err, &gerr))
				se := resolveStructured(err)
				assert.NotZero(t, se.Status())
				assert.Equal(t, "req-123", se.RequestID())
				assert.Empty(t, se.NextStep())
			})
		})
	}
}

// The bare sentinel still means 403, and the message says so.
func TestRunAgentsBalance_NoBillingPermission(t *testing.T) {
	setAPIURL(t, "")
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.prepayment.EXPECT().Get(gomock.Any()).Return(nil, nil, do.ErrNoBillingPermission)
		tm.oauth.EXPECT().TokenInfoContext(gomock.Any(), "").Return(&do.OAuthTokenInfo{Scopes: []string{"billing:read"}}, nil)

		assert.EqualError(t, RunAgentsBalance(config), balMsgRole)
	})
}

// A failure that is not a 401 or 403 is not about the token, so it passes
// through untouched and the token is never checked.
func TestRunAgentsBalance_ServerErrorSkipsTokenCheck(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		boom := balanceHTTPError(http.StatusInternalServerError, "boom")
		tm.prepayment.EXPECT().Get(gomock.Any()).Return(nil, nil, boom)

		assert.Same(t, boom, RunAgentsBalance(config))
	})
}

// -o json must print the structured error and no balance data when the
// balance could not be read.
func TestRunAgentsBalance_JSONPrintsStructuredError(t *testing.T) {
	setAPIURL(t, "")
	defer func(a func()) { errAction = a }(errAction)
	errAction = func() {}
	viper.Set("output", "json")
	t.Cleanup(func() { viper.Set("output", "") })

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		var out bytes.Buffer
		config.Out = &out
		tm.prepayment.EXPECT().Get(gomock.Any()).Return(nil, nil, forbiddenBalanceErr())
		tm.oauth.EXPECT().TokenInfoContext(gomock.Any(), "").Return(&do.OAuthTokenInfo{Scopes: []string{"billing:read"}}, nil)

		err := RunAgentsBalance(config)
		assert.EqualError(t, err, balMsgRole)
		assert.Empty(t, out.String())

		old := os.Stdout
		r, w, perr := os.Pipe()
		require.NoError(t, perr)
		os.Stdout = w
		checkErr(err)
		require.NoError(t, w.Close())
		os.Stdout = old
		raw, rerr := io.ReadAll(r)
		require.NoError(t, rerr)

		var payload outputErrors
		require.NoError(t, json.Unmarshal(raw, &payload))
		require.Len(t, payload.Errors, 1)
		assert.Equal(t, balMsgRole, payload.Errors[0].Detail)
		assert.Equal(t, http.StatusForbidden, payload.Errors[0].Status)
		assert.Equal(t, "req-123", payload.Errors[0].RequestID)
		assert.NotContains(t, string(raw), "balance\":")
	})
}

// With no validation server configured and an API URL that is not the default,
// the token is never sent anywhere for checking.
func TestRunAgentsBalance_CustomAPIURLSkipsTokenCheck(t *testing.T) {
	setAPIURL(t, "https://api.devcloud.example")

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.prepayment.EXPECT().Get(gomock.Any()).Return(nil, nil, balanceHTTPError(http.StatusUnauthorized, "Permission denied"))

		assert.EqualError(t, RunAgentsBalance(config), balMsgUnsure)
	})
}

// A configured validation server is the user's choice, so it is used even
// when the API URL is not the default.
func TestRunAgentsBalance_ConfiguredValidationServerIsUsed(t *testing.T) {
	setAPIURL(t, "https://api.devcloud.example")

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set("", doctl.ArgTokenValidationServer, "https://validate.example")
		tm.prepayment.EXPECT().Get(gomock.Any()).Return(nil, nil, forbiddenBalanceErr())
		tm.oauth.EXPECT().TokenInfoContext(gomock.Any(), "https://validate.example").
			Return(&do.OAuthTokenInfo{Scopes: []string{"billing:read"}}, nil)

		assert.EqualError(t, RunAgentsBalance(config), balMsgRole)
	})
}

// The token check is bounded; hitting the deadline falls back to the
// "may lack the scope" message.
func TestRunAgentsBalance_TokenCheckTimeout(t *testing.T) {
	setAPIURL(t, "")
	old := balanceTokenCheckTimeout
	balanceTokenCheckTimeout = 20 * time.Millisecond
	t.Cleanup(func() { balanceTokenCheckTimeout = old })

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.prepayment.EXPECT().Get(gomock.Any()).Return(nil, nil, balanceHTTPError(http.StatusUnauthorized, "Permission denied"))
		tm.oauth.EXPECT().TokenInfoContext(gomock.Any(), "").DoAndReturn(
			func(ctx context.Context, _ string) (*do.OAuthTokenInfo, error) {
				_, hasDeadline := ctx.Deadline()
				assert.True(t, hasDeadline)
				<-ctx.Done()
				return nil, ctx.Err()
			})

		assert.EqualError(t, RunAgentsBalance(config), balMsgUnsure)
	})
}

func TestServerSaid(t *testing.T) {
	nested := &godo.ErrorResponse{}
	nested.NestedError = &struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}{Message: "nested text"}
	assert.Equal(t, "nested text", serverSaid(nested, 401))
	assert.Equal(t, "plain.", serverSaid(&godo.ErrorResponse{Message: "plain."}, 401))
	assert.Equal(t, "Unauthorized", serverSaid(&godo.ErrorResponse{Message: "<html>bad gateway</html>"}, 401))
	assert.Equal(t, "Forbidden", serverSaid(&godo.ErrorResponse{}, 403))
}

func TestRunAgentsBalance_OtherErrorsPropagate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.prepayment.EXPECT().Get(gomock.Any()).Return(nil, nil, errors.New("billing is down"))

		assert.EqualError(t, RunAgentsBalance(config), "billing is down")
	})
}
