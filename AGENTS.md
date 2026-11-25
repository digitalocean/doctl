# AGENTS.md

`doctl` is the official command-line interface for the DigitalOcean API. It is
a Go CLI built on [Cobra](https://github.com/spf13/cobra) and the
[godo](https://github.com/digitalocean/godo) API client.
User docs: https://docs.digitalocean.com/reference/doctl/

## Layout

- `cmd/doctl/`: `main` package
- `commands/`: one file per command group (`droplets.go`, `kubernetes.go`, ...)
  with its `_test.go` unit tests. Top-level groups are registered in `commands/doit.go`.
- `commands/displayers/`: table and JSON output for each resource type
- `do/`: service interfaces that wrap godo (`do.AccountService`, ...). Commands
  call these through `CmdConfig`, never godo directly.
- `do/mocks/`: gomock mocks generated from `do/*.go`. Do not hand-edit.
- `integration/`: tests that run the compiled binary against an `httptest` server
- `args.go`: flag name constants (`ArgXxx`)
- `vendor/`: vendored modules. Do not hand-edit.
- `pkg/`, `internal/`: shared helpers

## Commands

Run `make help` for the full list. Dependencies are vendored, so plain `go`
commands need `-mod=vendor`.

- `make build`: build `builds/doctl`
- `make test_unit`: unit tests for `commands`, `do`, `pkg`, `internal` and the root package
- `make test_integration`: integration tests (builds the binary first)
- `make gofmt_check`: formatting check that CI runs (also rewrites `interface{}` to `any`)
- `make check_focused`: fails if an integration test is left focused
- `make shellcheck`: lint `scripts/` (needs `shellcheck`)
- `make mocks`: regenerate `do/mocks/` after changing a `do/` interface
- `make vendor`: `go mod tidy` and `go mod vendor` after any dependency change
- `make upgrade_godo`: bump godo, re-vendor and regenerate mocks
- `make docs`: generate YAML reference docs into `builds/docs/yaml`

CI (`.github/workflows/ci.yml`) runs `gofmt_check`, `shellcheck`, `test_unit`,
`check_focused` and `test_integration`. `typos` runs as a separate spell check.

## Adding or changing a command

1. Add or extend the service interface and implementation in `do/`, then run
   `make mocks`. If a new service is added, also add its `mockgen` line to
   `scripts/regenmocks.sh` and wire it into `CmdConfig` (`commands/command_config.go`)
   and the test mocks (`commands/commands_test.go`).
2. Define the command with `CmdBuilder` (see `commands/account.go`). Add flags
   with `AddStringFlag` and friends, using constants from `args.go`.
3. Add a displayer in `commands/displayers/` for any new output type.
4. Unit test with `withTestClient` and the gomock mocks. Add an integration
   test under `integration/` that asserts on the binary's output.

Follow the help-text rules in `CONTRIBUTING.md`: short descriptions are
imperative phrases with no trailing period, long descriptions are full
sentences, and command and flag names go in backticks.

## Conventions

- PRs must come from a fork, even with write access.
- PR titles usually start with the command area, for example `agents: ...` or `auth: ...`.
- `CHANGELOG.md` is frozen. Release notes are generated from merged PRs
  (`make changes`) and published on GitHub Releases.
