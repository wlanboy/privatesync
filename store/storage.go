package store

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"golang.org/x/crypto/bcrypt"
	"strings"

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
		setting.WorkPath = askQuestion("Path of files to protect")
		setting.SyncPath = askQuestion("Path to store crypted files (for sync like dropbox)")
		setting.Secret, _ = hashPassword(askQuestion("secret for AES encryption"))
		storage.Settings = &setting
		storage.Save()
	}
}

/*Save Settings*/
func (storage *Storage) Save() {
	file, _ := json.MarshalIndent(storage.Settings, "", " ")
	_ = ioutil.WriteFile(storage.SettingsPath, file, 0640)
}

func askQuestion(question string) string {
	fmt.Printf("Enter %s: \n", question)
	reader := bufio.NewReader(os.Stdin)
	readText, _ := reader.ReadString('\n')
	fmt.Println()

	response := string(readText)
	response = strings.TrimSuffix(response, "\n")
	return response
}

func hashPassword(password string) (string, error) {
    bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
    return string(bytes), err
}