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
		err := extractTarGz(source, target)
		if err != nil {
			return err
		}

	case ".zip":
		err := extractZip(source, target)
		if err != nil {
			return err
		}

	default:
		return errors.New("unexpected file type")
	}

	return nil
}

func extractTarGz(source, target string) error {
	s, err := os.Open(source)
	if err != nil {
		return err
	}
	defer s.Close()

	gzReader, err := gzip.NewReader(s)
	if err != nil {
		return err
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		info := header.FileInfo()
		path := filepath.Join(target, header.Name)
		if !strings.HasPrefix(path, filepath.Clean(target)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", path)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			err := os.MkdirAll(path, info.Mode())
			if err != nil {
				return err
			}

		case tar.TypeReg:
			if _, err := os.Stat(filepath.Dir(path)); os.IsNotExist(err) {
				os.MkdirAll(filepath.Dir(path), 0700)
			}

			f, err := os.Create(path)
			if err != nil {
				return err
			}

			f.Chmod(info.Mode())
			if err != nil {
				return err
			}

			_, err = io.Copy(f, tarReader)
			if err != nil {
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
	zReader, err := zip.OpenReader(source)
	if err != nil {
		return err
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

		f, err := os.Create(path)
		if err != nil {
			return err
		}

		f.Chmod(zf.Mode())
		if err != nil {
			return err
		}

		zippedFile, err := zf.Open()
		if err != nil {
			return err
		}

		_, err = io.Copy(f, zippedFile)
		if err != nil {
			return err
		}
		f.Close()
		zippedFile.Close()
	}

	return nil
}
