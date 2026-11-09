---
title: Roadmap
description: What Stencil does today and what is planned.
---

Source: `ROADMAP.md` in the repository, from a code audit at commit `7decabc`.
Everything below marked **Planned** is not implemented yet.

## Implemented today

- Command tree, typed flags, layered resolution, provenance
- YAML config loader
- Generated help, `--version`
- `pkg/ui` messages and spinners
- Developer CLI: `version` and `hook`

## P1: Bugs to fix

1. **Planned:** env values are not cast or validated.
2. **Planned:** `--version` and `-h` match anywhere in argv, including after `--`.
3. **Planned:** unknown subcommands are treated as positionals.
4. **Planned:** `no-` prefix handling is wrong for flags named `no-*`.
5. **Planned:** `Required` is ineffective for non-string and persistent flags.
6. **Planned:** PostRun hooks are skipped when `Run` fails.

## P2: Gaps

- **Planned:** bool flags accepting a separate value (`--flag false`)
- **Planned:** `-n5` and `-vn 5` short-flag forms
- **Planned:** a way to declare "no positional args"
- **Planned:** looser config typing and null handling
- **Planned:** JSON and TOML config loaders
- **Planned:** help improvements (usage line, required flags, aliases, alignment)
- **Planned:** runtime warning for deprecated commands
- **Planned:** consistent exit codes and custom codes from `Run`
- **Planned:** shell completion (bash, zsh, fish)
- **Planned:** `context.Context`, signal handling, cancellation

## P3: Housekeeping

- **Planned:** tests for `cmd`, `pkg/ui`, `internal/initializer`
- **Planned:** fill the `api/`, `pkg/`, `examples/`, `internal/` READMEs; add runnable examples
- **Planned:** generate security artifacts (SBOM, VEX) in CI
- **Planned:** publish this docs site

## Suggested order

Bugs 1-5 first, then 6 with the exit-code and no-args changes (they alter
public semantics ahead of 1.0), then config and help, then completion.
