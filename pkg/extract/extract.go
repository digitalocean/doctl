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

// sanitizePath validates that the resolved path falls within the extraction root.
func sanitizePath(target, name string) (string, error) {
	cleanTarget := filepath.Clean(target) + string(os.PathSeparator)
	path := filepath.Join(target, name)
	if !strings.HasPrefix(path, cleanTarget) {
		return "", fmt.Errorf("illegal file path: %s", path)
	}
	return path, nil
}

// sanitizeLinkTarget validates that a hardlink or symlink target resolves to
// a path within the extraction root. For symlinks the linkname is resolved
// relative to the directory that will contain the link.
func sanitizeLinkTarget(target, linkname, entryPath string, isSymlink bool) error {
	resolved := linkname
	if isSymlink {
		if !filepath.IsAbs(linkname) {
			resolved = filepath.Join(filepath.Dir(entryPath), linkname)
		}
	}
	resolved = filepath.Clean(resolved)
	cleanTarget := filepath.Clean(target) + string(os.PathSeparator)
	if !strings.HasPrefix(resolved, cleanTarget) {
		return fmt.Errorf("illegal link target: %s -> %s", entryPath, linkname)
	}
	return nil
}

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
		path, err := sanitizePath(target, header.Name)
		if err != nil {
			return err
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

			f, err := os.Create(path)
			if err != nil {
				return err
			}

			err = f.Chmod(info.Mode())
			if err != nil {
				f.Close()
				return err
			}

			_, err = io.Copy(f, tarReader)
			if err != nil {
				f.Close()
				return err
			}
			f.Close()

		case tar.TypeLink:
			if err := sanitizeLinkTarget(target, header.Linkname, path, false); err != nil {
				return err
			}
			if err := os.Link(header.Linkname, path); err != nil {
				return err
			}

		case tar.TypeSymlink:
			if err := sanitizeLinkTarget(target, header.Linkname, path, true); err != nil {
				return err
			}
			if err := os.Symlink(header.Linkname, path); err != nil {
				return err
			}

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
		path, err := sanitizePath(target, zf.Name)
		if err != nil {
			return err
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

		err = f.Chmod(zf.Mode())
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
