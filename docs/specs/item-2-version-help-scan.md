# Spec: scope `--version` and `-h` matching (ROADMAP item 2)

Status: awaiting approval

## Problem
`Execute` (parser.go) scans all of argv for `--version`/`-V` before anything is
parsed, and for `-h`/`--help` after `matchPath`. So these fire after `--`
(`app run -- --version`) and when they are the value of another flag
(`app --name --version`). `-V` also can't be used as an app flag, because the
built-in check wins.

## Behavior
- Add one helper, `scanBuiltin(path, argv)`, that walks the tokens the same way
  `parseFlags` does and reports whether a built-in version or help token was
  seen as a real flag.
- The scan stops at the first `--`. Tokens after it are never built-ins.
- A token consumed as a flag value is skipped. That means the next token after
  a non-bool long or short flag with no `=`. Bool flags consume nothing.
  Lookup uses `flagLookups(path)` and `lookupLong`, so the scan and
  `parseFlags` agree on which flag takes a value.
- Built-in tokens: `--version`, `-V`, `--help`, `-h`. Short bundles (`-vh`) are
  not matched (unchanged).
- User flags win. If the effective flags define a long or short name equal to a
  built-in (for example `Short: "V"`), the token is the user's flag and is not a
  built-in. `--version` stays reserved for the built-in unless the app defines a
  long flag `version`. This fixes `-V` reuse.
- Precedence is unchanged: version before help. If both appear, version wins.
- `app help ...` is unchanged.

## Changes
- `parser.go`: version check moves from the top of `Execute` to after
  `matchPath`, so the effective flags are known. Path and help check use the
  same helper. An unknown subcommand token isn't an error today (item 3), so
  moving it is safe.
- New helper in `parser.go`, about 30 lines. `parseFlags` is not changed.

## Tests (new `builtin_test.go`)
- `--version`, `-V`, `-h`, `--help` before `--` still work, at root and in a
  subcommand, including after other flags and positionals.
- `app run -- --version` and `-- -h` reach `Run` as positionals.
- `--name --version` and `-n -h` (non-bool flag) treat the token as the value;
  `Run` sees it.
- A bool flag before `--version` doesn't swallow it.
- `--name=--version` is a value, not a built-in.
- An app flag with `Short: "V"` is parsed as the app flag and doesn't print
  the version.
- Version wins over help when both are present.
- `go test ./...` passes.

## Out of scope
Unknown-subcommand errors (item 3), `no-` handling (item 4), bundled short
flags (item 8), making `-h`/`-V` real flags in help output.

## Open questions
1. Is a user flag named `help`/`h` allowed to override the built-in help too?
   Default: yes, same rule as `-V`.
2. Alternative: register version/help as real flags in the parser. It's more
   uniform, but it changes `parseFlags` and help output. Default: not now.
