package main

import (
	"github.com/bryanl/doit"
	"github.com/codegangsta/cli"
)

func domainCommands() cli.Command {
	return cli.Command{
		Name:  "domain",
		Usage: "domain commands",
		Subcommands: []cli.Command{
			domainList(),
			domainCreate(),
			domainGet(),
			domainDelete(),
			recordCommands(),
		},
	}
}

func domainList() cli.Command {
	return cli.Command{
		Name:   "list",
		Usage:  "list domains",
		Action: doit.DomainList,
	}
}

func domainCreate() cli.Command {
	return cli.Command{
		Name:  "create",
		Usage: "create domain",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  doit.ArgDomainName,
				Usage: "domain name",
			},
			cli.StringFlag{
				Name:  doit.ArgIPAddress,
				Usage: "domain ip address",
			},
		},
		Action: doit.DomainCreate,
	}
}

func domainGet() cli.Command {
	return cli.Command{
		Name:  "get",
		Usage: "get domain",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  doit.ArgDomainName,
				Usage: "domain name",
			},
		},
		Action: doit.DomainGet,
	}
}

func domainDelete() cli.Command {
	return cli.Command{
		Name:  "delete",
		Usage: "delete domain",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  doit.ArgDomainName,
				Usage: "domain name",
			},
		},
		Action: doit.DomainDelete,
	}
}

func recordCommands() cli.Command {
	return cli.Command{
		Name:  "records",
		Usage: "domain record commands",
		Subcommands: []cli.Command{
			recordList(),
			recordCreate(),
			recordGet(),
			recordUpdate(),
			recordDelete(),
		},
	}
}

func recordList() cli.Command {
	return cli.Command{
		Name:  "list",
		Usage: "list records",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  doit.ArgDomainName,
				Usage: "domain name",
			},
		},
		Action: doit.RecordList,
	}
}

func recordCreate() cli.Command {
	return cli.Command{
		Name:  "create",
		Usage: "create record",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  doit.ArgDomainName,
				Usage: "record domain (required)",
			},
			cli.StringFlag{
				Name:  doit.ArgRecordType,
				Usage: "record type (required)",
			},
			cli.StringFlag{
				Name:  doit.ArgRecordName,
				Usage: "record name (required for A, AAAA, CNAME, TXT, SRV records)",
			},
			cli.StringFlag{
				Name:  doit.ArgRecordData,
				Usage: "record data (required for A, AAAA, CNAME, MX, TXT, SRV, NS records)",
			},
			cli.IntFlag{
				Name:  doit.ArgRecordPriority,
				Usage: "record priority (required for MX, SRV records)",
			},
			cli.IntFlag{
				Name:  doit.ArgRecordPort,
				Usage: "record port (required for SRV records)",
			},
			cli.IntFlag{
				Name:  doit.ArgRecordWeight,
				Usage: "record weight (required for SRV records)",
			},
		},
		Action: doit.RecordCreate,
	}
}

func recordGet() cli.Command {
	return cli.Command{
		Name:  "get",
		Usage: "get domain record",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  doit.ArgDomainName,
				Usage: "domain name (required)",
			},
			cli.IntFlag{
				Name:  doit.ArgRecordID,
				Usage: "domain id (required)",
			},
		},
		Action: doit.RecordGet,
	}
}

func recordUpdate() cli.Command {
	return cli.Command{
		Name:  "update",
		Usage: "update domain record",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  doit.ArgDomainName,
				Usage: "record domain (required)",
			},
			cli.IntFlag{
				Name:  doit.ArgRecordID,
				Usage: "record id (required)",
			},
			cli.StringFlag{
				Name:  doit.ArgRecordType,
				Usage: "record type (required)",
			},
			cli.StringFlag{
				Name:  doit.ArgRecordName,
				Usage: "record name (required for A, AAAA, CNAME, TXT, SRV records)",
			},
			cli.StringFlag{
				Name:  doit.ArgRecordData,
				Usage: "record data (required for A, AAAA, CNAME, MX, TXT, SRV, NS records)",
			},
			cli.IntFlag{
				Name:  doit.ArgRecordPriority,
				Usage: "record priority (required for MX, SRV records)",
			},
			cli.IntFlag{
				Name:  doit.ArgRecordPort,
				Usage: "record port (required for SRV records)",
			},
			cli.IntFlag{
				Name:  doit.ArgRecordWeight,
				Usage: "record weight (required for SRV records)",
			},
		},
		Action: doit.RecordUpdate,
	}
}

func recordDelete() cli.Command {
	return cli.Command{
		Name:  "delete",
		Usage: "delete domain record",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  doit.ArgDomainName,
				Usage: "domain (required)",
			},
			cli.IntFlag{
				Name:  doit.ArgRecordID,
				Usage: "record id (required)",
			},
		},
		Action: doit.RecordDelete,
	}
}
