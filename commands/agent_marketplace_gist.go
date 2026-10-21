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

// Listing storage for `marketplace publish/deploy`. A listing is a secret
// GitHub gist: its ID is the listing ID and each gist revision SHA is a
// pinned version. Secret gists are readable by anyone holding the ID, which
// is what a shared deploy line needs, and unlisted everywhere else.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

const mkGistDescriptionPrefix = "doctl agent marketplace: "

// mkGitHubAPI is a var so tests can point the client at httptest.
var mkGitHubAPI = "https://api.github.com"

// mkGitHubToken resolves a github.com token: GITHUB_TOKEN, GH_TOKEN, then the
// gh CLI's stored login. Reading needs none; publishing needs one.
var mkGitHubToken = func() (string, error) {
	for _, k := range []string{"GITHUB_TOKEN", "GH_TOKEN"} {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			return v, nil
		}
	}
	out, err := exec.Command("gh", "auth", "token", "--hostname", "github.com").Output()
	if err == nil && strings.TrimSpace(string(out)) != "" {
		return strings.TrimSpace(string(out)), nil
	}
	return "", errors.New("publishing stores the listing as a secret GitHub gist and needs a github.com login: run `gh auth login`, or set GITHUB_TOKEN")
}

type mkGistFile struct {
	Content   string `json:"content"`
	Truncated bool   `json:"truncated,omitempty"`
	RawURL    string `json:"raw_url,omitempty"`
}

type mkGist struct {
	ID          string                `json:"id"`
	HTMLURL     string                `json:"html_url"`
	Description string                `json:"description"`
	UpdatedAt   time.Time             `json:"updated_at"`
	Files       map[string]mkGistFile `json:"files"`
	Owner       struct {
		Login string `json:"login"`
	} `json:"owner"`
	History []struct {
		Version     string    `json:"version"`
		CommittedAt time.Time `json:"committed_at"`
	} `json:"history"`
}

// Version is the gist's newest revision SHA.
func (g *mkGist) Version() string {
	if len(g.History) == 0 {
		return ""
	}
	return g.History[0].Version
}

type mkGistClient struct {
	http  *http.Client
	token string
}

func newMKGistClient(token string) *mkGistClient {
	return &mkGistClient{http: &http.Client{Timeout: 30 * time.Second}, token: token}
}

func (c *mkGistClient) do(ctx context.Context, method, path string, body any, out any) error {
	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, mkGitHubAPI+path, rd)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("github: %w", err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if resp.StatusCode >= 300 {
		var e struct {
			Message string `json:"message"`
		}
		_ = json.Unmarshal(data, &e)
		if e.Message == "" {
			e.Message = strings.TrimSpace(string(data))
		}
		return &mkHTTPError{Status: resp.StatusCode, Msg: fmt.Sprintf("github %s %s: %d %s", method, path, resp.StatusCode, e.Message),
			RateLimited: resp.Header.Get("X-RateLimit-Remaining") == "0"}
	}
	if out != nil {
		return json.Unmarshal(data, out)
	}
	return nil
}

type mkHTTPError struct {
	Status      int
	Msg         string
	RateLimited bool
}

func (e *mkHTTPError) Error() string {
	switch {
	case e.RateLimited:
		return e.Msg + " (GitHub rate limit; run `gh auth login` or set GITHUB_TOKEN to raise it)"
	case e.Status == http.StatusUnauthorized:
		return e.Msg + " (if GITHUB_TOKEN or GH_TOKEN is set for another GitHub host, unset it to use your gh login)"
	}
	return e.Msg
}

func httpStatus(err error) int {
	var he *mkHTTPError
	if errors.As(err, &he) {
		return he.Status
	}
	return 0
}

var errMKListingNotFound = errors.New("listing not found: check the listing ID, or ask the creator for a fresh deploy line")

