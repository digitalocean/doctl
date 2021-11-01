<h1 align="center">doctl</h1>

<p align="center">
  <img width="200" height="170" src="https://api-engineering.nyc3.cdn.digitaloceanspaces.com/doctl-mascot.png" alt="The doctl mascot." />
</p>

<p align="center">
  <a href="https://travis-ci.org/digitalocean/doctl">
    <img src="https://travis-ci.org/digitalocean/doctl.svg?branch=main" alt="Build Status" />
  </a>
  <a href="https://godoc.org/github.com/digitalocean/doctl">
    <img src="https://godoc.org/github.com/digitalocean/doctl?status.svg" alt="GoDoc" />
  </a>
  <a href="https://goreportcard.com/report/github.com/digitalocean/doctl">
    <img src="https://goreportcard.com/badge/github.com/digitalocean/doctl" alt="Go Report Card" />
  </a>
</p>

```
doctl is a command-line interface (CLI) for the DigitalOcean API.

Usage:
  doctl [command]

Available Commands:
  1-click         Display commands that pertain to 1-click applications
  account         Display commands that retrieve account details
  apps            Display commands for working with apps
  auth            Display commands for authenticating doctl with an account
  balance         Display commands for retrieving your account balance
  billing-history Display commands for retrieving your billing history
  completion      Modify your shell so doctl commands autocomplete with TAB
  compute         Display commands that manage infrastructure
  databases       Display commands that manage databases
  help            Help about any command
  invoice         Display commands for retrieving invoices for your account
  kubernetes      Displays commands to manage Kubernetes clusters and configurations
  monitoring      [Beta] Display commands to manage monitoring
  projects        Manage projects and assign resources to them
  registry        Display commands for working with container registries
  version         Show the current version
  vpcs            Display commands that manage VPCs

Flags:
  -t, --access-token string   API V2 access token
  -u, --api-url string        Override default API endpoint
  -c, --config string         Specify a custom config file (default "$HOME/.config/doctl/config.yaml")
      --context string        Specify a custom authentication context name
  -h, --help                  help for doctl
  -o, --output string         Desired output format [text|json] (default "text")
      --trace                 Show a log of network activity while performing a command
  -v, --verbose               Enable verbose output

Use "doctl [command] --help" for more information about a command.
```

