package application

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestCryptNameRoundtrip(t *testing.T) {
	names := []string{"file.txt", "archive.tar.gz", "foo", ".env", "Makefile.", "dir/sub/file", "a..b"}
	for _, name := range names {
		got, ok := plainName(cryptName(name))
		if !ok || got != name {
			t.Errorf("plainName(cryptName(%q)) = %q, %v", name, got, ok)
		}
	}
}

func TestPlainNameRejectsForeignFiles(t *testing.T) {
	for _, name := range []string{"desktop.ini", ".DS_Store", "x", ".crypt", "dir/.crypt", "file.crypt.bak"} {
		if got, ok := plainName(name); ok {
			t.Errorf("plainName(%q) = %q, want rejection", name, got)
		}
	}
}

func TestValidatePaths(t *testing.T) {
	base := t.TempDir()
	tests := []struct {
		work, sync string
		wantErr    bool
	}{
		{filepath.Join(base, "work"), filepath.Join(base, "sync"), false},
		{filepath.Join(base, "work"), filepath.Join(base, "work2"), false},
		{filepath.Join(base, "work"), filepath.Join(base, "work"), true},
		{filepath.Join(base, "work"), filepath.Join(base, "work", "sync"), true},
		{filepath.Join(base, "sync", "work"), filepath.Join(base, "sync"), true},
		{filepath.Join(base, "work") + string(filepath.Separator), filepath.Join(base, "work"), true},
	}
	for _, tt := range tests {
		err := validatePaths(tt.work, tt.sync)
		if (err != nil) != tt.wantErr {
			t.Errorf("validatePaths(%q, %q) err = %v, wantErr %v", tt.work, tt.sync, err, tt.wantErr)
		}
	}
}

func TestWriteAtomicSuccess(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, "out.txt")
	if err := os.WriteFile(dst, []byte("old"), 0600); err != nil {
		t.Fatal(err)
	}
	err := writeAtomic(dst, func(w io.Writer) error {
		_, err := w.Write([]byte("new"))
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(dst); string(got) != "new" {
		t.Errorf("content = %q", got)
	}
	assertNoTempFiles(t, dir)
}

func TestWriteAtomicFailureKeepsOldFile(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, "out.txt")
	if err := os.WriteFile(dst, []byte("old"), 0600); err != nil {
		t.Fatal(err)
	}
	boom := errors.New("boom")
	err := writeAtomic(dst, func(w io.Writer) error {
		w.Write([]byte("partial"))
		return boom
	})
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v, want boom", err)
	}
	if got, _ := os.ReadFile(dst); string(got) != "old" {
		t.Errorf("content = %q, old file must stay untouched", got)
	}
	assertNoTempFiles(t, dir)
}

func assertNoTempFiles(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if isTempFile(e.Name()) {
			t.Errorf("temp file left behind: %s", e.Name())
		}
	}
}
