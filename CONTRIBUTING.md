# Contributing to doctl

<!-- Non emacs users, feel free to update the toc by hand. -->
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
            - [Writing Tests](#writing-tests)
            - [`godo` mocks](#godo-mocks)
            - [Build Scripts](#build-scripts)
    - [Releasing](#releasing)
        - [Prerequisites](#prerequisites)
        - [Cutting a release](#cutting-a-release)
            - [Oops! What now?](#oops-what-now)
            - [Docker Hub](#docker-hub)
            - [Snap](#snap)
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

#### Writing Tests

In doctl, we have two kinds of tests: unit tests and integration tests. Both are run with Go's
built-in `go test` tool.

Unit tests live in the `_test.go` files. The bulk of these tests live in the `commands` package,
and exist to ensure that a CLI command produces an expected output. For each unit test, we
typically rely on an accompanying mocked API call. These mocks are generated via `gomock`, and
can be set to return different values from the API calls to test how our commands behave when
given different inputs.

Writing a unit test for a new command typically looks like this,

1. Write your new command.
2. If your new command depends on a mocked `godo` call, generate a mock for it. See
[the section below](#godo-mocks) about regenerating mocks to learn how to do this.
3. Use your new mocks to stub out the API call, and write a test case. We use
`github.com/stretchr/testify/assert` for our assertions. Test cases typically look like the following:
    ```go
    func TestMyNewCommand(t *testing.T) {
        // Use the `withTestClient` helper to access our tets config, as well as the godo API
        // mocks.
        withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
            // Mock the godo API call.
            tm.myNewCommandMock.EXPECT().Get("some-value").Return("some-other-value")

            // Optionally add a CLI argument.
            config.Args = append(config.Args, "some-value")

            // Optionally add a CLI flag.
            config.Doit.Set(config.NS, "--my-flag", "some-value")

            // Execute your command.
            err := RunMyNewCommand(config)

            // Add assertions to check if your test passes.
            assert.NoError(t, err)
        })
    }
    ```

Integration tests live under the top-level `integration` directory. These tests exist to ensure
that an invocation of a command though this CLI produces the expected output. These tests use a
mocked HTTP client, but run the actual compiled doctl binary.

Writing an integration test typically looks like this,

1. Write your new command.
2. Mock the API's JSON response that your command depends on.
3. Execute your command using `exec.Command` on the test CLI binary.
4. Add assertions to check the output from the CLI command.

Rather than providing an example here, please have a look at the [`integration/account_test.go`](/integration/account_test.go)
file to see what an integration test typically looks like.

#### `godo` mocks

When you upgrade `godo` you have to re-generate the mocks. 

    ```
    make mocks
    ```

#### Build Scripts

If you modify the build scripts, you can use `make shellcheck` to
check your changes. You'll need to install [shellcheck](https://github.com/koalaman/shellcheck)
to do so. Alternatively, you can open a "WIP" (Work In Progress) pull request
and let TravisCI run shellcheck for you.

## Releasing

### Prerequisites

* [goreleaser](https://goreleaser.com/install/)

* [docker](https://docs.docker.com/install/)

* a valid `GITHUB_TOKEN` environment variable with access to the
  `digitalocean/doctl` repo. You can generate a token
  [here](https://github.com/settings/tokens), it needs the `public_repo`
  access.

* a valid [Docker Hub](dockerhub.com) login with access to the `digitalocean` account. Post
  in #it_support to request access.

* a valid [ubuntu one](https://login.ubuntu.com) login with access to the `digitalocean` snapcraft account. 
  Post in #it_support to request access.

### Cutting a release

1. Run `make changes` to review the changes since the last
   release. Based on the changes, decide what kind of release you are
   doing (bugfix, feature or breaking). 
   `doctl` follows [semantic versioning](semver.org), ask if you aren't sure.

1. Cut a release using `BUMP=(bugfix|feature|breaking) make bump_and_release`. 
   (Bugfix, feature and breaking are aliases for semver's patch, minor and major.
   BUMP will also accept `patch`, `minor` and `major`, if you prefer). The command
   assumes you have a remote repository named `origin` pointing to this
   repository. If you'd prefer to specify a different remote repository, you can
   do so by setting `ORIGIN=(preferred remote name)`.

#### Oops! What now?

`make bump_and_release` calls a series of smaller tasks under the
hood. If the target fails, fix the problem and use the smaller tasks
to finish the release. `make release` may be of particular interest; 
it releases the most recent existing tag. Check `Makefile` for other
internal targets of interest.

#### Docker Hub

`make bump_and_release` and `make release` push new releases to dockerhub. Publishing
to Docker Hub uses `goreleaser` integration. If something goes wrong, you can run
`make release` to try again or fall back to `goreleaser`.

#### Snap

`make bump_and_release` and `make release` push new releases to the snap store. You
can also build and push the snap using `make _snap`. Specify the release channel using
the environment variable `CHANNEL`, which defaults to `stable`:

    CHANNEL=candidate make _snap

#### Updating Homebrew

Using the url and sha from the github release, update the 
[homebrew formula](https://github.com/Homebrew/homebrew-core/blob/master/Formula/doctl.rb).
You can use `brew bump-formula-pr doctl`, or 

1. fork `homebrew-core`
1. create a branch named `doctl-<version>`
1. update the url and the sha256 using the values for the archive in the github release
1. commit your changes
1. submit a PR to homebrew

