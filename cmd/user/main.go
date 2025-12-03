package main

import (
	"fmt"
	"os"
	"rankr/cmd/auth/command"
)

func main() {
	if err := command.RootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
