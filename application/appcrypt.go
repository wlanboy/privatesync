package application

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
)

func crypt(sync *Sync) {
	workdir := sync.Storage.Settings.WorkPath
	syncdir := sync.Storage.Settings.SyncPath
	secret := sync.Storage.Settings.Secret

    items, _ := ioutil.ReadDir(workdir)
    for _, item := range items {
        if item.IsDir() {
			testpath := filepath.Join(syncdir, item.Name())
			test,_ := os.Stat(testpath)
			if  test == nil {
				os.Mkdir(testpath, os.ModePerm)
			}
            subitems, _ := ioutil.ReadDir(item.Name())
            for _, subitem := range subitems {
                if !subitem.IsDir() {
					oldpath := filepath.Join(workdir, item.Name())
					newpath := filepath.Join(testpath, item.Name())

                    cryptFile(oldpath,newFileName(newpath),secret)
                }
            }
        } else {
			oldpath := filepath.Join(workdir, item.Name())
			newpath := filepath.Join(syncdir, item.Name())

			cryptFile(oldpath,newFileName(newpath),secret)
        }
    }
}

func newFileName(oldfilename string) string {
	ext := path.Ext(oldfilename)
	outfile := oldfilename[0:len(oldfilename)-len(ext)] + ".crypt"
	return outfile
}

func cryptFile(infile string, outfile string, secret string) {
	f, _ := os.Create(outfile)
	defer f.Close()
	data, _ := ioutil.ReadFile(infile)
	f.Write(encryptData(data, secret))
}

func encryptData(data []byte, secret string) []byte {
	block, _ := aes.NewCipher([]byte(createHash(secret)))
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		panic(err.Error())
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		panic(err.Error())
	}
	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return ciphertext
}