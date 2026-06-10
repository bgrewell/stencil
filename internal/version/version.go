// Package version manages a project's semantic version, stored as plain-text
// files under the .stencil directory at the project root. The version is split
// across three files (version_major, version_minor, version_patch) so that the
// individual components can be consumed directly by build tooling (e.g. via Go
// ldflags) without parsing.
package version

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Dir is the directory, relative to the project root, that holds version state.
const Dir = ".stencil"

const (
	fileMajor = "version_major"
	fileMinor = "version_minor"
	filePatch = "version_patch"
)

// Part identifies which component of a semantic version to bump.
type Part int

const (
	Patch Part = iota
	Minor
	Major
)

// Version is a semantic version (major.minor.patch).
type Version struct {
	Major, Minor, Patch int
}

// String renders the version as "major.minor.patch".
func (v Version) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

func dir(root string) string { return filepath.Join(root, Dir) }

// Init creates the .stencil directory and seeds any missing version files with
// 0. It is idempotent: existing files are left untouched. It returns the
// resulting version.
func Init(root string) (Version, error) {
	d := dir(root)
	if err := os.MkdirAll(d, 0o755); err != nil {
		return Version{}, fmt.Errorf("create %s: %w", d, err)
	}
	for _, name := range []string{fileMajor, fileMinor, filePatch} {
		p := filepath.Join(d, name)
		_, err := os.Stat(p)
		switch {
		case err == nil:
			continue // already present
		case os.IsNotExist(err):
			if werr := os.WriteFile(p, []byte("0\n"), 0o644); werr != nil {
				return Version{}, fmt.Errorf("write %s: %w", p, werr)
			}
		default:
			return Version{}, err
		}
	}
	return Read(root)
}

// Read loads the current version from the .stencil directory.
func Read(root string) (Version, error) {
	d := dir(root)
	if _, err := os.Stat(d); os.IsNotExist(err) {
		return Version{}, fmt.Errorf("%s not found; run `version init` first", d)
	}
	major, err := readInt(filepath.Join(d, fileMajor))
	if err != nil {
		return Version{}, err
	}
	minor, err := readInt(filepath.Join(d, fileMinor))
	if err != nil {
		return Version{}, err
	}
	patch, err := readInt(filepath.Join(d, filePatch))
	if err != nil {
		return Version{}, err
	}
	return Version{Major: major, Minor: minor, Patch: patch}, nil
}

// Write persists v to the .stencil directory, creating it if necessary.
func Write(root string, v Version) error {
	d := dir(root)
	if err := os.MkdirAll(d, 0o755); err != nil {
		return err
	}
	files := []struct {
		name string
		val  int
	}{
		{fileMajor, v.Major},
		{fileMinor, v.Minor},
		{filePatch, v.Patch},
	}
	for _, f := range files {
		p := filepath.Join(d, f.name)
		if err := os.WriteFile(p, []byte(strconv.Itoa(f.val)+"\n"), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", p, err)
		}
	}
	return nil
}

// Bump increments the given component, resetting all lower-order components to
// 0, and persists the result.
func Bump(root string, part Part) (Version, error) {
	v, err := Read(root)
	if err != nil {
		return Version{}, err
	}
	switch part {
	case Major:
		v.Major++
		v.Minor = 0
		v.Patch = 0
	case Minor:
		v.Minor++
		v.Patch = 0
	default: // Patch
		v.Patch++
	}
	if err := Write(root, v); err != nil {
		return Version{}, err
	}
	return v, nil
}

func readInt(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, fmt.Errorf("read %s: %w", path, err)
	}
	s := strings.TrimSpace(string(data))
	if s == "" {
		return 0, nil
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("invalid version number in %s: %q", path, s)
	}
	return n, nil
}
