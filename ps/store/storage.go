package store

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	settings "../settings"
)

/*Settings containing everything*/
type Storage struct {
	Settings   *settings.Setting
	SettingsPath   string
}

/*Load Settings*/
func (storage *Storage) Load() {
	var setting settings.Setting = settings.Setting{}

	info, err := os.Stat(storage.SettingsPath)
	if err != nil {
		fmt.Println(err)
	}
	if info != nil {
		file, _ := ioutil.ReadFile(storage.SettingsPath)
		json.Unmarshal(file, &setting)
		storage.Settings = &setting
	} else {
		setting.WorkPath = ""
		setting.SyncPath = ""
		setting.Secret = ""
		storage.Settings = &setting
		storage.Save()
	}
}

/*Save Settings*/
func (storage *Storage) Save() {
	file, _ := json.MarshalIndent(storage.Settings, "", " ")
	_ = ioutil.WriteFile(storage.SettingsPath, file, 0640)
}
