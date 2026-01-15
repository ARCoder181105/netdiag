/*
Copyright © 2026 ARCoder181105 <EMAIL ADDRESS>
*/
// Package cmd implements the CLI commands.
package cmd

import (
	"fmt"
	"strings"

	"github.com/likexian/whois"
	"github.com/spf13/cobra"

	"github.com/ARCoder181105/netdiag/pkg/output"
)

// whoisCmd represents the whois command
var whoisCmd = &cobra.Command{
	Use:   "whois <domain>",
	Short: "Retrieve domain registration information",
	Long: `Query the WHOIS database to find information about a domain name,
including the registrar, creation date, and expiration date.`,
	Args: cobra.ExactArgs(1),
	Run: func(_ *cobra.Command, args []string) {
		domain := args[0]

		output.PrintInfo(fmt.Sprintf("Querying WHOIS for %s...", domain))

		result, err := whois.Whois(domain)
		if err != nil {
			output.PrintError(fmt.Sprintf("Whois query failed: %v", err))
			return
		}

		fmt.Println("\n" + strings.TrimSpace(result))
	},
}

func init() {
	rootCmd.AddCommand(whoisCmd)
}
