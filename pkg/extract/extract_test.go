package extract

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestExtractTarGz(t *testing.T) {
	tmp, err := ioutil.TempDir("", "test-materials")
	require.NoError(t, err, "error creating tmp dir")
	out, err := ioutil.TempDir("", "output")
	require.NoError(t, err, "error creating out dir")
	defer func() {
		err := os.RemoveAll(tmp)
		require.NoError(t, err, "error cleaning tmp dir")
		err = os.RemoveAll(out)
		require.NoError(t, err, "error cleaning out dir")
	}()

	files := []string{"test.txt", "test/test.txt"}
	testTarGz := setUpTarGz(t, tmp, files)
	err = Extract(testTarGz, out)
	require.NoError(t, err, "error extracting archive")

	for _, f := range files {
		path := filepath.Join(out, f)
		_, err := os.Stat(path)
		require.NoError(t, err, "expected files not found")
	}
}

func TestExtractZip(t *testing.T) {
	tmp, err := ioutil.TempDir("", "test-materials")
	require.NoError(t, err, "error creating tmp dir")
	out, err := ioutil.TempDir("", "output")
	require.NoError(t, err, "error creating out dir")
	defer func() {
		err := os.RemoveAll(tmp)
		require.NoError(t, err, "error cleaning tmp dir")
		err = os.RemoveAll(out)
		require.NoError(t, err, "error cleaning out dir")
	}()

	files := []string{"test.txt", "test/test.txt"}
	testZip := setUpZip(t, tmp, files)
	err = Extract(testZip, out)
	require.NoError(t, err, "error extracting archive")

	for _, f := range files {
		path := filepath.Join(out, f)
		_, err := os.Stat(path)
		require.NoError(t, err, "expected files not found")
	}
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
		require.NoError(t, err, "error writting header")

		_, err = io.Copy(tarWriter, file)
		require.NoError(t, err, "error writting tar")
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
		require.NoError(t, err, "error writting zip")

		_, err = io.Copy(writer, file)

		require.NoError(t, err, "error writting file to zip")
	}

	return zipPath
}
