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
## Config Files and Value Provenance

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
