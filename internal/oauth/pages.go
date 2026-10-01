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

package oauth

import (
	"html/template"
	"strings"
)

// pageTemplate renders the page the local callback server shows in the
// browser. It is an html/template rather than a format string so every field
// is escaped for the context it appears in: the message repeats text from the
// authorization server, which doctl does not control.
var pageTemplate = template.Must(template.New("callback").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>doctl</title>
<style>
  :root { color-scheme: light dark; }
  body {
    margin: 0;
    min-height: 100vh;
    display: flex;
    align-items: center;
    justify-content: center;
    background: #f9fafb;
    color: #12203c;
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Helvetica, Arial, sans-serif;
  }
  main {
    max-width: 30rem;
    padding: 3rem 2.5rem;
    background: #ffffff;
    border: 1px solid #e5e8ed;
    border-radius: 0.75rem;
    text-align: center;
  }
  h1 { margin: 0 0 0.75rem; font-size: 1.25rem; font-weight: 600; }
  p { margin: 0; line-height: 1.6; color: #4b5563; }
  .mark { font-size: 2rem; line-height: 1; margin-bottom: 1rem; }
  .ok { color: #0069ff; }
  .fail { color: #d0021b; }
  @media (prefers-color-scheme: dark) {
    body { background: #0b1020; color: #e7ecf5; }
    main { background: #131a2e; border-color: #24304d; }
    p { color: #9aa7bd; }
  }
</style>
</head>
<body>
  <main>
    <div class="mark {{.MarkClass}}">{{.Mark}}</div>
    <h1>{{.Title}}</h1>
    <p>{{.Message}}</p>
  </main>
</body>
</html>
`))

type pageContent struct {
	MarkClass string
	Mark      string
	Title     string
	Message   string
}

func renderPage(content pageContent) string {
	var page strings.Builder
	if err := pageTemplate.Execute(&page, content); err != nil {
		// The template and this struct are both compiled in, so a failure here
		// is a programming error rather than anything the response can cause.
		// Fall back to a bare message so the browser is not left with a
		// half-written page.
		return "Return to your terminal to continue."
	}

	return page.String()
}

var successPage = renderPage(pageContent{
	MarkClass: "ok",
	Mark:      "\u2713",
	Title:     "You're signed in",
	Message:   "doctl is now authenticated with your DigitalOcean account. You can close this tab and return to your terminal.",
})

func errorPage(reason string) string {
	return renderPage(pageContent{
		MarkClass: "fail",
		Mark:      "\u2717",
		Title:     "Authorization failed",
		Message:   reason + " You can close this tab and try again in your terminal.",
	})
}
