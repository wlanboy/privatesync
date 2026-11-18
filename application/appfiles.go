package application

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	cryptExt  = ".crypt"
	tmpPrefix = ".privatesync-"
	tmpSuffix = ".tmp"
	dirPerm   = 0700
)

/*options of a crypt or decrypt run*/
type options struct {
	/*Prune removes encrypted files whose source file no longer exists (crypt only)*/
	Prune bool
	/*Force overwrites local files that are newer than the encrypted copy (decrypt only)*/
	Force bool
}

/*stats of a crypt or decrypt run*/
type stats struct {
	Written   int
	Unchanged int
	Conflicts int
	Ignored   int
	Removed   int
	Failed    int
}

func cryptName(name string) string {
	return name + cryptExt
}

/*plainName strips the .crypt extension; ok is false for names without it*/
func plainName(name string) (string, bool) {
	base := filepath.Base(name)
	if !strings.HasSuffix(base, cryptExt) || base == cryptExt {
		return "", false
	}
	return strings.TrimSuffix(name, cryptExt), true
}

func isTempFile(name string) bool {
	return strings.HasPrefix(name, tmpPrefix) && strings.HasSuffix(name, tmpSuffix)
}

/*writeAtomic writes dst via a temporary file in the same directory and renames it on success*/
func writeAtomic(dst string, fill func(w io.Writer) error) (err error) {
	f, err := os.CreateTemp(filepath.Dir(dst), tmpPrefix+"*"+tmpSuffix)
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer func() {
		if err != nil {
			f.Close()
			os.Remove(tmp)
		}
	}()
	if err = fill(f); err != nil {
		return err
	}
	if err = f.Sync(); err != nil {
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, dst)
}

/*validatePaths makes sure both directories differ and are not nested in each other*/
func validatePaths(workdir, syncdir string) error {
	work, err := filepath.Abs(workdir)
	if err != nil {
		return err
	}
	sync, err := filepath.Abs(syncdir)
	if err != nil {
		return err
	}
	if isWithin(work, sync) || isWithin(sync, work) {
		return fmt.Errorf("work path %q and sync path %q must not be the same or nested", work, sync)
	}
	return nil
}

func isWithin(parent, child string) bool {
	rel, err := filepath.Rel(parent, child)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

/*requireDir returns an error if path does not exist or is no directory*/
func requireDir(path string) error {
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("directory %q does not exist", path)
	}
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%q is not a directory", path)
	}
	return nil
}
