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
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/doctl/internal/ui"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func stubOpenPaymentMethodURL(t *testing.T, fn func(string) error) {
	t.Helper()
	prev := openPaymentMethodURL
	openPaymentMethodURL = fn
	t.Cleanup(func() { openPaymentMethodURL = prev })
}

func paymentMethodTestAccount(emailVerified, isTrial bool) *do.Account {
	return &do.Account{Account: &godo.Account{
		Email:         "sammy@digitalocean.com",
		EmailVerified: emailVerified,
		IsTrial:       isTrial,
	}}
}

// capturePaymentMethodOutput points the command's stdout and the check
// spinners' stderr at buffers.
func capturePaymentMethodOutput(config *CmdConfig) (out, checks *bytes.Buffer) {
	out, checks = &bytes.Buffer{}, &bytes.Buffer{}
	config.Out = out
	config.UI = ui.Env{Out: out, Err: checks}
	return out, checks
}

func TestPaymentMethodCommand(t *testing.T) {
	cmd := PaymentMethod()
	assert.NotNil(t, cmd)
	assert.Equal(t, "payment-method", cmd.Name())
	assert.Equal(t, viewBillingGroup, cmd.GroupID)
	assertCommandNames(t, cmd, "add")
}

func TestPaymentMethodAddOpensBrowserAfterChecksPass(t *testing.T) {
	var opened string
	stubOpenPaymentMethodURL(t, func(url string) error {
		opened = url
		return nil
	})

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.account.EXPECT().Get().Return(paymentMethodTestAccount(true, true), nil)
		out, checks := capturePaymentMethodOutput(config)

		err := RunPaymentMethodAdd(config)
		require.NoError(t, err)

		assert.Equal(t, digitalOceanAddPaymentMethodURL, opened)
		assert.Contains(t, out.String(), digitalOceanAddPaymentMethodURL)

		got := checks.String()
		signedIn := bytes.Index(checks.Bytes(), []byte("Signed in as sammy@digitalocean.com"))
		verified := bytes.Index(checks.Bytes(), []byte("Email address verified"))
		noPayment := bytes.Index(checks.Bytes(), []byte("Payment method not added for this team"))
		assert.Contains(t, got, "Checking your account")
		assert.NotContains(t, got, "(0s)")
		require.True(t, signedIn >= 0 && verified >= 0 && noPayment >= 0, got)
		assert.True(t, signedIn < verified && verified < noPayment, "checks run in order: %s", got)
	})
}

func TestPaymentMethodAddNoBrowserSkipsOpen(t *testing.T) {
	openCalled := false
	stubOpenPaymentMethodURL(t, func(string) error {
		openCalled = true
		return nil
	})

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.account.EXPECT().Get().Return(paymentMethodTestAccount(true, true), nil)
		config.Doit.Set(config.NS, doctl.ArgOAuthNoBrowser, true)
		out, _ := capturePaymentMethodOutput(config)

		err := RunPaymentMethodAdd(config)
		require.NoError(t, err)

		assert.False(t, openCalled)
		assert.Contains(t, out.String(), digitalOceanAddPaymentMethodURL)
	})
}

func TestPaymentMethodAddContinuesWhenBrowserOpenFails(t *testing.T) {
	stubOpenPaymentMethodURL(t, func(string) error {
		return errors.New("no display")
	})

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.account.EXPECT().Get().Return(paymentMethodTestAccount(true, true), nil)
		out, _ := capturePaymentMethodOutput(config)

		err := RunPaymentMethodAdd(config)
		require.NoError(t, err)

		assert.Contains(t, out.String(), "Could not open a browser automatically")
		assert.Contains(t, out.String(), digitalOceanAddPaymentMethodURL)
	})
}

func TestPaymentMethodAddSuggestsAuthLoginWithoutToken(t *testing.T) {
	openCalled := false
	stubOpenPaymentMethodURL(t, func(string) error {
		openCalled = true
		return nil
	})

	withTestClient(t, func(config *CmdConfig, _ *tcMocks) {
		config.initServices = func(*CmdConfig) error {
			return fmt.Errorf("Unable to initialize DigitalOcean API client: %w", doctl.NewMissingAccessTokenErr())
		}
		out, checks := capturePaymentMethodOutput(config)

		err := RunPaymentMethodAdd(config)
		require.Error(t, err)

		var ns NextStepper
		require.True(t, errors.As(err, &ns))
		assert.Contains(t, ns.NextStep(), "doctl auth login")
		assert.Contains(t, err.Error(), "not signed in")
		assert.Contains(t, checks.String(), "Not signed in to a DigitalOcean account")
		assert.NotContains(t, checks.String(), "Checking email verification")
		assert.False(t, openCalled)
		assert.Empty(t, out.String())
	})
}

