package godo

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// SandboxFileMaxBytes is the largest file or archive that UploadFile and
// DownloadFile transfer in one call: 50 MiB.
const SandboxFileMaxBytes = 50 << 20

// sandboxFileRetryDelays are the waits before each retry of an upload or
// download that the sandbox answered with 503 (too many concurrent
// transfers). A Retry-After header overrides the wait. Tests shorten them.
var sandboxFileRetryDelays = []time.Duration{500 * time.Millisecond, time.Second, 2 * time.Second}

// sandboxFileMaxRetryAfter caps how long a Retry-After header can make a
// transfer wait before it retries.
const sandboxFileMaxRetryAfter = 30 * time.Second

// SandboxFileEntryType is the type of a file system entry in a sandbox.
type SandboxFileEntryType string

const (
	// SandboxFileEntryTypeUnspecified is the zero type.
	SandboxFileEntryTypeUnspecified SandboxFileEntryType = "FILE_ENTRY_TYPE_UNSPECIFIED"
	// SandboxFileEntryTypeFile is a regular file.
	SandboxFileEntryTypeFile SandboxFileEntryType = "FILE_ENTRY_TYPE_FILE"
	// SandboxFileEntryTypeDirectory is a directory.
	SandboxFileEntryTypeDirectory SandboxFileEntryType = "FILE_ENTRY_TYPE_DIRECTORY"
	// SandboxFileEntryTypeSymlink is a symbolic link. See SymlinkTarget.
	SandboxFileEntryTypeSymlink SandboxFileEntryType = "FILE_ENTRY_TYPE_SYMLINK"
)

// SandboxFileEntry is a file, directory or symlink in a sandbox workspace.
type SandboxFileEntry struct {
	// Name is the base name of the entry.
	Name string               `json:"name"`
	Type SandboxFileEntryType `json:"type"`
	// Path is the absolute path of the entry in the sandbox.
	Path string `json:"path"`
	// Size is the size in bytes. The API encodes it as a string.
	Size int64 `json:"size,omitempty,string"`
	// Mode is the Unix file mode bits, such as 420 (0644).
	Mode int64 `json:"mode,omitempty"`
	// Permissions are the permissions in ls form, such as "-rw-r--r--".
	Permissions string `json:"permissions,omitempty"`
	// Owner is the user that owns the entry.
	Owner string `json:"owner,omitempty"`
	// Group is the group that owns the entry.
	Group string `json:"group,omitempty"`
	// ModifiedTime is when the entry was last modified.
	ModifiedTime Timestamp `json:"modified_time"`
	// SymlinkTarget is the path a symlink points to. It is empty unless Type
	// is SandboxFileEntryTypeSymlink.
	SymlinkTarget string `json:"symlink_target,omitempty"`
}

// SandboxFileUploadRequest is the request of UploadFile.
type SandboxFileUploadRequest struct {
	// Path is where to write the file or, when IsArchive is set, the directory
	// to extract the archive into. It is required.
	Path string
	// Body is the file content, or a tar archive when IsArchive is set. It is
	// read into memory, so a 503 can be retried, and must not be larger than
	// SandboxFileMaxBytes. A nil Body writes an empty file.
	Body io.Reader
	// IsArchive means Body is a tar archive to extract at Path.
	IsArchive bool
	// SHA256 is the lowercase hex SHA-256 of Body. When set, it is sent as the
	// X-Content-Sha256 header and the sandbox checks it after the write.
	SHA256 string
}

// SandboxFileUpload is the result of UploadFile.
type SandboxFileUpload struct {
	// Path is the absolute path the file was written to.
	Path string `json:"path"`
	// BytesWritten is the number of bytes written. The API encodes it as a
	// string.
	BytesWritten int64 `json:"bytes_written,omitempty,string"`
}

// SandboxFileDownloadRequest is the request of DownloadFile.
type SandboxFileDownloadRequest struct {
	// Path is the file or directory to download. It is required.
	Path string
	// AsArchive downloads Path as a tar archive. It is required to download a
	// directory.
	AsArchive bool
}

