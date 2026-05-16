package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	settings "github.com/wlanboy/privatesync/settings"
)

func TestHashPasswordAndCheckHashes(t *testing.T) {
	password := "mysecret"
	hash, err := hashPassword(password)
	if err != nil {
		t.Fatalf("hashPassword returned error: %v", err)
	}
	if hash == "" {
		t.Fatal("hash should not be empty")
	}
	if !checkHashes(password, hash) {
		t.Error("checkHashes should return true for matching password")
	}
}

func TestCheckHashesWrongPassword(t *testing.T) {
	hash, _ := hashPassword("correctpassword")
	if checkHashes("wrongpassword", hash) {
		t.Error("checkHashes should return false for wrong password")
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")

	setting := settings.Setting{
		WorkPath: "/work",
		SyncPath: "/sync",
		Secret:   "hashed-secret",
	}
	storage := &Storage{
		Settings:     &setting,
		SettingsPath: path,
	}
	storage.Save()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("settings file not created: %v", err)
	}

	var loaded settings.Setting
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("could not parse saved JSON: %v", err)
	}
	if loaded.WorkPath != setting.WorkPath {
		t.Errorf("WorkPath: got %q, want %q", loaded.WorkPath, setting.WorkPath)
	}
	if loaded.SyncPath != setting.SyncPath {
		t.Errorf("SyncPath: got %q, want %q", loaded.SyncPath, setting.SyncPath)
	}
	if loaded.Secret != setting.Secret {
		t.Errorf("Secret: got %q, want %q", loaded.Secret, setting.Secret)
	}
}

func TestHashPasswordDifferentHashes(t *testing.T) {
	h1, _ := hashPassword("samepassword")
	h2, _ := hashPassword("samepassword")
	if h1 == h2 {
		t.Error("bcrypt should produce different hashes per call (random salt)")
	}
}
