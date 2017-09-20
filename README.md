# doctl

[![Build Status](https://travis-ci.org/digitalocean/doctl.svg?branch=master)](https://travis-ci.org/digitalocean/doctl)

```
doctl is a command line interface for the DigitalOcean API.

Usage:
  doctl [command]

Available Commands:
  account     account commands
  auth        auth commands
  completion  completion commands
  compute     compute commands
  version     show the current version

Flags:
  -t, --access-token string   API V2 Access Token
  -c, --config string         config file (default is $HOME/.config/doctl/config.yaml)
  -o, --output string         output format [text|json] (default "text")
      --trace                 trace api access
  -v, --verbose               verbose output

Use "doctl [command] --help" for more information about a command.
```

## Installation

### Option 1 - Use a Package Manager (preferred method)

#### OS X

You can use [Homebrew](http://brew.sh) to install `doctl` on Mac OS X by using the command below:

```
brew install doctl
```

#### Linux

You can use [snap](https://snapcraft.io) to install `doctl` on Ubuntu and [snap supported](https://snapcraft.io/docs/core/install) systems by using the command below:

```
snap install doctl
```

#### Windows

Integrations with package managers for Windows are to come.

### Option 2 – Download a Release from GitHub

Visit the [Releases page](https://github.com/digitalocean/doctl/releases) for the `doctl` GitHub project, and find the appropriate archive for your operating system and architecture.  (For OS X systems, remember to use the `darwin` archive.)

#### OS X and GNU/Linux

You can download the archive from your browser, or copy its URL and retrieve it to your home directory with `wget` or `curl`:

```
cd ~

# OS X
curl -L https://github.com/digitalocean/doctl/releases/download/v1.7.0/doctl-1.7.0-darwin-10.6-amd64.tar.gz | tar xz

# linux (with wget)
wget -qO- https://github.com/digitalocean/doctl/releases/download/v1.7.0/doctl-1.7.0-linux-amd64.tar.gz  | tar xz
# linux (with curl)
curl -L https://github.com/digitalocean/doctl/releases/download/v1.7.0/doctl-1.7.0-linux-amd64.tar.gz  | tar xz
```

Move the `doctl` binary to somewhere in your path.  For example:

```
sudo mv ./doctl /usr/local/bin
```

#### Windows

On Windows systems, you should be able to [download the Windows release](windows-release), and then double-click the zip archive to extract the `doctl.exe` executable.

### Option 3 – Build From Source

Alternatively, if you have a Go environment configured, you can install the development version of `doctl` from the command line like so:

```
go get github.com/digitalocean/doctl/cmd/doctl
```

### Option 4 – Build with Docker

If you have Docker installed, you can build with the Dockerfile a Docker image and run `doctl` within a Docker container.

```
# build Docker image
docker build -t doctl .

# usage
docker run -e DIGITALOCEAN_ACCESS_TOKEN doctl <followed by doctl commands>
```

## Initialization

To use `doctl`, a DigitalOcean access token is required. [Generate](https://cloud.digitalocean.com/settings/api/tokens)
a new token and run `doctl auth init`, or set the environment variable, `DIGITALOCEAN_ACCESS_TOKEN`, with your new
token.

## Configuration

By default, `doctl` will load a configuration file from `$XDG_CONFIG_HOME/doctl/config.yaml` if found. If
the `XDG_CONFIG_HOME` environment variable is not, the path will default to `$HOME/.config/doctl/config.yaml` on
Unix-like systems, and `%APPDATA%/doctl/config/config.yaml` on Windows.

The configuration file has changed locations in recent versions, and a warning will be displayed if your configuration
exists at the legacy location.

### Configuration OPTIONS

* `access-token` - The DigitalOcean access token. You can generate a token in the
[Apps & API](https://cloud.digitalocean.com/settings/api/tokens) section of the DigitalOcean control panel and then use it
with `doctl auth init`.
* `output` - Type of output to display results in. Choices are `json` or `text`. If not supplied, `doctl` will default
 to `text`.

Example:

```yaml
access-token: MY_TOKEN
output: text
```

## Examples

`doctl` is able to interact with all of your DigitalOcean resources. Below are a few common usage examples. To learn more about the features available, see [the full tutorial on the DigitalOcean community site](https://www.digitalocean.com/community/tutorials/how-to-use-doctl-the-official-digitalocean-command-line-client).

* List all Droplets on your account:

    `doctl compute droplet list`

* Create a Droplet:

    `doctl compute droplet create <name> --region <region-slug> --image <image-slug> --size <size-slug>`

* Assign a Floating IP to a Droplet:

    `doctl compute floating-ip-action assign <ip-addr> <droplet-id>`

* Create a new A record for an existing domain:

    `doctl compute domain records create --record-type A --record-name www --record-data <ip-addr> <domain-name>`

`doctl` also simplifies actions without an API endpoint. For instance, it allows you to SSH to your Droplet by name:

    doctl compute ssh <droplet-name>

By default, it assumes you are using the `root` user. If you want to SSH as a specific user, you can do that as well:

    doctl compute ssh <user>@<droplet-name>

## Building and dependencies

`doctl`'s dependencies are managed by [gvt](https://github.com/FiloSottile/gvt). To add dependencies, use `gvt fetch`.

## Releasing

First, make sure the [CHANGELOG](https://github.com/digitalocean/doctl/blob/master/CHANGELOG.md)
contains all changes for the version you're going to release.

### Setup

To release `doctl` you need to install:

* [xgo](https://github.com/karalabe/xgo)
* [github-release](https://github.com/aktau/github-release)

And make them available at your `PATH`. You can use `go get -u` for both of them and add your
`$GOPATH/bin` to your `PATH` so your scripts will find them.

You will also need valid `GITHUB_TOKEN` environment variable with access to the `digitalocean/doctl` repo.

### Scripts

To build `doctl` for all its platforms run `scripts/stage.sh major minor patch` 
(ie. `scripts/stage.sh 1 5 0`). This will place all files and its checksums 
at `builds/major.minor.patch/release`.

Then mark the release on github with `scripts/release.sh v<version>` (ie. `scripts/release.sh v1.5.0`, _note_ the `v`).

Then upload using `scripts/upload.sh <version>` to mark it on github.

Now go to [releases](https://github.com/digitalocean/doctl/releases) and update the release
description to contain all changelog entries for this specific release.

Also don't forget to update:
- Dockerfile
- snapcraft
- homebrew formula

## More info

* [Tutorial](https://www.digitalocean.com/community/tutorials/how-to-use-doctl-the-official-digitalocean-command-line-client)
* [doctl Releases](https://github.com/digitalocean/doctl/releases)
* [windows Release](https://github.com/digitalocean/doctl/releases/download/v1.7.0/doctl-1.7.0-windows-4.0-amd64.zip)
