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

package main

import (
	"flag"
	"log"
	"os"

	"github.com/digitalocean/github-changelog-generator/ghcl"
)

var (
	org   = flag.String("org", "", "organization (required)")
	repo  = flag.String("repo", "", "repository (required)")
	token = flag.String("token", os.Getenv("GITHUB_TOKEN"), "GitHub token")
	url   = flag.String("url", "", "alternative GitHub API URL, must be a fully qualified URL with a trailing slash (optional)")
)

func main() {
	flag.Parse()

	if *org == "" || *repo == "" {
		flag.Usage()
		os.Exit(1)
	}

	cs := ghcl.NewGitHubChangelogService(*org, *repo, *token, *url)
	if err := ghcl.Build(cs); err != nil {
		log.Fatalln(err)
	}
}
