# Stencil

Stencil is a Go library for building command-line applications. You describe
your commands and typed flags; Stencil parses arguments, generates help,
resolves flag values from several sources, and gives you a small console UI
for messages and spinners.

It also ships a developer CLI (the `stencil` binary) that tracks your
project's semantic version and can install a git hook to bump it on commit.

## Features

- Nested command tree with aliases, hidden and deprecated commands, and
  pre/post run hooks (including persistent ones inherited by subcommands)
- Typed flags: `Bool`, `String`, `Int`, `Duration`, `StringSlice`,
  `StringArray`, `IntSlice`, with short names, env vars, required flags,
  enum values and custom validation
- Layered value resolution: default, then config file, then environment,
  then command line, with provenance (`Source`, `Changed`)
- Pluggable config files (`ConfigLoader`; a YAML loader is included)
- Generated help (`-h`, `--help`, `app help ...`) and `--version` / `-V`
- Color modes (auto, on, off; honors `NO_COLOR`)
- `pkg/ui`: console messages (`Info`, `Warn`, `Error`) and spinner tasks

## Install

```bash
go get github.com/bgrewell/stencil
```

Requires Go 1.22 or newer.

## Quick start

```go
package main

import (
    "fmt"
    "os"

    "github.com/bgrewell/stencil"
)

func main() {
    flags := stencil.NewFlagSet()
    flags.String("name", "n", "who to greet", "world").Env = "GREET_NAME"

    app := stencil.NewApp(
        stencil.WithName("greet"),
        stencil.WithDescription("A friendly greeter"),
        stencil.WithRootCommand(&stencil.Command{
            Name:    "greet",
            Summary: "Say hello",
            Flags:   flags,
            Run: func(ctx *stencil.Context) error {
                fmt.Printf("hello, %s\n", ctx.Flags.String("name"))
                return nil
            },
        }),
    )

    os.Exit(app.Execute(os.Args[1:]))
}
```

```console
$ greet                    # hello, world
$ GREET_NAME=env greet     # hello, env
$ greet --name cli         # hello, cli
$ greet --help             # generated help
```

## Concepts

- **App**: created with `NewApp` and options (`WithName`, `WithDescription`,
  `WithVersionInfo`, `WithColorMode`, `WithIO`, `WithUI`, `WithRootCommand`,
  `WithConfigFile`). `app.Execute(args)` returns an exit code.
- **Command**: a `Name`, `Summary`, optional `Run`, `Flags` (local),
  `PersistentFlags` (inherited by subcommands), positional `Args`
  (`ArgSpec`) and `Sub` commands.
- **Context**: passed to `Run`; holds `Args`, the command `Path` and the
  resolved `Flags`.

## Console UI

```go
ctx.App.UI.Info("starting")
task, err := ctx.App.UI.Task("building")
if err == nil {
    // ... work ...
    task.Complete() // or task.Fail()
}
```

Supply your own implementation of `ui.UI` with `WithUI`. A runnable demo
lives in `cmd/example`:

```bash
go run ./cmd/example
```

## Development

```bash
go build ./...
go test ./...
```

## Developer CLI: versioning

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

## Setting version information

Feed the tracked version into your build via ldflags:

```bash
VERSION="$(cat .stencil/version_major).$(cat .stencil/version_minor).$(cat .stencil/version_patch)"

go build -ldflags "\
  -X 'github.com/bgrewell/stencil.appVersion=v${VERSION}' \
  -X 'github.com/bgrewell/stencil.appBuildDate=2025-07-23' \
  -X 'github.com/bgrewell/stencil.appCommitHash=abc1234' \
  -X 'github.com/bgrewell/stencil.appBranch=main'"
```

## Config files and value provenance

Flag values resolve in layers, lowest to highest:

1. the flag's declared `Default`
2. the config file (opt-in, see below)
3. the flag's `Env` variable
4. the command line

### Provenance

`ctx.Flags.Source(name)` reports which layer supplied a value
(`SourceDefault`, `SourceConfig`, `SourceEnv`, `SourceFlag`), and
`ctx.Flags.Changed(name)` is true for anything other than the default. This
lets an application overlay only the values a user actually supplied onto its
own configuration.

### Config file layer

Enable it with `WithConfigFile`, naming the String flag that carries the path,
and bind individual flags to dotted keys with `ConfigKey`:

```go
persistent := stencil.NewFlagSet()
cfg := persistent.String("config", "c", "path to config file", "app.yaml")
cfg.Env = "APP_CONFIG"

flags := stencil.NewFlagSet()
flags.Int("workers", "w", "worker count", 1).ConfigKey = "server.workers"

app := stencil.NewApp(
    stencil.WithName("app"),
    stencil.WithRootCommand(&stencil.Command{
        Name: "app", PersistentFlags: persistent,
        Sub: []*stencil.Command{{Name: "run", Flags: flags, Run: run}},
    }),
    stencil.WithConfigFile("config", stencil.YAMLLoader),
)
```

```yaml
# app.yaml
server:
  workers: 4
```

The config path is taken from the command line, then the flag's `Env`
variable, then its default. A missing file is an error only when the path was
given explicitly (command line or environment). Any `ConfigLoader` can be
supplied; `YAMLLoader` is provided.

### Repeatable flags

`FlagSet.StringArray` declares a flag that accumulates one element per
occurrence and never splits on commas, which suits `--set key=value`
overrides:

```go
flags.StringArray("set", "", "override a config key (repeatable)", nil)
// app run --set a.b=[1,2] --set c=x,y  =>  ["a.b=[1,2]", "c=x,y"]
```

The first command-line occurrence replaces any value from lower layers;
later occurrences append.
