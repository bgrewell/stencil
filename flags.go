package stencil

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type FlagType int

const (
	FlagBool FlagType = iota
	FlagString
	FlagInt
	FlagDuration
	FlagStringSlice
	FlagIntSlice
	// FlagStringArray accumulates one element per command-line occurrence
	// and never splits on commas (suited to key=value overrides).
	FlagStringArray
)

type Flag struct {
	Type     FlagType
	Name     string // long --name
	Short    string // short -n
	Usage    string
	Default  any
	Env      string // optional env var name
	Required bool
	Enum     []string        // allowed values (for strings)
	Validate func(any) error // per-flag custom validation
	Hidden   bool

	// ConfigKey is an optional dotted path (e.g. "server.port") that binds the
	// flag to a value in the config file loaded via WithConfigFile.
	ConfigKey string
}

type FlagSet struct {
	order  []*Flag
	byLong map[string]*Flag
	bySh   map[string]*Flag
}

func NewFlagSet() *FlagSet {
	return &FlagSet{byLong: map[string]*Flag{}, bySh: map[string]*Flag{}}
}

func (fs *FlagSet) add(f *Flag) *Flag {
	if f.Name == "" {
		panic("flag must have a long name")
	}
	if _, exists := fs.byLong[f.Name]; exists {
		panic("duplicate flag: " + f.Name)
	}
	fs.order = append(fs.order, f)
	fs.byLong[f.Name] = f
	if f.Short != "" {
		if _, ok := fs.bySh[f.Short]; ok {
			panic("duplicate short flag: " + f.Short)
		}
		fs.bySh[f.Short] = f
	}
	return f
}

func (fs *FlagSet) Bool(name, short, usage string, def bool) *Flag {
	return fs.add(&Flag{Type: FlagBool, Name: name, Short: short, Usage: usage, Default: def})
}
func (fs *FlagSet) String(name, short, usage, def string) *Flag {
	return fs.add(&Flag{Type: FlagString, Name: name, Short: short, Usage: usage, Default: def})
}
func (fs *FlagSet) Int(name, short, usage string, def int) *Flag {
	return fs.add(&Flag{Type: FlagInt, Name: name, Short: short, Usage: usage, Default: def})
}
func (fs *FlagSet) Duration(name, short, usage string, def time.Duration) *Flag {
	return fs.add(&Flag{Type: FlagDuration, Name: name, Short: short, Usage: usage, Default: def})
}
func (fs *FlagSet) StringSlice(name, short, usage string, def []string) *Flag {
	return fs.add(&Flag{Type: FlagStringSlice, Name: name, Short: short, Usage: usage, Default: def})
}
func (fs *FlagSet) StringArray(name, short, usage string, def []string) *Flag {
	return fs.add(&Flag{Type: FlagStringArray, Name: name, Short: short, Usage: usage, Default: def})
}
func (fs *FlagSet) IntSlice(name, short, usage string, def []int) *Flag {
	return fs.add(&Flag{Type: FlagIntSlice, Name: name, Short: short, Usage: usage, Default: def})
}

// Lookup (internal)
func (fs *FlagSet) list() []*Flag                    { return fs.order }
func (fs *FlagSet) getLong(n string) (*Flag, bool)   { f, ok := fs.byLong[n]; return f, ok }
func (fs *FlagSet) getShort(sh string) (*Flag, bool) { f, ok := fs.bySh[sh]; return f, ok }

// ResolvedFlags is the merged, final view.
// Source identifies the layer that supplied a resolved flag value.
// Layers are applied in ascending order: default < config < env < flag.
type Source int

const (
	SourceDefault Source = iota
	SourceConfig
	SourceEnv
	SourceFlag
)

func (s Source) String() string {
	switch s {
	case SourceDefault:
		return "default"
	case SourceConfig:
		return "config"
	case SourceEnv:
		return "env"
	case SourceFlag:
		return "flag"
	default:
		return "unknown"
	}
}

type ResolvedFlags struct {
	values  map[string]any
	sources map[string]Source
}

