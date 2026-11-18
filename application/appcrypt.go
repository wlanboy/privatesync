package application

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
)

/*run carries the state shared by one crypt or decrypt pass*/
type run struct {
	key  []byte
	opts options
	out  io.Writer
	st   stats
}

func (r *run) fail(rel string, d fs.DirEntry, err error) error {
	fmt.Fprintf(r.out, "error: %s: %v\n", rel, err)
	r.st.Failed++
	if d != nil && d.IsDir() {
		return fs.SkipDir
	}
	return nil
}

/*cryptTree encrypts every regular file below workdir into syncdir, keeping the directory structure*/
func cryptTree(workdir, syncdir string, key []byte, opts options, out io.Writer) (stats, error) {
	if err := requireDir(workdir); err != nil {
		return stats{}, err
	}
	if err := os.MkdirAll(syncdir, dirPerm); err != nil {
		return stats{}, err
	}

	r := &run{key: key, opts: opts, out: out}
	expected := map[string]bool{}
	err := filepath.WalkDir(workdir, func(path string, d fs.DirEntry, err error) error {
		rel, relErr := filepath.Rel(workdir, path)
		if relErr != nil {
			return relErr
		}
		if err != nil {
			return r.fail(rel, d, err)
		}
		if rel == "." {
			return nil
		}
		if d.IsDir() {
			if err := os.MkdirAll(filepath.Join(syncdir, rel), dirPerm); err != nil {
				return r.fail(rel, d, err)
			}
			return nil
		}
		if isTempFile(d.Name()) {
			return nil
		}
		if !d.Type().IsRegular() {
			fmt.Fprintf(out, "skip (no regular file): %s\n", rel)
			r.st.Ignored++
			return nil
		}
		dst := filepath.Join(syncdir, cryptName(rel))
		expected[dst] = true
		if err := r.cryptFile(path, dst, rel, d); err != nil {
			return r.fail(rel, d, err)
		}
		return nil
	})
	if err != nil {
		return r.st, err
	}

	if opts.Prune {
		if r.st.Failed > 0 {
			fmt.Fprintln(out, "prune skipped because of errors")
		} else if err := r.prune(workdir, syncdir, expected); err != nil {
			return r.st, err
		}
	}
	return r.st, nil
}

func (r *run) cryptFile(src, dst, rel string, d fs.DirEntry) error {
	info, err := d.Info()
	if err != nil {
		return err
	}
	modTime := info.ModTime().Unix()

	if existing, err := os.Stat(dst); err == nil && existing.Size() == encryptedSize(info.Size()) {
		if h, err := readFileHeader(dst); err == nil && h.ModTime == modTime {
			r.st.Unchanged++
			return nil
		}
	}

	err = writeAtomic(dst, func(w io.Writer) error {
		f, err := os.Open(src)
		if err != nil {
			return err
		}
		defer f.Close()
		return encryptStream(w, f, r.key, filepath.ToSlash(rel), modTime)
	})
	if err != nil {
		return err
	}
	fmt.Fprintf(r.out, "crypt: %s\n", rel)
	r.st.Written++
	return nil
}

/*
prune removes encrypted files without source, leftover temp files and
directories that no longer exist in workdir and became empty
*/
func (r *run) prune(workdir, syncdir string, expected map[string]bool) error {
	var dirs []string
	err := filepath.WalkDir(syncdir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == syncdir {
			return nil
		}
		if d.IsDir() {
			dirs = append(dirs, path)
			return nil
		}
		_, isCrypt := plainName(d.Name())
		if (isCrypt && !expected[path]) || isTempFile(d.Name()) {
			if err := os.Remove(path); err != nil {
				return err
			}
			rel, _ := filepath.Rel(syncdir, path)
			fmt.Fprintf(r.out, "removed: %s\n", rel)
			r.st.Removed++
		}
		return nil
	})
	if err != nil {
		return err
	}

	// deepest directories first, so parents can become empty
	slices.Reverse(dirs)
	for _, dir := range dirs {
		rel, _ := filepath.Rel(syncdir, dir)
		if _, err := os.Stat(filepath.Join(workdir, rel)); err == nil {
			continue
		}
		if entries, err := os.ReadDir(dir); err == nil && len(entries) == 0 {
			if err := os.Remove(dir); err != nil {
				return err
			}
		}
	}
	return nil
}
