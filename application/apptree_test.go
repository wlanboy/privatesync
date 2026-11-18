package application

import (
	"bytes"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var baseTime = time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

/*testTree is a set of relative paths with content used to build a work directory*/
var testTree = map[string]string{
	"file.txt":              "hello",
	"noextension":           "no extension",
	".env":                  "SECRET=1",
	"archive.tar.gz":        "gz",
	"empty":                 "",
	"bilder/foto.jpg":       "jpg",
	"bilder/2024/jan/a.png": "deep",
	"docs/sub/notes.md":     strings.Repeat("x", chunkSize+123),
}

func writeTree(t *testing.T, dir string, tree map[string]string) {
	t.Helper()
	for rel, content := range tree {
		writeFile(t, filepath.Join(dir, rel), content, baseTime)
	}
}

func writeFile(t *testing.T, path, content string, mtime time.Time) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, mtime, mtime); err != nil {
		t.Fatal(err)
	}
}

/*readTree returns all regular files below dir with their content*/
func readTree(t *testing.T, dir string) map[string]string {
	t.Helper()
	tree := map[string]string{}
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(dir, path)
		tree[filepath.ToSlash(rel)] = string(data)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return tree
}

func assertTree(t *testing.T, dir string, want map[string]string) {
	t.Helper()
	got := readTree(t, dir)
	for rel, content := range want {
		if g, ok := got[rel]; !ok {
			t.Errorf("missing %s", rel)
		} else if g != content {
			t.Errorf("%s: content differs (got %d bytes, want %d)", rel, len(g), len(content))
		}
	}
	for rel := range got {
		if _, ok := want[rel]; !ok {
			t.Errorf("unexpected file %s", rel)
		}
	}
}

type testDirs struct {
	work, sync, restore string
}

func newTestDirs(t *testing.T) testDirs {
	base := t.TempDir()
	return testDirs{
		work:    filepath.Join(base, "work"),
		sync:    filepath.Join(base, "sync"),
		restore: filepath.Join(base, "restore"),
	}
}

func mustCrypt(t *testing.T, d testDirs, opts options) (stats, string) {
	t.Helper()
	key, _ := testKeys(t)
	var out bytes.Buffer
	st, err := cryptTree(d.work, d.sync, key, opts, &out)
	if err != nil {
		t.Fatalf("cryptTree: %v\n%s", err, out.String())
	}
	return st, out.String()
}

func mustDecrypt(t *testing.T, work, sync string, opts options) (stats, string) {
	t.Helper()
	key, _ := testKeys(t)
	var out bytes.Buffer
	st, err := decryptTree(work, sync, key, opts, &out)
	if err != nil {
		t.Fatalf("decryptTree: %v\n%s", err, out.String())
	}
	return st, out.String()
}

func TestTreeRoundtrip(t *testing.T) {
	d := newTestDirs(t)
	writeTree(t, d.work, testTree)

	st, _ := mustCrypt(t, d, options{})
	if st.Written != len(testTree) || st.Failed != 0 {
		t.Fatalf("crypt stats = %+v", st)
	}

	encrypted := map[string]string{}
	for rel := range readTree(t, d.sync) {
		if !strings.HasSuffix(rel, cryptExt) {
			t.Errorf("sync dir contains non .crypt file %s", rel)
		}
		encrypted[rel] = ""
	}
	if len(encrypted) != len(testTree) {
		t.Errorf("sync dir has %d files, want %d", len(encrypted), len(testTree))
	}
	if strings.Contains(readTree(t, d.sync)["docs/sub/notes.md.crypt"], strings.Repeat("x", 32)) {
		t.Error("encrypted file contains plaintext")
	}

	st, _ = mustDecrypt(t, d.restore, d.sync, options{})
	if st.Written != len(testTree) || st.Failed != 0 {
		t.Fatalf("decrypt stats = %+v", st)
	}
	assertTree(t, d.restore, testTree)

	info, err := os.Stat(filepath.Join(d.restore, "bilder", "2024", "jan", "a.png"))
	if err != nil {
		t.Fatal(err)
	}
	if !info.ModTime().Equal(baseTime) {
		t.Errorf("mtime = %v, want %v", info.ModTime(), baseTime)
	}
}

