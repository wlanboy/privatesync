package application

import (
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
	store "github.com/wlanboy/privatesync/store"
)

/*Initialize configuration*/
func (sync *Sync) Initialize() {
	godotenv.Load()

	homedir, _ := os.Getwd()
	sync.SettingsFilePath = filepath.Join(homedir, "syncsettings.json")

	var storage store.Storage = store.Storage{}
	storage.SettingsPath = sync.SettingsFilePath
	sync.Storage = &storage

	sync.Storage.Load()
}
