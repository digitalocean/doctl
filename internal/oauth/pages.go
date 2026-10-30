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
	_ "embed"
	"encoding/base64"
	"html/template"
)

// successBackgroundPNG is the image shown behind the signed-in card.
// It is inlined into that one response: the callback server shuts down as
// soon as the page is written, so a follow-up request for the file would miss.
//
//go:embed assets/success_background.png
var successBackgroundPNG []byte

// successBackgroundDataURL is the embedded image. It is emitted with
// template.HTMLAttr so the base64 is not rewritten (+ would otherwise become
// &#43; and the file would no longer match the image byte for byte).
var successBackgroundDataURL = "data:image/png;base64," + base64.StdEncoding.EncodeToString(successBackgroundPNG)

var successBackgroundSrc = template.HTMLAttr(`src="` + successBackgroundDataURL + `"`)

// digitalOceanLogoSVG is the white DigitalOcean wordmark from the control
// panel header. It is only legible on the dark success background.
//
//go:embed assets/digitalocean_logo.svg
var digitalOceanLogoSVG []byte

var digitalOceanLogoSrc = template.HTMLAttr(`src="data:image/svg+xml;base64,` + base64.StdEncoding.EncodeToString(digitalOceanLogoSVG) + `"`)

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
    position: relative;
    z-index: 1;
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
  .sea {
    position: fixed;
    inset: 0;
    width: 100%;
    height: 100%;
    object-fit: cover;
    object-position: center;
    z-index: 0;
    pointer-events: none;
  }
  .logo {
    position: fixed;
    top: 1.5rem;
    left: 2rem;
    z-index: 1;
    height: 1.75rem;
  }
  body.ocean {
    background: #031414;
    color-scheme: light;
  }
  body.ocean main {
    border: none;
    box-shadow: 0 24px 64px rgba(0, 0, 0, 0.35);
    color: #031b4e;
  }
  @media (prefers-color-scheme: dark) {
    body:not(.ocean) { background: #0b1020; color: #e7ecf5; }
    body:not(.ocean) main { background: #131a2e; border-color: #24304d; }
    body:not(.ocean) p { color: #9aa7bd; }
  }
</style>
</head>
<body{{if .Ocean}} class="ocean"{{end}}>
  {{if .Ocean}}<img class="sea" alt="" {{.Background}}>
  <img class="logo" alt="DigitalOcean" {{.Logo}}>{{end}}
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
	// Ocean is set once authorization succeeds, when Background is shown.
	Ocean bool
	// Background is the src attribute for assets/success_background.png.
	Background template.HTMLAttr
	// Logo is the src attribute for assets/digitalocean_logo.svg.
	Logo template.HTMLAttr
}

var successContent = pageContent{
	MarkClass:  "ok",
	Mark:       "\u2713",
	Title:      "You're signed in",
	Message:    "doctl is now authenticated with your DigitalOcean account. You can close this tab and return to your terminal.",
	Ocean:      true,
	Background: successBackgroundSrc,
	Logo:       digitalOceanLogoSrc,
}

func errorContent(reason string) pageContent {
	return pageContent{
		MarkClass: "fail",
		Mark:      "\u2717",
		Title:     "Authorization failed",
		Message:   reason + " You can close this tab and try again in your terminal.",
	}
}
