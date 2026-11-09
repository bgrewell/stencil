---
title: Developer CLI
description: The stencil binary for version tracking and git hooks.
---

The `stencil` binary is developer tooling, independent of the CLIs you build
with the library.

```bash
go build -o stencil ./cmd
stencil version init            # create .stencil/ at 0.0.0
stencil version show
stencil version bump            # patch (default)
stencil version bump --minor
stencil version bump --major
stencil hook install            # pre-commit patch bump
stencil hook uninstall
```

Put the subcommand first, then flags: `stencil version show -C path/to/repo`.

The version lives in `.stencil/version_major`, `version_minor` and
`version_patch`, and can be fed to builds via ldflags.
