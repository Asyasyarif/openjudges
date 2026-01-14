package main

import (
	"os"
	"strings"

	"openllmjudge/cmd"
)

func main() {
	// Check for slash commands before cobra processing
	if len(os.Args) > 1 {
		firstArg := os.Args[1]

		// Handle slash commands
		if strings.HasPrefix(firstArg, "/") {
			cmd.HandleSlashCommand(firstArg, os.Args[2:])
			return
		}
	}

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
