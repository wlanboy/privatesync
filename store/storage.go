package store

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/bcrypt"

	settings "github.com/wlanboy/privatesync/settings"
)

const (
	settingsFileName = "syncsettings.json"
	settingsDirName  = "privatesync"
	bcryptCost       = 14
)

var (
	/*ErrWrongSecret is returned when the entered secret does not match the stored hash*/
	ErrWrongSecret = errors.New("wrong secret")
	/*ErrSecretMismatch is returned when the secret confirmation differs during setup*/
	ErrSecretMismatch = errors.New("secrets do not match")
)

/*Storage containing everything*/
type Storage struct {
	Settings     *settings.Setting
	SettingsPath string
	/*PlainSecret holds the secret entered at runtime; it is never persisted*/
	PlainSecret string
}

/*
DefaultSettingsPath returns the settings file in the user config directory.
If that file does not exist but a syncsettings.json exists in the current
directory (location used by older versions), that file is returned and legacy is true.
*/
func DefaultSettingsPath() (path string, legacy bool, err error) {
	cfgDir, err := os.UserConfigDir()
	if err != nil {
		return "", false, err
	}
	path = filepath.Join(cfgDir, settingsDirName, settingsFileName)
	if _, err := os.Stat(path); err == nil {
		return path, false, nil
	}
	if cwd, err := os.Getwd(); err == nil {
		old := filepath.Join(cwd, settingsFileName)
		if _, err := os.Stat(old); err == nil {
			return old, true, nil
		}
	}
	return path, false, nil
}

/*Load reads the settings and verifies the secret; runs the interactive setup if no settings exist*/
func (storage *Storage) Load(p Prompter, out io.Writer) error {
	file, err := os.ReadFile(storage.SettingsPath)
	if errors.Is(err, os.ErrNotExist) {
		return storage.setup(p, out)
	}
	if err != nil {
		return err
	}

	var setting settings.Setting
	if err := json.Unmarshal(file, &setting); err != nil {
		return fmt.Errorf("%s: %w", storage.SettingsPath, err)
	}
	if setting.WorkPath == "" || setting.SyncPath == "" || setting.Secret == "" {
		return fmt.Errorf("%s: WorkPath, SyncPath and Secret must be set", storage.SettingsPath)
	}
	storage.Settings = &setting

	input, err := p.AskSecret("secret for AES encryption")
	if err != nil {
		return err
	}
	if !checkHashes(input, setting.Secret) {
		return ErrWrongSecret
	}
	storage.PlainSecret = input

	if setting.Salt == "" {
		fmt.Fprintln(out, "no salt found in settings, generating one now (files crypted before this point can no longer be decrypted)")
		salt, err := generateSalt()
		if err != nil {
			return err
		}
		setting.Salt = salt
		return storage.Save()
	}
	return nil
}

func (storage *Storage) setup(p Prompter, out io.Writer) error {
	fmt.Fprintf(out, "no settings found, creating %s\n", storage.SettingsPath)

	var setting settings.Setting
	var err error
	if setting.WorkPath, err = askPath(p, "path of files to protect"); err != nil {
		return err
	}
	if setting.SyncPath, err = askPath(p, "path to store crypted files (for sync like dropbox)"); err != nil {
		return err
	}

	secret, err := p.AskSecret("secret for AES encryption")
	if err != nil {
		return err
	}
	if secret == "" {
		return errors.New("secret must not be empty")
	}
	confirm, err := p.AskSecret("secret again")
	if err != nil {
		return err
	}
	if secret != confirm {
		return ErrSecretMismatch
	}

	if setting.Secret, err = hashPassword(secret); err != nil {
		return err
	}
	if setting.Salt, err = generateSalt(); err != nil {
		return err
	}

	storage.Settings = &setting
	storage.PlainSecret = secret
	return storage.Save()
}

func askPath(p Prompter, question string) (string, error) {
	answer, err := p.Ask(question)
	if err != nil {
		return "", err
	}
	answer = strings.TrimSpace(answer)
	if answer == "" {
		return "", fmt.Errorf("%s must not be empty", question)
	}
	expanded, err := ExpandPath(answer)
	if err != nil {
		return "", err
	}
	return filepath.Abs(expanded)
}

/*ExpandPath replaces a leading ~ with the home directory*/
func ExpandPath(path string) (string, error) {
	if path != "~" && !strings.HasPrefix(path, "~/") && !strings.HasPrefix(path, `~\`) {
		return path, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, path[1:]), nil
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
func (storage *Storage) Save() error {
	file, err := json.MarshalIndent(storage.Settings, "", " ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(storage.SettingsPath), 0700); err != nil {
		return err
	}
	return os.WriteFile(storage.SettingsPath, file, 0600)
}

func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	return string(bytes), err
}
