package main

import (
	"os"

	app "./application"
)

func main() {
	sync := app.Sync{}
	sync.Initialize()
	sync.RunCommands(os.Args)
}
