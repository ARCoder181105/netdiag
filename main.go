/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package main

import "github.com/ARCoder181105/netdiag/cmd"

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	// Set version information in cmd package
	cmd.Version = version
	cmd.Commit = commit
	cmd.Date = date

	cmd.Execute()
}