See the [full reference documentation](https://www.digitalocean.com/docs/apis-clis/doctl/reference/) for information about each available command.

- [Installing `doctl`](#installing-doctl)
  - [Using a Package Manager (Preferred)](#using-a-package-manager-preferred)
    - [MacOS](#macos)
    - [Snap supported OS](#snap-supported-os)
      - [Use with `kubectl`](#use-with-kubectl)
      - [Using `doctl compute ssh`](#using-doctl-compute-ssh)
      - [Use with Docker](#use-with-docker)
    - [Arch Linux](#arch-linux)
    - [Fedora](#fedora)
    - [Nix supported OS](#nix-supported-os)
  - [Docker Hub](#docker-hub)
  - [Downloading a Release from GitHub](#downloading-a-release-from-github)
  - [Building with Docker](#building-with-docker)
  - [Building the Development Version from Source](#building-the-development-version-from-source)
  - [Dependencies](#dependencies)
- [Authenticating with DigitalOcean](#authenticating-with-digitalocean)
  - [Logging into multiple DigitalOcean accounts](#logging-into-multiple-digitalocean-accounts)
- [Configuring Default Values](#configuring-default-values)
  - [Environment Variables](#environment-variables)
- [Enabling Shell Auto-Completion](#enabling-shell-auto-completion)
  - [Linux Auto Completion](#linux-auto-completion)
  - [MacOS](#macos-1)
- [Uninstalling `doctl`](#uninstalling-doctl)
  - [Using a Package Manager](#using-a-package-manager)
    - [MacOS Uninstall](#macos-uninstall)
- [Examples](#examples)
- [Tutorials](#tutorials)


## Installing `doctl`

### Using a Package Manager (Preferred)

A package manager allows you to install and keep up with new `doctl` versions using only a few commands.
Our community distributes `doctl` via a growing set of package managers in addition to the officially
supported set listed below; chances are good a solution exists for your platform.

#### MacOS

Use [Homebrew](https://brew.sh/) to install `doctl` on macOS:

```
brew install doctl
```

`doctl` is also available via [MacPorts](https://www.macports.org/ports.php?by=name&substr=doctl). Note that
the port is community maintained and may not be on the latest version.

#### Snap supported OS

Use [Snap](https://snapcraft.io/) on [Snap-supported](https://snapcraft.io/docs/core/install) systems to
install `doctl`:

```
sudo snap install doctl
```

##### Use with `kubectl`

Using `kubectl` requires the `kube-config` personal-files connection for `doctl`:

    sudo snap connect doctl:kube-config

##### Using `doctl compute ssh`

Using `doctl compute ssh` requires the core [ssh-keys interface](https://docs.snapcraft.io/ssh-keys-interface):

    sudo snap connect doctl:ssh-keys :ssh-keys

##### Use with Docker

Using `doctl registry login` requires the `dot-docker` personal-files connection for `doctl`:

    sudo snap connect doctl:dot-docker

This allows `doctl` to add DigitalOcean container registry credentials to your Docker configuration file.

#### Arch Linux

`doctl` is available in the official Arch Linux repository:

    sudo pacman -S doctl

As an alternative, you can install it from the [AUR](https://aur.archlinux.org/packages/doctl-bin/).

#### Fedora

`doctl` is available in the official Fedora repository:

    sudo dnf install doctl

#### Nix supported OS

Users of NixOS or other [supported
platforms](https://nixos.org/nixpkgs/) may install ```doctl``` from
[Nixpkgs](https://nixos.org/nixos/packages.html#doctl). Please note
this package is also community maintained and may not be on the latest
version.

### Docker Hub

Containers for each release are available under the `digitalocean`
organization on [Docker Hub](https://hub.docker.com/r/digitalocean/doctl).
Links to the containers are available in the GitHub releases.

### Downloading a Release from GitHub

Visit the [Releases
page](https://github.com/digitalocean/doctl/releases) for the
[`doctl` GitHub project](https://github.com/digitalocean/doctl), and find the
appropriate archive for your operating system and architecture.
Download the archive from your browser or copy its URL and
retrieve it to your home directory with `wget` or `curl`.

For example, with `wget`:

```
cd ~
wget https://github.com/digitalocean/doctl/releases/download/v<version>/doctl-<version>-linux-amd64.tar.gz
```

Or with `curl`:

```
cd ~
curl -OL https://github.com/digitalocean/doctl/releases/download/v<version>/doctl-<version>-linux-amd64.tar.gz
```

Extract the binary:

```
tar xf ~/doctl-<version>-linux-amd64.tar.gz
```

Or download and extract with this oneliner:
```
curl -sL https://github.com/digitalocean/doctl/releases/download/v<version>/doctl-<version>-linux-amd64.tar.gz | tar -xzv
```

where `<version>` is the full semantic version, e.g., `1.17.0`.

On Windows systems, you should be able to double-click the zip archive to extract the `doctl` executable.

Move the `doctl` binary to somewhere in your path. For example, on GNU/Linux and OS X systems:

```
sudo mv ~/doctl /usr/local/bin
```

Windows users can follow [How to: Add Tool Locations to the PATH Environment Variable](https://msdn.microsoft.com/en-us/library/office/ee537574(v=office.14).aspx) in order to add `doctl` to their `PATH`.

### Building with Docker

If you have
[Docker](https://www.digitalocean.com/community/tutorials/how-to-install-and-use-docker-on-ubuntu-16-04)
configured, you can build a local Docker image using `doctl`'s
[Dockerfile](https://github.com/digitalocean/doctl/blob/main/Dockerfile)
and run `doctl` within a container.

```
docker build --tag=doctl .
```

Then you can run it within a container.

```
docker run --rm --interactive --tty --env=DIGITALOCEAN_ACCESS_TOKEN="your_DO_token" doctl any_doctl_command
```

### Building the Development Version from Source

If you have a [Go environment](https://www.digitalocean.com/community/tutorials/how-to-install-go-1-6-on-ubuntu-16-04)
configured, you can install the development version of `doctl` from
the command line.

```
go install github.com/digitalocean/doctl/cmd/doctl@latest
```

While the development version is a good way to take a peek at
`doctl`'s latest features before they get released, be aware that it
may have bugs. Officially released versions will generally be more
stable.

### Dependencies

`doctl` uses Go modules with vendoring.

## Authenticating with DigitalOcean

To use `doctl`, you need to authenticate with DigitalOcean by providing an access token, which can be created from the [Applications & API](https://cloud.digitalocean.com/account/api/tokens) section of the Control Panel. You can learn how to generate a token by following the [DigitalOcean API guide](https://www.digitalocean.com/community/tutorials/how-to-use-the-digitalocean-api-v2).

Docker users will have to use the `DIGITALOCEAN_ACCESS_TOKEN` environmental variable to authenticate, as explained in the Installation section of this document.

If you're not using Docker to run `doctl`, authenticate with the `auth init` command.

```
doctl auth init
```

You will be prompted to enter the DigitalOcean access token that you generated in the DigitalOcean control panel.

```
DigitalOcean access token: your_DO_token
```

After entering your token, you will receive confirmation that the credentials were accepted. If the token doesn't validate, make sure you copied and pasted it correctly.

```
Validating token: OK
```

This will create the necessary directory structure and configuration file to store your credentials.

### Logging into multiple DigitalOcean accounts

`doctl` allows you to log in to multiple DigitalOcean accounts at the same time and easily switch between them with the use of authentication contexts.

By default, a context named `default` is used. To create a new context, run `doctl auth init --context <new-context-name>`. You may also pass the new context's name using the `DIGITALOCEAN_CONTEXT` [environment variable](#environment-variables). You will be prompted for your API access token which will be associated with the new context.

To use a non-default context, pass the context name to any `doctl` command. For example:

```
doctl compute droplet list --context <new-context-name>
```

To set a new default context, run `doctl auth switch --context <new-context-name>`. This command will save the current context to the config file and use it for all commands by default if a context is not specified.

The `--access-token` flag or `DIGITALOCEAN_ACCESS_TOKEN` [environment variable](#environment-variables) are acknowledged only if the `default` context is used. Otherwise, they will have no effect on what API access token is used. To temporarily override the access token if a different context is set as default, use `doctl --context default --access-token your_DO_token ...`.

## Configuring Default Values

The `doctl` configuration file is used to store your API Access Token as well as the defaults for command flags. If you find yourself using certain flags frequently, you can change their default values to avoid typing them every time. This can be useful when, for example, you want to change the username or port used for SSH.

On OS X, `doctl` saves its configuration as `${HOME}/Library/Application Support/doctl/config.yaml`. The `${HOME}/Library/Application Support/doctl/` directory will be created once you run `doctl auth init`.

On Linux, `doctl` saves its configuration as `${XDG_CONFIG_HOME}/doctl/config.yaml` if the `${XDG_CONFIG_HOME}` environmental variable is set, or `~/.config/doctl/config.yaml` if it is not. On Windows, the config file location is `%APPDATA%\doctl\config.yaml`.

The configuration file is automatically created and populated with default properties when you authenticate with `doctl` for the first time. The typical format for a property is `category.command.sub-command.flag: value`. For example, the property for the `force` flag with tag deletion is `tag.delete.force`.

To change the default SSH user used when connecting to a Droplet with `doctl`, look for the `compute.ssh.ssh-user` property and change the value after the colon. In this example, we changed it to the username **sammy**.

```
. . .
compute.ssh.ssh-user: sammy
. . .
```

Save and close the file. The next time you use `doctl`, the new default values you set will be in effect. In this example, that means that it will SSH as the **sammy** user (instead of the default **root** user) next time you log into a Droplet.

### Environment variables

In addition to specifying configuration using `config.yaml` file or program arguments, it is also possible to override values just for the given session with environment variables:

```
# Use instead of --context argument
DIGITALOCEAN_CONTEXT=my-context doctl auth list
```

```
# Use instead of --access-token argument
DIGITALOCEAN_ACCESS_TOKEN=my-do-token doctl
```

## Enabling Shell Auto-Completion

`doctl` also has auto-completion support. It can be set up so that if you partially type a command and then press `TAB`, the rest of the command is automatically filled in. For example, if you type `doctl comp<TAB><TAB> drop<TAB><TAB>` with auto-completion enabled, you'll see `doctl compute droplet` appear on your command prompt.

**Note:** Shell auto-completion is not available for Windows users.

How you enable auto-completion depends on which operating system you're using. If you installed `doctl` via Homebrew, auto-completion is activated automatically, though you may need to configure your local environment to enable it.

`doctl` can generate an auto-completion script with the `doctl completion your_shell_here` command. Valid arguments for the shell are Bash (`bash`), ZSH (`zsh`), and fish (`fish`). By default, the script will be printed to the command line output.  For more usage examples for the `completion` command, use `doctl completion --help`.

### Linux Auto Completion

The most common way to use the `completion` command is by adding a line to your local profile configuration. At the end of your `~/.profile` file, add this line:

```
source <(doctl completion your_shell_here)
```

Then refresh your profile.

```
source ~/.profile
```

### MacOS

macOS users will have to install the `bash-completion` framework to use the auto-completion feature.

```
brew install bash-completion
```

After it's installed, load `bash_completion` by adding the following line to your `.profile` or `.bashrc`/`.zshrc` file.

```
source $(brew --prefix)/etc/bash_completion
```

Then refresh your profile using the appropriate command for the bash configurations file.

```
source ~/.profile
source ~/.bashrc
source ~/.zshrc
```


## Uninstalling `doctl`

### Using a Package Manager

#### MacOS Uninstall

Use [Homebrew](https://brew.sh/) to uninstall all current and previous versions of the `doctl` formula on macOS:

```
brew uninstall -f doctl
```

To completely remove the configuration, also remove the following directory:

```
rm -rf "$HOME/Library/Application Support/doctl"
```


## Examples

`doctl` is able to interact with all of your DigitalOcean resources. Below are a few common usage examples. To learn more about the features available, see [the full tutorial on the DigitalOcean community site](https://www.digitalocean.com/community/tutorials/how-to-use-doctl-the-official-digitalocean-command-line-client).

* List all Droplets on your account:
```
doctl compute droplet list
```
* Create a Droplet:
```
doctl compute droplet create <name> --region <region-slug> --image <image-slug> --size <size-slug>
```
* Assign a Floating IP to a Droplet:
```
doctl compute floating-ip-action assign <ip-addr> <droplet-id>
```
* Create a new A record for an existing domain:
```
doctl compute domain records create --record-type A --record-name www --record-data <ip-addr> <domain-name>
```

`doctl` also simplifies actions without an API endpoint. For instance, it allows you to SSH to your Droplet by name:
```
doctl compute ssh <droplet-name>
```

By default, it assumes you are using the `root` user. If you want to SSH as a specific user, you can do that as well:
```
doctl compute ssh <user>@<droplet-name>
```

## Tutorials

* [How To Use Doctl, the Official DigitalOcean Command-Line Client](https://www.digitalocean.com/community/tutorials/how-to-use-doctl-the-official-digitalocean-command-line-client)
* [How To Work with DigitalOcean Load Balancers Using Doctl](https://www.digitalocean.com/community/tutorials/how-to-work-with-digitalocean-load-balancers-using-doctl)
* [How To Secure Web Server Infrastructure With DigitalOcean Cloud Firewalls Using Doctl](https://www.digitalocean.com/community/tutorials/how-to-secure-web-server-infrastructure-with-digitalocean-cloud-firewalls-using-doctl)
* [How To Work with DigitalOcean Block Storage Using Doctl](https://www.digitalocean.com/community/tutorials/how-to-work-with-digitalocean-block-storage-using-doctl)
