package application

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
)

func crypt(sync *Sync) {
	workdir := sync.Storage.Settings.WorkPath
	syncdir := sync.Storage.Settings.SyncPath

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

                    cryptFile(oldpath,newFileName(newpath))
                }
            }
        } else {
			oldpath := filepath.Join(workdir, item.Name())
			newpath := filepath.Join(syncdir, item.Name())

			cryptFile(oldpath,newFileName(newpath))
        }
    }
}

func newFileName(oldfilename string) string {
	ext := path.Ext(oldfilename)
	outfile := oldfilename[0:len(oldfilename)-len(ext)] + ".crypt"
	return outfile
}

func cryptFile(infile string, outfile string) {

}