func TestCryptIsIncremental(t *testing.T) {
	d := newTestDirs(t)
	writeTree(t, d.work, testTree)
	mustCrypt(t, d, options{})

	st, out := mustCrypt(t, d, options{})
	if st.Written != 0 || st.Unchanged != len(testTree) {
		t.Errorf("second run stats = %+v\n%s", st, out)
	}

	writeFile(t, filepath.Join(d.work, "file.txt"), "changed", baseTime.Add(time.Hour))
	st, out = mustCrypt(t, d, options{})
	if st.Written != 1 || !strings.Contains(out, "crypt: file.txt") {
		t.Errorf("after change stats = %+v\n%s", st, out)
	}

	// same mtime, different size must be detected as well
	writeFile(t, filepath.Join(d.work, "file.txt"), "changed again", baseTime.Add(time.Hour))
	if st, _ = mustCrypt(t, d, options{}); st.Written != 1 {
		t.Errorf("size change stats = %+v", st)
	}
}

func TestCryptPrune(t *testing.T) {
	d := newTestDirs(t)
	writeTree(t, d.work, testTree)
	mustCrypt(t, d, options{})

	if err := os.RemoveAll(filepath.Join(d.work, "bilder")); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(d.work, "file.txt")); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(d.sync, "desktop.ini"), "foreign", baseTime)
	writeFile(t, filepath.Join(d.sync, tmpPrefix+"123"+tmpSuffix), "leftover", baseTime)

	st, _ := mustCrypt(t, d, options{})
	if st.Removed != 0 {
		t.Errorf("without --prune nothing must be removed, stats = %+v", st)
	}
	if _, err := os.Stat(filepath.Join(d.sync, "file.txt.crypt")); err != nil {
		t.Errorf("file.txt.crypt removed without --prune: %v", err)
	}

	st, out := mustCrypt(t, d, options{Prune: true})
	if st.Removed != 4 {
		t.Errorf("removed = %d, want 4 (3 files + temp file)\n%s", st.Removed, out)
	}
	for _, rel := range []string{"file.txt.crypt", "bilder"} {
		if _, err := os.Stat(filepath.Join(d.sync, rel)); !os.IsNotExist(err) {
			t.Errorf("%s should be gone: %v", rel, err)
		}
	}
	if _, err := os.Stat(filepath.Join(d.sync, "desktop.ini")); err != nil {
		t.Errorf("foreign file must not be pruned: %v", err)
	}
}

func TestCryptPruneSkippedOnErrors(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root can read unreadable files")
	}
	d := newTestDirs(t)
	writeTree(t, d.work, testTree)
	mustCrypt(t, d, options{})

	locked := filepath.Join(d.work, "bilder")
	if err := os.Chmod(locked, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(locked, 0700) })

	key, _ := testKeys(t)
	var out bytes.Buffer
	st, err := cryptTree(d.work, d.sync, key, options{Prune: true}, &out)
	if err != nil {
		t.Fatal(err)
	}
	if st.Failed == 0 || st.Removed != 0 {
		t.Errorf("stats = %+v, want failures and no removal\n%s", st, out.String())
	}
	if _, err := os.Stat(filepath.Join(d.sync, "bilder", "foto.jpg.crypt")); err != nil {
		t.Errorf("crypt copy of unreadable dir must survive: %v", err)
	}
}

func TestDecryptSkipsForeignAndBrokenFiles(t *testing.T) {
	d := newTestDirs(t)
	writeTree(t, d.work, testTree)
	mustCrypt(t, d, options{})

	writeFile(t, filepath.Join(d.sync, "desktop.ini"), "foreign", baseTime)
	writeFile(t, filepath.Join(d.sync, "x"), "", baseTime)
	writeFile(t, filepath.Join(d.sync, ".crypt"), "", baseTime)
	writeFile(t, filepath.Join(d.sync, "short.crypt"), "abc", baseTime)
	writeFile(t, filepath.Join(d.sync, tmpPrefix+"1"+tmpSuffix), "leftover", baseTime)

	key, _ := testKeys(t)
	var out bytes.Buffer
	st, err := decryptTree(d.restore, d.sync, key, options{}, &out)
	if err != nil {
		t.Fatal(err)
	}
	if st.Written != len(testTree) || st.Ignored != 3 || st.Failed != 1 {
		t.Errorf("stats = %+v\n%s", st, out.String())
	}
	assertTree(t, d.restore, testTree)
}

