package extract

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Benign archive tests – verify normal extraction still works end-to-end
// ---------------------------------------------------------------------------

func TestExtractTarGz(t *testing.T) {
	tmp := t.TempDir()
	out := t.TempDir()

	files := []string{"test.txt", "test/test.txt"}
	testTarGz := setUpTarGz(t, tmp, files)
	err := Extract(testTarGz, out)
	require.NoError(t, err, "error extracting archive")

	for _, f := range files {
		path := filepath.Join(out, f)
		_, err := os.Stat(path)
		require.NoError(t, err, "expected files not found")
	}
}

func TestExtractZip(t *testing.T) {
	tmp := t.TempDir()
	out := t.TempDir()

	files := []string{"test.txt", "test/test.txt"}
	testZip := setUpZip(t, tmp, files)
	err := Extract(testZip, out)
	require.NoError(t, err, "error extracting archive")

	for _, f := range files {
		path := filepath.Join(out, f)
		_, err := os.Stat(path)
		require.NoError(t, err, "expected files not found")
	}
}

func TestExtractTarGzWithContent(t *testing.T) {
	out := t.TempDir()
	archive := buildTarGz(t, []tarEntry{
		{typeflag: tar.TypeDir, name: "myapp/", mode: 0755},
		{typeflag: tar.TypeDir, name: "myapp/bin/", mode: 0755},
		{typeflag: tar.TypeReg, name: "myapp/README.md", content: []byte("# My App\nversion 1.0\n"), mode: 0644},
		{typeflag: tar.TypeReg, name: "myapp/bin/run.sh", content: []byte("#!/bin/sh\necho hello\n"), mode: 0755},
		{typeflag: tar.TypeReg, name: "myapp/config.json", content: []byte(`{"port":8080}`), mode: 0644},
	})
	err := Extract(archive, out)
	require.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(out, "myapp/README.md"))
	require.NoError(t, err)
	require.Equal(t, "# My App\nversion 1.0\n", string(content))

	content, err = os.ReadFile(filepath.Join(out, "myapp/bin/run.sh"))
	require.NoError(t, err)
	require.Equal(t, "#!/bin/sh\necho hello\n", string(content))

	info, err := os.Stat(filepath.Join(out, "myapp/bin/run.sh"))
	require.NoError(t, err)
	require.NotZero(t, info.Mode()&0111, "run.sh should be executable")

	content, err = os.ReadFile(filepath.Join(out, "myapp/config.json"))
	require.NoError(t, err)
	require.Equal(t, `{"port":8080}`, string(content))
}

func TestExtractZipWithContent(t *testing.T) {
	out := t.TempDir()
	archive := buildZip(t, []zipEntry{
		{name: "data/", isDir: true},
		{name: "data/hello.txt", content: []byte("hello world")},
		{name: "data/nested/deep.txt", content: []byte("deep content")},
	})
	err := Extract(archive, out)
	require.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(out, "data/hello.txt"))
	require.NoError(t, err)
	require.Equal(t, "hello world", string(content))

	content, err = os.ReadFile(filepath.Join(out, "data/nested/deep.txt"))
	require.NoError(t, err)
	require.Equal(t, "deep content", string(content))
}

func TestExtractTarGzValidSymlink(t *testing.T) {
	out := t.TempDir()
	archive := buildTarGz(t, []tarEntry{
		{typeflag: tar.TypeReg, name: "real.txt", content: []byte("target content"), mode: 0644},
		{typeflag: tar.TypeSymlink, name: "link.txt", linkname: "real.txt"},
	})
	err := Extract(archive, out)
	require.NoError(t, err)

	target, err := os.Readlink(filepath.Join(out, "link.txt"))
	require.NoError(t, err)
	require.Equal(t, "real.txt", target)

	content, err := os.ReadFile(filepath.Join(out, "link.txt"))
	require.NoError(t, err)
	require.Equal(t, "target content", string(content))
}

func TestExtractTarGzValidHardlink(t *testing.T) {
	out := t.TempDir()

	realFile := filepath.Join(out, "original.txt")
	err := os.WriteFile(realFile, []byte("hardlink target"), 0644)
	require.NoError(t, err)

	archive := buildTarGz(t, []tarEntry{
		{typeflag: tar.TypeLink, name: "linked.txt", linkname: realFile},
	})
	err = Extract(archive, out)
	require.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(out, "linked.txt"))
	require.NoError(t, err)
	require.Equal(t, "hardlink target", string(content))
}

