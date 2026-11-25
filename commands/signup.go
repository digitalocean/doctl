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
	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/charm/template"
	"github.com/pkg/browser"
)

const (
	// digitalOceanSignupURL is the control-panel page that creates a new
	// DigitalOcean account. Account creation is web-only; doctl only opens
	// or prints this URL.
	digitalOceanSignupURL = "https://cloud.digitalocean.com/registrations/new"
)

// openSignupURL opens a URL in the user's browser. Tests replace it.
var openSignupURL = browser.OpenURL

// Signup creates the top-level signup command under Configure doctl.
func Signup() *Command {
	cmd := cmdBuilderWithInit(nil, RunSignup, "signup", "Create a DigitalOcean account in your browser", `This command opens the DigitalOcean signup page so you can create an account.

Account creation happens in the browser; doctl does not collect signup details itself. After your account is ready, run `+"`"+`doctl auth login`+"`"+` to authorize doctl with OAuth, or `+"`"+`doctl auth init`+"`"+` if you prefer a personal access token.

Use `+"`"+`--no-browser`+"`"+` to print the signup URL instead of opening it.`, Writer, false)
	cmd.GroupID = configureDoctlGroup
	AddBoolFlag(cmd, doctl.ArgOAuthNoBrowser, "", false, "Print the signup URL instead of opening a browser")
	cmd.Example = `The following example opens the DigitalOcean signup page: doctl signup
The following example prints the signup URL without opening a browser: doctl signup --no-browser`
	return cmd
}

// RunSignup opens (or prints) the DigitalOcean signup page, then tells the
// customer to run doctl auth login once the account exists.
func RunSignup(c *CmdConfig) error {
	noBrowser, err := c.Doit.GetBool(c.NS, doctl.ArgOAuthNoBrowser)
	if err != nil {
		return err
	}

	if noBrowser {
		template.Render(c.Out, `{{nl}}Visit the link below to create a DigitalOcean account.{{nl}}{{nl}}  {{underline .}}{{nl}}{{nl}}`, digitalOceanSignupURL)
	} else {
		template.Render(c.Out, `{{nl}}Opening your browser to create a DigitalOcean account. If it does not open, visit the link below.{{nl}}{{nl}}  {{underline .}}{{nl}}{{nl}}`, digitalOceanSignupURL)
		if err := openSignupURL(digitalOceanSignupURL); err != nil {
			template.Render(c.Out, `{{muted "Could not open a browser automatically; open the URL above manually."}}{{nl}}{{nl}}`, nil)
		}
	}

	displaySignupNextSteps(c)
	return nil
}

// displaySignupNextSteps is the post-signup helper: once the account exists,
// customers authenticate doctl with the OAuth browser login.
func displaySignupNextSteps(c *CmdConfig) {
	template.Render(c.Out, `{{muted "After your account is ready, complete doctl setup with:"}}{{nl}}  {{highlight "doctl auth login"}}{{nl}}{{nl}}`, nil)
	template.Render(c.Out, `{{muted "That opens your browser so you can authorize doctl. Prefer an API token instead? Run"}} {{highlight "doctl auth init"}}{{muted "."}}{{nl}}`, nil)
}
