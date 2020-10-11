package application

import (
	"fmt"
	"os"
	"io/ioutil"
	"encoding/json"
	"path/filepath"

	store "../store"
	"github.com/joho/godotenv"
)

/*Initialize configuration*/
func (sync *Sync) Initialize() {
	godotenv.Load()

	homedir, _ := os.Getwd()
	sync.SettingsFilePath = filepath.Join(homedir, "syncsettings.json")

	var storage store.Storage = store.Storage{}

	fileExists, err := os.Stat(sync.SettingsFilePath)
	if err != nil {
		fmt.Println(err)
	}
	if fileExists != nil {
		file, _ := ioutil.ReadFile(sync.SettingsFilePath)
		json.Unmarshal(file, &storage)
    } else {
		fmt.Println("Example file does not exist (or is a directory)")
    }
	sync.Storage = &storage

	sync.Storage.Save()
}
