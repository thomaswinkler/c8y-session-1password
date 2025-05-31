package main

import "github.com/thomaswinkler/c8y-session-1password/cmd"

// Version information set by build process
var (
	version = "1.0.0"
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
