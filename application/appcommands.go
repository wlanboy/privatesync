package application

import (
	"fmt"
	"os"
	"crypto/md5"
	"encoding/hex"
)

/*RunCommands for Sync*/
func (sync *Sync) RunCommands(args []string) {
	if len(args) == 1 {
		showHelp()
	} else if len(args) == 2 {
		object := args[1]
		if "crypt" == object {
			crypt(sync)
		} else if "decrypt" == object {
			decrypt(sync)
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

func createHash(key string) string {
	hasher := md5.New()
	hasher.Write([]byte(key))
	return hex.EncodeToString(hasher.Sum(nil))
}