// create stores a new listing. files are keyed by listing path.
func (c *mkGistClient) create(ctx context.Context, name string, files map[string]string) (*mkGist, error) {
	var g mkGist
	err := c.do(ctx, http.MethodPost, "/gists", map[string]any{
		"description": mkGistDescriptionPrefix + name,
		"public":      false,
		"files":       gistPayload(files, nil),
	}, &g)
	if httpStatus(err) == http.StatusNotFound {
		return nil, errors.New("GitHub refused to create the gist: the token needs the gist scope (run `gh auth refresh -s gist`, or use a token that has it)")
	}
	return &g, err
}

// update writes a new revision of an existing listing, deleting files that
// are no longer part of it (a deselected skill must not linger).
func (c *mkGistClient) update(ctx context.Context, id, name string, files map[string]string) (*mkGist, error) {
	cur, err := c.get(ctx, id, "")
	if errors.Is(err, errMKListingNotFound) {
		return nil, fmt.Errorf("listing %s no longer exists on GitHub; delete %s in the listing folder to publish it as a new listing", id, mkStateFile)
	}
	if err != nil {
		return nil, err
	}
	var g mkGist
	err = c.do(ctx, http.MethodPatch, "/gists/"+id, map[string]any{
		"description": mkGistDescriptionPrefix + name,
		"files":       gistPayload(files, cur.Files),
	}, &g)
	return &g, err
}

func gistPayload(files map[string]string, existing map[string]mkGistFile) map[string]any {
	out := map[string]any{}
	for path, content := range files {
		out[gistFileName(path)] = map[string]string{"content": content}
	}
	for name := range existing {
		if _, ok := out[name]; !ok {
			out[name] = nil
		}
	}
	return out
}

// get fetches a listing at version (a revision SHA, full or a prefix of at
// least 7 characters) or, when empty, latest. Truncated files (over 1 MB)
// are fetched from their raw URL.
func (c *mkGistClient) get(ctx context.Context, id, version string) (*mkGist, error) {
	if version != "" && len(version) < 40 {
		full, err := c.resolveVersion(ctx, id, version)
		if err != nil {
			return nil, err
		}
		version = full
	}
	path := "/gists/" + id
	if version != "" {
		path += "/" + version
	}
	var g mkGist
	err := c.do(ctx, http.MethodGet, path, nil, &g)
	if httpStatus(err) == http.StatusUnauthorized && c.token != "" {
		// A token for another GitHub (GHES) is refused by github.com, but a
		// secret gist is readable without one.
		err = (&mkGistClient{http: c.http}).do(ctx, http.MethodGet, path, nil, &g)
	}
	if httpStatus(err) == http.StatusNotFound || httpStatus(err) == http.StatusUnprocessableEntity {
		return nil, errMKListingNotFound
	}
	if err != nil {
		return nil, err
	}
	for name, f := range g.Files {
		if !f.Truncated || f.RawURL == "" {
			continue
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, f.RawURL, nil)
		if err != nil {
			return nil, err
		}
		resp, err := c.http.Do(req)
		if err != nil {
			return nil, err
		}
		b, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20+1))
		resp.Body.Close()
		switch {
		case err != nil:
			return nil, err
		case resp.StatusCode != http.StatusOK:
			return nil, fmt.Errorf("github: fetching listing file %s: HTTP %d", name, resp.StatusCode)
		case len(b) > 8<<20:
			return nil, fmt.Errorf("listing file %s is over 8 MiB", name)
		}
		f.Content, f.Truncated = string(b), false
		g.Files[name] = f
	}
	return &g, nil
}

// resolveVersion expands a short revision SHA, which the gist API refuses,
// against the listing's history.
func (c *mkGistClient) resolveVersion(ctx context.Context, id, short string) (string, error) {
	if len(short) < 7 {
		return "", fmt.Errorf("version %q is too short; use at least 7 characters of it", short)
	}
	latest, err := c.get(ctx, id, "")
	if err != nil {
		return "", err
	}
	match := ""
	for _, h := range latest.History {
		if strings.HasPrefix(h.Version, short) {
			if match != "" {
				return "", fmt.Errorf("version %q matches more than one version; use more characters", short)
			}
			match = h.Version
		}
	}
	if match == "" {
		return "", fmt.Errorf("listing %s has no version %s", id, short)
	}
	return match, nil
}

