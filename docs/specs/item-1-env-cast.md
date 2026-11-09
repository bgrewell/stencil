# Spec: cast and validate env values (ROADMAP item 1)

Status: awaiting approval

## Problem
`applyEnv` (flags.go) stores the raw env string in `values`. `Enum` and
`Validate` are skipped, and a bad int or duration is later read as `0` with exit
code 0. Env is the only source (besides defaults) not passed through `castValue`.

## Behavior
- For each flag with `Env` set and a non-empty env value, call `castValue(f, raw)`.
- On success store the cast value, `sources[name] = SourceEnv`.
- On failure, `applyEnv` returns an error. The message names the variable, e.g.
  `invalid env FOO_PORT for --port: invalid int for --port: "abc"`.
- `parser.go` handles the error like a config error: print to `cfg.Err`, return
  `ExitUsage` (no help dump).
- Unchanged: empty env value is ignored; hidden flags and shadowed persistent
  flags are skipped; precedence stays default < config < env < argv.
- A command-line value still overrides a bad env value only if the env value is
  valid; a bad env is an error even when argv overrides it (simple and
  predictable; stated as a decision, easy to relax).

## Changes
- `flags.go`: `applyEnv` returns `error`, uses `castValue`.
- `parser.go` (line ~78): handle the error.
- `config.go` env fallback for the config path stays raw (it is a path string).

## Tests (regression, in a new `env_test.go` or existing test file)
- bad int, bad duration, bad bool from env -> exit `ExitUsage`, message on Err.
- Enum violation and `Validate` failure from env -> `ExitUsage`.
- valid int/duration/bool env -> cast value, `SourceEnv`.
- empty env var ignored; argv still overrides valid env.
- `go test ./...` passes.

## Out of scope
Other roadmap items (config typing, `Required`, `--` handling).

## Open question
Should a bad env value be an error when argv overrides it? Default: yes.
