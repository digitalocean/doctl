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




## Configuration

By default, `doit` will load a configuration file from `$HOME/.doitcfg` if found.

### Configuration OPTIONS

* `access-token` - The DigitalOcean access token. You can generate a token in the [Apps & API](https://cloud.digitalocean.com/settings/applications) Of the DigitalOcean control panel.
* `output` - Type of output to display results in. Choices are `json` or `text`. If not supplied, `doit` will default to `text`.

Example:

```yaml
{
  access-token: MY_TOKEN
  output: text
}
```

## Building and dependencies

`doit`'s dependencies are managed by [godep](https:/.com/tools/godep). To add new packages, you must
run `godep save ./...` to update the vendored dependencies. External dependencies have been rewritten using `godep`.