// listMine returns the authenticated user's marketplace listings.
func (c *mkGistClient) listMine(ctx context.Context) ([]mkGist, error) {
	var out []mkGist
	for page := 1; page <= 30; page++ {
		var batch []mkGist
		if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/gists?per_page=100&page=%d", page), nil, &batch); err != nil {
			return nil, err
		}
		for _, g := range batch {
			if strings.HasPrefix(g.Description, mkGistDescriptionPrefix) {
				out = append(out, g)
			}
		}
		if len(batch) < 100 {
			break
		}
	}
	return out, nil
}

// listingFromGist decodes listing.json and maps gist file names back to
// listing paths, so review and deploy see the same layout publish wrote.
func listingFromGist(g *mkGist) (*mkListing, map[string]string, error) {
	lf, ok := g.Files["listing.json"]
	if !ok {
		return nil, nil, fmt.Errorf("gist %s is not a marketplace listing (no listing.json)", g.ID)
	}
	var l mkListing
	if err := json.Unmarshal([]byte(lf.Content), &l); err != nil {
		return nil, nil, fmt.Errorf("listing.json is not valid: %w", err)
	}
	if l.Schema != mkListingSchema {
		return nil, nil, fmt.Errorf("listing schema %q is not supported by this doctl (want %s); update doctl", mkCleanText(l.Schema), mkListingSchema)
	}
	cleanMKListing(&l)
	if err := validateMKLayout(&l); err != nil {
		return nil, nil, fmt.Errorf("listing is malformed: %w", err)
	}
	files := map[string]string{"listing.json": lf.Content}
	paths := []string{}
	for _, s := range l.Skills {
		paths = append(paths, s.File)
	}
	if l.ClaudeMD != "" {
		paths = append(paths, l.ClaudeMD)
	}
	for _, p := range paths {
		f, ok := g.Files[gistFileName(p)]
		if !ok {
			return nil, nil, fmt.Errorf("listing is incomplete: %s is missing", p)
		}
		files[p] = f.Content
	}
	return &l, files, nil
}

// cleanMKListing strips terminal control characters from every listing field
// that deploy or show prints. The listing is written by someone else.
func cleanMKListing(l *mkListing) {
	for _, p := range []*string{&l.Name, &l.Slug, &l.Description, &l.Creator, &l.Agent, &l.ClaudeMD, &l.PermissionMode, &l.CreatedAt} {
		*p = mkCleanText(*p)
	}
	for i := range l.Skills {
		l.Skills[i].Name, l.Skills[i].Description, l.Skills[i].File = mkCleanText(l.Skills[i].Name), mkCleanText(l.Skills[i].Description), mkCleanText(l.Skills[i].File)
	}
	for i := range l.SecretSlots {
		s := &l.SecretSlots[i]
		s.Name, s.Label, s.Reason = mkCleanText(s.Name), mkCleanText(s.Label), mkCleanText(s.Reason)
	}
	for i := range l.EgressAllowList {
		l.EgressAllowList[i] = mkCleanText(l.EgressAllowList[i])
	}
	l.Dropped = nil
}

// parseListingRef splits `<id>` or `<id>@<version>`, and accepts a pasted
// gist URL so a link from a tweet works as-is.
func parseListingRef(ref string) (id, version string, err error) {
	ref = strings.TrimSpace(ref)
	ref = strings.TrimPrefix(ref, "https://")
	ref = strings.TrimPrefix(ref, "gist.github.com/")
	if i := strings.LastIndex(ref, "/"); i >= 0 {
		ref = ref[i+1:]
	}
	id, version, _ = strings.Cut(ref, "@")
	if id == "" || strings.ContainsAny(id, " ?#") {
		return "", "", fmt.Errorf("%q is not a listing ID; use the ID from the creator's deploy line", ref)
	}
	return id, version, nil
}
