package main

import (
	"github.com/bryanl/docli"
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
		Action: docli.DomainList,
	}
}

func domainCreate() cli.Command {
	return cli.Command{
		Name:  "create",
		Usage: "create domain",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  docli.ArgDomainName,
				Usage: "domain name",
			},
			cli.StringFlag{
				Name:  docli.ArgIPAddress,
				Usage: "domain ip address",
			},
		},
		Action: docli.DomainCreate,
	}
}

func domainGet() cli.Command {
	return cli.Command{
		Name:  "get",
		Usage: "get domain",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  docli.ArgDomainName,
				Usage: "domain name",
			},
			jsonFlag(),
			textFlag(),
		},
		Action: docli.DomainGet,
	}
}

func domainDelete() cli.Command {
	return cli.Command{
		Name:  "delete",
		Usage: "delete domain",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  docli.ArgDomainName,
				Usage: "domain name",
			},
		},
		Action: docli.DomainDelete,
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
				Name:  docli.ArgDomainName,
				Usage: "domain name",
			},
		},
		Action: docli.RecordList,
	}
}

func recordCreate() cli.Command {
	return cli.Command{
		Name:  "create",
		Usage: "create record",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  docli.ArgDomainName,
				Usage: "record domain (required)",
			},
			cli.StringFlag{
				Name:  docli.ArgRecordType,
				Usage: "record type (required)",
			},
			cli.StringFlag{
				Name:  docli.ArgRecordName,
				Usage: "record name (required for A, AAAA, CNAME, TXT, SRV records)",
			},
			cli.StringFlag{
				Name:  docli.ArgRecordData,
				Usage: "record data (required for A, AAAA, CNAME, MX, TXT, SRV, NS records)",
			},
			cli.IntFlag{
				Name:  docli.ArgRecordPriority,
				Usage: "record priority (required for MX, SRV records)",
			},
			cli.IntFlag{
				Name:  docli.ArgRecordPort,
				Usage: "record port (required for SRV records)",
			},
			cli.IntFlag{
				Name:  docli.ArgRecordWeight,
				Usage: "record weight (required for SRV records)",
			},
		},
		Action: docli.RecordCreate,
	}
}

func recordGet() cli.Command {
	return cli.Command{
		Name:  "get",
		Usage: "get domain record",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  docli.ArgDomainName,
				Usage: "domain name (required)",
			},
			cli.IntFlag{
				Name:  docli.ArgRecordID,
				Usage: "domain id (required)",
			},
		},
		Action: docli.RecordGet,
	}
}

func recordUpdate() cli.Command {
	return cli.Command{
		Name:  "update",
		Usage: "update domain record",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  docli.ArgDomainName,
				Usage: "record domain (required)",
			},
			cli.IntFlag{
				Name:  docli.ArgRecordID,
				Usage: "record id (required)",
			},
			cli.StringFlag{
				Name:  docli.ArgRecordType,
				Usage: "record type (required)",
			},
			cli.StringFlag{
				Name:  docli.ArgRecordName,
				Usage: "record name (required for A, AAAA, CNAME, TXT, SRV records)",
			},
			cli.StringFlag{
				Name:  docli.ArgRecordData,
				Usage: "record data (required for A, AAAA, CNAME, MX, TXT, SRV, NS records)",
			},
			cli.IntFlag{
				Name:  docli.ArgRecordPriority,
				Usage: "record priority (required for MX, SRV records)",
			},
			cli.IntFlag{
				Name:  docli.ArgRecordPort,
				Usage: "record port (required for SRV records)",
			},
			cli.IntFlag{
				Name:  docli.ArgRecordWeight,
				Usage: "record weight (required for SRV records)",
			},
		},
		Action: docli.RecordUpdate,
	}
}

func recordDelete() cli.Command {
	return cli.Command{
		Name:  "delete",
		Usage: "delete domain record",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  docli.ArgDomainName,
				Usage: "domain (required)",
			},
			cli.IntFlag{
				Name:  docli.ArgRecordID,
				Usage: "record id (required)",
			},
		},
		Action: docli.RecordDelete,
	}
}
