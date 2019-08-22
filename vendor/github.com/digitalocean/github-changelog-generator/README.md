# GHCL – GitHub ChangeLog Generator

[![Build Status](https://travis-ci.org/digitalocean/github-changelog-generator.svg?branch=master)](https://travis-ci.org/digitalocean/github-changelog-generator)

A changelog generator for GitHub.

The generator works by,
  * Fetching the time of your repository's most recent release,
  * Fetching all pull requests merged _after_ the time of your most recent release,
  * Outputting a summary of those pull requests.

### Installation

`github-changelog-generator` must be installed from source. Before doing this, you'll need Go 1.11 or later. To install, run `go get -u github.com/digitalocean/github-changelog-generator`. A `github-changelog-generator` binary will then be available under your `$GOBIN` directory.

### Usage

```
Usage of github-changelog-generator
  -org string
    	organization (required)
  -repo string
    	repository (required)
  -token string
    	GitHub token (default env GITHUB_TOKEN)
  -url string
    	alternative GitHub API URL, must be a fully qualified URL with a trailing slash (optional)
```

The output is in the format `- #<pull request number> - @<github username> - <pull request title>`. An example of the output is shown below.

```
- #3 - @some_contributor - update contibuting file
- #2 - @myteammate - update README.md
- #1 - @me - First PR
```

If there haven't been any changes to the repository since the latest release, the changelog generator will not show any output.

### Testing

Some tests use the network to test against real data. To avoid running these with the rest of the suite, use `go test -short ./...`. See [CONTRIBUTING.md](./CONTRIBUTING.md) for more information.

### License

GitHub changelog generator is [Apache-2.0 licensed](./LICENSE).
