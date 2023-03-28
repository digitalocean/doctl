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
			Short: "Display commands for retrieving invoices for your account",
			Long:  "The subcommands of `doctl invoice` retrieve details about invoices for your account.",
		},
	}

	getInvoiceDesc := `This command retrieves a detailed list of all the items on a specific invoice.

Use the ` + "`" + `doctl invoice list` + "`" + ` command to find the UUID of the invoice to retrieve.`
	CmdBuilder(cmd, RunInvoicesGet, "get <invoice-uuid>", "Retrieve a list of all the items on an invoice",
		getInvoiceDesc, Writer, aliasOpt("g"), displayerType(&displayers.Invoice{}))

	listInvoiceDesc := "This command lists all of the invoices on your account including the UUID, amount in USD, and time period for each."
	CmdBuilder(cmd, RunInvoicesList, "list", "List all of the invoices for your account",
		listInvoiceDesc, Writer, aliasOpt("ls"), displayerType(&displayers.Invoice{}))

	invoiceSummaryDesc := `This command retrieves a summary of a specific invoice including the following details:

- The invoice's UUID
- The year and month of the billing period
- The total amount of the invoice, in USD
- The name of the user associated with the invoice
- The company associated with the invoice
- The email address associated with the invoice
- The amount of product usage charges contributing to the invoice
- The amount of overage charges contributing to the invoice (e.g. bandwidth)
- The amount of taxes contributing to the invoice
- The amount of any credits or other adjustments contributing to the invoice

Use the ` + "`" + `doctl invoice list` + "`" + ` command to find the UUID of the invoice to retrieve.`
	CmdBuilder(cmd, RunInvoicesSummary, "summary <invoice-uuid>", "Get a summary of an invoice",
		invoiceSummaryDesc, Writer, aliasOpt("s"), displayerType(&displayers.Invoice{}))

	pdfInvoiceDesc := `This command downloads a PDF summary of a specific invoice to the provided location.

Use the ` + "`" + `doctl invoice list` + "`" + ` command to find the UUID of the invoice to retrieve.`
	CmdBuilder(cmd, RunInvoicesGetPDF, "pdf <invoice-uuid> <output-file.pdf>", "Download a PDF file of an invoice",
		pdfInvoiceDesc, Writer, aliasOpt("p"))

	csvInvoiceDesc := `This command downloads a CSV formatted file for a specific invoice to the provided location.

Use the ` + "`" + `doctl invoice list` + "`" + ` command to find the UUID of the invoice to retrieve.`
	CmdBuilder(cmd, RunInvoicesGetCSV, "csv <invoice-uuid> <output-file.csv>", "Download a CSV file of an invoice",
		csvInvoiceDesc, Writer, aliasOpt("c"))

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
