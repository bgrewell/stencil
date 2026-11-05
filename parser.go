package stencil

import (
	"fmt"
	"io"
	"strings"
	"time"
)

type ParserConfig struct {
	App      *App
	Colored  bool
	Out      io.Writer
	Err      io.Writer
	TimeNow  func() time.Time
	ShowInfo AppShow
}

type Parser struct {
	cfg ParserConfig
}

func NewParser(cfg ParserConfig) *Parser { return &Parser{cfg: cfg} }

func (p *Parser) Execute(argv []string) int {
	// Global "version" handling
	for _, a := range argv {
		if a == "--version" || a == "-V" {
			renderVersion(p.cfg.Out, AppInfo{
				Name:        p.cfg.App.Name,
				Description: p.cfg.App.Description,
				Version:     p.cfg.App.Version,
				Show:        p.cfg.ShowInfo,
				Colored:     p.cfg.Colored,
			})
			return int(ExitOK)
		}
	}

	// Built-in "help" command: `app help ...`
	if len(argv) > 0 && argv[0] == "help" {
		argv = argv[1:]
		if len(argv) == 0 {
			renderHelp(p.cfg.Out, p.appInfo(), []*Command{p.cfg.App.Root})
			return int(ExitOK)
		}
		// Resolve path for help target
		path, _, _ := p.matchPath(argv)
		renderHelp(p.cfg.Out, p.appInfo(), path)
		return int(ExitOK)
	}

	// Normal flow
	path, rest, err := p.matchPath(argv)
	if err != nil {
		fmt.Fprintln(p.cfg.Err, err.Error())
		renderHelp(p.cfg.Out, p.appInfo(), path)
		return int(ExitUsage)
	}
	leaf := path[len(path)-1]

	// Recognize -h/--help anywhere
	for _, a := range argv {
		if a == "-h" || a == "--help" {
			renderHelp(p.cfg.Out, p.appInfo(), path)
			return int(ExitOK)
		}
	}

	// Merged defaults
	values, sources := mergeFlagDefaults(path)
	// Apply config file values if configured
	if err := p.applyConfig(path, rest, values, sources); err != nil {
		fmt.Fprintln(p.cfg.Err, err.Error())
		return int(ExitUsage)
	}
	// Apply env if set
	applyEnv(path, values, sources)

	// Parse flags and positionals
	vals, pos, err := p.parseFlags(path, rest, values, sources)
	if err != nil {
		fmt.Fprintln(p.cfg.Err, err.Error())
		renderHelp(p.cfg.Out, p.appInfo(), path)
		return int(ExitUsage)
	}

	// Validate positional args
	if err := validateArgs(leaf.Args, pos); err != nil {
		fmt.Fprintln(p.cfg.Err, err.Error())
		renderHelp(p.cfg.Out, p.appInfo(), path)
		return int(ExitUsage)
	}
	if leaf.Args.Validate != nil {
		if err := leaf.Args.Validate(pos); err != nil {
			fmt.Fprintln(p.cfg.Err, err.Error())
			renderHelp(p.cfg.Out, p.appInfo(), path)
			return int(ExitUsage)
		}
	}

	// If not runnable but has subs → help
	if leaf.Run == nil && len(leaf.Sub) > 0 {
		renderHelp(p.cfg.Out, p.appInfo(), path)
		return int(ExitUsage)
	}

	// Build context
	ctx := &Context{
		App:   p.cfg.App,
		Path:  path,
		Args:  pos,
		Flags: &ResolvedFlags{values: vals, sources: sources},
	}

	// Hooks: PersistentPreRun (root→leaf), PreRun(leaf), Run, PostRun(leaf), PersistentPostRun (leaf→root)
	for _, c := range path {
		if c.PersistentPreRun != nil {
			if err := c.PersistentPreRun(ctx); err != nil {
				fmt.Fprintln(p.cfg.Err, err.Error())
				return int(ExitRuntime)
			}
		}
	}
	if leaf.PreRun != nil {
		if err := leaf.PreRun(ctx); err != nil {
			fmt.Fprintln(p.cfg.Err, err.Error())
			return int(ExitRuntime)
		}
	}

	if leaf.Run == nil {
		renderHelp(p.cfg.Out, p.appInfo(), path)
		return int(ExitUsage)
	}
	if err := leaf.Run(ctx); err != nil {
		fmt.Fprintln(p.cfg.Err, err.Error())
		return int(ExitRuntime)
	}

	if leaf.PostRun != nil {
		if err := leaf.PostRun(ctx); err != nil {
			fmt.Fprintln(p.cfg.Err, err.Error())
			return int(ExitRuntime)
		}
	}
	for i := len(path) - 1; i >= 0; i-- {
		if path[i].PersistentPostRun != nil {
			if err := path[i].PersistentPostRun(ctx); err != nil {
				fmt.Fprintln(p.cfg.Err, err.Error())
				return int(ExitRuntime)
			}
		}
	}
	return int(ExitOK)
}

