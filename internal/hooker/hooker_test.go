package hooker

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// newRepo creates a temp dir with a .git/ directory and returns its path.
func newRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestVersionHookInstallUninstall(t *testing.T) {
	root := newRepo(t)
	h := VersionHook{Bin: "/usr/local/bin/stencil"}

	if err := h.Install(root); err != nil {
		t.Fatalf("Install: %v", err)
	}

	path := filepath.Join(root, ".git", "hooks", HookName)
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("hook not written: %v", err)
	}
	if info.Mode().Perm()&0o100 == 0 {
		t.Fatalf("hook not executable: %v", info.Mode())
	}

	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{marker, "/usr/local/bin/stencil", "version bump --patch", "git add .stencil"} {
		if !strings.Contains(string(body), want) {
			t.Fatalf("hook missing %q:\n%s", want, body)
		}
	}

	if err := h.Uninstall(root); err != nil {
		t.Fatalf("Uninstall: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("hook still present after uninstall: %v", err)
	}
}

func TestInstallDefaultsToPathLookup(t *testing.T) {
	root := newRepo(t)
	if err := (VersionHook{}).Install(root); err != nil {
		t.Fatal(err)
	}
	body, _ := os.ReadFile(filepath.Join(root, ".git", "hooks", HookName))
	if !strings.Contains(string(body), `"stencil" version bump`) {
		t.Fatalf("expected PATH lookup of stencil:\n%s", body)
	}
}

func TestRefusesToClobberForeignHook(t *testing.T) {
	root := newRepo(t)
	hookPath := filepath.Join(root, ".git", "hooks", HookName)
	if err := os.MkdirAll(filepath.Dir(hookPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(hookPath, []byte("#!/bin/sh\necho mine\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := (VersionHook{}).Install(root); err == nil {
		t.Fatal("expected Install to refuse overwriting a foreign hook")
	}
	// And uninstall must not delete a hook stencil doesn't own.
	if err := (VersionHook{}).Uninstall(root); err == nil {
		t.Fatal("expected Uninstall to refuse removing a foreign hook")
	}
	if _, err := os.Stat(hookPath); err != nil {
		t.Fatalf("foreign hook was removed: %v", err)
	}
}

func TestUninstallMissingIsNoop(t *testing.T) {
	if err := (VersionHook{}).Uninstall(newRepo(t)); err != nil {
		t.Fatalf("uninstall of absent hook should be a no-op, got %v", err)
	}
}

func TestNotAGitRepo(t *testing.T) {
	if err := (VersionHook{}).Install(t.TempDir()); err == nil {
		t.Fatal("expected error when .git is absent")
	}
}

func TestGitFileIndirection(t *testing.T) {
	// Simulate a worktree/submodule where .git is a file pointing elsewhere.
	root := t.TempDir()
	realGit := t.TempDir()
	gitContent := "gitdir: " + realGit + "\n"
	if err := os.WriteFile(filepath.Join(root, ".git"), []byte(gitContent), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := (VersionHook{}).Install(root); err != nil {
		t.Fatalf("Install with .git file: %v", err)
	}
	if _, err := os.Stat(filepath.Join(realGit, "hooks", HookName)); err != nil {
		t.Fatalf("hook not written to resolved gitdir: %v", err)
	}
}
