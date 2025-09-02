package stencil

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/bgrewell/stencil/pkg/ui"
	"github.com/fatih/color"
)

var (
	appVersion    string = "dev"
	appBuildDate  string = "unknown"
	appCommitHash string = "unknown"
	appBranch     string = "unknown"
)

type Option func(*Stencil)

// Flag represents a command-line flag.
type Flag struct {
	Short       string
	Name        string
	Usage       string
	Value       interface{}
	Default     interface{}
	SetFunction func(value interface{}) error
}

// NewStencil creates a new Stencil instance with default values. Options can be provided to customize the instance.
func NewStencil(opts ...Option) *Stencil {
	s := &Stencil{
		AppName:        filepath.Base(os.Args[0]),
		ShowVersion:    true,
		ShowBuildDate:  true,
		ShowCommitHash: true,
		ShowBranch:     true,
		ColoredOutput:  true,
		Output:         os.Stdout,
		flags:          make(map[string]*Flag),
	}

	// apply user options
	for _, opt := range opts {
		opt(s)
	}

	// initialize UI after options are set
	s.UI = ui.NewConsoleUI(os.Stdout)
	return s
}

// Stencil represents the helper structure for commonly used command-line application features.
type Stencil struct {
	UI             ui.UI
	AppName        string
	AppDesc        string
	ShowVersion    bool
	ShowBuildDate  bool
	ShowCommitHash bool
	ShowBranch     bool
	ColoredOutput  bool
	Output         io.Writer
	flags          map[string]*Flag
}

// BoolFlag registers a boolean flag.
func (s *Stencil) BoolFlag(name, short, usage string, defaultValue bool) *bool {
	value := new(bool)
	s.flags[name] = &Flag{
		Name:    name,
		Short:   short,
		Usage:   usage,
		Value:   value,
		Default: defaultValue,
		SetFunction: func(val interface{}) error {
			if v, ok := val.(bool); ok {
				*value = v
				return nil
			}
			return errors.New("invalid value type")
		},
	}
	return value
}

// StringFlag registers a string flag.
func (s *Stencil) StringFlag(name, short, usage string, defaultValue string) *string {
	value := new(string)
	s.flags[name] = &Flag{
		Name:    name,
		Short:   short,
		Usage:   usage,
		Value:   value,
		Default: defaultValue,
		SetFunction: func(val interface{}) error {
			if v, ok := val.(string); ok {
				*value = v
				return nil
			}
			return errors.New("invalid value type")
		},
	}
	return value
}

// IntFlag registers an integer flag.
func (s *Stencil) IntFlag(name, short, usage string, defaultValue int) *int {
	value := new(int)
	s.flags[name] = &Flag{
		Name:    name,
		Short:   short,
		Usage:   usage,
		Value:   value,
		Default: defaultValue,
		SetFunction: func(val interface{}) error {
			if v, ok := val.(int); ok {
				*value = v
				return nil
			}
			return errors.New("invalid value type")
		},
	}
	return value
}

// convertFlagValue converts a string argument value to the appropriate type.
func (s *Stencil) convertFlagValue(defaultValue interface{}, arg string) (interface{}, error) {
	switch defaultValue.(type) {
	case bool:
		return strconv.ParseBool(arg)
	case string:
		return arg, nil
	case int:
		return strconv.Atoi(arg)
	default:
		return nil, errors.New("unsupported flag type")
	}
}

// ParseFlags parses the command-line arguments and sets flag values.
func (s *Stencil) ParseFlags(args []string) error {
	for _, arg := range args {
		if strings.HasPrefix(arg, "--") {
			parts := strings.SplitN(arg[2:], "=", 2)
			name := parts[0]
			flag, exists := s.flags[name]
			if !exists {
				return fmt.Errorf("unknown flag: --%s", name)
			}

			var value interface{}
			if len(parts) == 2 {
				value, _ = s.convertFlagValue(flag.Default, parts[1])
			} else {
				value, _ = s.convertFlagValue(flag.Default, "true")
			}

			if err := flag.SetFunction(value); err != nil {
				return fmt.Errorf("failed to set flag --%s: %v", name, err)
			}
		}
	}
	return nil
}

func (s *Stencil) ShowHelp() {
	// Setup color functions with conditional disabling
	if !s.ColoredOutput {
		color.NoColor = true // Disable color globally if ColoredOutput is false
	}

	blue := color.New(color.FgBlue).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()

	appName := blue(s.AppName)
	fmt.Fprintf(s.Output, "Usage: %s [OPTIONS]\n\n", appName)

	if s.AppDesc != "" {
		desc := green("Description:")
		fmt.Fprintf(s.Output, "%s %s\n\n", desc, s.AppDesc)
	}

	versionSectionPrinted := false
	printVersionInfo := func(header, value string) {
		if !versionSectionPrinted {
			fmt.Fprintln(s.Output, yellow("Version Information:"))
			versionSectionPrinted = true
		}
		fmt.Fprintf(s.Output, "  %s: %s\n", header, value)
	}

	if s.ShowVersion {
		printVersionInfo("Version", appVersion)
	}
	if s.ShowBuildDate {
		printVersionInfo("Build Date", appBuildDate)
	}
	if s.ShowCommitHash {
		printVersionInfo("Commit Hash", appCommitHash)
	}
	if s.ShowBranch {
		printVersionInfo("Branch", appBranch)
	}

	if versionSectionPrinted {
		fmt.Fprintln(s.Output)
	}

	fmt.Fprintln(s.Output, yellow("Options:"))
	for _, flag := range s.flags {
		fmt.Fprintf(s.Output, "  --%s: %s (default: %v)\n", flag.Name, flag.Usage, flag.Default)
	}
}