func TestExtractTarGzValidNestedSymlink(t *testing.T) {
	out := t.TempDir()
	archive := buildTarGz(t, []tarEntry{
		{typeflag: tar.TypeDir, name: "subdir/", mode: 0755},
		{typeflag: tar.TypeReg, name: "subdir/target.txt", content: []byte("nested"), mode: 0644},
		{typeflag: tar.TypeSymlink, name: "subdir/alias.txt", linkname: "target.txt"},
	})
	err := Extract(archive, out)
	require.NoError(t, err)

	resolved, err := os.Readlink(filepath.Join(out, "subdir/alias.txt"))
	require.NoError(t, err)
	require.Equal(t, "target.txt", resolved)

	content, err := os.ReadFile(filepath.Join(out, "subdir/alias.txt"))
	require.NoError(t, err)
	require.Equal(t, "nested", string(content))
}

// ---------------------------------------------------------------------------
// Malicious archive tests – verify attacks are blocked
// ---------------------------------------------------------------------------

func TestExtractTarGzHardlinkEscape(t *testing.T) {
	tmp := t.TempDir()
	out := t.TempDir()

	outsideFile := filepath.Join(tmp, "outside.txt")
	require.NoError(t, os.WriteFile(outsideFile, []byte("original"), 0644))

	archive := buildTarGz(t, []tarEntry{
		{typeflag: tar.TypeLink, name: "link.txt", linkname: outsideFile},
	})
	err := Extract(archive, out)
	require.Error(t, err)
	require.Contains(t, err.Error(), "illegal link target")

	content, err := os.ReadFile(outsideFile)
	require.NoError(t, err)
	require.Equal(t, "original", string(content), "outside file must not be modified")
}

func TestExtractTarGzHardlinkThenOverwrite(t *testing.T) {
	tmp := t.TempDir()
	out := t.TempDir()

	victim := filepath.Join(tmp, "victim.txt")
	require.NoError(t, os.WriteFile(victim, []byte("sensitive data"), 0644))

	archive := buildTarGz(t, []tarEntry{
		{typeflag: tar.TypeLink, name: "innocent.txt", linkname: victim},
		{typeflag: tar.TypeReg, name: "innocent.txt", content: []byte("pwned"), mode: 0644},
	})
	err := Extract(archive, out)
	require.Error(t, err)
	require.Contains(t, err.Error(), "illegal link target")

	content, err := os.ReadFile(victim)
	require.NoError(t, err)
	require.Equal(t, "sensitive data", string(content), "victim file must survive the full attack chain")
}

func TestExtractTarGzSymlinkAbsoluteEscape(t *testing.T) {
	out := t.TempDir()
	archive := buildTarGz(t, []tarEntry{
		{typeflag: tar.TypeSymlink, name: "escape", linkname: "/etc"},
	})
	err := Extract(archive, out)
	require.Error(t, err)
	require.Contains(t, err.Error(), "illegal link target")

	_, statErr := os.Lstat(filepath.Join(out, "escape"))
	require.True(t, os.IsNotExist(statErr), "symlink must not be created")
}

func TestExtractTarGzSymlinkRelativeEscape(t *testing.T) {
	out := t.TempDir()
	archive := buildTarGz(t, []tarEntry{
		{typeflag: tar.TypeSymlink, name: "escape", linkname: "../../etc/passwd"},
	})
	err := Extract(archive, out)
	require.Error(t, err)
	require.Contains(t, err.Error(), "illegal link target")

	_, statErr := os.Lstat(filepath.Join(out, "escape"))
	require.True(t, os.IsNotExist(statErr), "symlink must not be created")
}

func TestExtractTarGzSymlinkDirThenWriteThrough(t *testing.T) {
	tmp := t.TempDir()
	out := t.TempDir()

	victimDir := filepath.Join(tmp, "victimdir")
	require.NoError(t, os.MkdirAll(victimDir, 0755))
	victim := filepath.Join(victimDir, "secret.conf")
	require.NoError(t, os.WriteFile(victim, []byte("secret"), 0644))

	archive := buildTarGz(t, []tarEntry{
		{typeflag: tar.TypeSymlink, name: "jailbreak", linkname: victimDir},
		{typeflag: tar.TypeReg, name: "jailbreak/secret.conf", content: []byte("overwritten"), mode: 0644},
	})
	err := Extract(archive, out)
	require.Error(t, err)
	require.Contains(t, err.Error(), "illegal link target")

	content, err := os.ReadFile(victim)
	require.NoError(t, err)
	require.Equal(t, "secret", string(content), "file behind symlinked dir must not be modified")
}

func TestExtractTarGzPathTraversal(t *testing.T) {
	out := t.TempDir()
	archive := buildTarGz(t, []tarEntry{
		{typeflag: tar.TypeReg, name: "../../../tmp/pwned.txt", content: []byte("bad"), mode: 0644},
	})
	err := Extract(archive, out)
	require.Error(t, err)
	require.Contains(t, err.Error(), "illegal file path")
}

