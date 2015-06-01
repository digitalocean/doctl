package main

import (
	"github.com/bryanl/docli/domainrecs"
	"github.com/bryanl/docli/domains"
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
		Action: domains.List,
	}
}

func domainCreate() cli.Command {
	return cli.Command{
		Name:  "create",
		Usage: "create domain",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "name",
				Usage: "domain name",
			},
			cli.StringFlag{
				Name:  "ip-address",
				Usage: "domain ip address",
			},
		},
		Action: domains.Create,
	}
}

func domainGet() cli.Command {
	return cli.Command{
		Name:  "get",
		Usage: "get domain",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "domain-name",
				Usage: "domain name",
			},
		},
		Action: domains.Get,
	}
}

func domainDelete() cli.Command {
	return cli.Command{
		Name:  "delete",
		Usage: "delete domain",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "domain-name",
				Usage: "domain name",
			},
		},
		Action: domains.Delete,
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
				Name:  "name",
				Usage: "domain name",
			},
		},
		Action: domainrecs.List,
	}
}

func recordCreate() cli.Command {
	return cli.Command{
		Name:  "create",
		Usage: "create record",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "domain",
				Usage: "record domain (required)",
			},
			cli.StringFlag{
				Name:  "type",
				Usage: "record type (required)",
			},
			cli.StringFlag{
				Name:  "name",
				Usage: "record name (required for A, AAAA, CNAME, TXT, SRV records)",
			},
			cli.StringFlag{
				Name:  "data",
				Usage: "record data (required for A, AAAA, CNAME, MX, TXT, SRV, NS records)",
			},
			cli.IntFlag{
				Name:  "priority",
				Usage: "record priority (required for MX, SRV records)",
			},
			cli.IntFlag{
				Name:  "port",
				Usage: "record port (required for SRV records)",
			},
			cli.IntFlag{
				Name:  "weight",
				Usage: "record weight (required for SRV records)",
			},
		},
		Action: domainrecs.Create,
	}
}

func recordGet() cli.Command {
	return cli.Command{
		Name:  "get",
		Usage: "get domain record",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "domain-name",
				Usage: "domain name (required)",
			},
			cli.IntFlag{
				Name:  "record-id",
				Usage: "domain id (required)",
			},
		},
		Action: domainrecs.Get,
	}
}

func recordUpdate() cli.Command {
	return cli.Command{
		Name:  "update",
		Usage: "update domain record",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "domain",
				Usage: "record domain (required)",
			},
			cli.IntFlag{
				Name:  "id",
				Usage: "record id (required)",
			},
			cli.StringFlag{
				Name:  "type",
				Usage: "record type (required)",
			},
			cli.StringFlag{
				Name:  "name",
				Usage: "record name (required for A, AAAA, CNAME, TXT, SRV records)",
			},
			cli.StringFlag{
				Name:  "data",
				Usage: "record data (required for A, AAAA, CNAME, MX, TXT, SRV, NS records)",
			},
			cli.IntFlag{
				Name:  "priority",
				Usage: "record priority (required for MX, SRV records)",
			},
			cli.IntFlag{
				Name:  "port",
				Usage: "record port (required for SRV records)",
			},
			cli.IntFlag{
				Name:  "weight",
				Usage: "record weight (required for SRV records)",
			},
		},
		Action: domainrecs.Update,
	}
}

func recordDelete() cli.Command {
	return cli.Command{
		Name:  "delete",
		Usage: "delete domain record",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "domain",
				Usage: "domain (required)",
			},
			cli.IntFlag{
				Name:  "id",
				Usage: "record id (required)",
			},
		},
		Action: domainrecs.Delete,
	}
}
