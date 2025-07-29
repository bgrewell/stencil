package ui

import (
	"io"
)

// Spinner is the handle you get back when you start a spinner.
type Spinner interface {
	Update(format string, args ...interface{})
	Complete()
	Stop()
	Fail()
}

// UI is the interface your app code talks to.
type UI interface {
	Info(format string, args ...interface{})
	Warn(format string, args ...interface{})
	Error(format string, args ...interface{})
	Task(format string, args ...interface{}) (Spinner, error)
}

// config holds internal settings for Console UI behavior.
type config struct {
	spinnerIndex    int
	taskColumnWidth int
	minPadding      int
	paddingChar     string
}

// defaultConfig returns the default config.
func defaultConfig() config {
	return config{
		spinnerIndex:    14,
		taskColumnWidth: -1,
		minPadding:      3,
		paddingChar:     ".",
	}
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

// WithSpinnerStyle allows customization of the spinner style.
func WithSpinnerStyle(style int) Option {
	return func(c *config) { c.spinnerIndex = style }
}
