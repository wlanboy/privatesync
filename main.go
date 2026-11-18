package main

import (
	"os"

	app "github.com/wlanboy/privatesync/application"
	store "github.com/wlanboy/privatesync/store"
)

func main() {
	sync := app.Sync{
		Prompter: store.NewTerminalPrompter(os.Stdin, os.Stdout),
		Out:      os.Stdout,
		ErrOut:   os.Stderr,
	}
	os.Exit(sync.RunCommands(os.Args[1:]))
}
