package main

import (
	"fmt"
	"github.com/bgrewell/stencil"
	"time"
)

func main() {
	// Instantiate your helper (spinners & colored output enabled)
	app := stencil.NewStencil(stencil.WithColor(true))
	ui := app.UI

	// 1) Simple messages
	ui.Info("Starting %s", app.AppName)

	// 2) Spinner for setup phase
	setupSp, err := ui.Task("Preparing build environment")
	if err != nil {
		ui.Error("failed to start spinner: %v", err)
		return
	}
	time.Sleep(800 * time.Millisecond) // simulate work
	setupSp.Update("Environment ready")
	setupSp.Complete()

	// 3) Download loop with spinner‑based progress
	dlSp, _ := ui.Task("Downloading assets")
	total := int64(100)
	for got := int64(0); got <= total; got += 1 {
		time.Sleep(300 * time.Millisecond) // simulate chunk
		dlSp.Update(fmt.Sprintf("Downloading | Progress %.1f%%", float64(got)/float64(total)*100))
	}
	dlSp.Complete()

	// 4) Print a warning message
	ui.Warn("This is a warning message, but the build will continue")

	// 5) Print an error message
	ui.Error("An error occurred, but we can still proceed with the build")

	// 6) Final message
	ui.Info("Build complete")
}
