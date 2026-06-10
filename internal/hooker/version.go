package hooker

import "fmt"

// HookName is the git hook used to auto-bump the project version.
//
// We use pre-commit (not post-commit): the hook bumps the patch version and
// stages the change *before* the commit is created, so the new version lands in
// that same commit. A post-commit hook would have to `git commit --amend`,
// which re-fires post-commit and loops forever.
const HookName = "pre-commit"

// VersionHook installs a git pre-commit hook that bumps the project's patch
// version (by calling back into the stencil binary) and stages the change.
type VersionHook struct {
	// Bin is the path to the stencil binary the hook should invoke. If empty,
	// the hook resolves "stencil" from PATH.
	Bin string
}

func (v VersionHook) bin() string {
	if v.Bin == "" {
		return "stencil"
	}
	return v.Bin
}

func (v VersionHook) script() string {
	return fmt.Sprintf(`#!/bin/sh
%s

# Auto-bump the patch version and stage it so it is part of this commit.
"%s" version bump --patch || exit 1
git add .stencil
`, marker, v.bin())
}

// Install writes the pre-commit hook into the repository at rootPath.
func (v VersionHook) Install(rootPath string) error {
	return InstallHook(rootPath, HookName, v.script())
}

// Uninstall removes the pre-commit hook from the repository at rootPath.
func (v VersionHook) Uninstall(rootPath string) error {
	return UninstallHook(rootPath, HookName)
}
