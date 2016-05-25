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
  -o, --output string         output format [text|json] (default "text")
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

#### Windows and GNU/Linux

Integrations with package managers for GNU/Linux and Windows are to come.

### Option 2 – Download a Release from GitHub

Visit the [Releases page][doctl-releases] for the `doctl` GitHub project, and find the appropriate archive for your operating system and architecture.  (For OS X systems, remember to use the `darwin` archive.)

#### OS X and GNU/Linux

You can download the archive from your browser, or copy its URL and retrieve it to your home directory with `wget` or `curl`:

```
cd ~

# OS X
curl -L https://github.com/digitalocean/doctl/releases/download/v1.1.0/doctl-1.1.0-darwin-10.6-amd64.tar.gz | tar xz

# linux (with wget)
wget -qO- https://github.com/digitalocean/doctl/releases/download/v1.1.0/doctl-1.1.0-linux-amd64.tar.gz  | tar xz
# linux (with curl)
curl -L https://github.com/digitalocean/doctl/releases/download/v1.1.0/doctl-1.1.0-linux-amd64.tar.gz  | tar xz
```

Move the `doctl` binary to somewhere in your path.  For example:

```
sudo mv ./doctl /usr/local/bin
```

#### Windows

On Windows systems, you should be able to [download the Windows release][windows-release], and then double-click the zip archive to extract the `doctl.exe` executable.

### Option 3 – Build From Source

Alternatively, if you have a Go environment configured, you can install the development version of `doctl` from the command line like so:

```
go get github.com/digitalocean/doctl/cmd/doctl
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
access-token: MY_TOKEN
output: text
```

## Examples

`doctl` is able to interact with all of your DigitalOcean resources. Below are a few common usage examples. To learn more about the features available, see [the full tutorial on the DigitalOcean community site][tutorial].

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

To build `doctl` for all its platforms, run `script/build.sh <version>`. To upload `doctl` to Github,
run `script/release.sh <version>`. A valid `GITHUB_TOKEN` environment variable with access to the `bryanl/doctl`
repository is required.

[tutorial]: https://www.digitalocean.com/community/tutorials/how-to-use-doctl-the-official-digitalocean-command-line-client
[doctl-releases]: https://github.com/digitalocean/doctl/releases
[windows-release]: https://github.com/digitalocean/doctl/releases/download/v1.1.0/doctl-1.1.0-windows-4.0-amd64.zip