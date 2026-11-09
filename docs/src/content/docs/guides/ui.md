---
title: Console UI
description: Messages and spinner tasks from pkg/ui.
---

```go
ctx.App.UI.Info("starting")
task, err := ctx.App.UI.Task("building")
if err == nil {
    // ... work ...
    task.Complete() // or task.Fail()
}
```

Messages: `Info`, `Warn`, `Error`. Supply your own `ui.UI` with `WithUI`.
Color modes are auto, on and off, and honor `NO_COLOR`.

A runnable demo: `go run ./cmd/example`.
