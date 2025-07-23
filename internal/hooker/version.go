package hooker

const (
	postCommitScript = `#!/bin/bash

VERSION_FILE="version.txt"

if [[ ! -f "$VERSION_FILE" ]]; then
  echo "1" > "$VERSION_FILE"
else
  VERSION=$(cat "$VERSION_FILE")
  VERSION=$((VERSION + 1))
  echo "$VERSION" > "$VERSION_FILE"
fi

git add "$VERSION_FILE"
git commit --amend --no-edit --no-verify
`
)

type Version struct {
}

func (v Version) Install(rootPath string) error {
	return InstallHook(rootPath, "post-commit", postCommitScript)
}

func (v Version) Uninstall(rootPath string) error {
	return UninstallHook(rootPath, "post-commit")
}
