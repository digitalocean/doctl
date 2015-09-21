# DOIT

![Travis Build Status](https://travis-ci.org/bryanl/doit.svg?branch=master)
[![Coverage Status]
(https://coveralls.io/repos/bryanl/doit/badge.svg?branch=master)]
(https://coveralls.io/r/bryanl/doit?branch=master)

```
NAME:
   doit - DigitalOcean Interactive Tool

USAGE:
   doit [global options] command [command options] [arguments...]

VERSION:
   0.4.0

COMMANDS:
   account		account commands
   action		action commands
   domain		domain commands
   droplet		droplet commands
   droplet-action	droplet action commands
   image-action		image action commands
   image		image commands
   key			ssh key commands
   region		region commands
   size			size commands
   ssh			SSH to droplet. Provide name or id
   help, h		Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --access-token 		DigitalOcean API V2 Access Token [$DIGITALOCEAN_ACCESS_TOKEN]
   --debug		Debug
   --output 		output format (json or text)
   --help, -h		show help
   --version, -v	print the version

```
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

## Plugins

`doit` functionality can be enhanced using plugins.