func TestDecryptRenamedFileFails(t *testing.T) {
	d := newTestDirs(t)
	writeTree(t, d.work, map[string]string{"a.txt": "a", "b.txt": "b"})
	mustCrypt(t, d, options{})

	// swap the two encrypted files
	a := filepath.Join(d.sync, "a.txt.crypt")
	b := filepath.Join(d.sync, "b.txt.crypt")
	tmp := filepath.Join(d.sync, "swap")
	for _, mv := range [][2]string{{a, tmp}, {b, a}, {tmp, b}} {
		if err := os.Rename(mv[0], mv[1]); err != nil {
			t.Fatal(err)
		}
	}

	key, _ := testKeys(t)
	var out bytes.Buffer
	st, _ := decryptTree(d.restore, d.sync, key, options{}, &out)
	if st.Failed != 2 || st.Written != 0 {
		t.Errorf("stats = %+v\n%s", st, out.String())
	}
	assertTree(t, d.restore, map[string]string{})
}

func TestDecryptConflicts(t *testing.T) {
	d := newTestDirs(t)
	writeTree(t, d.work, map[string]string{"a.txt": "remote"})
	mustCrypt(t, d, options{})

	target := filepath.Join(d.restore, "a.txt")

	// unchanged local copy is not rewritten
	mustDecrypt(t, d.restore, d.sync, options{})
	st, _ := mustDecrypt(t, d.restore, d.sync, options{})
	if st.Unchanged != 1 || st.Written != 0 {
		t.Errorf("unchanged stats = %+v", st)
	}

	// local file is newer: keep it
	writeFile(t, target, "local edit", baseTime.Add(time.Hour))
	st, out := mustDecrypt(t, d.restore, d.sync, options{})
	if st.Conflicts != 1 || !strings.Contains(out, "--force") {
		t.Errorf("conflict stats = %+v\n%s", st, out)
	}
	if got, _ := os.ReadFile(target); string(got) != "local edit" {
		t.Errorf("newer local file overwritten: %q", got)
	}

	// --force overwrites
	st, _ = mustDecrypt(t, d.restore, d.sync, options{Force: true})
	if st.Written != 1 {
		t.Errorf("force stats = %+v", st)
	}
	if got, _ := os.ReadFile(target); string(got) != "remote" {
		t.Errorf("content after --force = %q", got)
	}

	// local file is older: overwrite
	writeFile(t, target, "old local", baseTime.Add(-time.Hour))
	if st, _ = mustDecrypt(t, d.restore, d.sync, options{}); st.Written != 1 {
		t.Errorf("older local stats = %+v", st)
	}
	if got, _ := os.ReadFile(target); string(got) != "remote" {
		t.Errorf("content = %q", got)
	}
}

func TestMissingSourceDirectory(t *testing.T) {
	d := newTestDirs(t)
	key, _ := testKeys(t)
	if _, err := cryptTree(d.work, d.sync, key, options{Prune: true}, io.Discard); err == nil {
		t.Error("crypt with missing work dir must fail")
	}
	if _, err := decryptTree(d.work, d.sync, key, options{}, io.Discard); err == nil {
		t.Error("decrypt with missing sync dir must fail")
	}
}

func TestCryptCreatesSyncDir(t *testing.T) {
	d := newTestDirs(t)
	writeTree(t, d.work, map[string]string{"a": "a"})
	d.sync = filepath.Join(d.sync, "not", "yet", "there")
	mustCrypt(t, d, options{})
	if _, err := os.Stat(filepath.Join(d.sync, "a.crypt")); err != nil {
		t.Error(err)
	}
}

func TestCryptSkipsSymlinks(t *testing.T) {
	d := newTestDirs(t)
	writeTree(t, d.work, map[string]string{"a": "a"})
	if err := os.Symlink(filepath.Join(d.work, "a"), filepath.Join(d.work, "link")); err != nil {
		t.Skip("symlinks not supported:", err)
	}
	st, _ := mustCrypt(t, d, options{})
	if st.Written != 1 || st.Ignored != 1 {
		t.Errorf("stats = %+v", st)
	}
}
