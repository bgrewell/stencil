---
title: Commands and flags
description: The command tree and typed flags.
---

## App options

`NewApp` takes `WithName`, `WithDescription`, `WithVersionInfo`,
`WithColorMode`, `WithIO`, `WithUI`, `WithRootCommand` and `WithConfigFile`.
`app.Execute(args)` returns an exit code.

## Commands

A `Command` has a `Name`, `Summary`, optional `Run`, `Flags` (local),
`PersistentFlags` (inherited by subcommands), positional `Args` (`ArgSpec`)
and `Sub` commands. It also supports aliases, hidden and deprecated commands,
and pre/post run hooks.

## Flags

Types: `Bool`, `String`, `Int`, `Duration`, `StringSlice`, `StringArray`,
`IntSlice`. Flags can have short names, an `Env` variable, enum values,
validation and a `ConfigKey`.

`StringArray` accumulates one element per occurrence and never splits on
commas. The first command-line occurrence replaces lower layers; later ones
append.

## Known issues

Some parser behavior is under repair: see the [roadmap](/roadmap/) (P1
bugs), for example env values are not yet validated.
