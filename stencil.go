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

func NewStencil(opts ...Option) *Stencil {
	s := &Stencil{
		AppName:        filepath.Base(os.Args[0]),
		AppDesc:        "",
		ShowVersion:    true,
		ShowBuildDate:  true,
		ShowCommitHash: true,
		ShowBranch:     true,
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
}