// matchPath walks command names greedily: root already implied.
func (p *Parser) matchPath(argv []string) ([]*Command, []string, error) {
	path := []*Command{p.cfg.App.Root}
	cur := p.cfg.App.Root
	i := 0
outer:
	for i < len(argv) {
		if strings.HasPrefix(argv[i], "-") || argv[i] == "--" {
			break
		}
		name := argv[i]
		for _, c := range cur.Sub {
			if c.Name == name || inEnum(name, c.Aliases) {
				path = append(path, c)
				cur = c
				i++
				continue outer
			}
		}
		// no subcommand match: stop
		break
	}
	return path, argv[i:], nil
}

// parseFlags respects persistent + local flags; supports --no-<boolflag>
func (p *Parser) parseFlags(path []*Command, argv []string, defaults map[string]any, sources map[string]Source) (map[string]any, []string, error) {
	longLkp, shortLkp := flagLookups(path)

	values := map[string]any{}
	for k, v := range defaults {
		values[k] = v
	}

	pos := []string{}
	i := 0
	inPos := false

	for i < len(argv) {
		tok := argv[i]
		if inPos {
			pos = append(pos, tok)
			i++
			continue
		}
		if tok == "--" {
			inPos = true
			i++
			continue
		}

		if strings.HasPrefix(tok, "--") {
			body := strings.TrimPrefix(tok, "--")
			name, sval, hasEq := splitEq(body)
			f, name, neg, ok := lookupLong(longLkp, name)
			if !ok {
				return values, pos, newUsagef("unknown flag: --%s", name)
			}
			var val any
			consumed := 1

			if f.Type == FlagBool && (neg || !hasEq) {
				val = !neg // --flag => true, --no-flag => false
			} else if hasEq {
				val = sval
			} else {
				if i+1 >= len(argv) {
					return values, pos, newUsagef("flag --%s requires a value", name)
				}
				val = argv[i+1]
				consumed = 2
			}

			casted, err := castValue(f, val)
			if err != nil {
				return values, pos, err
			}
			setFlagValue(f, values, sources, casted)
			i += consumed
			continue
		}

		if strings.HasPrefix(tok, "-") && len(tok) > 1 {
			body := strings.TrimPrefix(tok, "-")
			// Bundled short bools: -abc
			if !strings.Contains(body, "=") && len(body) > 1 {
				for _, ch := range strings.Split(body, "") {
					f, ok := shortLkp[ch]
					if !ok {
						return values, pos, newUsagef("unknown flag: -%s", ch)
					}
					if f.Type != FlagBool {
						return values, pos, newUsagef("flag -%s requires a value", ch)
					}
					values[f.Name] = true
					sources[f.Name] = SourceFlag
				}
				i++
				continue
			}
			// -f or -f=val or -f val
			name, sval, hasEq := splitEq(body)
			f, ok := shortLkp[name]
			if !ok {
				return values, pos, newUsagef("unknown flag: -%s", name)
			}
			var val any
			consumed := 1
			if f.Type == FlagBool && !hasEq {
				val = true
			} else if hasEq {
				val = sval
			} else {
				if i+1 >= len(argv) {
					return values, pos, newUsagef("flag -%s requires a value", name)
				}
				val = argv[i+1]
				consumed = 2
			}
			casted, err := castValue(f, val)
			if err != nil {
				return values, pos, err
			}
			setFlagValue(f, values, sources, casted)
			i += consumed
			continue
		}

		// positional
		pos = append(pos, tok)
		i++
	}

	// Required flags validation for *leaf only*
	leaf := path[len(path)-1]
	if leaf.Flags != nil {
		for _, f := range leaf.Flags.list() {
			if f.Required {
				if _, ok := values[f.Name]; !ok || fmt.Sprint(values[f.Name]) == "" {
					return values, pos, newUsagef("missing required flag --%s", f.Name)
				}
			}
		}
	}
	return values, pos, nil
}

