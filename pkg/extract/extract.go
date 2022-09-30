package extract

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Extract extracts files from an archive to the specified location. It supports
// .tar.gz and .zip archives.
func Extract(source, target string) error {
	if _, err := os.Stat(target); os.IsNotExist(err) {
		return errors.New("target directory does not exist")
	}

	if _, err := os.Stat(source); os.IsNotExist(err) {
		return errors.New("source archive does not exist")
	}

	switch filepath.Ext(source) {
	case ".gz":
		if err := extractTarGz(source, target); err != nil {
			return err
		}

	case ".zip":
		if err := extractZip(source, target); err != nil {
			return err
		}

	default:
		return errors.New("unexpected file type")
	}

	return nil
}

func extractTarGz(source, target string) error {
	s, openErr := os.Open(source)
	if openErr != nil {
		return openErr
	}
	defer s.Close()

	gzReader, gzReaderErr := gzip.NewReader(s)
	if gzReaderErr != nil {
		return gzReaderErr
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)
	for {
		header, rNextErr := tarReader.Next()
		if rNextErr != nil {
			if errors.Is(rNextErr, io.EOF) {
				break
			}

			return rNextErr
		}

		info := header.FileInfo()
		path := filepath.Join(target, header.Name)
		if !strings.HasPrefix(path, filepath.Clean(target)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", path)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(path, info.Mode()); err != nil {
				return err
			}

		case tar.TypeReg:
			if _, err := os.Stat(filepath.Dir(path)); os.IsNotExist(err) {
				os.MkdirAll(filepath.Dir(path), 0700)
			}

			f, createErr := os.Create(path)
			if createErr != nil {
				return createErr
			}

			if err := f.Chmod(info.Mode()); err != nil {
				return err
			}

			if _, err := io.Copy(f, tarReader); err != nil {
				return err
			}
			f.Close()

		case tar.TypeLink:
			os.Link(header.Linkname, path)

		case tar.TypeSymlink:
			os.Symlink(header.Linkname, path)

		default:
			return fmt.Errorf("unknown type %s in %s", string(header.Typeflag), header.Name)
		}
	}

	return nil
}

func extractZip(source, target string) error {
	zReader, openReaderErr := zip.OpenReader(source)
	if openReaderErr != nil {
		return openReaderErr
	}
	defer zReader.Close()

	for _, zf := range zReader.File {
		path := filepath.Join(target, zf.Name)
		if !strings.HasPrefix(path, filepath.Clean(target)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", path)
		}

		if zf.FileInfo().IsDir() {
			if err := os.MkdirAll(path, zf.Mode()); err != nil {
				return err
			}
			continue
		}

		if _, err := os.Stat(filepath.Dir(path)); os.IsNotExist(err) {
			os.MkdirAll(filepath.Dir(path), 0700)
		}

		f, createErr := os.Create(path)
		if createErr != nil {
			return createErr
		}

		if err := f.Chmod(zf.Mode()); err != nil {
			return err
		}

		zippedFile, openErr := zf.Open()
		if openErr != nil {
			return openErr
		}

		if _, err := io.Copy(f, zippedFile); err != nil {
			return err
		}
		f.Close()
		zippedFile.Close()
	}

	return nil
}
