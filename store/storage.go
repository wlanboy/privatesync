package store

import (
	"bufio"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"golang.org/x/crypto/bcrypt"

	settings "github.com/wlanboy/privatesync/settings"
)

/*Storage containing everything*/
type Storage struct {
	Settings     *settings.Setting
	SettingsPath string
	/*PlainSecret holds the secret entered at runtime; it is never persisted*/
	PlainSecret string
}

/*Load Settings*/
func (storage *Storage) Load() {
	var setting settings.Setting = settings.Setting{}
	reader := bufio.NewReader(os.Stdin)

	info, err := os.Stat(storage.SettingsPath)
	if err != nil && !os.IsNotExist(err) {
		fmt.Println(err)
		os.Exit(1)
	}
	if info != nil {
		file, err := os.ReadFile(storage.SettingsPath)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		if err := json.Unmarshal(file, &setting); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		storage.Settings = &setting

		input := askQuestion(reader, "reenter secret for AES encryption")
		if !checkHashes(input, storage.Settings.Secret) {
			fmt.Println("wrong secret")
			os.Exit(1)
		}
		storage.PlainSecret = input

		if storage.Settings.Salt == "" {
			fmt.Println("no salt found in settings, generating one now (files crypted before this point can no longer be decrypted)")
			salt, err := generateSalt()
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			storage.Settings.Salt = salt
			storage.Save()
		}
	} else {
		setting.WorkPath = askQuestion(reader, "Path of files to protect")
		setting.SyncPath = askQuestion(reader, "Path to store crypted files (for sync like dropbox)")

		secret := askQuestion(reader, "secret for AES encryption")
		hash, err := hashPassword(secret)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		setting.Secret = hash

		salt, err := generateSalt()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		setting.Salt = salt

		storage.Settings = &setting
		storage.PlainSecret = secret
		storage.Save()
	}
}

func generateSalt() (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(salt), nil
}

func checkHashes(input string, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(input))
	return err == nil
}

/*Save Settings*/
func (storage *Storage) Save() {
	file, err := json.MarshalIndent(storage.Settings, "", " ")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if err := os.WriteFile(storage.SettingsPath, file, 0640); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func askQuestion(reader *bufio.Reader, question string) string {
	fmt.Printf("Enter %s: \n", question)
	readText, _ := reader.ReadString('\n')
	fmt.Println()

	response := strings.TrimSuffix(readText, "\n")
	response = strings.Replace(response, "\r", "", 1)
	return response
}

func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}