// Source reports which layer supplied the value of the named flag.
// Unknown flags report SourceDefault.
func (rf *ResolvedFlags) Source(name string) Source {
	return rf.sources[name]
}

// Changed reports whether the named flag was set by any layer other than its
// declared default (config file, environment variable, or command line).
func (rf *ResolvedFlags) Changed(name string) bool {
	return rf.Source(name) != SourceDefault
}

func (rf *ResolvedFlags) Bool(name string) bool {
	v, _ := rf.values[name]
	switch t := v.(type) {
	case bool:
		return t
	case string:
		b, _ := strconv.ParseBool(t)
		return b
	default:
		return false
	}
}
func (rf *ResolvedFlags) String(name string) string {
	v, _ := rf.values[name]
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
func (rf *ResolvedFlags) Int(name string) int {
	v, _ := rf.values[name]
	switch t := v.(type) {
	case int:
		return t
	case string:
		i, _ := strconv.Atoi(t)
		return i
	default:
		return 0
	}
}
func (rf *ResolvedFlags) Duration(name string) time.Duration {
	v, _ := rf.values[name]
	switch t := v.(type) {
	case time.Duration:
		return t
	case string:
		d, _ := time.ParseDuration(t)
		return d
	default:
		return 0
	}
}
func (rf *ResolvedFlags) StringSlice(name string) []string {
	v, _ := rf.values[name]
	switch t := v.(type) {
	case []string:
		return append([]string(nil), t...)
	case string:
		return csvSplit(t)
	default:
		return nil
	}
}
func (rf *ResolvedFlags) StringArray(name string) []string {
	v, _ := rf.values[name]
	switch t := v.(type) {
	case []string:
		return append([]string(nil), t...)
	case string:
		if t == "" {
			return nil
		}
		return []string{t}
	default:
		return nil
	}
}
func (rf *ResolvedFlags) IntSlice(name string) []int {
	v, _ := rf.values[name]
	switch t := v.(type) {
	case []int:
		return append([]int(nil), t...)
	case string:
		var out []int
		for _, s := range csvSplit(t) {
			i, _ := strconv.Atoi(s)
			out = append(out, i)
		}
		return out
	default:
		return nil
	}
}
func (rf *ResolvedFlags) Get(name string) (any, bool) { v, ok := rf.values[name]; return v, ok }

// Merge defaults across path (persistent) and leaf locals. Every flag starts
// with SourceDefault provenance.
func mergeFlagDefaults(path []*Command) (map[string]any, map[string]Source) {
	out := map[string]any{}
	src := map[string]Source{}
	for _, c := range path {
		if c.PersistentFlags != nil {
			for _, f := range c.PersistentFlags.list() {
				out[f.Name] = f.Default
				src[f.Name] = SourceDefault
			}
		}
	}
	leaf := path[len(path)-1]
	if leaf.Flags != nil {
		for _, f := range leaf.Flags.list() {
			out[f.Name] = f.Default
			src[f.Name] = SourceDefault
		}
	}
	return out, src
}

// Apply env on top of defaults if Env is set
func applyEnv(path []*Command, values map[string]any, sources map[string]Source) {
	seen := map[string]struct{}{}
	for _, c := range path {
		if c.PersistentFlags != nil {
			for _, f := range c.PersistentFlags.list() {
				if f.Env == "" || f.Hidden {
					continue
				}
				if _, ok := seen[f.Name]; ok {
					continue
				}
				if v, ok := os.LookupEnv(f.Env); ok && v != "" {
					values[f.Name] = v
					sources[f.Name] = SourceEnv
				}
				seen[f.Name] = struct{}{}
			}
		}
	}
	leaf := path[len(path)-1]
	if leaf.Flags != nil {
		for _, f := range leaf.Flags.list() {
			if f.Env == "" || f.Hidden {
				continue
			}
			if _, ok := seen[f.Name]; ok {
				continue
			}
			if v, ok := os.LookupEnv(f.Env); ok && v != "" {
				values[f.Name] = v
				sources[f.Name] = SourceEnv
			}
			seen[f.Name] = struct{}{}
		}
	}
}

func castValue(f *Flag, v any) (any, error) {
	switch f.Type {
	case FlagBool:
		switch t := v.(type) {
		case bool:
			return t, nil
		case string:
			l := strings.ToLower(strings.TrimSpace(t))
			switch l {
			case "", "true", "1", "yes", "on":
				return true, nil
			case "false", "0", "no", "off":
				return false, nil
			default:
				return nil, fmt.Errorf("invalid bool for --%s: %q", f.Name, t)
			}
		default:
			return nil, fmt.Errorf("invalid bool for --%s", f.Name)
		}
	case FlagString:
		if s, ok := v.(string); ok {
			if len(f.Enum) > 0 && !inEnum(s, f.Enum) {
				return nil, fmt.Errorf("--%s must be one of %v", f.Name, f.Enum)
			}
			if f.Validate != nil {
				if err := f.Validate(s); err != nil {
					return nil, err
				}
			}
			return s, nil
		}
		return nil, fmt.Errorf("invalid string for --%s", f.Name)
	case FlagInt:
		switch t := v.(type) {
		case int:
			if f.Validate != nil {
				if err := f.Validate(t); err != nil {
					return nil, err
				}
			}
			return t, nil
		case string:
			i, err := strconv.Atoi(strings.TrimSpace(t))
			if err != nil {
				return nil, fmt.Errorf("invalid int for --%s: %q", f.Name, t)
			}
			if f.Validate != nil {
				if err := f.Validate(i); err != nil {
					return nil, err
				}
			}
			return i, nil
		default:
			return nil, fmt.Errorf("invalid int for --%s", f.Name)
		}
	case FlagDuration:
		switch t := v.(type) {
		case time.Duration:
			return t, nil
		case string:
			d, err := time.ParseDuration(strings.TrimSpace(t))
			if err != nil {
				return nil, fmt.Errorf("invalid duration for --%s: %q", f.Name, t)
			}
			return d, nil
		default:
			return nil, fmt.Errorf("invalid duration for --%s", f.Name)
		}
	case FlagStringSlice:
		switch t := v.(type) {
		case []string:
			return t, nil
		case []any:
			out := make([]string, 0, len(t))
			for _, e := range t {
				s, ok := e.(string)
				if !ok {
					return nil, fmt.Errorf("invalid string in --%s: %v", f.Name, e)
				}
				out = append(out, s)
			}
			return out, nil
		case string:
			return csvSplit(t), nil
		default:
			return nil, fmt.Errorf("invalid string slice for --%s", f.Name)
		}
	case FlagStringArray:
		switch t := v.(type) {
		case []string:
			return t, nil
		case []any:
			out := make([]string, 0, len(t))
			for _, e := range t {
				s, ok := e.(string)
				if !ok {
					return nil, fmt.Errorf("invalid string in --%s: %v", f.Name, e)
				}
				out = append(out, s)
			}
			return out, nil
		case string:
			return []string{t}, nil
		default:
			return nil, fmt.Errorf("invalid string array for --%s", f.Name)
		}
	case FlagIntSlice:
		switch t := v.(type) {
		case []int:
			return t, nil
		case []any:
			out := make([]int, 0, len(t))
			for _, e := range t {
				i, ok := e.(int)
				if !ok {
					return nil, fmt.Errorf("invalid int in --%s: %v", f.Name, e)
				}
				out = append(out, i)
			}
			return out, nil
		case string:
			raw := csvSplit(t)
			out := make([]int, 0, len(raw))
			for _, s := range raw {
				i, err := strconv.Atoi(s)
				if err != nil {
					return nil, fmt.Errorf("invalid int in --%s: %q", f.Name, s)
				}
				out = append(out, i)
			}
			return out, nil
		default:
			return nil, fmt.Errorf("invalid int slice for --%s", f.Name)
		}
	default:
		return nil, fmt.Errorf("unsupported flag type for --%s", f.Name)
	}
}

func inEnum(s string, set []string) bool {
	for _, v := range set {
		if s == v {
			return true
		}
	}
	return false
}

func csvSplit(s string) []string {
	if s = strings.TrimSpace(s); s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}
