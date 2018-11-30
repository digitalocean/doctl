/*
Copyright 2018 The Doctl Authors All rights reserved.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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
