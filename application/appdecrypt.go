package application

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
)

func decrypt(sync *Sync) {
    workdir := sync.Storage.Settings.WorkPath
    syncdir := sync.Storage.Settings.SyncPath

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

                    cryptFile(oldpath,oldFileName(newpath))
                }
            }
        } else {

			oldpath := filepath.Join(syncdir, item.Name())
			newpath := filepath.Join(workdir, item.Name())

			cryptFile(oldpath,oldFileName(newpath))
        }
    }    
}

func oldFileName(oldfilename string) string {
	ext := path.Ext(oldfilename)
	outfile := oldfilename[0:len(oldfilename)-len(ext)]
	return outfile
}

func decryptFile(infile string, outfile string) {

}