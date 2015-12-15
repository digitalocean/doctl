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
