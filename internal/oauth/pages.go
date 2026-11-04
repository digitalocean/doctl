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
	"strings"
	"unicode"
	"unicode/utf8"
)

// successBackgroundPNG is the image shown behind the callback card, for both
// a successful sign-in and a failed authorization. It is inlined into that
// response: the callback server shuts down as soon as the page is written,
// so a follow-up request for the file would miss.
//
//go:embed assets/success_background.png
var successBackgroundPNG []byte

// successBackgroundDataURL is the embedded image. It is emitted with
// template.HTMLAttr so the base64 is not rewritten (+ would otherwise become
// &#43; and the file would no longer match the image byte for byte).
var successBackgroundDataURL = "data:image/png;base64," + base64.StdEncoding.EncodeToString(successBackgroundPNG)

var successBackgroundSrc = template.HTMLAttr(`src="` + successBackgroundDataURL + `"`)

// digitalOceanLogoSVG is the white DigitalOcean wordmark from the control
// panel header. It is only legible on the dark callback background.
//
//go:embed assets/digitalocean_logo.svg
var digitalOceanLogoSVG []byte

var digitalOceanLogoSrc = template.HTMLAttr(`src="data:image/svg+xml;base64,` + base64.StdEncoding.EncodeToString(digitalOceanLogoSVG) + `"`)

// The agent marks on the signed-in page are inlined for the same reason as
// the background: the callback server is gone before the browser can ask for
// another file. Each one is drawn on black so the row of circles matches.
//
//go:embed assets/agent_openai.png
var agentOpenAIPNG []byte

//go:embed assets/agent_cursor.png
var agentCursorPNG []byte

//go:embed assets/agent_claude.png
var agentClaudePNG []byte

