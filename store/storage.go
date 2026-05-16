package store

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"golang.org/x/crypto/bcrypt"

	settings "github.com/wlanboy/privatesync/settings"
)

/*Settings containing everything*/
type Storage struct {
	Settings     *settings.Setting
	SettingsPath string
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

		input := askQuestion("reenter secret for AES encryption")
		ok := checkHashes(input, storage.Settings.Secret)
		if !ok {
			fmt.Println("wrong secret")
			os.Exit(0)
		}
	} else {
		setting.WorkPath = askQuestion("Path of files to protect")
		setting.SyncPath = askQuestion("Path to store crypted files (for sync like dropbox)")
		setting.Secret, _ = hashPassword(askQuestion("secret for AES encryption"))
		storage.Settings = &setting
		storage.Save()
	}
}

func checkHashes(input string, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(input))
	return err == nil
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
	filterd := strings.Replace(response, "\r", "", 1)
	return filterd
}

func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}
