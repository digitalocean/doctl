# doctl

![Travis Build Status](https://travis-ci.org/bryanl/doit.svg?branch=master)

```
doctl is a command line interface for the DigitalOcean API.

Usage:
  doctl [command]

Available Commands:
  account     account commands
  auth        auth commands
  compute     compute commands
  version     show the current version

Flags:
  -t, --access-token string   DigitalOcean API V2 Access Token
  -h, --help                  help for doctl
  -o, --output string         output formt [text|json] (default "text")
  -v, --verbose               verbose output

Use "doctl [command] --help" for more information about a command.

```

## Initialization

To automatically retrieve your access token from DigitalOcean, run `doctl auth login`. This process will authenticate 
you with DigitalOcean and retrieve an access token. If your shell does not have access to a web browser 
(because of a remote Linux shell with no DISPLAY environment variable or you've specified the CLIAUTH=1 flag), `doctl` 
will give you a link for offline authentication.


## Configuration

By default, `doctl` will load a configuration file from `$HOME/.doctlcfg` if found.

### Configuration OPTIONS

* `access-token` - The DigitalOcean access token. You can generate a token in the 
[Apps & API](https://cloud.digitalocean.com/settings/applications) section of the DigitalOcean control panel or use 
`doctl auth login`.
* `output` - Type of output to display results in. Choices are `json` or `text`. If not supplied, `doctl` will default
 to `text`.

Example:

```yaml
{
  access-token: MY_TOKEN
  output: text
}
```

## Building and dependencies

`doctl`'s dependencies are managed by [gvt](https://github.com/FiloSottile/gvt). To add dependencies, use `gvt fetch`.

## Releasing

To build `doctl` for all it's platforms, run `script/build.sh <version>`. To upload `doctl` to Github, 
run `script/release.sh <version>`. A valid `GITHUB_TOKEN` environment variable with access to the `bryanl/doctl` 
repository is required.