func pngSrc(b []byte) template.HTMLAttr {
	return template.HTMLAttr(`src="data:image/png;base64,` + base64.StdEncoding.EncodeToString(b) + `"`)
}

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
    box-sizing: border-box;
    width: calc(100% - 2rem);
    max-width: 32rem;
    padding: 3rem 2.2rem 2rem;
    background: #ffffff;
    border-radius: 1.3rem;
    text-align: center;
  }
  h1 {
    margin: 0;
    color: #031b4e;
    font-size: 2.55rem;
    font-weight: 700;
    letter-spacing: -0.035em;
    line-height: 1.08;
  }
  p { margin: 0; }
  .status {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 0.45rem;
    margin-top: 1.3rem;
    color: #5c6a80;
    font-size: 1.1rem;
    font-weight: 500;
    line-height: 1.3;
  }
  .status.failed { color: #031b4e; font-weight: 600; }
  .badge {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 1.15rem;
    height: 1.15rem;
    border-radius: 50%;
    background: #1f9d55;
    flex: none;
  }
  .status.failed .badge { background: #d43f3a; }
  .badge svg { width: 0.78rem; height: 0.78rem; }
  .message {
    max-width: 25rem;
    margin: 0.75rem auto 0;
    color: #5c6a80;
    font-size: 1rem;
    line-height: 1.55;
  }
  .detail {
    margin-top: 1.1rem;
    color: #8b97a8;
    font-size: 0.8rem;
    font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
    overflow-wrap: anywhere;
  }
  .copy {
    display: inline-flex;
    align-items: center;
    gap: 0.6rem;
    margin-top: 1.55rem;
    padding: 0.45rem 0.9rem 0.45rem 0.5rem;
    border: 2.5px solid #5d6c6c;
    border-radius: 999px;
    background: #fff;
    color: #031b4e;
    font-family: inherit;
    font-size: 1rem;
    font-weight: 600;
    line-height: 1.2;
    cursor: pointer;
  }
  .copy:hover { background: #f6f8fb; }
  .copy:focus-visible { outline: 2px solid #0069ff; outline-offset: 3px; }
  .agents { display: flex; align-items: center; }
  .agents img {
    width: 1.75rem;
    height: 1.75rem;
    margin-left: -0.43rem;
    border-radius: 50%;
    background: #111;
    object-fit: cover;
    box-shadow: 0 0 0 2.5px #fff;
  }
  .agents img:first-child { margin-left: 0; }
  .label { display: grid; }
  .label > span { grid-area: 1 / 1; }
  .label .done { visibility: hidden; text-align: center; }
  .copy.copied .label .idle { visibility: hidden; }
  .copy.copied .label .done { visibility: visible; }
  .copy-icon, .copied-icon { width: 1.05rem; height: 1.05rem; flex: none; color: #8b97a8; }
  .copied-icon { display: none; color: #1f9d55; }
  .copy.copied .copy-icon { display: none; }
  .copy.copied .copied-icon { display: block; }
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
    box-shadow: 0 24px 64px rgba(0, 0, 0, 0.35);
  }
</style>
</head>
<body{{if .Ocean}} class="ocean"{{end}}>
  {{if .Ocean}}<img class="sea" alt="" {{.Background}}>
  <img class="logo" alt="DigitalOcean" {{.Logo}}>{{end}}
  <main>
    <h1>Authorization<br>{{.Outcome}}</h1>
    <p class="status{{if .Failed}} failed{{end}}"><span class="badge" aria-hidden="true">{{if .Failed}}<svg viewBox="0 0 16 16" fill="none"><path d="M5 5 11 11M11 5 5 11" stroke="#fff" stroke-width="2" stroke-linecap="round"/></svg>{{else}}<svg viewBox="0 0 16 16" fill="none"><path d="M3.4 8.2 6.5 11.2 12.6 4.8" stroke="#fff" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/></svg>{{end}}</span> {{.Status}}</p>
    {{if .Message}}<p class="message">{{.Message}}</p>{{end}}
    {{if .Detail}}<p class="detail">{{.Detail}}</p>{{end}}
    {{if .Prompt}}
    <button type="button" class="copy" data-prompt="{{.Prompt}}">
      <span class="agents" aria-hidden="true">
        <img alt="" {{.AgentOpenAI}}>
        <img alt="" {{.AgentCursor}}>
        <img alt="" {{.AgentClaude}}>
      </span>
      <span class="label"><span class="idle">Copy setup prompt for your agent</span><span class="done" aria-live="polite">Copied. Paste it into your agent</span></span>
      <svg class="copy-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" aria-hidden="true"><rect x="8.5" y="8.5" width="11" height="11" rx="2"/><path d="M6.5 15.5h-1a2 2 0 0 1-2-2v-8a2 2 0 0 1 2-2h8a2 2 0 0 1 2 2v1"/></svg>
      <svg class="copied-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path d="M5 12.5 9.2 17 19 7" stroke-linecap="round" stroke-linejoin="round"/></svg>
    </button>
    {{end}}
  </main>
  {{if .Prompt}}<script>
(function () {
  var button = document.querySelector(".copy");
  if (!button) return;
  var timer;
  button.addEventListener("click", function () {
    var text = button.getAttribute("data-prompt");
    var finish = function () {
      button.classList.add("copied");
      window.clearTimeout(timer);
      timer = window.setTimeout(function () {
        button.classList.remove("copied");
      }, 2500);
    };
    var fallback = function () {
      var area = document.createElement("textarea");
      area.value = text;
      area.setAttribute("readonly", "");
      area.style.position = "fixed";
      area.style.left = "-9999px";
      document.body.appendChild(area);
      area.select();
      try { document.execCommand("copy"); } catch (e) {}
      document.body.removeChild(area);
      finish();
    };
    if (navigator.clipboard && navigator.clipboard.writeText) {
      navigator.clipboard.writeText(text).then(finish, fallback);
      return;
    }
    fallback();
  });
})();
</script>{{end}}
</body>
</html>
`))

type pageContent struct {
	// Failed switches the status line from a green check to a red cross.
	Failed bool
	// Outcome completes the "Authorization ..." heading.
	Outcome string
	// Status is the short line beside the check or cross.
	Status string
	// Message and Detail are optional and may repeat text from the
	// authorization server; the template escapes them.
	Message string
	Detail  string
	// Ocean selects the photographed background shared by the signed-in and
	// failed-authorization pages.
	Ocean bool
	// Background is the src attribute for assets/success_background.png.
	Background template.HTMLAttr
	// Logo is the src attribute for assets/digitalocean_logo.svg.
	Logo template.HTMLAttr
	// Agent marks are the src attributes for the three icons on the copy button.
	AgentOpenAI template.HTMLAttr
	AgentCursor template.HTMLAttr
	AgentClaude template.HTMLAttr
	// Prompt, when set, is the agent setup text the signed-in page copies.
	// It is doctl's own text, escaped into the button, and read back by the
	// inline script. Authorization-server text never goes here.
	Prompt string
}

// AgentSetupPrompt is what "Copy setup prompt for your agent" puts on the
// clipboard: a one-line instruction to fetch DigitalOcean's getting-started
// guide and follow it. doctl auth login also prints it in the terminal.
const AgentSetupPrompt = "Fetch https://docs.digitalocean.com/get-started.md and follow it."

var successContent = pageContent{
	Outcome:     "successful",
	Status:      "You can close this tab.",
	Ocean:       true,
	Background:  successBackgroundSrc,
	Logo:        digitalOceanLogoSrc,
	Prompt:      AgentSetupPrompt,
	AgentOpenAI: pngSrc(agentOpenAIPNG),
	AgentCursor: pngSrc(agentCursorPNG),
	AgentClaude: pngSrc(agentClaudePNG),
}

// tryAgain closes the failure messages that have no more specific next step.
const tryAgain = "You can close this tab and try again from your terminal."

func errorContent(status, message, detail string) pageContent {
	return pageContent{
		Failed:     true,
		Outcome:    "failed",
		Status:     status,
		Message:    message,
		Detail:     detail,
		Ocean:      true,
		Background: successBackgroundSrc,
		Logo:       digitalOceanLogoSrc,
	}
}

var stateMismatchContent = errorContent(
	"Sign-in couldn't be verified",
	"This page came from a different sign-in attempt, so doctl ignored it for your safety. Close this tab and run doctl auth login again.",
	"",
)

var missingCodeContent = errorContent(
	"Sign-in didn't finish",
	"DigitalOcean didn't send back an authorization code. "+tryAgain,
	"",
)

// authorizationErrorContent describes an error the authorization server sent
// back. Codes people commonly hit get their own wording; anything else shows
// the server's description and keeps the raw code as a small detail line.
func authorizationErrorContent(e *AuthorizationError) pageContent {
	switch {
	case e.Code == "access_denied":
		return errorContent("Access declined", "doctl wasn't given access to your DigitalOcean account. "+tryAgain, "")
	case isInvalidClientCode(e.Code):
		return errorContent("doctl wasn't recognized", "DigitalOcean didn't accept doctl's sign-in request. "+tryAgain, "Error code: "+e.Code)
	}
	message := "DigitalOcean couldn't complete the sign-in."
	if e.Description != "" {
		message = sentence(e.Description)
	}
	return errorContent("Sign-in didn't finish", message+" "+tryAgain, "Error code: "+e.Code)
}

// sentence capitalizes s and ends it with a period, so a server description
// such as "the request was invalid" reads as a sentence before tryAgain.
func sentence(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	first, size := utf8.DecodeRuneInString(s)
	s = string(unicode.ToUpper(first)) + s[size:]
	if !strings.ContainsAny(s[len(s)-1:], ".!?") {
		s += "."
	}
	return s
}
