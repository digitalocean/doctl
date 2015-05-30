# Digital Ocean Control TooL

[![Build Status](https://travis-ci.org/digitalocean/doctl.svg)](https://travis-ci.org/digitalocean/doctl.svg?branch=master)


doctl is a tool for controlling your DigitalOcean resources from the command line. You can find your personal access tokens [on the control panel](https://cloud.digitalocean.com/settings/applications) or [generate a new one](https://cloud.digitalocean.com/settings/tokens/new).

## Installation

Download [pre-built binaries](https://github.com/digitalocean/doctl/releases) from this repository, or clone and build yourself:

```
$ git clone
$ go get
$ make all # Note that this compiles binaries for several architectures, make sure your go is pre-compiled with support, on homebrew: `brew install go --with-cc-common`
```

Or using `go get`:

```
$ go get github.com/digitalocean/doctl
```

## Usage

More details:

```
NAME:
   doctl - Digital Ocean Control TooL.

USAGE:
   doctl [global options] command [command options] [arguments...]

VERSION:
   0.0.9

COMMANDS:
   action   Action commands.
   domain   Domain commands.
   droplet, d  Droplet commands. Lists by default.
   region   Region commands.
   size     Size commands.
   sshkey   SSH Key commands.
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --api-key, -k  API Key for DO APIv2. [$DIGITALOCEAN_API_KEY, $DIGITAL_OCEAN_API_KEY]
   --format, -f 'yaml'  Format for output.
   --help, -h       show help
   --version, -v    print the version

```

Don't forget the shortcuts! Try `doctl d l`.

### Actions
```
NAME:
   doctl action - Action commands.

USAGE:
   doctl action [global options] command [command options] [arguments...]

VERSION:
   0.0.11

COMMANDS:
   show     Show an action.
   list     List all actions.
   help, h  Shows a list of commands or help for one command

```

### Domains
```
NAME:
   doctl domain - Domain commands.

USAGE:
   doctl domain [global options] command [command options] [arguments...]

VERSION:
   0.0.15

COMMANDS:
   show, s        <name> Show an domain.
   list, l        List all domains.
   create, c         <domain> <Droplet name> Create new domain.
   destroy, d        <name> Destroy a domain.
   list-records, records, r   <domain> List domain records for a domain.
   show-record, record     <domain> <id> Show a domain record.
   add, create-record      <domain> Create domain record.
   destroy-record    <domain> <id> Destroy domain record.

```

### Droplets
```
NAME:
   doctl droplet - Droplet commands. Lists by default.

USAGE:
   doctl droplet [global options] command [command options] [arguments...]

VERSION:
   0.0.15

COMMANDS:
   create, c   (--domain | --add-region) --user-data --ssh-keys --size "512mb" --region "nyc3" --image "ubuntu-14-04-x64" --backups --ipv6 --private-networking Create Droplet
   list, l     List droplets.
   find, f     <Droplet name> Find the first Droplet whose name matches the first argument.
   destroy, d     [--id | <name>] Destroy droplet.
   reboot      [--id | <name>] Reboot droplet.
   power_cycle    [--id | <name>] Powercycle droplet.
   shutdown    [--id | <name>] Shutdown droplet.
   poweroff, off  [--id | <name>] Power off droplet.
   poweron, on    [--id | <name>] Power on droplet.
   password_reset [--id | <name>] Reset password for droplet.
   resize      [--id | <name>] Resize droplet.
```

### Regions
```
NAME:
   doctl region - Region commands.

USAGE:
   doctl region [global options] command [command options] [arguments...]

VERSION:
   0.0.11

COMMANDS:
   list     List All Regions.
```

### Sizes
```
NAME:
   doctl size - Size commands.

USAGE:
   doctl size [global options] command [command options] [arguments...]

VERSION:
   0.0.11

COMMANDS:
   list     List all sizes.
```

### SSH Keys
```
NAME:
   doctl sshkey - SSH Key commands.

USAGE:
   doctl sshkey [global options] command [command options] [arguments...]

VERSION:
   0.0.11

COMMANDS:
   create   Create SSH key.
   list     List all SSH keys.
   show     Show SSH key.
   destroy  Destroy SSH key.

```


## Licensing

doctl is licensed under the Apache License, Version 2.0. See LICENSE.txt for full license text.

## Author

Phillip Baker <phillip@digitalocean.com>
Steve Rude <steve@slantview.com>
