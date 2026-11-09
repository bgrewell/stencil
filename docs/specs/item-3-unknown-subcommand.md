# Spec: unknown subcommand is an error, not a positional (ROADMAP item 3)

Status: spec (awaiting approval)

## Problem
`matchPath` (parser.go) stops at the first token that matches no subcommand and
never returns an error. So `app bogus` runs the root `Run` with `["bogus"]`
when the root is runnable. When the root is not runnable, it prints help and
exits 2 without saying what was wrong. A typo in a subcommand name is
indistinguishable from a positional argument.

## Behavior
- In `matchPath`, when the current command has `Sub` and the next token is not
  a flag, not `--`, and matches no subcommand name or alias, return an
  unknown-command usage error.
- A command with no `Sub` is unchanged: its tokens are positionals.
- Tokens starting with `-` and `--` still end path matching with no error.
  Flags before the subcommand (`app --verbose bogus`) are not covered; the
  walk stops at the first flag, as today. See Decisions.
- Error text: `unknown command "bogus" for "app"`, using the full path names
  joined by spaces (`"app remote"`). If a close match exists, add
  `\nDid you mean "build"?`. Close match: smallest Levenshtein distance over
  names and aliases of the current command's `Sub`, accepted when distance <= 2
  (and < len(token)). Ties go to the first declared.
- Exit code is `ExitUsage` (2). The error goes to stderr, then help for the
  deepest matched command, as other usage errors do.
- Hidden subcommands still match, but are never suggested.
- `app help bogus` is unchanged (falls back to the parent; item 12).

## Changes
- `parser.go`: `matchPath` returns `newUsagef(...)` in the no-match branch when
  `len(cur.Sub) > 0`. `Execute` already handles the error from `matchPath` in
  the normal flow. The `help` flow ignores it, as today.
- New small helpers `suggestCommand(cur, name)` and `levenshtein(a, b)` in
  `parser.go`, about 25 lines, no new dependency.

## Compatibility
Behavior change: an app whose root has both `Sub` and a `Run` that takes
positionals will now fail on a first positional. Such apps must either drop
`Sub` or put the positional after `--`. Call this out in the changelog. A
`Command` option to opt out (for example `AllowArgsWithSub`) is not added.

## Tests (new `unknown_cmd_test.go`)
- Root with `Sub` and `Run`: `app bogus` returns 2, `Run` not called, stderr
  has `unknown command "bogus" for "app"`.
- Nested: `app remote bogus` names `"app remote"`.
- Typo `buidl` suggests `build`; alias is suggested by its own name;
  `zzzzzz` gives no suggestion.
- Leaf without `Sub`: `app run foo` still passes `["foo"]` to `Run`.
- `app -- bogus` and `app --flag bogus` unchanged (no error).
- Valid subcommand and alias still run.
- Existing tests (including `builtin_test.go`) pass: `go test ./...`.

## Out of scope
Help for unknown paths (item 12), flags before subcommands, shell completion,
`no-` handling (item 4).

## Decisions needed
1. Is the compatibility break acceptable, or should an opt-out exist?
2. Should `app --verbose bogus` also error? That requires the matcher to know
   which flags take values; proposed to defer, since it needs the flag lookup
   used by `scanBuiltin`.
