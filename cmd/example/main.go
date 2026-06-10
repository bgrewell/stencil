// File: examples/ui_demo/main.go
package main

import (
	"fmt"
	"time"

	"github.com/bgrewell/stencil"
)

func main() {
	// Create the app (color on; defaults to stdout/stderr; root command auto-created)
	app := stencil.NewApp(
		stencil.WithName("ui-demo"),
		stencil.WithDescription("Stencil UI demo (messages + spinners)"),
		stencil.WithColorMode(stencil.ColorOn),
	)

	ui := app.UI

	// 1) Simple messages
	ui.Info("Starting %s", app.Name)

	// 2) Spinner for setup phase
	setupSp, err := ui.Task("Preparing build environment")
	if err != nil {
		ui.Error("failed to start spinner: %v", err)
		return
	}
	time.Sleep(800 * time.Millisecond) // simulate work
	setupSp.Update("Environment ready")
	setupSp.Complete()

	// 3) Download loop with spinner-based progress
	dlSp, err := ui.Task("Downloading assets")
	if err != nil {
		ui.Error("failed to start spinner: %v", err)
		return
	}
	total := int64(100)
	for got := int64(0); got <= total; got++ {
		time.Sleep(75 * time.Millisecond) // simulate chunk
		pct := float64(got) / float64(total) * 100
		dlSp.Update(fmt.Sprintf("Downloading | Progress %.1f%%", pct))
	}
	dlSp.Complete()

	// 4) Print a warning message
	ui.Warn("This is a warning message, but the build will continue")

	// 5) Print an error message
	ui.Error("An error occurred, but we can still proceed with the build")

	// 6) Final message
	ui.Info("Build complete")
}
