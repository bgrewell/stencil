package stencil

import (
	"os"
	"path/filepath"
)

var (
	appVersion    string
	appBuildDate  string
	appCommitHash string
	appBranch     string
)

type Option func(*Stencil)

// NewStencil creates a new Stencil instance with default values. Options can be provided to customize the instance.
func NewStencil(opts ...Option) *Stencil {
	s := &Stencil{
		AppName:        filepath.Base(os.Args[0]),
		ShowVersion:    true,
		ShowBuildDate:  true,
		ShowCommitHash: true,
		ShowBranch:     true,
		ColoredOutput:  true,
	}

	// apply user options
	for _, opt := range opts {
		opt(s)
	}

	// initialize UI after options are set
	s.UI = NewConsoleUI(os.Stdout, WithSpinner(s.ColoredOutput))
	return s
}

// Stencil represents the helper structure for commonly used command-line application features.
type Stencil struct {
	UI             UI
	AppName        string
	AppDesc        string
	ShowVersion    bool
	ShowBuildDate  bool
	ShowCommitHash bool
	ShowBranch     bool
	ColoredOutput  bool
}
