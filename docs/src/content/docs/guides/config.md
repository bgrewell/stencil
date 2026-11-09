---
title: Config files and provenance
description: Layered flag values and where each value came from.
---

Values resolve in layers, lowest to highest: default, config file,
environment variable, command line.

## Provenance

`ctx.Flags.Source(name)` returns `SourceDefault`, `SourceConfig`, `SourceEnv`
or `SourceFlag`. `ctx.Flags.Changed(name)` is true for anything but the
default.

## Config file

Enable with `WithConfigFile("config", stencil.YAMLLoader)`, naming the String
flag that holds the path, and bind flags to dotted keys with `ConfigKey`:

```yaml
server:
  workers: 4
```

The path comes from the command line, then the flag's `Env`, then its default.
A missing file is an error only when the path was given explicitly.

Any `ConfigLoader` can be supplied. Only `YAMLLoader` ships today.

:::note[Planned]
JSON and TOML loaders are planned. See the [roadmap](/roadmap/).
:::
