package install

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"

	"github.com/mitchellh/ioprogress"
)

func Download(localPath, remoteURL string) (*os.File, error) {
	f, err := os.Create(localPath)
	if err != nil {
		return nil, err
	}

	resp, err := http.Get(remoteURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	size, _ := strconv.Atoi(resp.Header.Get("Content-Length"))

	progressR := &ioprogress.Reader{
		Reader: resp.Body,
		Size:   int64(size),
	}

	_, err = io.Copy(f, progressR)
	if err != nil {
		return nil, err
	}

	err = f.Close()
	if err != nil {
		return nil, err
	}

	return os.Open(localPath)
}

func Validate(f, cs io.Reader) error {
	h := sha256.New()

	if _, err := io.Copy(h, f); err != nil {
		return err
	}

	scanner := bufio.NewScanner(cs)
	scanner.Split(bufio.ScanWords)

	var words []string
	for scanner.Scan() {
		words = append(words, scanner.Text())
	}

	if sum, wantedSum := hex.EncodeToString(h.Sum(nil)), words[0]; sum != wantedSum {
		return fmt.Errorf("invalid checksum: %s != %s", sum, wantedSum)
	}

	return nil
}

func URL(ver, filename string) string {
	u := url.URL{
		Scheme: "http",
		Host:   "github.com",
		Path:   fmt.Sprintf("/bryanl/doit/releases/download/v%s/%s", ver, filename),
	}

	return u.String()
}
