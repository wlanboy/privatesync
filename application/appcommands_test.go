package application

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	store "github.com/wlanboy/privatesync/store"
)

/*fakePrompter returns the given answers in order and fails once they are used up*/
type fakePrompter struct {
	answers []string
	asked   int
}

func (f *fakePrompter) next() (string, error) {
	if f.asked >= len(f.answers) {
		return "", store.ErrNoInput
	}
	f.asked++
	return f.answers[f.asked-1], nil
}

func (f *fakePrompter) Ask(string) (string, error)       { return f.next() }
func (f *fakePrompter) AskSecret(string) (string, error) { return f.next() }

func newTestSync(answers ...string) (*Sync, *fakePrompter, *bytes.Buffer, *bytes.Buffer) {
	p := &fakePrompter{answers: answers}
	var out, errOut bytes.Buffer
	return &Sync{Prompter: p, Out: &out, ErrOut: &errOut}, p, &out, &errOut
}

func TestRunCommandsUsageDoesNotPrompt(t *testing.T) {
	tests := []struct {
		args []string
		want int
	}{
		{nil, exitUsage},
		{[]string{"foo"}, exitUsage},
		{[]string{"help"}, exitOK},
		{[]string{"--help"}, exitOK},
		{[]string{"crypt", "-h"}, exitOK},
		{[]string{"crypt", "--force"}, exitUsage},
		{[]string{"decrypt", "--prune"}, exitUsage},
		{[]string{"crypt", "extra"}, exitUsage},
	}
	for _, tt := range tests {
		sync, p, _, _ := newTestSync()
		if got := sync.RunCommands(tt.args); got != tt.want {
			t.Errorf("RunCommands(%v) = %d, want %d", tt.args, got, tt.want)
		}
		if p.asked != 0 {
			t.Errorf("RunCommands(%v) asked for input", tt.args)
		}
	}
}

func TestRunCommandsEndToEnd(t *testing.T) {
	base := t.TempDir()
	work := filepath.Join(base, "work")
	syncDir := filepath.Join(base, "sync")
	config := filepath.Join(base, "config", "syncsettings.json")
	writeTree(t, work, map[string]string{"a.txt": "a", "dir/b.txt": "b"})

	// first run: setup asks for both paths and the secret twice
	sync, _, out, errOut := newTestSync(work, syncDir, "pw", "pw")
	if code := sync.RunCommands([]string{"crypt", "--config", config}); code != exitOK {
		t.Fatalf("setup crypt exit %d\n%s%s", code, out, errOut)
	}
	if info, err := os.Stat(config); err != nil || info.Mode().Perm() != 0600 {
		t.Errorf("config file: %v, %v", info, err)
	}
	if !strings.Contains(out.String(), "crypt: 2 written") {
		t.Errorf("output = %s", out)
	}

	// wrong secret
	sync, _, _, errOut = newTestSync("wrong")
	if code := sync.RunCommands([]string{"decrypt", "--config", config}); code != exitError {
		t.Errorf("wrong secret exit %d", code)
	}
	if !strings.Contains(errOut.String(), store.ErrWrongSecret.Error()) {
		t.Errorf("stderr = %s", errOut)
	}

	// decrypt after deleting the work copy
	if err := os.RemoveAll(work); err != nil {
		t.Fatal(err)
	}
	sync, _, out, errOut = newTestSync("pw")
	if code := sync.RunCommands([]string{"decrypt", "--config", config}); code != exitOK {
		t.Fatalf("decrypt exit %d\n%s%s", code, out, errOut)
	}
	assertTree(t, work, map[string]string{"a.txt": "a", "dir/b.txt": "b"})
}

func TestRunCommandsRejectsNestedPaths(t *testing.T) {
	base := t.TempDir()
	work := filepath.Join(base, "work")
	writeTree(t, work, map[string]string{"a": "a"})

	sync, _, _, errOut := newTestSync(work, filepath.Join(work, "sync"), "pw", "pw")
	if code := sync.RunCommands([]string{"crypt", "--config", filepath.Join(base, "c.json")}); code != exitError {
		t.Errorf("exit %d, want %d", code, exitError)
	}
	if !strings.Contains(errOut.String(), "nested") {
		t.Errorf("stderr = %s", errOut)
	}
}
