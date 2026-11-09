---
title: Getting started
description: Install Stencil and build a first command.
---

```bash
go get github.com/bgrewell/stencil
```

Requires Go 1.22 or newer.

```go
flags := stencil.NewFlagSet()
flags.String("name", "n", "who to greet", "world").Env = "GREET_NAME"

app := stencil.NewApp(
    stencil.WithName("greet"),
    stencil.WithDescription("A friendly greeter"),
    stencil.WithRootCommand(&stencil.Command{
        Name:  "greet",
        Flags: flags,
        Run: func(ctx *stencil.Context) error {
            fmt.Printf("hello, %s\n", ctx.Flags.String("name"))
            return nil
        },
    }),
)
os.Exit(app.Execute(os.Args[1:]))
```

`greet --help` prints generated help; `greet --name cli` prints `hello, cli`.
