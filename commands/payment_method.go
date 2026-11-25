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
	"errors"
	"fmt"
	"net/http"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/charm/template"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/doctl/internal/ui"
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
)

const (
	// digitalOceanAddPaymentMethodURL is the control-panel page where a
	// customer adds a payment method. Payment details are web-only; doctl only
	// opens or prints this URL.
	digitalOceanAddPaymentMethodURL = "https://cloud.digitalocean.com/account/billing/add-payment-method"

	// digitalOceanBillingURL is the control-panel billing overview.
	digitalOceanBillingURL = "https://cloud.digitalocean.com/account/billing"
)

// openPaymentMethodURL opens a URL in a browser popup window. Tests replace it.
var openPaymentMethodURL = openURLInPopupWindow

// paymentMethodAccountScope is the token permission payment-method add needs
// to read the account it checks.
const paymentMethodAccountScope = "account:read"

// paymentMethodCheckError is a failed pre-flight check of payment-method add,
// carrying the step that resolves it.
type paymentMethodCheckError struct {
	msg      string
	nextStep string
	// failure replaces the check's default failure line when set.
	failure string
}

func (e *paymentMethodCheckError) Error() string    { return e.msg }
func (e *paymentMethodCheckError) NextStep() string { return e.nextStep }

// PaymentMethod creates the payment-method commands hierarchy.
func PaymentMethod() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "payment-method",
			Short:   "Display commands for managing your account's payment methods",
			Long:    "The subcommands of `doctl payment-method` help you manage the payment methods on your DigitalOcean account.",
			GroupID: viewBillingGroup,
		},
	}

	// Not initialized up front so a missing token can be reported as a
	// failed sign-in check rather than a generic client error.
	cmdAdd := cmdBuilderWithInit(cmd, RunPaymentMethodAdd, "add", "Add a payment method in a browser popup window", `This command opens the DigitalOcean control panel in a popup window so you can add a payment method to your account.

The popup is a Chrome, Edge, Brave, or Chromium app window. If none of those browsers is installed, the page opens in a regular tab of your default browser instead.

Before opening the page, doctl checks that:

- doctl is signed in to your account with the `+"`"+paymentMethodAccountScope+"`"+` permission. If it is not, run `+"`"+`doctl auth login`+"`"+` first.
- Your account's email address is verified. If it is not, run `+"`"+`doctl auth login`+"`"+` to verify it, then run this command again.
- Your team does not have a payment method yet. Teams that already have one can manage payment methods at `+digitalOceanBillingURL+`.

Payment details are entered in the browser; doctl does not collect them itself.

Use `+"`"+`--no-browser`+"`"+` to print the URL instead of opening it.`, Writer, false)
	AddBoolFlag(cmdAdd, doctl.ArgOAuthNoBrowser, "", false, "Print the payment method URL instead of opening a browser")
	cmdAdd.Example = `The following example opens a popup window to add a payment method: doctl payment-method add
The following example prints the URL without opening a browser: doctl payment-method add --no-browser`

	return cmd
}

// RunPaymentMethodAdd runs the sign-in, email, and payment-method checks, then
// opens (or prints) the add-payment-method page once they all pass.
func RunPaymentMethodAdd(c *CmdConfig) error {
	noBrowser, err := c.Doit.GetBool(c.NS, doctl.ArgOAuthNoBrowser)
	if err != nil {
		return err
	}

	check := c.UI.NewSpinner("Checking sign-in", ui.WithHeading("Checking your account"), ui.WithoutElapsed())
	check.Start()
	account, err := paymentMethodAccount(c)
	if err != nil {
		failure := "Could not verify sign-in"
		var checkErr *paymentMethodCheckError
		if errors.As(err, &checkErr) && checkErr.failure != "" {
			failure = checkErr.failure
		}
		check.Fail("%s", failure)
		return err
	}
	check.Succeed("Signed in as %s", account.Email)

	check = c.UI.NewSpinner("Checking email verification", ui.WithoutElapsed())
	check.Start()
	if !account.EmailVerified {
		check.Fail("Email address %s is not verified", account.Email)
		return &paymentMethodCheckError{
			msg:      fmt.Sprintf("the email address %s on your account is not verified", account.Email),
			nextStep: "run doctl auth login to verify your email, then run doctl payment-method add again",
		}
	}
	check.Succeed("Email address verified")

	team := paymentMethodTeam(account)
	check = c.UI.NewSpinner("Checking payment method", ui.WithoutElapsed())
	check.Start()
	if !account.IsTrial {
		check.Succeed("Payment method already added for %s", team)
		template.Render(c.Out, `{{nl}}Manage your payment methods at:{{nl}}{{nl}}  {{underline .}}{{nl}}{{nl}}`, digitalOceanBillingURL)
		return nil
	}
	check.Succeed("Payment method not added for %s", team)

	if noBrowser {
		template.Render(c.Out, `{{nl}}Visit the link below to add a payment method to your DigitalOcean account.{{nl}}{{nl}}  {{underline .}}{{nl}}{{nl}}`, digitalOceanAddPaymentMethodURL)
		return nil
	}

	template.Render(c.Out, `{{nl}}Opening a browser window to add a payment method. If it does not open, visit the link below.{{nl}}{{nl}}  {{underline .}}{{nl}}{{nl}}`, digitalOceanAddPaymentMethodURL)
	if err := openPaymentMethodURL(digitalOceanAddPaymentMethodURL); err != nil {
		template.Render(c.Out, `{{muted "Could not open a browser automatically; open the URL above manually."}}{{nl}}{{nl}}`, nil)
	}
	return nil
}

// paymentMethodTeam names the team a payment method applies to.
func paymentMethodTeam(account *do.Account) string {
	if account.Team != nil && account.Team.Name != "" {
		return "team - " + account.Team.Name
	}
	return "this team"
}

// paymentMethodAccount initializes the API client and fetches the account,
// pointing a customer with no access token, or one without account:read, at
// doctl auth login.
func paymentMethodAccount(c *CmdConfig) (*do.Account, error) {
	if err := c.initServices(c); err != nil {
		var missing *doctl.MissingAccessTokenErr
		if errors.As(err, &missing) {
			return nil, &paymentMethodCheckError{
				msg:      "doctl is not signed in to a DigitalOcean account",
				nextStep: "run doctl auth login (or doctl auth init to use an API token)",
				failure:  "Not signed in to a DigitalOcean account",
			}
		}
		return nil, err
	}

	account, err := c.Account().Get()
	var errResp *godo.ErrorResponse
	if errors.As(err, &errResp) && errResp.Response != nil && errResp.Response.StatusCode == http.StatusForbidden {
		return nil, &paymentMethodCheckError{
			msg:      fmt.Sprintf("your access token does not have the %s permission that doctl payment-method add needs", paymentMethodAccountScope),
			nextStep: fmt.Sprintf("run doctl auth login and grant the %s permission (or use an API token that includes it)", paymentMethodAccountScope),
			failure:  fmt.Sprintf("Access token is missing the %s permission", paymentMethodAccountScope),
		}
	}
	return account, err
}
