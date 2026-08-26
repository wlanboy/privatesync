package application

import (
	"crypto/aes"
	"crypto/cipher"
	"fmt"
	"os"
	"path"
	"path/filepath"
)

func decrypt(sync *Sync) error {
	workdir := sync.Storage.Settings.WorkPath
	syncdir := sync.Storage.Settings.SyncPath

	key, err := deriveKey(sync.Storage.PlainSecret, sync.Storage.Settings.Salt)
	if err != nil {
		return err
	}

	items, err := os.ReadDir(syncdir)
	if err != nil {
		return err
	}
	for _, item := range items {
		if item.IsDir() {
			srcdir := filepath.Join(syncdir, item.Name())
			dstdir := filepath.Join(workdir, item.Name())

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

					if err := decryptFile(oldpath, oldFileName(newpath), key); err != nil {
						return err
					}
				}
			}
		} else {
			oldpath := filepath.Join(syncdir, item.Name())
			newpath := filepath.Join(workdir, item.Name())

			if err := decryptFile(oldpath, oldFileName(newpath), key); err != nil {
				return err
			}
		}
	}
	return nil
}

func oldFileName(oldfilename string) string {
	withoutCrypt := oldfilename[:len(oldfilename)-len(".crypt")]
	innerExt := path.Ext(withoutCrypt)
	return withoutCrypt[:len(withoutCrypt)-len(innerExt)-1] + innerExt
}

func decryptFile(infile string, outfile string, key []byte) error {
	data, err := os.ReadFile(infile)
	if err != nil {
		return err
	}
	plaintext, err := decryptData(data, key)
	if err != nil {
		return err
	}
	if err := os.WriteFile(outfile, plaintext, 0640); err != nil {
		return err
	}
	fmt.Println("*")
	return nil
}

func decryptData(data []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := gcm.NonceSize()
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}
