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
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/pelletier/go-toml/v2"
)

const (
	harborMaxArchiveBytes = 100 << 20
	harborMaxFiles        = 10000
	harborIgnoreFile      = ".harborignore"
)

var harborNameRE = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,127}$`)

// harborDefaultIgnores applies when a task has no root .gitignore.
var harborDefaultIgnores = []string{"__pycache__/", "*.pyc", ".DS_Store", "*.swp", "*.swo", "*~"}

// harborSingleFiles and harborTreeDirs are the only parts of a task folder
// Harbor publishes; anything else at the top level is left out.
var (
	harborSingleFiles = []string{"task.toml", "instruction.md", "README.md", "trajectory.json"}
	harborTreeDirs    = []string{"environment", "tests", "solution", "steps"}
)

var harborIgnoreScopes = mustHarborIgnoreSpec(harborIgnoreGit, []string{
	"/tests/", "/solution/", "/steps/*/tests/", "/steps/*/solution/",
})

func mustHarborIgnoreSpec(flavour harborIgnoreFlavour, lines []string) *harborIgnoreSpec {
	spec, err := newHarborIgnoreSpec(flavour, lines)
	if err != nil {
		panic(err)
	}
	return spec
}

// harborFile is one file of a packed task.
type harborFile struct {
	// Path is relative to the task folder, with forward slashes.
	Path string
	Abs  string
	Size int64
	Exec bool
}

// harborPackage is a task folder packed the way `harbor publish` packs it.
type harborPackage struct {
	Name     string
	Dir      string
	Files    []harborFile
	Manifest map[string]any
	// Digest is sha256:<hex> over the sorted "<path>\0<sha256 of file>\n"
	// lines, which is Harbor's content hash. It does not depend on the
	// archive bytes.
	Digest      string
	FileDigests map[string]string
	ArchivePath string
	ArchiveSize int64
	Warnings    []string
}

// Close removes the temporary archive.
func (p *harborPackage) Close() {
	if p != nil && p.ArchivePath != "" {
		os.Remove(p.ArchivePath)
	}
}

// packHarborTask selects, hashes, and archives a task folder. nameOverride is
// --name, which takes priority over task.toml and the folder name.
func packHarborTask(dir, nameOverride string) (*harborPackage, error) {
	root, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	if root, err = filepath.EvalSymlinks(root); err != nil {
		return nil, fmt.Errorf("reading task folder: %w", err)
	}
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a folder; pass the path to a Harbor task folder", dir)
	}

	configPath := filepath.Join(root, "task.toml")
	config, err := os.ReadFile(configPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("task.toml not found in %s", dir)
		}
		return nil, err
	}
	manifest := map[string]any{}
	if err := toml.Unmarshal(config, &manifest); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", configPath, err)
	}

	pkg := &harborPackage{Dir: root, Manifest: manifest}
	pkg.Name, err = resolveHarborTaskName(root, manifest, nameOverride, &pkg.Warnings)
	if err != nil {
		return nil, err
	}

	files, err := collectHarborFiles(root)
	if err != nil {
		return nil, err
	}
	if len(files) > harborMaxFiles {
		return nil, fmt.Errorf("the task has %d files; the limit is %d", len(files), harborMaxFiles)
	}
	pkg.Files = files

	if err := writeHarborArchive(pkg); err != nil {
		pkg.Close()
		return nil, err
	}
	if pkg.ArchiveSize > harborMaxArchiveBytes {
		pkg.Close()
		return nil, fmt.Errorf("the task archive is %s; the limit is 100 MiB", displayers.BytesToHumanReadableUnitBinary(uint64(pkg.ArchiveSize)))
	}
	return pkg, nil
}

// resolveHarborTaskName applies RFC order: --name, then [task] name with any
// org prefix stripped, then the folder name.
func resolveHarborTaskName(root string, manifest map[string]any, override string, warnings *[]string) (string, error) {
	name, source := strings.TrimSpace(override), "--name"
	if name == "" {
		if task, ok := manifest["task"].(map[string]any); ok {
			if n, ok := task["name"].(string); ok && strings.TrimSpace(n) != "" {
				name, source = strings.TrimSpace(n), "[task] name in task.toml"
				if i := strings.LastIndex(name, "/"); i >= 0 {
					*warnings = append(*warnings, fmt.Sprintf("Publishing %q as %q: tasks belong to your team, so the org prefix is dropped.", name, name[i+1:]))
					name = name[i+1:]
				}
			}
		}
	}
	if name == "" {
		name, source = filepath.Base(root), "the folder name"
	}
	if !harborNameRE.MatchString(name) {
		return "", fmt.Errorf("task name %q (from %s) is invalid: use 1-128 lowercase letters, digits, '.', '_' or '-', starting with a letter or digit, or pass --name", name, source)
	}
	return name, nil
}

// collectHarborFiles ports Packager.collect_files.
func collectHarborFiles(root string) ([]harborFile, error) {
	filter := &harborFileFilter{root: root, specs: map[string]*harborIgnoreSpec{}}
	var rels []string

	for _, name := range harborSingleFiles {
		if _, err := os.Stat(filepath.Join(root, name)); err == nil {
			rels = append(rels, name)
		}
	}

	for _, dir := range harborTreeDirs {
		info, err := os.Stat(filepath.Join(root, dir))
		if err != nil || !info.IsDir() {
			continue
		}
		selected, err := filter.selectFiles(dir)
		if err != nil {
			return nil, err
		}
		for _, rel := range selected {
			if info, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel))); err == nil && info.Mode().IsRegular() {
				rels = append(rels, rel)
			}
		}
	}

	gitignore := harborDefaultIgnores
	if data, err := os.ReadFile(filepath.Join(root, ".gitignore")); err == nil {
		gitignore = harborSplitLines(string(data))
	} else if !errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("reading .gitignore: %w", err)
	}
	rootSpec, err := newHarborIgnoreSpec(harborIgnoreBasic, gitignore)
	if err != nil {
		return nil, fmt.Errorf(".gitignore: %w", err)
	}

	files := make([]harborFile, 0, len(rels))
	for _, rel := range rels {
		if rootSpec.excludes(rel) {
			continue
		}
		abs := filepath.Join(root, filepath.FromSlash(rel))
		if err := ensureHarborInsideTask(root, abs, rel); err != nil {
			return nil, err
		}
		info, err := os.Stat(abs)
		if err != nil {
			return nil, err
		}
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("%s must be a regular file", rel)
		}
		files = append(files, harborFile{Path: rel, Abs: abs, Size: info.Size(), Exec: info.Mode().Perm()&0o111 != 0})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, nil
}

// ensureHarborInsideTask refuses a file whose symlinks lead outside the task,
// which the server would reject anyway.
func ensureHarborInsideTask(root, abs, rel string) error {
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return fmt.Errorf("resolving %s: %w", rel, err)
	}
	if !harborWithin(root, resolved) {
		return fmt.Errorf("%s is a symlink to a file outside the task folder; copy the file into the task instead", rel)
	}
	return nil
}

func harborWithin(root, p string) bool {
	r, err := filepath.Rel(root, p)
	return err == nil && r != ".." && !strings.HasPrefix(r, ".."+string(filepath.Separator)) && !filepath.IsAbs(r)
}

// harborFileFilter ports TaskFileFilter: .harborignore files apply only under
// tests/, solution/, and each step's tests/ and solution/, and rules are
// inherited from parent folders.
type harborFileFilter struct {
	root  string
	specs map[string]*harborIgnoreSpec
}

// selectFiles ports select_files: a sorted walk that keeps every entry not
// excluded and does not descend into symlinked folders.
func (f *harborFileFilter) selectFiles(dir string) ([]string, error) {
	excluded, err := f.isExcluded(dir)
	if err != nil || excluded {
		return nil, err
	}
	var out []string
	var visit func(rel string) error
	visit = func(rel string) error {
		entries, err := os.ReadDir(filepath.Join(f.root, filepath.FromSlash(rel)))
		if err != nil {
			return err
		}
		for _, e := range entries {
			child := path.Join(rel, e.Name())
			excluded, err := f.isExcluded(child)
			if err != nil {
				return err
			}
			if excluded {
				continue
			}
			out = append(out, child)
			if e.IsDir() && e.Type()&fs.ModeSymlink == 0 {
				if err := visit(child); err != nil {
					return err
				}
			}
		}
		return nil
	}
	if err := visit(dir); err != nil {
		return nil, err
	}
	return out, nil
}

func (f *harborFileFilter) isDir(rel string) bool {
	info, err := os.Stat(filepath.Join(f.root, filepath.FromSlash(rel)))
	return err == nil && info.IsDir()
}

// isExcluded checks a path and, for a symlink inside the task, its target.
func (f *harborFileFilter) isExcluded(rel string) (bool, error) {
	excluded, err := f.matches(rel)
	if err != nil || excluded {
		return excluded, err
	}
	abs := filepath.Join(f.root, filepath.FromSlash(rel))
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil || resolved == abs || !harborWithin(f.root, resolved) {
		return false, nil
	}
	target, err := filepath.Rel(f.root, resolved)
	if err != nil || target == "." {
		return false, nil
	}
	return f.matches(filepath.ToSlash(target))
}

func (f *harborFileFilter) matches(rel string) (bool, error) {
	if path.Base(rel) == harborIgnoreFile {
		return true, nil
	}
	suffix := ""
	if f.isDir(rel) {
		suffix = "/"
	}
	if !harborIgnoreScopes.excludes(rel + suffix) {
		return false, nil
	}

	type dirSpec struct {
		dir  string
		spec *harborIgnoreSpec
	}
	var specs []dirSpec
	parts := strings.Split(rel, "/")
	candidate := ""
	for i, part := range parts {
		spec, err := f.rules(candidate)
		if err != nil {
			return false, err
		}
		if spec != nil {
			specs = append(specs, dirSpec{candidate, spec})
		}
		candidate = path.Join(candidate, part)
		suf := ""
		if i < len(parts)-1 || suffix == "/" {
			suf = "/"
		}
		if !harborIgnoreScopes.excludes(candidate + suf) {
			continue
		}
		excluded := false
		for _, s := range specs {
			sub := candidate
			if s.dir != "" {
				sub = strings.TrimPrefix(candidate, s.dir+"/")
			}
			if ex, matched := s.spec.check(sub + suf); matched {
				excluded = ex
			}
		}
		if excluded {
			return true, nil
		}
	}
	return false, nil
}

// rules loads the .harborignore of a folder, or returns nil outside the
// folders where .harborignore applies.
func (f *harborFileFilter) rules(dir string) (*harborIgnoreSpec, error) {
	scopePath := "./"
	if dir != "" {
		scopePath = dir + "/"
	}
	if !harborIgnoreScopes.excludes(scopePath) {
		return nil, nil
	}
	if spec, ok := f.specs[dir]; ok {
		return spec, nil
	}

	rel := path.Join(dir, harborIgnoreFile)
	abs := filepath.Join(f.root, filepath.FromSlash(rel))
	var lines []string
	if _, err := os.Lstat(abs); err == nil {
		if resolved, err := filepath.EvalSymlinks(abs); err != nil || !harborWithin(f.root, resolved) {
			return nil, fmt.Errorf("%s must stay within the task folder", rel)
		}
		info, err := os.Stat(abs)
		if err != nil || !info.Mode().IsRegular() {
			return nil, fmt.Errorf("%s must be a regular file", rel)
		}
		data, err := os.ReadFile(abs)
		if err != nil {
			return nil, err
		}
		lines = harborSplitLines(string(data))
	}
	spec, err := newHarborIgnoreSpec(harborIgnoreGit, lines)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", rel, err)
	}
	f.specs[dir] = spec
	return spec, nil
}

// writeHarborArchive writes the tar.gz Harbor's publisher writes (file entries
// only, uid/gid 0, empty owner names, mtime 0, mode 0755 or 0644) and computes
// the digest from the same bytes it archives.
func writeHarborArchive(pkg *harborPackage) error {
	tmp, err := os.CreateTemp("", "doctl-harbor-*.tar.gz")
	if err != nil {
		return err
	}
	pkg.ArchivePath = tmp.Name()

	gz := gzip.NewWriter(tmp)
	tw := tar.NewWriter(gz)
	outer := sha256.New()
	pkg.FileDigests = make(map[string]string, len(pkg.Files))

	for _, f := range pkg.Files {
		mode := int64(0o644)
		if f.Exec {
			mode = 0o755
		}
		hdr := &tar.Header{
			Typeflag: tar.TypeReg,
			Name:     f.Path,
			Size:     f.Size,
			Mode:     mode,
			ModTime:  time.Unix(0, 0),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			tmp.Close()
			return err
		}
		sum, err := copyHarborFile(tw, f)
		if err != nil {
			tmp.Close()
			return err
		}
		pkg.FileDigests[f.Path] = "sha256:" + sum
		fmt.Fprintf(outer, "%s\x00%s\n", f.Path, sum)
	}

	if err := errors.Join(tw.Close(), gz.Close()); err != nil {
		tmp.Close()
		return err
	}
	info, err := tmp.Stat()
	if err != nil {
		tmp.Close()
		return err
	}
	pkg.ArchiveSize = info.Size()
	pkg.Digest = "sha256:" + hex.EncodeToString(outer.Sum(nil))
	return tmp.Close()
}

func copyHarborFile(w io.Writer, f harborFile) (string, error) {
	in, err := os.Open(f.Abs)
	if err != nil {
		return "", err
	}
	defer in.Close()
	h := sha256.New()
	n, err := io.Copy(io.MultiWriter(w, h), io.LimitReader(in, f.Size+1))
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", f.Path, err)
	}
	if n != f.Size {
		return "", fmt.Errorf("%s changed while it was being packed; try again", f.Path)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// harborContentDigest computes Harbor's digest from (path, sha256 hex) pairs.
func harborContentDigest(fileSums map[string]string) string {
	paths := make([]string, 0, len(fileSums))
	for p := range fileSums {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	outer := sha256.New()
	for _, p := range paths {
		fmt.Fprintf(outer, "%s\x00%s\n", p, fileSums[p])
	}
	return "sha256:" + hex.EncodeToString(outer.Sum(nil))
}
