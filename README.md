# Digital Ocean Control TooL

[![Build Status](https://travis-ci.org/slantview/doctl.svg)](https://travis-ci.org/slantview/doctl)


doctl is a tool for controlling your digital ocean resources from the command line.  As an added benefit you get an API library for v2 of the DO API.

## Installation

Download [pre-built binaries](https://github.internal.digitalocean.com/phillip/doctl/releases) from this repository, or clone and build yourself:

```
$ git clone 
$ go get 
$ make all # Note that this compiles binaries for several architectures, make sure your go is pre-compiled with support, on homebrew: 
```

Or using `go get`:

```
$ go get github.com/slantview/doctl
```

## Usage

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
   --api-key, -k    API Key for DO APIv2.
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
   0.0.11

COMMANDS:
   record   Domain record commands.
   show     Show an domain.
   list     List all domains.
   create   Create new domain.
   destroy  Destroy a domain.

```

### Droplets
```
NAME:
   doctl droplet - Droplet commands. Lists by default.

USAGE:
   doctl droplet [global options] command [command options] [arguments...]

VERSION:
   0.0.11

COMMANDS:
   action   Droplet Action Commands.
   create, c   Create droplet.
   list, l  List droplets.
   find, f  Find the first Droplet whose name matches the first argument.
   destroy, d  Destroy droplet.
   help, h  Shows a list of commands or help for one command
   

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
   show     Show a Region.
   list     List All Regions.
   help, h  Shows a list of commands or help for one command
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
   show     Show a size.
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

Steve Rude <steve@slantview.com>