// SandboxDirectoryCreated is the result of MakeDirectory.
type SandboxDirectoryCreated struct {
	// Created is false when the directory already existed.
	Created bool `json:"created,omitempty"`
	// Entry describes the directory, whether or not it was created.
	Entry SandboxFileEntry `json:"entry"`
}

type sandboxFileEntryRoot struct {
	Entry *SandboxFileEntry `json:"entry"`
}

func sandboxFilesPath(sandboxID, action string, query url.Values) string {
	p := fmt.Sprintf("%s/%s/files/%s", sandboxesBasePath, sandboxID, action)
	if len(query) > 0 {
		p += "?" + query.Encode()
	}
	return p
}

// UploadFile writes a file to a sandbox, or extracts a tar archive into a
// directory when IsArchive is set. Missing parent directories are created. A
// 503 (too many concurrent transfers) is retried up to 3 times unless the
// client was built with RetryConfig, which already retries it.
func (s *SandboxesServiceOp) UploadFile(ctx context.Context, sandboxID string, upload *SandboxFileUploadRequest) (*SandboxFileUpload, *Response, error) {
	if sandboxID == "" {
		return nil, nil, errors.New("sandboxes: sandbox id is required")
	}
	if upload == nil || upload.Path == "" {
		return nil, nil, errors.New("sandboxes: path is required")
	}
	var data []byte
	if upload.Body != nil {
		var err error
		data, err = io.ReadAll(io.LimitReader(upload.Body, SandboxFileMaxBytes+1))
		if err != nil {
			return nil, nil, fmt.Errorf("sandboxes: reading the upload body: %w", err)
		}
		if len(data) > SandboxFileMaxBytes {
			return nil, nil, fmt.Errorf("sandboxes: the upload body is larger than %d bytes", SandboxFileMaxBytes)
		}
	}
	q := url.Values{}
	q.Set("path", upload.Path)
	if upload.IsArchive {
		q.Set("is_archive", "true")
	}
	path := sandboxFilesPath(sandboxID, "upload", q)

	var out *SandboxFileUpload
	resp, err := s.transferWithRetry(ctx, func() (*Response, error) {
		req, err := s.client.NewRequest(ctx, http.MethodPost, path, nil)
		if err != nil {
			return nil, err
		}
		req.Body = io.NopCloser(bytes.NewReader(data))
		req.ContentLength = int64(len(data))
		req.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(data)), nil }
		req.Header.Set("Content-Type", "application/octet-stream")
		if upload.SHA256 != "" {
			req.Header.Set(workspaceContentSHA256Header, upload.SHA256)
		}
		out = new(SandboxFileUpload)
		return s.client.Do(ctx, req, out)
	})
	if err != nil {
		return nil, resp, err
	}
	return out, resp, nil
}

