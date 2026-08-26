package application

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
)

func crypt(sync *Sync) error {
	workdir := sync.Storage.Settings.WorkPath
	syncdir := sync.Storage.Settings.SyncPath

	key, err := deriveKey(sync.Storage.PlainSecret, sync.Storage.Settings.Salt)
	if err != nil {
		return err
	}

	items, err := os.ReadDir(workdir)
	if err != nil {
		return err
	}
	for _, item := range items {
		if item.IsDir() {
			srcdir := filepath.Join(workdir, item.Name())
			dstdir := filepath.Join(syncdir, item.Name())

			if _, err := os.Stat(dstdir); os.IsNotExist(err) {
				if err := os.Mkdir(dstdir, os.ModePerm); err != nil {
					return err
				}
			}

			subitems, err := os.ReadDir(srcdir)
			if err != nil {
				return err
			}
			for _, subitem := range subitems {
				if !subitem.IsDir() {
					oldpath := filepath.Join(srcdir, subitem.Name())
					newpath := filepath.Join(dstdir, subitem.Name())

					if err := cryptFile(oldpath, newFileName(newpath), key); err != nil {
						return err
					}
				}
			}
		} else {
			oldpath := filepath.Join(workdir, item.Name())
			newpath := filepath.Join(syncdir, item.Name())

			if err := cryptFile(oldpath, newFileName(newpath), key); err != nil {
				return err
			}
		}
	}
	return nil
}

func newFileName(oldfilename string) string {
	ext := path.Ext(oldfilename)
	outfile := oldfilename[0:len(oldfilename)-len(ext)] + "." + ext + ".crypt"
	return outfile
}

func cryptFile(infile string, outfile string, key []byte) error {
	data, err := os.ReadFile(infile)
	if err != nil {
		return err
	}
	ciphertext, err := encryptData(data, key)
	if err != nil {
		return err
	}
	if err := os.WriteFile(outfile, ciphertext, 0640); err != nil {
		return err
	}
	fmt.Println("*")
	return nil
}

func encryptData(data []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, data, nil), nil
}
