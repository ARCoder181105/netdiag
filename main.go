/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package main

import "github.com/ARCoder181105/netdiag/cmd"

// These variables are injected at build time via -ldflags
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	// Pass version info to the cmd package
	cmd.SetVersionInfo(version, commit, date)
	cmd.Execute()
}