// DownloadFile reads a file from a sandbox, or a file or directory as a tar
// archive when AsArchive is set. It strips the integrity footer the API
// appends and checks the payload against its SHA-256; a missing footer or a
// mismatch is an error. A 503 (too many concurrent transfers) is retried up to
// 3 times unless the client was built with RetryConfig, which already retries
// it. The response headers X-Workspace-Is-Archive and X-Workspace-Size-Bytes
// are on the returned Response.
func (s *SandboxesServiceOp) DownloadFile(ctx context.Context, sandboxID string, download *SandboxFileDownloadRequest) ([]byte, *Response, error) {
	if sandboxID == "" {
		return nil, nil, errors.New("sandboxes: sandbox id is required")
	}
	if download == nil || download.Path == "" {
		return nil, nil, errors.New("sandboxes: path is required")
	}
	q := url.Values{}
	q.Set("path", download.Path)
	if download.AsArchive {
		q.Set("as_archive", "true")
	}
	path := sandboxFilesPath(sandboxID, "download", q)

	var buf *bytes.Buffer
	resp, err := s.transferWithRetry(ctx, func() (*Response, error) {
		req, err := s.client.NewRequest(ctx, http.MethodGet, path, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Accept", "application/octet-stream")
		buf = new(bytes.Buffer)
		return s.client.Do(ctx, req, buf)
	})
	if err != nil {
		return nil, resp, err
	}
	payload, err := verifySandboxDownload(buf.Bytes())
	if err != nil {
		return nil, resp, err
	}
	return payload, resp, nil
}

// verifySandboxDownload strips the DOWSSHA1 integrity footer from a download
// body and checks the payload against the SHA-256 in it.
func verifySandboxDownload(body []byte) ([]byte, error) {
	if len(body) < workspaceDownloadFooterLen {
		return nil, errors.New("sandboxes: the download is missing its integrity footer")
	}
	split := len(body) - workspaceDownloadFooterLen
	want, ok := parseWorkspaceDownloadFooter(body[split:])
	if !ok {
		return nil, errors.New("sandboxes: the download has an invalid integrity footer")
	}
	payload := body[:split]
	sum := sha256.Sum256(payload)
	if got := hex.EncodeToString(sum[:]); got != strings.ToLower(want) {
		return nil, fmt.Errorf("sandboxes: the download failed its integrity check: SHA-256 is %s, the footer says %s", got, want)
	}
	return payload, nil
}

// transferWithRetry runs a file transfer and retries it when the sandbox
// answers 503. It does not retry when the client has RetryConfig set: the
// go-retryablehttp client then already retries 503, honoring Retry-After.
func (s *SandboxesServiceOp) transferWithRetry(ctx context.Context, do func() (*Response, error)) (*Response, error) {
	for attempt := 0; ; attempt++ {
		resp, err := do()
		if err == nil || s.client.RetryConfig.RetryMax > 0 || attempt >= len(sandboxFileRetryDelays) ||
			resp == nil || resp.StatusCode != http.StatusServiceUnavailable {
			return resp, err
		}
		wait := sandboxFileRetryDelays[attempt]
		if d, ok := parseSandboxRetryAfter(resp.Header.Get("Retry-After")); ok {
			wait = d
		}
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return resp, ctx.Err()
		case <-timer.C:
		}
	}
}

// parseSandboxRetryAfter reads a Retry-After header in seconds or as an HTTP
// date, capped at sandboxFileMaxRetryAfter.
func parseSandboxRetryAfter(v string) (time.Duration, bool) {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0, false
	}
	var d time.Duration
	if secs, err := strconv.ParseInt(v, 10, 64); err == nil {
		if secs < 0 {
			return 0, false
		}
		d = time.Duration(secs) * time.Second
	} else if at, err := http.ParseTime(v); err == nil {
		d = time.Until(at)
		if d < 0 {
			d = 0
		}
	} else {
		return 0, false
	}
	if d > sandboxFileMaxRetryAfter {
		d = sandboxFileMaxRetryAfter
	}
	return d, true
}

// ListFiles lists the entries of a directory in a sandbox. depth sets how many
// levels to list: 0 or 1 lists the directory's own entries, 2 also lists the
// entries of its subdirectories, and so on. Symlinks are listed, not followed.
func (s *SandboxesServiceOp) ListFiles(ctx context.Context, sandboxID, path string, depth int) ([]SandboxFileEntry, *Response, error) {
	if sandboxID == "" {
		return nil, nil, errors.New("sandboxes: sandbox id is required")
	}
	if path == "" {
		return nil, nil, errors.New("sandboxes: path is required")
	}
	if depth < 0 {
		return nil, nil, errors.New("sandboxes: depth must not be negative")
	}
	q := url.Values{}
	q.Set("path", path)
	if depth > 0 {
		q.Set("depth", strconv.Itoa(depth))
	}
	req, err := s.client.NewRequest(ctx, http.MethodGet, sandboxFilesPath(sandboxID, "list", q), nil)
	if err != nil {
		return nil, nil, err
	}
	root := new(struct {
		Entries []SandboxFileEntry `json:"entries"`
	})
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}
	return root.Entries, resp, nil
}

