# Stencil Roadmap

Prioritized from a code audit of `parser.go`, `flags.go`, `config.go`, `app.go`,
`help.go` and `command.go` at HEAD `7decabc`. Items marked *(verified)* were
reproduced with a throwaway test; the rest come from reading the code.

## Status key

Every item carries one status tag, written right after its number:

- `[open]`: not started; candidates for the next spec.
- `[spec]`: a spec has been drafted and awaits approval.
- `[in-progress]`: approved and being implemented.
- `[in-review]`: a pull request is open.
- `[done]`: merged; keep the line and add the PR link.
- `[dropped]`: will not be done; add a one-line reason.

Priority is the section (P1 > P2 > P3) and then the number within it. The top
open item is the lowest-numbered `[open]` item (items 1 and 2 are `[done]`; item 3 is `[spec]`: `docs/specs/item-3-unknown-subcommand.md`). Update the tag
in the same change that moves the item along.

## P1: Bugs that produce wrong behavior

1. `[done]` **Env values are never cast or validated** *(verified)*. Merged in [#7](https://github.com/bgrewell/stencil/pull/7). `applyEnv` stores the raw
   string. `Enum` and `Validate` are bypassed, and a bad int or duration from the
   environment silently reads as `0` with exit code 0. Fix: run `castValue` on env
   values and return a usage error on failure.
2. `[done]` **`--version`/`-V` and `-h`/`--help` are matched anywhere in argv** *(verified)*. Merged in [#9](https://github.com/bgrewell/stencil/pull/9).
   This includes after `--` and as the value of another flag (`--name --version`
   prints the version). `-V` also can't be reused as an app flag. Fix: scan only
   until `--` and skip tokens consumed as flag values, or move these into the flag
   parser.
3. `[spec]` **Unknown subcommands are treated as positionals** *(verified)*. `matchPath`
   never returns an error, so `app bogus` runs the root `Run` with `["bogus"]`
   when the root is runnable. Fix: if a command has `Sub` and the first token isn't
   a flag or match, return "unknown command" (with a suggestion).
4. `[open]` **`no-` prefix handling is wrong** *(verified)*. `lookupLong` strips `no-` from
   every name. A flag actually named `no-cache` can't be set (`unknown flag:
   --cache`). `--no-name v` on a non-bool flag is silently accepted as `--name v`.
   Fix: look up the exact name first, and allow negation only for bool flags.
5. `[open]` **`Required` is ineffective for most flags**. Defaults are always present in
   `values`. The check only catches an empty string, so required bool, int,
   duration and slice flags always pass. It also only looks at the leaf's local
   `Flags`, never at persistent flags. Fix: check `sources[name] != SourceDefault`
   and include effective persistent flags.
6. `[open]` **`PersistentPostRun` and `PostRun` don't run when `Run` fails**. Hooks cannot
   do cleanup on error. Fix: define the semantics (always-run cleanup hook or
   documented skip) and test them.

## P2: Gaps and rough edges

7. `[open]` **Bool flags don't accept a space-separated value**. `--flag false` leaves
   `false` as a positional. Document it or accept `true|false` as the next token.
8. `[open]` **Short flag forms**: `-n5` (attached value) and `-vn 5` (bundle ending in a
   value flag) are unsupported. The error for a non-bool in a bundle says
   "requires a value", which is misleading.
9. `[open]` **`ArgSpec.Max == 0` means unlimited**, so a command can't declare "no
   positionals". The sanity check at the end of `validateArgs` is an empty branch.
   Fix: use `-1` or a separate `NoArgs` option; implement or delete the check.
10. `[open]` **Config typing is strict**. YAML `port: 8080` into a String flag, an int into
    a Duration, or ints in a StringSlice are errors, and `null` values error out.
    `Validate` is not applied to slice, bool or duration values. `--config=` (empty
    value) is treated as an explicit path and silently skipped.
11. `[open]` **Only `YAMLLoader` ships**. Add JSON and TOML loaders; `ConfigLoader` is already
    a plain function type.
12. `[open]` **Help output**:
    - The usage line is always `[FLAGS] [COMMAND] [ARGS]` and uses only the root
      name.
    - Required flags, aliases and arg counts aren't shown.
    - Columns use a raw `\t` instead of a tabwriter.
    - The root help prints an empty "Version:" block when no version is set.
    - Flags shadowed in the path can be listed twice.
    - Help for an unknown path silently falls back to the parent.
13. `[open]` **`Command.Deprecated` is never surfaced** apart from a help label; no runtime
    warning is printed.
14. `[open]` **Exit-code constants are inconsistent**. `ExitOK` is typed `ErrCode` but the
    others are untyped, and `ExecError` is unused. `Run` errors are always exit 1;
    there's no way to return a specific code (for example `ExitNoChange`).
15. `[open]` **No shell completion** (bash/zsh/fish) and no auto-generated `completion`
    command.
16. `[open]` **No `context.Context`, signal handling, or cancellation** in `Context`.

## P3: Quality, tests and housekeeping

17. `[open]` **Missing tests**: `cmd`, `pkg/ui`, `internal/initializer`; parser edge cases
    above (add a regression test for each fix).
18. `[open]` **Stale docs and comments**: `README.md` still has the template title, and
    `api/`, `pkg/`, `examples/` and `internal/` READMEs are placeholders. The
    header comment in `cmd/example/main.go` points to a non-existent path.
19. `[open]` **Empty directories**: `api/` and `examples/` hold no code. Add runnable
    examples or remove them.
20. `[open]` **Security artifacts** (`SBOM.spdx`, `VEX.yaml`, `vulnerability-report.txt`) are
    committed and can go stale; generate them in CI instead.
21. `[open]` **Docs site**: scaffolded in `docs/` (Starlight); starter pages and publishing
    are still to do.

## Suggested order

Items 1-5 first (each is small and self-contained, with a test). Then 6, 9 and 14,
which change public semantics and should land before a 1.0. After that, the
config and help improvements (10-12), then completion (15).
