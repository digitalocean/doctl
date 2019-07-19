# Contributing to doctl

<!-- markdown-toc start - Don't edit this section. Run M-x markdown-toc-refresh-toc -->
**Table of Contents**

- [Contributing to doctl](#contributing-to-doctl)
    - [Issues](#issues)
        - [Reporting an Issue](#reporting-an-issue)
        - [Issue Lifecycle](#issue-lifecycle)
    - [Developing](#developing)
        - [Go environment](#go-environment)
        - [Docker](#docker)
        - [Testing](#testing)
            - [`godo` mocks](#godo-mocks)
        - [Build Scripts](#build-scripts)
    - [Releasing](#releasing)
        - [Setup](#setup)
        - [Cutting a release](#cutting-a-release)
        - [Updating Homebrew](#updating-homebrew)

<!-- markdown-toc end -->

**First:** if you're unsure or afraid of _anything_, just ask
or submit the issue or pull request anyways. You won't be yelled at for
giving your best effort. The worst that can happen is that you'll be
politely asked to change something. We appreciate all contributions!

For those folks who want a bit more guidance on the best way to
contribute to the project, read on. Addressing the points below
lets us merge or address your contributions quickly.

## Issues

### Reporting an Issue

* Make sure you test against the latest released version. It is possible
  we already fixed the bug you're experiencing.

* If you experienced a panic, please create a [gist](https://gist.github.com)
  of the *entire* generated crash log for us to look at. Double check
  no sensitive items were in the log.

* Respond as promptly as possible to any questions made by the _doctl_
  team to your issue. Stale issues will be closed.

### Issue Lifecycle

1. The issue is reported.

2. The issue is verified and categorized by a _doctl_ collaborator.
   Categorization is done via labels. For example, bugs are marked as "bugs".

3. Unless it is critical, the issue is left for a period of time (sometimes
   many weeks), giving outside contributors a chance to address the issue.

4. The issue is addressed in a pull request or commit. The issue will be
   referenced in the commit message so that the code that fixes it is clearly
   linked.

5. The issue is closed. Sometimes, valid issues will be closed to keep
   the issue tracker clean. The issue is still indexed and available for
   future viewers, or can be re-opened if necessary.

## Developing

`doctl` has `make` commands for most tooling in the `Makefile`. Run `make`
or `make help` for a list of available commands with descriptions.

### Go environment

The minimal version of Golang for `doctl` is 1.11. `doctl` uses [Go
modules](https://github.com/golang/go/wiki/Modules) for dependency
management [with vendoring](https://github.com/golang/go/wiki/Modules#how-do-i-use-vendoring-with-modules-is-vendoring-going-away). 
Please run `make vendor` after any dependency modifications.

Be sure to run `go fmt` on your code before submitting a pull request.

### Docker

You can create a local Docker container via `make docker_build`.

### Testing

Run the tests locally via `make test`, or on Travis CI by opening a PR.

#### `godo` mocks

When you upgrade `godo` you have to re-generate the mocks. 

    ```
    make mocks
    ```

### Build Scripts

If you modify the build scripts, use `make shellcheck` to check your changes.

## Releasing

### Setup

To release `doctl`, you need to install:

* [gothub](https://github.com/itchio/gothub)

And make it available in your `PATH`. You can use `go get -u` and add your
`$GOPATH/bin` to your `PATH` so your scripts will find it.

You will also need a valid `GITHUB_TOKEN` environment variable with access to the `digitalocean/doctl` repo. You can generate a token [here](https://github.com/settings/tokens), it needs the `public_repo` access.

### Cutting a release

1. Run `make changelog` and add the results to the [CHANGELOG](https://github.com/digitalocean/doctl/blob/master/CHANGELOG.md)
   under the version you're going to release if they aren't already there.

   Update the version in:

   * `doit.go`
   * `Dockerfile`

1. Generate a PR, get it reviewed, and merge

1. To build `doctl` for all its platforms, run `scripts/stage.sh major minor patch` 
(e.g. `scripts/stage.sh 1 5 0`). This will place all files and their checksums 
in `builds/major.minor.patch/release`.

1. Mark the release on GitHub with `scripts/release.sh v<version>` (e.g. `scripts/release.sh v1.5.0`, _note_ the `v`),

1. Upload using `scripts/upload.sh <version>`.

1. Go to [releases](https://github.com/digitalocean/doctl/releases) and update the release
   description to contain all changelog entries for this specific release. Uncheck the pre-release checkbox.

### Updating Homebrew

Using the url and sha from the github release, update the 
[homebrew formula](https://github.com/Homebrew/homebrew-core/blob/master/Formula/doctl.rb).
You can use `brew bump-formula-pr doctl`, or 

1. fork `homebrew-core`
1. create a branch named `doctl-<version>`
1. update the url and the sha256 using the values for the archive in the github release
1. commit your changes
1. submit a PR to homebrew
