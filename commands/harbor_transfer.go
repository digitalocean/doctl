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

package commands

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// harborHTTPClient talks to presigned Spaces URLs. It has no timeout so large
// transfers are not cut off; each request is bounded by the URL's expiry.
var harborHTTPClient = &http.Client{}

// harborTransferAttempts is how many times one upload or download is tried.
var harborTransferAttempts = 3

// harborRetryBackoff is the wait before the first retry; it doubles after.
var harborRetryBackoff = time.Second

// harborSleep is a variable so tests can wait without sleeping.
var harborSleep = time.Sleep

// harborMaxExtractBytes bounds what task pull will unpack, so a hostile
// archive cannot fill the disk.
const harborMaxExtractBytes = 1 << 30

// errHarborURLRejected marks a request refused by storage, usually because the
// presigned URL expired. Retrying the same URL cannot succeed.
var errHarborURLRejected = errors.New("storage rejected the request")

func harborRetry(what string, fn func() error) error {
	var lastErr error
	for attempt := 1; attempt <= harborTransferAttempts; attempt++ {
		if attempt > 1 {
			harborSleep(harborRetryBackoff << (attempt - 2))
		}
		if lastErr = fn(); lastErr == nil {
			return nil
		}
		if errors.Is(lastErr, errHarborURLRejected) {
			return fmt.Errorf("%s: %w", what, lastErr)
		}
	}
	return fmt.Errorf("%s failed after %d attempts: %w", what, harborTransferAttempts, lastErr)
}

func harborStatusError(resp *http.Response) error {
	if resp.StatusCode >= 400 && resp.StatusCode < 500 {
		return fmt.Errorf("%w with status %s; the link may have expired, so run the command again", errHarborURLRejected, resp.Status)
	}
	return fmt.Errorf("storage returned status %s", resp.Status)
}

// putHarborArchive uploads the archive at path to a presigned PUT URL.
func putHarborArchive(uploadURL, path string, size int64) error {
	return harborRetry("uploading the task archive", func() error {
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		req, err := http.NewRequest(http.MethodPut, uploadURL, f)
		if err != nil {
			return redactURLError(err)
		}
		req.ContentLength = size
		req.Header.Set("Content-Type", "application/gzip")

		resp, err := harborHTTPClient.Do(req)
		if err != nil {
			return redactURLError(err)
		}
		defer resp.Body.Close()
		io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))
		if resp.StatusCode < 200 || resp.StatusCode > 299 {
			return harborStatusError(resp)
		}
		return nil
	})
}

