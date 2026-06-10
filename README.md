# golang-repo-template
This project is a template repository for my typical golang repositories

## Versioning

Stencil ships a small developer CLI (`cmd/`, the `stencil` binary) that tracks a
project's semantic version in plain-text files under `.stencil/`
(`version_major`, `version_minor`, `version_patch`). This is developer tooling —
it is run in your repository and is independent of the user-facing CLI you build
with the stencil library.

```bash
go build -o stencil ./cmd

# create .stencil/ seeded at 0.0.0
stencil version init

# print the current version
stencil version show

# bump components (patch is the default; major/minor reset lower components)
stencil version bump            # 0.0.0 -> 0.0.1
stencil version bump --minor    # 0.0.1 -> 0.1.0
stencil version bump --major    # 0.1.0 -> 1.0.0

# install a git pre-commit hook that auto-bumps the patch version on every
# commit and stages the change so it lands in that commit
stencil hook install
stencil hook uninstall
```

> Run the subcommand first, then flags: `stencil version show -C path/to/repo`
> (use `-C`/`--dir` to operate on a repo other than the current directory).

## Setting Versioning Information

Feed the tracked version into your build via ldflags:

```bash
VERSION="$(cat .stencil/version_major).$(cat .stencil/version_minor).$(cat .stencil/version_patch)"

go build -ldflags "\
  -X 'github.com/bgrewell/stencil.appVersion=v${VERSION}' \
  -X 'github.com/bgrewell/stencil.appBuildDate=2025-07-23' \
  -X 'github.com/bgrewell/stencil.appCommitHash=abc1234' \
  -X 'github.com/bgrewell/stencil.appBranch=main'"
```