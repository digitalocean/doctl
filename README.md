# doctl [![Build Status](https://travis-ci.org/digitalocean/doctl.svg?branch=master)](https://travis-ci.org/digitalocean/doctl) [![GoDoc](https://godoc.org/github.com/digitalocean/doctl?status.svg)](https://godoc.org/github.com/digitalocean/doctl) [![Go Report Card](https://goreportcard.com/badge/github.com/digitalocean/doctl)](https://goreportcard.com/report/github.com/digitalocean/doctl)

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

=======

## Installing `doctl`

There are four ways to install `doctl`: using a package manager, downloading a GitHub release, building a development version from source, or building it with [Docker](https://www.digitalocean.com/community/tutorials/the-docker-ecosystem-an-introduction-to-common-components).

### Option 1 – Using a Package Manager (Preferred)

A package manager allows you to install and keep up with new `doctl` versions using only a few commands. Currently, `doctl` is available as part of [Homebrew](homebrew) for macOS users and [Snap](snap) for GNU/Linux users.

You can use [Homebrew](homebrew) to install `doctl` on macOS with this command:

```command
brew install doctl
```

You can use [Snap](snap) on [Snap-supported](snap-supported-systems) systems to install `doctl` with this command:

```
sudo snap install doctl
```

Support for Windows package managers are on the way.

### Option 2 — Downloading a Release from GitHub

Visit the [Releases page][doctl-releases] for the [`doctl` GitHub project](doctl-github), and find the appropriate archive for your operating system and architecture.  You can download the archive from from your browser, or copy its URL and retrieve it to your home directory with `wget` or `curl`.

For example, with `wget`:

```command
cd ~
wget https://github.com/digitalocean/doctl/releases/download/v1.7.0/doctl-1.7.0-linux-amd64.tar.gz
```

Or with `curl`:

```command
cd ~
curl -OL https://github.com/digitalocean/doctl/releases/download/v1.7.0/doctl-1.7.0-linux-amd64.tar.gz
```

Extract the binary.  On GNU/Linux or OS X systems, you can use `tar`.

```command
tar xf ~/doctl-1.7.0-linux-amd64.tar.gz
```

On Windows systems, you should be able to double-click the zip archive to extract the `doctl` executable.

Move the `doctl` binary to somewhere in your path. For example, on GNU/Linux and OS X systems:

```command
sudo mv ~/doctl /usr/local/bin
```

### Option 3 — Building the Development Version from Source

If you have a [Go environment][install-go] configured, you can install the development version of `doctl` from the command line.

```command
go get -u github.com/digitalocean/doctl/cmd/doctl
```

While the development version is a good way to take a peek at `doctl`'s latest features before they get released, be aware that it may have bugs. Officially released versions will generally be more stable.

### Option 4 — Building with Docker

If you have [Docker](install-docker) configured, you can build a Docker image using `doctl`'s [Dockerfile](doctl-dockerfile) and run `doctl` within a container.

```command
docker build -t doctl .
```

Then you can run it within a container.

```command
docker run --rm -e DIGITALOCEAN_ACCESS_TOKEN="your_DO_token" doctl any_doctl_command
```

## Authenticating with DigitalOcean

In order to use `doctl`, you need to authenticate with DigitalOcean.

In case you're not using Docker to run `doctl`, this is done using the `auth init` command.
Docker users will have to use `DIGITALOCEAN_ACCESS_TOKEN` environmental variable, as exlained in the Installation part of this document.

```command
doctl auth init
```

You will be prompted to enter the DigitalOcean access token that you generated in the DigitalOcean control panel.

```
[secondary_label Output]
DigitalOcean access token: your_DO_token
```

After entering your token, you will receive confirmation that the credentials were accepted. If the token doesn't validate, make sure you copied and pasted it correctly.

```
[secondary_label Output]
Validating token: OK
```

This will create the necessary directory structure and configuration file to store your credentials.

## Configuring Default Values

The `doctl` configuration file is used to store your API Access Token as well as the defaults for command flags. If you find yourself using certain flags frequently, you can change their default values to avoid typing them every time. This can be useful when, for example, you want to change the username or port used for SSH.

On OS X and Linux, `doctl`'s configuration file can be found at `${XDG_CONFIG_HOME}/doctl/config.yaml` if the `${XDG_CONFIG_HOME}` environmental variable is set. Otherwise, the config will be written to `~/.config/doctl/config.yaml`. For Windows users, the config will be available at `%LOCALAPPDATA%/doctl/config/config.yaml`.

The configuration file was automatically created and populated with default properties when you authenticated with `doctl` for the first time. The typical format for a property is `<^>category<^>.<^>command<^>.<^>sub-command<^>.<^>flag<^>: <^>value<^>`. For example, the property for the `force` flag with tag deletion is `tag.delete.force`.

To change the default SSH user used when connecting to a Droplet with `doctl`, look for the `compute.ssh.ssh-user` property and change the value after the colon. In this example, we changed it to the username **sammy**.

```
[label doctl configuration file]
. . .
compute.ssh.ssh-user: sammy
. . .
```

Save and close the file. The next time you use `doctl`, the new default values you set will be in effect. In this example, that means that it will SSH as the **sammy** user (instead of the default **root** user) next time you log into a Droplet.

## Enabling Shell Auto-Completion

`doctl` also has auto-completion support. It can be set up so that if you partially type a command and then press `TAB`, the rest of the command is automatically filled in.

For example, if you type `doctl comp<TAB><TAB> drop<TAB><TAB>` with auto-completion enabled, you'll see `doctl compute droplet` appear on your command prompt.

**Note:** Shell auto-completion is not available for Windows users.

How you enable auto-completion depends on which operating system you're using. If you installed `doctl` via Homebrew or Snap, auto-completion is activated automatically, though you may need to configure your local environment to enable it.

`doctl` can generate an auto-completion script with the `doctl completion your_shell_here` command. Valid arguments for the shell are Bash (`bash`) and ZSH (`zsh`). By default, the script will be printed to the command line output.  For more usage examples for the `completion` command, use `doctl completion --help`.

### Linux

The most common way to use the `completion` command is by adding a line to your local profile configuration. Open `~/.profile` for editing.

```command
nano ~/.profile
```

At the end of the file, add this line:

```
source <(doctl completion your_shell_here)
```

Save file and close the editor. Finally, refresh your profile.

```command
source ~/.profile
```

### macOS

macOS users will have to install the `bash-completion` framework to use the auto-completion feature..

```command
brew install bash-completion
```

After it's installed, load `bash_completion` by adding following line to your `.profile` or `.bashrc`/`.zshrc` file.

```
source $(brew --prefix)/etc/bash_completion
```
