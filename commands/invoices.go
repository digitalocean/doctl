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
	"os"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/spf13/cobra"
)

// Invoices creates the invoices commands hierarchy.
func Invoices() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "invoice",
			Short:   "Display commands for retrieving invoices for your account",
			Long:    "The subcommands of `doctl invoice` retrieve details about invoices for your account.",
			GroupID: viewBillingGroup,
		},
	}

	getInvoiceDesc := `Retrieves an itemized list of resources and their costs on the specified invoice, including each resource's:
- ID
- UUID (if applicable)
- Product name
- Description
- Group description
- Amount charged, in USD
- Duration of usage for the invoice period
- Duration unit of measurement, such as hours
- The start time of the invoice period, in ISO8601 combined date and time format
- The end time of the invoice period, in ISO8601 combined date and time format
- The project name the resource belongs to
- Category, such as "iaas"

Use the ` + "`" + `doctl invoice list` + "`" + ` command to find the UUID of the invoice to retrieve.`
	cmdInvoicesGet := CmdBuilder(cmd, RunInvoicesGet, "get <invoice-uuid>", "Retrieve a list of all the items on an invoice",
		getInvoiceDesc, Writer, aliasOpt("g"), displayerType(&displayers.Invoice{}))
	cmdInvoicesGet.Example = `The following example retrieves details about an invoice with the ID ` + "`" + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + "`" + `: doctl invoice get f81d4fae-7dec-11d0-a765-00a0c91e6bf6`

	listInvoiceDesc := "Lists all of the invoices on your account including the UUID, amount in USD, and time period for each."
	cmdInvoicesList := CmdBuilder(cmd, RunInvoicesList, "list", "List all of the invoices for your account",
		listInvoiceDesc, Writer, aliasOpt("ls"), displayerType(&displayers.Invoice{}))
	cmdInvoicesList.Example = `The following example lists all of the invoices on your account and uses the ` + "`" + `--format` + "`" + ` flag to only return the product name and the amount charged for it: doctl invoice list --format Product,Amount`

	invoiceSummaryDesc := `Retrieves a summary of an invoice, including the following details:

- The invoice's UUID
- The year and month of the billing period
- The total amount of the invoice, in USD
- The name of the user associated with the invoice
- The company associated with the invoice
- The email address associated with the invoice
- The amount of product usage charges contributing to the invoice
- The amount of overage charges contributing to the invoice, such as bandwidth overages
- The amount of taxes contributing to the invoice
- The amount of any credits or other adjustments contributing to the invoice

Use the ` + "`" + `doctl invoice list` + "`" + ` command to find the UUID of the invoice to retrieve.`

	cmdInvoicesSummary := CmdBuilder(cmd, RunInvoicesSummary, "summary <invoice-uuid>", "Get a summary of an invoice",
		invoiceSummaryDesc, Writer, aliasOpt("s"), displayerType(&displayers.Invoice{}))
	cmdInvoicesSummary.Example = `The following example retrieves a summary of an invoice with the ID ` + "`" + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + "`" + `: doctl invoice summary f81d4fae-7dec-11d0-a765-00a0c91e6bf6`

	pdfInvoiceDesc := `This command downloads a PDF summary of a specific invoice to the provided location.

Use the ` + "`" + `doctl invoice list` + "`" + ` command to find the UUID of the invoice to retrieve.`

	cmdInvoicesGetPDF := CmdBuilder(cmd, RunInvoicesGetPDF, "pdf <invoice-uuid> <output-file.pdf>", "Downloads a PDF file of a specific invoice to your local machine",
		pdfInvoiceDesc, Writer, aliasOpt("p"))
	cmdInvoicesGetPDF.Example = `The following example downloads a PDF summary of an invoice with the ID ` + "`" + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + "`" + ` to the file ` + "`" + `invoice.pdf` + "`" + `: doctl invoice pdf f81d4fae-7dec-11d0-a765-00a0c91e6bf6 invoice.pdf`

	csvInvoiceDesc := `Downloads a CSV-formatted file of a specific invoice to your local machine.

Use the ` + "`" + `doctl invoice list` + "`" + ` command to find the UUID of the invoice to retrieve.`
	cmdInvoicesGetCSV := CmdBuilder(cmd, RunInvoicesGetCSV, "csv <invoice-uuid> <output-file.csv>", "Downloads a CSV file of a specific invoice to you local machine",
		csvInvoiceDesc, Writer, aliasOpt("c"))
	cmdInvoicesGetCSV.Example = `The following example downloads a CSV summary of an invoice with the ID ` + "`" + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + "`" + ` to the file ` + "`" + `invoice.csv` + "`" + `: doctl invoice csv f81d4fae-7dec-11d0-a765-00a0c91e6bf6 invoice.csv`

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

	err = os.WriteFile(outputFile, pdf, 0644)
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

	err = os.WriteFile(outputFile, csv, 0644)
	if err != nil {
		return err
	}

	return nil
}