func TestExtractZipPathTraversal(t *testing.T) {
	out := t.TempDir()
	archive := buildZip(t, []zipEntry{
		{name: "../../../tmp/pwned.txt", content: []byte("bad")},
	})
	err := Extract(archive, out)
	require.Error(t, err)
	require.Contains(t, err.Error(), "illegal file path")
}

// ---------------------------------------------------------------------------
// Helpers – tar
// ---------------------------------------------------------------------------

type tarEntry struct {
	typeflag byte
	name     string
	linkname string
	content  []byte
	mode     int64
}

func buildTarGz(t *testing.T, entries []tarEntry) string {
	t.Helper()
	archivePath := filepath.Join(t.TempDir(), "test.tar.gz")
	f, err := os.Create(archivePath)
	require.NoError(t, err)
	defer f.Close()

	gw := gzip.NewWriter(f)
	defer gw.Close()
	tw := tar.NewWriter(gw)
	defer tw.Close()

	for _, e := range entries {
		hdr := &tar.Header{
			Typeflag: e.typeflag,
			Name:     e.name,
			Linkname: e.linkname,
			Mode:     e.mode,
			Size:     int64(len(e.content)),
		}
		require.NoError(t, tw.WriteHeader(hdr))
		if len(e.content) > 0 {
			_, err := tw.Write(e.content)
			require.NoError(t, err)
		}
	}
	return archivePath
}

// ---------------------------------------------------------------------------
// Helpers – zip
// ---------------------------------------------------------------------------

type zipEntry struct {
	name    string
	isDir   bool
	content []byte
}

func buildZip(t *testing.T, entries []zipEntry) string {
	t.Helper()
	archivePath := filepath.Join(t.TempDir(), "test.zip")
	f, err := os.Create(archivePath)
	require.NoError(t, err)
	defer f.Close()

	zw := zip.NewWriter(f)
	defer zw.Close()

	for _, e := range entries {
		hdr := &zip.FileHeader{Name: e.name}
		if e.isDir {
			hdr.SetMode(0755 | os.ModeDir)
		} else {
			hdr.SetMode(0644)
		}
		w, err := zw.CreateHeader(hdr)
		require.NoError(t, err)
		if len(e.content) > 0 {
			_, err = w.Write(e.content)
			require.NoError(t, err)
		}
	}
	return archivePath
}

func setUpTarGz(t *testing.T, tmpDir string, files []string) string {
	tarballName := fmt.Sprintf("%s.tar.gz", uuid.New())
	tarballPath := filepath.Join(tmpDir, tarballName)
	tarball, err := os.Create(tarballPath)
	require.NoError(t, err, "error creating tar file")
	defer tarball.Close()

	gzipWriter := gzip.NewWriter(tarball)
	defer gzipWriter.Close()
	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	for _, f := range files {
		path := filepath.Join(tmpDir, f)
		if _, err := os.Stat(filepath.Dir(path)); os.IsNotExist(err) {
			os.MkdirAll(filepath.Dir(path), 0700)
		}

		file, err := os.Create(path)
		require.NoError(t, err, "error creating test file")
		defer file.Close()

		info, err := file.Stat()
		require.NoError(t, err, "error getting file info")

		header := &tar.Header{
			Name:    f,
			Size:    info.Size(),
			Mode:    int64(info.Mode()),
			ModTime: info.ModTime(),
		}
		err = tarWriter.WriteHeader(header)
		require.NoError(t, err, "error writing header")

		_, err = io.Copy(tarWriter, file)
		require.NoError(t, err, "error writing tar")
	}

	return tarballPath
}

func setUpZip(t *testing.T, tmpDir string, files []string) string {
	zipName := fmt.Sprintf("%s.zip", uuid.New())
	zipPath := filepath.Join(tmpDir, zipName)
	zipFile, err := os.Create(zipPath)
	require.NoError(t, err, "error creating tar file")
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	for _, f := range files {
		path := filepath.Join(tmpDir, f)
		if _, err := os.Stat(filepath.Dir(path)); os.IsNotExist(err) {
			os.MkdirAll(filepath.Dir(path), 0700)
		}

		file, err := os.Create(path)
		require.NoError(t, err, "error creating test file")
		defer file.Close()

		info, err := file.Stat()
		require.NoError(t, err)

		header, err := zip.FileInfoHeader(info)
		require.NoError(t, err)

		header.Name = f

		writer, err := zipWriter.CreateHeader(header)
		require.NoError(t, err, "error writing zip")

		_, err = io.Copy(writer, file)

		require.NoError(t, err, "error writing file to zip")
	}

	return zipPath
}
