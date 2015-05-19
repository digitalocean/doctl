# Digital Ocean Control TooL

[![Build Status](https://travis-ci.org/slantview/doctl.svg)](https://travis-ci.org/slantview/doctl)


doctl is a tool for controlling your digital ocean resources from the command line.  As an added benefit you get an API library for v2 of the DO API.

## Installation

Download [pre-built binaries](https://github.internal.digitalocean.com/phillip/doctl/releases) from this repository, or clone and build yourself:

```
$ git clone 
$ go get 
$ make all
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
   droplet  Droplet commands.
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

## Licensing

doctl is licensed under the Apache License, Version 2.0. See LICENSE.txt for full license text.

## Author

Steve Rude <steve@slantview.com>
