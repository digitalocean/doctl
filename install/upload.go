package install

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

type UserInfo struct {
	User, Apikey string
}

func Upload(ui UserInfo, ver, buildPath string) error {
	bt := NewBintray(ui.User, ui.Apikey)

	fis, err := ioutil.ReadDir(buildPath)
	if err != nil {
		return err
	}

	for _, fi := range fis {
		if fi.IsDir() {
			continue
		}

		p := filepath.Join(buildPath, fi.Name())

		f, err := os.Open(p)
		if err != nil {
			return err
		}
		defer f.Close()

		fmt.Println("uploading", fi.Name())
		err = bt.Upload(f, ver, fi.Name())
		if err != nil {
			return err
		}
	}

	return nil
}
