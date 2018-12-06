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
	"io"
	"net/http"
	"net/url"
)

var (
	bintrayHost    = "https://api.bintray.com"
	releaseSubject = "bryanliles"
	releseRepo     = "doit"
	releasePkg     = "doit"
)

type Bintray struct {
	Host string

	user, apikey       string
	subject, repo, pkg string
}

func NewBintray(user, apikey string) *Bintray {
	return &Bintray{
		Host:    bintrayHost,
		subject: releaseSubject,
		repo:    releseRepo,
		pkg:     releasePkg,

		user:   user,
		apikey: apikey,
	}
}

func (b *Bintray) Upload(r io.Reader, version, filePath string) error {
	u, err := url.Parse(bintrayHost)
	if err != nil {
		return err
	}

	u.User = url.UserPassword(b.user, b.apikey)
	v := u.Query()
	v.Set("publish", "1")
	v.Set("override", "1")
	u.RawQuery = v.Encode()

	u.Path = fmt.Sprintf("/content/%s/%s/%s/%s/%s",
		b.subject, b.repo, b.pkg, version, filePath)

	req, err := http.NewRequest("PUT", u.String(), r)
	if err != nil {
		return err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if sc := resp.StatusCode; sc != 201 {
		return fmt.Errorf("error uploading %q status: %d", filePath, sc)
	}

	return nil
}
