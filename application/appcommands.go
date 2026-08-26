package application

import (
	"fmt"
	"os"
)

/*RunCommands for Sync*/
func (sync *Sync) RunCommands(args []string) {
	if len(args) == 1 {
		showHelp()
		return
	}
	if len(args) == 2 {
		var err error
		switch args[1] {
		case "crypt":
			err = crypt(sync)
		case "decrypt":
			err = decrypt(sync)
		default:
			showHelp()
			return
		}
		if err != nil {
			fmt.Println("error:", err)
			os.Exit(1)
		}
		return
	}
	showHelp()
}

func showHelp() {
	fmt.Println("Usage:", os.Args[0], "verb")
	fmt.Println("verbs:")
	fmt.Println("crypt (crypts files from work directory to sync directory)")
	fmt.Println("decrypt (decrypts files from sync directory to work directory)")
}