// lookupLong resolves a long flag name against the effective definitions,
// normalizing a "no-" prefix the same way for every flag type so that every
// argv scanner agrees on which definition a token refers to and whether it
// consumes a value. It returns the definition, the normalized name, and
// whether the prefix was present.
func lookupLong(longLkp map[string]*Flag, name string) (f *Flag, norm string, neg bool, ok bool) {
	norm = name
	if strings.HasPrefix(norm, "no-") {
		neg = true
		norm = strings.TrimPrefix(norm, "no-")
	}
	f, ok = longLkp[norm]
	return f, norm, neg, ok
}

// flagLookups builds the effective long- and short-name lookup tables for a
// command path. Definitions are visited root-to-leaf so a leaf-local flag
// shadows a persistent flag of the same name, matching mergeFlagDefaults.
func flagLookups(path []*Command) (long, short map[string]*Flag) {
	long = map[string]*Flag{}
	short = map[string]*Flag{}
	for _, f := range collectFlags(path) {
		long[f.Name] = f
		if f.Short != "" {
			short[f.Short] = f
		}
	}
	return long, short
}

func (p *Parser) appInfo() AppInfo {
	return AppInfo{
		Name:        p.cfg.App.Name,
		Description: p.cfg.App.Description,
		Version:     p.cfg.App.Version,
		Show:        p.cfg.ShowInfo,
		Colored:     p.cfg.Colored,
	}
}

// setFlagValue stores a command-line value. String arrays accumulate across
// repeated occurrences; the first occurrence replaces any lower-layer value
// with a freshly cast slice that the parser owns, so later occurrences can
// append in place without copying.
func setFlagValue(f *Flag, values map[string]any, sources map[string]Source, v any) {
	if f.Type == FlagStringArray && sources[f.Name] == SourceFlag {
		prev, _ := values[f.Name].([]string)
		add, _ := v.([]string)
		values[f.Name] = append(prev, add...)
		return
	}
	values[f.Name] = v
	sources[f.Name] = SourceFlag
}

// small helpers
func splitEq(s string) (name, val string, hasEq bool) {
	parts := strings.SplitN(s, "=", 2)
	if len(parts) == 2 {
		return parts[0], parts[1], true
	}
	return s, "", false
}

func validateArgs(spec ArgSpec, args []string) error {
	n := len(args)
	if n < spec.Min {
		return newUsagef("requires at least %d arg(s), got %d", spec.Min, n)
	}
	if spec.Max > 0 && n > spec.Max {
		return newUsagef("accepts at most %d arg(s), got %d", spec.Max, n)
	}
	if spec.Max > 0 && len(spec.Names) > 0 && len(spec.Names) != spec.Max {
		// purely a dev sanity check for better help text; not exposed to users
	}
	return nil
}
