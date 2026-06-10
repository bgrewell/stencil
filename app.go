package stencil

import (
	"io"
	"os"
	"time"

	"github.com/bgrewell/stencil/pkg/ui"
)

type ColorMode int

const (
	ColorAuto ColorMode = iota
	ColorOn
	ColorOff
)

type VersionInfo struct {
	Version    string
	BuildDate  string
	CommitHash string
	Branch     string
}

type Stdio struct {
	In  io.Reader
	Out io.Writer
	Err io.Writer
}

type App struct {
	Name        string
	Description string
	Version     VersionInfo
	Color       ColorMode
	IO          Stdio
	UI          ui.UI

	Root *Command
}

type Option func(*App)

func NewApp(opts ...Option) *App {
	a := &App{
		Name:  filepathBase(os.Args[0]),
		Color: ColorAuto,
		IO:    Stdio{In: os.Stdin, Out: os.Stdout, Err: os.Stderr},
	}
	for _, o := range opts {
		o(a)
	}
	if a.Root == nil {
		a.Root = &Command{
			Name:            a.Name,
			Summary:         a.Description,
			PersistentFlags: NewFlagSet(),
			Flags:           NewFlagSet(),
		}
	}
	if a.UI == nil {
		a.UI = ui.NewConsoleUI(a.IO.Out)
	}
	return a
}

func (a *App) Execute(argv []string) int {
	cfg := ParserConfig{
		App:      a,
		Colored:  shouldColor(a.Color, a.IO.Out),
		Out:      a.IO.Out,
		Err:      a.IO.Err,
		TimeNow:  time.Now,
		ShowInfo: AppShow{Version: true, BuildDate: true, CommitHash: true, Branch: true},
	}
	return NewParser(cfg).Execute(argv)
}

// --- Options ---
func WithName(name string) Option          { return func(a *App) { a.Name = name } }
func WithDescription(desc string) Option   { return func(a *App) { a.Description = desc } }
func WithVersionInfo(v VersionInfo) Option { return func(a *App) { a.Version = v } }
func WithColorMode(m ColorMode) Option     { return func(a *App) { a.Color = m } }
func WithIO(in io.Reader, out, err io.Writer) Option {
	return func(a *App) { a.IO = Stdio{In: in, Out: out, Err: err} }
}
func WithUI(u ui.UI) Option             { return func(a *App) { a.UI = u } }
func WithRootCommand(c *Command) Option { return func(a *App) { a.Root = c } }

// --- Internal util ---
func filepathBase(p string) string {
	for i := len(p) - 1; i >= 0; i-- {
		if p[i] == '/' || p[i] == '\\' {
			return p[i+1:]
		}
	}
	return p
}

func isCharDevice(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

func shouldColor(mode ColorMode, out io.Writer) bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	switch mode {
	case ColorOn:
		return true
	case ColorOff:
		return false
	default:
		return isCharDevice(out)
	}
}