// GetFileInfo retrieves a file, directory or symlink in a sandbox. A missing
// path returns a 404 *ErrorResponse; FileExists tests for a path without one.
func (s *SandboxesServiceOp) GetFileInfo(ctx context.Context, sandboxID, path string) (*SandboxFileEntry, *Response, error) {
	if sandboxID == "" {
		return nil, nil, errors.New("sandboxes: sandbox id is required")
	}
	if path == "" {
		return nil, nil, errors.New("sandboxes: path is required")
	}
	q := url.Values{}
	q.Set("path", path)
	req, err := s.client.NewRequest(ctx, http.MethodGet, sandboxFilesPath(sandboxID, "info", q), nil)
	if err != nil {
		return nil, nil, err
	}
	return s.fileEntry(ctx, req)
}

// FileExists reports whether a file, directory or symlink exists at a path in
// a sandbox. A 404 *ErrorResponse means the sandbox was not found.
func (s *SandboxesServiceOp) FileExists(ctx context.Context, sandboxID, path string) (bool, *Response, error) {
	if sandboxID == "" {
		return false, nil, errors.New("sandboxes: sandbox id is required")
	}
	if path == "" {
		return false, nil, errors.New("sandboxes: path is required")
	}
	q := url.Values{}
	q.Set("path", path)
	req, err := s.client.NewRequest(ctx, http.MethodGet, sandboxFilesPath(sandboxID, "exists", q), nil)
	if err != nil {
		return false, nil, err
	}
	root := new(struct {
		Exists bool `json:"exists"`
	})
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return false, resp, err
	}
	return root.Exists, resp, nil
}

// MakeDirectory creates a directory in a sandbox, with any missing parents.
// When the directory already exists nothing changes and Created is false.
func (s *SandboxesServiceOp) MakeDirectory(ctx context.Context, sandboxID, path string) (*SandboxDirectoryCreated, *Response, error) {
	if sandboxID == "" {
		return nil, nil, errors.New("sandboxes: sandbox id is required")
	}
	if path == "" {
		return nil, nil, errors.New("sandboxes: path is required")
	}
	req, err := s.client.NewRequest(ctx, http.MethodPost, sandboxFilesPath(sandboxID, "make-directory", nil),
		map[string]string{"path": path})
	if err != nil {
		return nil, nil, err
	}
	root := new(struct {
		Created bool              `json:"created"`
		Entry   *SandboxFileEntry `json:"entry"`
	})
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}
	if root.Entry == nil {
		return nil, resp, errors.New("sandboxes: the response did not include the directory")
	}
	return &SandboxDirectoryCreated{Created: root.Created, Entry: *root.Entry}, resp, nil
}

// MoveFile moves or renames a file, directory or symlink in a sandbox and
// returns the entry at its new path. Missing parents of destination are
// created, and a file already at destination is replaced.
func (s *SandboxesServiceOp) MoveFile(ctx context.Context, sandboxID, source, destination string) (*SandboxFileEntry, *Response, error) {
	if sandboxID == "" {
		return nil, nil, errors.New("sandboxes: sandbox id is required")
	}
	if source == "" || destination == "" {
		return nil, nil, errors.New("sandboxes: source and destination are required")
	}
	req, err := s.client.NewRequest(ctx, http.MethodPost, sandboxFilesPath(sandboxID, "move", nil),
		map[string]string{"source": source, "destination": destination})
	if err != nil {
		return nil, nil, err
	}
	return s.fileEntry(ctx, req)
}

// RemoveFile removes a file, or a directory with everything in it, from a
// sandbox. Removing a path that does not exist succeeds.
func (s *SandboxesServiceOp) RemoveFile(ctx context.Context, sandboxID, path string) (*Response, error) {
	if sandboxID == "" {
		return nil, errors.New("sandboxes: sandbox id is required")
	}
	if path == "" {
		return nil, errors.New("sandboxes: path is required")
	}
	req, err := s.client.NewRequest(ctx, http.MethodPost, sandboxFilesPath(sandboxID, "remove", nil),
		map[string]string{"path": path})
	if err != nil {
		return nil, err
	}
	return s.client.Do(ctx, req, nil)
}

func (s *SandboxesServiceOp) fileEntry(ctx context.Context, req *http.Request) (*SandboxFileEntry, *Response, error) {
	root := new(sandboxFileEntryRoot)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}
	if root.Entry == nil {
		return nil, resp, errors.New("sandboxes: the response did not include the entry")
	}
	return root.Entry, resp, nil
}
