package application

import (
	"crypto/aes"
	"crypto/cipher"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
)

func decrypt(sync *Sync) {
    workdir := sync.Storage.Settings.WorkPath
    syncdir := sync.Storage.Settings.SyncPath
    secret := sync.Storage.Settings.Secret

    items, _ := ioutil.ReadDir(syncdir)
    for _, item := range items {
        if item.IsDir() {
			testpath := filepath.Join(workdir, item.Name())
			test,_ := os.Stat(testpath)
			if  test == nil {
				os.Mkdir(testpath, os.ModePerm)
			}
            subitems, _ := ioutil.ReadDir(item.Name())
            for _, subitem := range subitems {
                if !subitem.IsDir() {
					oldpath := filepath.Join(syncdir, item.Name())
					newpath := filepath.Join(testpath, item.Name())

                    decryptFile(oldpath,oldFileName(newpath), secret)
                }
            }
        } else {

			oldpath := filepath.Join(syncdir, item.Name())
			newpath := filepath.Join(workdir, item.Name())

			decryptFile(oldpath,oldFileName(newpath), secret)
        }
    }    
}

func oldFileName(oldfilename string) string {
	ext := path.Ext(oldfilename)
	outfile := oldfilename[0:len(oldfilename)-len(ext)]
	return outfile
}

func decryptFile(infile string, outfile string, secret string) {
    f, _ := os.Create(outfile)
    defer f.Close()
    
	data, _ := ioutil.ReadFile(infile)
    f.Write(decryptData(data, secret))
}

func decryptData(data []byte, secret string) []byte {
	key := []byte(createHash(secret))
	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err.Error())
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		panic(err.Error())
	}
	nonceSize := gcm.NonceSize()
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		panic(err.Error())
	}
	return plaintext
}