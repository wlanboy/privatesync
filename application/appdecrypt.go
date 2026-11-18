package application

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

/*decryptTree decrypts every .crypt file below syncdir into workdir, keeping the directory structure*/
func decryptTree(workdir, syncdir string, key []byte, opts options, out io.Writer) (stats, error) {
	if err := requireDir(syncdir); err != nil {
		return stats{}, err
	}
	if err := os.MkdirAll(workdir, dirPerm); err != nil {
		return stats{}, err
	}

	r := &run{key: key, opts: opts, out: out}
	err := filepath.WalkDir(syncdir, func(path string, d fs.DirEntry, err error) error {
		rel, relErr := filepath.Rel(syncdir, path)
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
			if err := os.MkdirAll(filepath.Join(workdir, rel), dirPerm); err != nil {
				return r.fail(rel, d, err)
			}
			return nil
		}
		if isTempFile(d.Name()) {
			return nil
		}
		plainRel, ok := plainName(rel)
		if !ok || !d.Type().IsRegular() {
			fmt.Fprintf(out, "skip (no .crypt file): %s\n", rel)
			r.st.Ignored++
			return nil
		}
		if err := r.decryptFile(path, filepath.Join(workdir, plainRel), plainRel, d); err != nil {
			return r.fail(rel, d, err)
		}
		return nil
	})
	return r.st, err
}

func (r *run) decryptFile(src, dst, plainRel string, d fs.DirEntry) error {
	info, err := d.Info()
	if err != nil {
		return err
	}
	h, err := readFileHeader(src)
	if err != nil {
		return err
	}

	local, err := os.Stat(dst)
	switch {
	case err == nil:
		if !local.Mode().IsRegular() {
			return errors.New("target exists and is no regular file")
		}
		localTime := local.ModTime().Unix()
		if localTime == h.ModTime && encryptedSize(local.Size()) == info.Size() {
			r.st.Unchanged++
			return nil
		}
		if localTime > h.ModTime && !r.opts.Force {
			fmt.Fprintf(r.out, "skip (local file is newer, use --force to overwrite): %s\n", plainRel)
			r.st.Conflicts++
			return nil
		}
	case !errors.Is(err, os.ErrNotExist):
		return err
	}

	err = writeAtomic(dst, func(w io.Writer) error {
		f, err := os.Open(src)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = decryptStream(w, f, r.key, filepath.ToSlash(plainRel))
		return err
	})
	if err != nil {
		return err
	}
	mtime := time.Unix(h.ModTime, 0)
	if err := os.Chtimes(dst, mtime, mtime); err != nil {
		return err
	}
	fmt.Fprintf(r.out, "decrypt: %s\n", plainRel)
	r.st.Written++
	return nil
}
