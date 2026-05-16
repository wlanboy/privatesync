package main

import (
	"os"

	app "github.com/wlanboy/privatesync/application"
)

func main() {
	sync := app.Sync{}
	sync.Initialize()
	sync.RunCommands(os.Args)
}
