/*
Copyright 2019 DigitalOcean
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

// Package ghcl provides functionality for generating a changelog entry from
// GitHub.
package ghcl

import (
	"context"
	"io"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

// A ChangelogEntry contains the data used to display changelog entries and sort them.
type ChangelogEntry struct {
	Number   int
	Body     string
	Username string
	MergedAt time.Time
}

// A ChangelogService implements the methods required to build a changelog.
type ChangelogService interface {
	FetchReleaseTime() (time.Time, error)
	FetchChangelogEntriesUntil(t time.Time) ([]*ChangelogEntry, error)
}

// A GitHubChangelogService implements the ChangelogService for public GitHub.
type GitHubChangelogService struct {
	organization string
	repository   string
	token        string
	client       *github.Client
}

// NewGitHubChangelogService returns a new instance of a
// GitHubChangelogService.
//
// The apiURL can be used to specify a different URL
// to access the GitHub API, typically used for GitHub enterprise customers.
func NewGitHubChangelogService(organization, repository, token, apiURL string) *GitHubChangelogService {
	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	client := github.NewClient(oauth2.NewClient(context.Background(),
		tokenSource))

	if apiURL != "" {
		baseURL, err := url.Parse(apiURL)
		if err != nil {
			panic("enterprise url is invalid: " + err.Error())
		}
		client.BaseURL = baseURL
	}

	return &GitHubChangelogService{
		organization: organization,
		repository:   repository,
		token:        token,
		client:       client,
	}
}

// FetchReleaseTime fetches the time of the latest release on GitHub.
func (gh *GitHubChangelogService) FetchReleaseTime() (time.Time, error) {
	release, _, err := gh.client.Repositories.GetLatestRelease(
		context.Background(), gh.organization, gh.repository)
	if err != nil {
		return time.Time{}, err
	}
	return release.GetPublishedAt().Time, nil
}

// FetchChangelogEntriesUntil fetches all of the pull requests merged before
// the given time.
func (gh *GitHubChangelogService) FetchChangelogEntriesUntil(t time.Time) ([]*ChangelogEntry, error) {
	prs, _, err := gh.client.PullRequests.List(context.Background(),
		gh.organization, gh.repository, &github.PullRequestListOptions{
			ListOptions: github.ListOptions{PerPage: 100},
			State:       "closed",
			Sort:        "updated",
			Direction:   "desc",
		})
	if err != nil {
		return nil, err
	}

	entries := make([]*ChangelogEntry, 0)
	for _, pr := range prs {
		if pr.MergedAt != nil {
			if pr.MergedAt.After(t) {
				entries = append(entries, &ChangelogEntry{
					Number:   pr.GetNumber(),
					Body:     pr.GetTitle(),
					Username: pr.GetUser().GetLogin(),
					MergedAt: pr.GetMergedAt(),
				})
			}
		}
	}
	return entries, nil
}

// FetchChangelogEntries fetches a list of all changelog entries using the
// given changelog service.
func FetchChangelogEntries(cs ChangelogService) ([]*ChangelogEntry, error) {
	releaseTime, err := cs.FetchReleaseTime()
	if err != nil {
		return nil, err
	}
	return cs.FetchChangelogEntriesUntil(releaseTime)
}

// FormatChangelogEntries formats a slice of changelog entries, returning a
// string that can be used to display them.
func FormatChangelogEntries(entries []*ChangelogEntry) string {
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].MergedAt.After(entries[j].MergedAt)
	})

	builder := strings.Builder{}
	for _, entry := range entries {
		builder.WriteString("- #")
		builder.WriteString(strconv.Itoa(entry.Number))
		builder.WriteString(" - @")
		builder.WriteString(entry.Username)
		builder.WriteString(" - ")
		builder.WriteString(entry.Body)
		builder.WriteString("\n")
	}
	return builder.String()
}

// Build fetches a list of changelog entries using the given changelog
// service, and prints them to stdout.
func Build(cs ChangelogService) error {
	entries, err := FetchChangelogEntries(cs)
	if err != nil {
		return err
	}

	notes := FormatChangelogEntries(entries)
	_, err = io.WriteString(os.Stdout, notes)
	return err
}
