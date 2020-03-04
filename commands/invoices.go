/*
Copyright 2018 The Doctl Authors All rights reserved.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package commands

import (
	"fmt"
	"io/ioutil"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/spf13/cobra"
)

// Invoices creates the invoices commands hierarchy.
func Invoices() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "invoice",
			Short: "invoice commands",
			Long:  "invoice is used to access invoice commands",
		},
	}

	CmdBuilder(cmd, RunInvoicesGet, "get <invoice-uuid>", "get paginated invoice items of an invoice", Writer,
		aliasOpt("g"), displayerType(&displayers.Invoice{}))

	CmdBuilder(cmd, RunInvoicesList, "list", "list invoices", Writer,
		aliasOpt("ls"), displayerType(&displayers.Invoice{}))

	CmdBuilder(cmd, RunInvoicesSummary, "summary <invoice-uuid>", "get a summary of an invoice", Writer,
		aliasOpt("s"), displayerType(&displayers.Invoice{}))

	CmdBuilder(cmd, RunInvoicesGetPDF, "pdf <invoice-uuid> <output-file.pdf>", "get a pdf of an invoice", Writer,
		aliasOpt("p"))

	CmdBuilder(cmd, RunInvoicesGetCSV, "csv <invoice-uuid> <output-file.csv>", "get a csv of an invoice", Writer,
		aliasOpt("c"))

	return cmd
}

func getInvoiceUUIDArg(ns string, args []string) (string, error) {
	if len(args) < 1 {
		return "", doctl.NewMissingArgsErr(ns)
	}

	return args[0], nil
}

func getOutputFileArg(ext string, args []string) string {
	if len(args) != 2 {
		return fmt.Sprintf("invoice.%s", ext)
	}

	return args[1]
}

// RunInvoicesGet runs invoice get.
func RunInvoicesGet(c *CmdConfig) error {
	uuid, err := getInvoiceUUIDArg(c.NS, c.Args)
	if err != nil {
		return err
	}

	invoice, err := c.Invoices().Get(uuid)
	if err != nil {
		return err
	}

	return c.Display(&displayers.Invoice{Invoice: invoice})
}

// RunInvoicesList runs invoice list.
func RunInvoicesList(c *CmdConfig) error {
	invoiceList, err := c.Invoices().List()
	if err != nil {
		return err
	}

	return c.Display(&displayers.InvoiceList{InvoiceList: invoiceList})
}

// RunInvoicesSummary runs an invoice summary.
func RunInvoicesSummary(c *CmdConfig) error {
	uuid, err := getInvoiceUUIDArg(c.NS, c.Args)
	if err != nil {
		return err
	}

	summary, err := c.Invoices().GetSummary(uuid)
	if err != nil {
		return err
	}

	return c.Display(&displayers.InvoiceSummary{InvoiceSummary: summary})
}

// RunInvoicesGetPDF runs an invoice get pdf.
func RunInvoicesGetPDF(c *CmdConfig) error {
	uuid, err := getInvoiceUUIDArg(c.NS, c.Args)
	if err != nil {
		return err
	}

	pdf, err := c.Invoices().GetPDF(uuid)
	if err != nil {
		return err
	}

	outputFile := getOutputFileArg("pdf", c.Args)

	err = ioutil.WriteFile(outputFile, pdf, 0644)
	if err != nil {
		return err
	}

	return nil
}

// RunInvoicesGetCSV runs an invoice get csv.
func RunInvoicesGetCSV(c *CmdConfig) error {
	uuid, err := getInvoiceUUIDArg(c.NS, c.Args)
	if err != nil {
		return err
	}

	csv, err := c.Invoices().GetCSV(uuid)
	if err != nil {
		return err
	}

	outputFile := getOutputFileArg("csv", c.Args)

	err = ioutil.WriteFile(outputFile, csv, 0644)
	if err != nil {
		return err
	}

	return nil
}