func TestPaymentMethodAddRejectedTokenSuggestsAuthLogin(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.account.EXPECT().Get().Return(nil, &godo.ErrorResponse{
			Response: &http.Response{StatusCode: http.StatusUnauthorized},
			Message:  "Unable to authenticate you",
		})
		_, checks := capturePaymentMethodOutput(config)

		err := RunPaymentMethodAdd(config)
		require.Error(t, err)

		assert.Contains(t, resolveStructured(err).NextStep(), "doctl auth login")
		assert.Contains(t, checks.String(), "Could not verify sign-in")
	})
}

func TestPaymentMethodAddExplainsMissingAccountReadScope(t *testing.T) {
	openCalled := false
	stubOpenPaymentMethodURL(t, func(string) error {
		openCalled = true
		return nil
	})

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.account.EXPECT().Get().Return(nil, &godo.ErrorResponse{
			Response: &http.Response{StatusCode: http.StatusForbidden},
			Message:  "You are not authorized to perform this operation",
		})
		_, checks := capturePaymentMethodOutput(config)

		err := RunPaymentMethodAdd(config)
		require.Error(t, err)

		assert.Contains(t, err.Error(), "account:read")
		assert.Contains(t, resolveStructured(err).NextStep(), "doctl auth login")
		assert.Contains(t, resolveStructured(err).NextStep(), "account:read")
		assert.Contains(t, checks.String(), "Access token is missing the account:read permission")
		assert.NotContains(t, checks.String(), "Checking email verification")
		assert.False(t, openCalled)

		// The failed check line already names the problem, so the error block
		// is only the next step.
		assert.Equal(t,
			"→ run doctl auth login and grant the account:read permission (or use an API token that includes it)\n",
			formatStructuredError(ui.Env{}, err))
	})
}

func TestPaymentMethodAddRequiresVerifiedEmail(t *testing.T) {
	openCalled := false
	stubOpenPaymentMethodURL(t, func(string) error {
		openCalled = true
		return nil
	})

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.account.EXPECT().Get().Return(paymentMethodTestAccount(false, true), nil)
		out, checks := capturePaymentMethodOutput(config)

		err := RunPaymentMethodAdd(config)
		require.Error(t, err)

		assert.Contains(t, err.Error(), "not verified")
		assert.Contains(t, err.Error(), "sammy@digitalocean.com")
		assert.Contains(t, resolveStructured(err).NextStep(), "run doctl auth login to verify your email")
		assert.Contains(t, checks.String(), "Signed in as sammy@digitalocean.com")
		assert.Contains(t, checks.String(), "Email address sammy@digitalocean.com is not verified")
		assert.NotContains(t, checks.String(), "Checking payment method")
		assert.False(t, openCalled)
		assert.NotContains(t, out.String(), digitalOceanAddPaymentMethodURL)
	})
}

func TestPaymentMethodAddSkipsRedirectWhenPaymentMethodOnFile(t *testing.T) {
	openCalled := false
	stubOpenPaymentMethodURL(t, func(string) error {
		openCalled = true
		return nil
	})

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.account.EXPECT().Get().Return(paymentMethodTestAccount(true, false), nil)
		out, checks := capturePaymentMethodOutput(config)

		err := RunPaymentMethodAdd(config)
		require.NoError(t, err)

		assert.False(t, openCalled)
		assert.Contains(t, checks.String(), "Payment method already added for this team")
		assert.Contains(t, out.String(), digitalOceanBillingURL)
		assert.NotContains(t, out.String(), digitalOceanAddPaymentMethodURL)
	})
}

func TestPaymentMethodAddNamesTheTeam(t *testing.T) {
	stubOpenPaymentMethodURL(t, func(string) error { return nil })

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		account := paymentMethodTestAccount(true, true)
		account.Team = &godo.TeamInfo{Name: "Sharks"}
		tm.account.EXPECT().Get().Return(account, nil)
		_, checks := capturePaymentMethodOutput(config)

		require.NoError(t, RunPaymentMethodAdd(config))

		assert.Contains(t, checks.String(), "Payment method not added for team Sharks")
	})
}

func TestPaymentMethodAddReturnsAccountError(t *testing.T) {
	openCalled := false
	stubOpenPaymentMethodURL(t, func(string) error {
		openCalled = true
		return nil
	})

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.account.EXPECT().Get().Return(nil, errors.New("connection refused"))
		capturePaymentMethodOutput(config)

		err := RunPaymentMethodAdd(config)
		require.Error(t, err)

		assert.Contains(t, err.Error(), "connection refused")
		assert.False(t, openCalled)
	})
}
