package stencil

import (
	"fmt"
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

func NewStencil(opts ...Option) *Stencil {
	s := &Stencil{
		AppName:        filepath.Base(os.Args[0]),
		AppDesc:        "",
		ShowVersion:    true,
		ShowBuildDate:  true,
		ShowCommitHash: true,
		ShowBranch:     true,
		ColoredOutput:  true,
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

type Stencil struct {
	AppName        string
	AppDesc        string
	ShowVersion    bool
	ShowBuildDate  bool
	ShowCommitHash bool
	ShowBranch     bool
	ColoredOutput  bool
}

func (s *Stencil) ShowHelp() {
	fmt.Printf("Usage: %s [options]\n\nDescription: %s\n  Version: %s\n  Build Date: %s\n  Commit Hash: %s\n  Branch: %s\n\nOptions:\n",
		s.AppName,
		s.AppDesc,
		appVersion,
		appBuildDate,
		appCommitHash,
		appBranch,
	)
}
