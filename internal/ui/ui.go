package ui

import (
	"fmt"
	"io"
	"time"

	"github.com/theckman/yacspin"
)

// Spinner is the handle you get back when you start a spinner.
type Spinner interface {
	Update(msg string) error
	Stop() error
}

// UI is the interface your app code talks to.
type UI interface {
	Info(format string, args ...interface{})
	Warn(format string, args ...interface{})
	Error(format string, args ...interface{})

	// StartSpinner launches a spinner; caller must Stop().
	StartSpinner(msg string) (Spinner, error)

	// Progress prints an overwritable progress line.
	Progress(current, total int64)
}

// Option tweaks behavior (verbosity, spinner on/off, etc.).
type Option func(*config)

// NewConsoleUI creates a console-based UI (uses yacspin under the hood).
func NewConsoleUI(out io.Writer, opts ...Option) UI {
	cfg := defaultConfig()
	for _, o := range opts {
		o(&cfg)
	}
	return &consoleUI{w: out, cfg: cfg}
}

// WithSpinner enables or disables spinner output.
func WithSpinner(enabled bool) Option {
	return func(c *config) { c.enableSpinner = enabled }
}
