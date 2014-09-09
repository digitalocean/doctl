# Digital Ocean Control TooL

doctl is a tool for controlling your digital ocean resources from the command line.  As an added benefit you get an API library for v2 of the DO API.

## Installation

Download pre-built binaries from this repository or simply run:

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
   0.0.7

COMMANDS:
   droplet  Droplet commands.
   sshkey   SSH Key commands.
   region   Region commands.
   size     Size commands.
   help, h  Shows a list of commands or help for one command
   
GLOBAL OPTIONS:
   --api-key, -k    API Key for DO APIv2.
   --help, -h       show help
   --version, -v    print the version

```

## Licensing

doctl is licensed under the Apache License, Version 2.0. See LICENSE.txt for full license text.

## Author

Steve Rude <steve@slantview.com>