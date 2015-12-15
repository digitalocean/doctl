package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/bryanl/doit"
	"github.com/bryanl/doit/install"
	"github.com/fatih/color"
)

var (
	ver  = flag.String("ver", doit.DoitVersion.String(), "version to install")
	bold = color.New(color.Bold, color.FgWhite).SprintfFunc()
)

func main() {
	flag.Parse()

	fmt.Printf("\nInstall doit %s\n\n", bold(*ver))

	var err error
	defer func() {
		if err != nil {
			log.Fatalf("error encountered: %v", err)
		}
	}()

	// get install directory
	home, err := homeDir()
	if err != nil {
		return
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("doit installation directory (this will create a doit subdirectory) (%s): ", bold(home))
	installDir, err := reader.ReadString('\n')
	if err != nil {
		return
	}

	installDir = strings.TrimSpace(installDir)

	if installDir == "" {
		installDir = home
	}

	// create install directory
	doitDir := filepath.Join(installDir, "doit")
	fmt.Printf("creating %s directory...\n\n", doitDir)
	err = os.MkdirAll(filepath.Join(doitDir, "bin"), 0755)
	if err != nil {
		return
	}

	// create temp directory
	tmpDir, err := ioutil.TempDir("", "doit-install-")
	if err != nil {
		return
	}
	defer func() {
		err := os.Remove(tmpDir)
		if err != nil {
			fmt.Printf("could not remove temp directory (%s): %v", tmpDir, err)
		}
	}()

	// retrieve doit binary
	filename := archiveName(*ver)

	fmt.Println("retrieving doit...")
	doitPath := filepath.Join(tmpDir, filename)
	file, err := install.Download(doitPath, install.URL(filename))
	if err != nil {
		return
	}
	file.Close()
	fmt.Println()

	fmt.Println("retrieving doit checksum...")
	checksumPath := filepath.Join(tmpDir, filename+".sha256")
	checksumFile, err := install.Download(checksumPath, install.URL(filename+".sha256"))
	if err != nil {
		log.Fatalf("could not download doit checksum file: %v", err)
	}
	checksumFile.Close()
	fmt.Println("\n")

	// validate binary
	fmt.Println("validating doit checksum...")
	f, err := os.Open(doitPath)
	if err != nil {
		return
	}
	defer f.Close()

	cs, err := os.Open(checksumPath)
	if err != nil {
		return
	}
	defer func() {
		cs.Close()
		os.Remove(checksumPath)
	}()

	err = install.Validate(f, cs)
	if err != nil {
		return
	}

	fmt.Println("checksum was valid\n")

	// place binary in install directory
	doitInstallPath := filepath.Join(doitDir, "bin", "doit")
	fmt.Println("placing doit in install path...")
	err = os.Rename(doitPath, doitInstallPath)
	if err != nil {
		return
	}
	os.Chmod(doitInstallPath, 0755)

	fmt.Println("install complete!\n")
}

func homeDir() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}

	return usr.HomeDir, nil
}

func archiveName(ver string) string {
	var suffix string

	if runtime.GOOS == "darwin" {
		suffix = "darwin-10.6-amd64"
	} else {
		suffix = fmt.Sprintf("%s-%s", runtime.GOOS, runtime.GOARCH)
	}

	return fmt.Sprintf("doit-%s-%s", ver, suffix)
}
