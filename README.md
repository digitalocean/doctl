# DOIT

![Travis Build Status](https://travis-ci.org/bryanl/doit.svg?branch=master)
[![Coverage Status]
(https://coveralls.io/repos/bryanl/doit/badge.svg?branch=master)]
(https://coveralls.io/r/bryanl/doit?branch=master)

```
Usage:
  doit [command]

Available Commands:
  account            account commands
  action             action commands
  auth               auth commands
  domain             domain commands
  droplet-action     droplet action commands
  droplet            droplet commands
  floating-ip        floating IP commands
  floating-ip-action floating IP action commands
  image              image commands
  region             region commands
  size               size commands
  ssh-key            sshkey commands
  ssh                ssh to droplet

Flags:
  -t, --access-token="": DigtialOcean API V2 Access Token
  -h, --help[=false]: help for doit
  -o, --output="text": output formt [text|json]

Use "doit [command] --help" for more information about a command.

```

## Initialization

To automatically retrieve your access token from DigitalOcean, run `doit auth login`. This process will authenticate you with DigitalOcean and retrieve an access token. If your shell does not have access to a web browser (because of a remote Linux shell with no DISPLAY environment variable or you've specified the CLIAUTH=1 flag), `doit` will provide you with a link for offline authentication.


## Configuration

By default, `doit` will load a configuration file from `$HOME/.doitcfg` if found.

### Configuration OPTIONS

* `access-token` - The DigitalOcean access token. You can generate a token in the [Apps & API](https://cloud.digitalocean.com/settings/applications) Of the DigitalOcean control panel or use `doit auth login`.
* `output` - Type of output to display results in. Choices are `json` or `text`. If not supplied, `doit` will default to `text`.

Example:

```yaml
{
  access-token: MY_TOKEN
  output: text
}
```

## Building and dependencies

`doit`'s dependencies are managed by [glide](https:/github.com/Mastermind/glide). To develop locally, an installation of glide is required. Once glide is installed, add new dependencies with `glide install <dep>`.

## Releasing

To build `doit` for all it's platforms, run `script/build.sh <version>`. To upload `doit` to Github, run `script/release.sh <version>`. A valid `GITHUB_TOKEN` environment variable with access to the `bryanl/doit` repository will be required.
