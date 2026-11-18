package store

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	settings "github.com/wlanboy/privatesync/settings"
)

type fakePrompter struct {
	answers []string
	asked   int
}

func (f *fakePrompter) next() (string, error) {
	if f.asked >= len(f.answers) {
		return "", ErrNoInput
	}
	f.asked++
	return f.answers[f.asked-1], nil
}

func (f *fakePrompter) Ask(string) (string, error)       { return f.next() }
func (f *fakePrompter) AskSecret(string) (string, error) { return f.next() }

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
	if checkHashes("wrongpassword", hash) {
		t.Error("checkHashes should return false for wrong password")
	}
}

func TestSaveWritesPrivateFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "settings.json")
	setting := settings.Setting{WorkPath: "/work", SyncPath: "/sync", Secret: "hashed-secret", Salt: "c2FsdA=="}
	storage := &Storage{Settings: &setting, SettingsPath: path}
	if err := storage.Save(); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("settings file not created: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("permissions = %v, want 0600", info.Mode().Perm())
	}
	data, _ := os.ReadFile(path)
	var loaded settings.Setting
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("could not parse saved JSON: %v", err)
	}
	if loaded != setting {
		t.Errorf("loaded %+v, want %+v", loaded, setting)
	}
}

func TestSetupAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	work := filepath.Join(dir, "work")

	storage := &Storage{SettingsPath: path}
	p := &fakePrompter{answers: []string{"  " + work + "  ", "relative/sync", "pw", "pw"}}
	if err := storage.Load(p, io.Discard); err != nil {
		t.Fatal(err)
	}
	if storage.PlainSecret != "pw" || storage.Settings.Salt == "" {
		t.Errorf("storage after setup = %+v", storage)
	}
	if storage.Settings.WorkPath != work {
		t.Errorf("WorkPath = %q, want trimmed %q", storage.Settings.WorkPath, work)
	}
	if !filepath.IsAbs(storage.Settings.SyncPath) {
		t.Errorf("SyncPath %q should be absolute", storage.Settings.SyncPath)
	}
	data, _ := os.ReadFile(path)
	if strings.Contains(string(data), `"pw"`) {
		t.Error("plain secret must not be stored")
	}

	loaded := &Storage{SettingsPath: path}
	if err := loaded.Load(&fakePrompter{answers: []string{"pw"}}, io.Discard); err != nil {
		t.Fatal(err)
	}
	if *loaded.Settings != *storage.Settings {
		t.Errorf("loaded %+v, want %+v", loaded.Settings, storage.Settings)
	}

	wrong := &Storage{SettingsPath: path}
	if err := wrong.Load(&fakePrompter{answers: []string{"nope"}}, io.Discard); !errors.Is(err, ErrWrongSecret) {
		t.Errorf("err = %v, want ErrWrongSecret", err)
	}
}

func TestSetupErrors(t *testing.T) {
	tests := []struct {
		name    string
		answers []string
		want    error
	}{
		{"secret mismatch", []string{"/w", "/s", "pw", "typo"}, ErrSecretMismatch},
		{"stdin closed", []string{"/w"}, ErrNoInput},
		{"empty secret", []string{"/w", "/s", ""}, nil},
		{"empty path", []string{"", "/s"}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "settings.json")
			storage := &Storage{SettingsPath: path}
			err := storage.Load(&fakePrompter{answers: tt.answers}, io.Discard)
			if err == nil {
				t.Fatal("expected error")
			}
			if tt.want != nil && !errors.Is(err, tt.want) {
				t.Errorf("err = %v, want %v", err, tt.want)
			}
			if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
				t.Error("settings must not be written on error")
			}
		})
	}
}

func TestLoadGeneratesMissingSalt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	hash, _ := hashPassword("pw")
	old := &Storage{SettingsPath: path, Settings: &settings.Setting{WorkPath: "/w", SyncPath: "/s", Secret: hash}}
	if err := old.Save(); err != nil {
		t.Fatal(err)
	}

	storage := &Storage{SettingsPath: path}
	if err := storage.Load(&fakePrompter{answers: []string{"pw"}}, io.Discard); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	var saved settings.Setting
	json.Unmarshal(data, &saved)
	if saved.Salt == "" || saved.Salt != storage.Settings.Salt {
		t.Errorf("salt not persisted: %+v", saved)
	}
}

func TestLoadInvalidSettings(t *testing.T) {
	dir := t.TempDir()
	tests := map[string]string{
		"broken.json":     "{not json",
		"incomplete.json": `{"WorkPath": "/w"}`,
	}
	for name, content := range tests {
		path := filepath.Join(dir, name)
		os.WriteFile(path, []byte(content), 0600)
		p := &fakePrompter{}
		if err := (&Storage{SettingsPath: path}).Load(p, io.Discard); err == nil {
			t.Errorf("%s: expected error", name)
		}
		if p.asked != 0 {
			t.Errorf("%s: secret must not be asked for invalid settings", name)
		}
	}
}

func TestExpandPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip(err)
	}
	tests := map[string]string{
		"~":           home,
		"~/Dropbox":   filepath.Join(home, "Dropbox"),
		"/abs/path":   "/abs/path",
		"rel/path":    "rel/path",
		"~other/path": "~other/path",
	}
	for in, want := range tests {
		if got, _ := ExpandPath(in); got != want {
			t.Errorf("ExpandPath(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestReadLine(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"a\nb\n", []string{"a", "b"}},
		{"windows\r\n", []string{"windows"}},
		{"no newline", []string{"no newline"}},
	}
	for _, tt := range tests {
		r := bufio.NewReader(strings.NewReader(tt.input))
		for _, want := range tt.want {
			if got, err := readLine(r); err != nil || got != want {
				t.Errorf("readLine(%q) = %q, %v; want %q", tt.input, got, err, want)
			}
		}
		if _, err := readLine(r); !errors.Is(err, ErrNoInput) {
			t.Errorf("readLine(%q) at end: err = %v, want ErrNoInput", tt.input, err)
		}
	}
}

func TestTerminalPrompterWithoutTTY(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	w.WriteString("/some/path\nsecret\n")
	w.Close()

	var out strings.Builder
	p := NewTerminalPrompter(r, &out)
	if got, err := p.Ask("path"); err != nil || got != "/some/path" {
		t.Errorf("Ask = %q, %v", got, err)
	}
	if got, err := p.AskSecret("secret"); err != nil || got != "secret" {
		t.Errorf("AskSecret = %q, %v", got, err)
	}
	if !strings.Contains(out.String(), "Enter path") {
		t.Errorf("prompt output = %q", out.String())
	}
}
