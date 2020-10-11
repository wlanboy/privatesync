package application

import (
	"fmt"
	"os"
)

/*RunCommands for Sync*/
func (sync *Sync) RunCommands(args []string) {
	if len(args) == 1 {
		showHelp()
	} else if len(args) == 2 {
		object := args[1]
		if "crypt" == object {
			crypt()
		} else if "decrypt" == object {
			decrypt()
		} else {
			showHelp()
		}
	} else {
		showHelp()
	}
}

func showHelp() {
	fmt.Println("Usage:", os.Args[0], "verb")
	fmt.Println("verbs: crypt (crypts files from work directory to sync directory), decrypt (decrypts files from sync directory to work directory)")
}

func crypt() {

}

func decrypt() {

}