// Package hooker installs and removes git hooks. Hooks written by this package
// carry a marker comment so that stencil never clobbers or removes a hook the
// user wrote themselves.
package hooker

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// marker identifies hooks installed by stencil.
const marker = "# installed by stencil"

// Hooker installs and removes a git hook for a repository rooted at rootPath.
type Hooker interface {
	Install(rootPath string) error
	Uninstall(rootPath string) error
}

// hooksDir returns the git hooks directory for the repo rooted at rootPath,
// resolving the gitdir indirection used by worktrees and submodules (where
// .git is a file rather than a directory).
func hooksDir(rootPath string) (string, error) {
	gitPath := filepath.Join(rootPath, ".git")
	info, err := os.Stat(gitPath)
	if err != nil {
		return "", fmt.Errorf("not a git repository (no .git in %s)", rootPath)
	}

	gitDir := gitPath
	if !info.IsDir() {
		data, err := os.ReadFile(gitPath)
		if err != nil {
			return "", err
		}
		const prefix = "gitdir: "
		line := strings.TrimSpace(string(data))
		if !strings.HasPrefix(line, prefix) {
			return "", fmt.Errorf("unrecognized .git file format in %s", gitPath)
		}
		gitDir = strings.TrimSpace(strings.TrimPrefix(line, prefix))
		if !filepath.IsAbs(gitDir) {
			gitDir = filepath.Join(rootPath, gitDir)
		}
	}
	return filepath.Join(gitDir, "hooks"), nil
}

// InstallHook writes an executable git hook named `name` containing `script`.
// It refuses to overwrite a pre-existing hook that stencil did not install.
func InstallHook(rootPath, name, script string) error {
	dir, err := hooksDir(rootPath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, name)
	switch existing, err := os.ReadFile(path); {
	case err == nil:
		if !strings.Contains(string(existing), marker) {
			return fmt.Errorf("refusing to overwrite existing %s hook not managed by stencil", name)
		}
	case !os.IsNotExist(err):
		return err
	}
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		return fmt.Errorf("write hook %s: %w", path, err)
	}
	return nil
}

// UninstallHook removes the git hook named `name`, but only if stencil installed
// it. Removing an absent hook is a no-op.
func UninstallHook(rootPath, name string) error {
	dir, err := hooksDir(rootPath)
	if err != nil {
		return err
	}
	path := filepath.Join(dir, name)
	existing, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if !strings.Contains(string(existing), marker) {
		return fmt.Errorf("refusing to remove %s hook not managed by stencil", name)
	}
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("remove hook %s: %w", path, err)
	}
	return nil
}
