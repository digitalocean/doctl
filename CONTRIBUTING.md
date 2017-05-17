# Contributing to doctl

**First:** if you're unsure or afraid of _anything_, just ask
or submit the issue or pull request anyways. You won't be yelled at for
giving your best effort. The worst that can happen is that you'll be
politely asked to change something. We appreciate any sort of contributions,
and don't want a wall of rules to get in the way of that.

However, for those people who want a bit more guidance on the
best way to contribute to the project, read on. This document will cover
what we're looking for. By addressing all the points we're looking for,
it raises the chances we can quickly merge or address your contributions.

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
   Categorization is done via tags. For example, bugs are marked as "bugs".

3. Unless it is critical, the issue is left for a period of time (sometimes
   many weeks), giving outside contributors a chance to address the issue.

4. The issue is addressed in a pull request or commit. The issue will be
   referenced in the commit message so that the code that fixes it is clearly
   linked.

5. The issue is closed. Sometimes, valid issues will be closed to keep
   the issue tracker clean. The issue is still indexed and available for
   future viewers, or can be re-opened if necessary.

## Setting up Go to work on doctl

If you have never worked with Go before, you will have to complete the
following steps in order to be able to compile and test doctl.

1. Install Go. Make sure the Go version is at least Go 1.6.
   On Mac OS X, you can `brew install go` to install Go 1.6.

1. Set and export the `GOPATH` environment variable and update your `PATH`.
   For example, you can add to your `.bash_profile`.

    ```
    export GOPATH=$HOME/Documents/golang
    export PATH=$PATH:$GOPATH/bin
    ```

1. Make your changes to the doctl source, being sure to run the basic
   tests.

1. If everything works well and the tests pass, run `go fmt` on your code
   before submitting a pull request.

## Contributing code

### Upgrading `godo`

When you upgrade `godo` you have to re-generate the mocks using [mockery](https://github.com/vektra/mockery),
so first install mockery in your `GOPATH` then run the `script/regenmocks.sh` script to produce them.
