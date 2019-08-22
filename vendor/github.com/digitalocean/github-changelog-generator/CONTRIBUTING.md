# Contributing

**First:** If you're unsure or afraid of _anything_, just ask or submit the issue or pull request anyway. You won't be yelled at for giving your best effort. The worst that can happen is that you'll be politely asked to change something. We appreciate all contributions!

For those folks who want a bit more guidance on the best way to contribute to the project, read on. Addressing the points below lets us merge or address your contributions quickly.

## Issues

### Reporting an Issue

* Make sure you test against the latest released version. It is possible we already fixed the bug you're experiencing.

* If you experienced a panic, please create a [gist](https://gist.github.com) of the *entire* generated crash log for us to look at. Double check no sensitive items were in the log.

* Respond as promptly as possible to any questions made by the `github-changelog-generator` team to your issue. Stale issues will be closed.

## Developing

`github-changelog-generator` uses the standard Go toolset. Run the command via `go run main.go [flags]`. See the section on testing below for information on how `github-changelog-generator` is tested.

### Go environment

The minimal version of Golang for `github-changelog-generator` is 1.11. `github-changelog-generator` uses [Go modules](https://github.com/golang/go/wiki/Modules) for dependency management [with vendoring](https://github.com/golang/go/wiki/Modules#how-do-i-use-vendoring-with-modules-is-vendoring-going-away). Please run `go mod vendor` and `go mod tidy` after any dependency modifications.

Be sure to run `go fmt` on your code before submitting a pull request.

### Testing

Run the unit tests locally via `go test -short ./...`, or on Travis CI by opening a PR. If you wish to run the suite with the integration tests, use `go test ./...`. Keep in mind that these tests use the network to issue real HTTP requests against the GitHub API.