// downloadHarborFile streams a presigned URL to dest, checking the size and
// checksum when known. Bytes go to a temp file first so a failed download
// never leaves a partial file in place.
func downloadHarborFile(rawURL, dest string, wantSize int64, wantSHA256 string) (int64, error) {
	if rawURL == "" {
		return 0, fmt.Errorf("the API returned no download link for %s", filepath.Base(dest))
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return 0, err
	}
	var written int64
	err := harborRetry("downloading "+filepath.Base(dest), func() error {
		req, err := http.NewRequest(http.MethodGet, rawURL, nil)
		if err != nil {
			return redactURLError(err)
		}
		resp, err := harborHTTPClient.Do(req)
		if err != nil {
			return redactURLError(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode > 299 {
			return harborStatusError(resp)
		}
		written, err = writeHarborFile(resp.Body, dest, wantSize, wantSHA256)
		return err
	})
	return written, err
}

func writeHarborFile(body io.Reader, dest string, wantSize int64, wantSHA256 string) (int64, error) {
	tmp, err := os.CreateTemp(filepath.Dir(dest), ".doctl-download-*")
	if err != nil {
		return 0, err
	}
	tmpName := tmp.Name()

	h := sha256.New()
	written, copyErr := io.Copy(tmp, io.TeeReader(body, h))
	if err := errors.Join(copyErr, tmp.Close()); err != nil {
		os.Remove(tmpName)
		return 0, err
	}
	if wantSize > 0 && written != wantSize {
		os.Remove(tmpName)
		return 0, fmt.Errorf("received %d bytes, expected %d", written, wantSize)
	}
	if want := strings.TrimPrefix(wantSHA256, "sha256:"); want != "" {
		if got := hex.EncodeToString(h.Sum(nil)); !strings.EqualFold(got, want) {
			os.Remove(tmpName)
			return 0, fmt.Errorf("checksum mismatch: got %s, want %s", got, want)
		}
	}
	if err := os.Rename(tmpName, dest); err != nil {
		os.Remove(tmpName)
		return 0, err
	}
	return written, nil
}

// harborFileMatches reports whether dest already holds the expected file, so
// an interrupted download can be resumed without fetching it again.
func harborFileMatches(dest string, wantSize int64, wantSHA256 string) bool {
	info, err := os.Stat(dest)
	if err != nil || !info.Mode().IsRegular() {
		return false
	}
	if wantSize > 0 && info.Size() != wantSize {
		return false
	}
	want := strings.TrimPrefix(wantSHA256, "sha256:")
	if want == "" {
		return wantSize > 0
	}
	f, err := os.Open(dest)
	if err != nil {
		return false
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return false
	}
	return strings.EqualFold(hex.EncodeToString(h.Sum(nil)), want)
}

// harborSafePath resolves a server-supplied relative path inside dir and
// refuses any that would land outside it.
func harborSafePath(dir, name string) (string, error) {
	clean := filepath.Clean(filepath.FromSlash(strings.TrimSpace(name)))
	if clean == "." || clean == "" || filepath.IsAbs(clean) || filepath.VolumeName(clean) != "" ||
		clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("refusing to write a file with the unsafe path %q", name)
	}
	root, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	dest := filepath.Join(root, clean)
	if !strings.HasPrefix(dest, root+string(filepath.Separator)) {
		return "", fmt.Errorf("refusing to write a file with the unsafe path %q", name)
	}
	return dest, nil
}

// extractHarborArchive unpacks a task archive into dir and returns the sha256
// of each file, keyed by its path in the archive. Only regular files are
// accepted, which is all Harbor writes.
func extractHarborArchive(archive, dir string) (map[string]string, error) {
	f, err := os.Open(archive)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return nil, fmt.Errorf("reading the task archive: %w", err)
	}
	defer gz.Close()

	sums := map[string]string{}
	var total int64
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return sums, nil
		}
		if err != nil {
			return nil, fmt.Errorf("reading the task archive: %w", err)
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			continue
		case tar.TypeReg:
		default:
			return nil, fmt.Errorf("the task archive contains %q, which is not a regular file", hdr.Name)
		}
		if len(sums) >= harborMaxFiles {
			return nil, fmt.Errorf("the task archive has more than %d files", harborMaxFiles)
		}
		total += hdr.Size
		if hdr.Size < 0 || total > harborMaxExtractBytes {
			return nil, fmt.Errorf("the task archive unpacks to more than 1 GiB")
		}
		dest, err := harborSafePath(dir, hdr.Name)
		if err != nil {
			return nil, err
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return nil, err
		}
		mode := os.FileMode(0o644)
		if hdr.Mode&0o111 != 0 {
			mode = 0o755
		}
		out, err := os.OpenFile(dest, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
		if err != nil {
			return nil, err
		}
		h := sha256.New()
		_, copyErr := io.Copy(io.MultiWriter(out, h), io.LimitReader(tr, hdr.Size))
		if err := errors.Join(copyErr, out.Close()); err != nil {
			return nil, err
		}
		sums[filepath.ToSlash(filepath.Clean(filepath.FromSlash(hdr.Name)))] = hex.EncodeToString(h.Sum(nil))
	}
}

// redactURLError strips the query string from a URL in err. Presigned URLs
// carry their signature there, and it must not end up in logs or CI output.
func redactURLError(err error) error {
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return &url.Error{Op: urlErr.Op, URL: redactURL(urlErr.URL), Err: urlErr.Err}
	}
	return err
}

func redactURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return "<redacted>"
	}
	u.RawQuery = ""
	u.Fragment = ""
	u.User = nil
	return u.String()
}
