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
	values := mergeFlagDefaults(path)
	// Apply env if set
	applyEnv(path, values)

	// Parse flags and positionals
	vals, pos, err := p.parseFlags(path, rest, values)
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
		Flags: &ResolvedFlags{values: vals},
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
func (p *Parser) parseFlags(path []*Command, argv []string, defaults map[string]any) (map[string]any, []string, error) {
	longLkp := map[string]*Flag{}
	shortLkp := map[string]*Flag{}
	for _, f := range collectFlags(path) {
		longLkp[f.Name] = f
		if f.Short != "" {
			shortLkp[f.Short] = f
		}
	}

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

			// support --no-flag for bools
			neg := false
			if strings.HasPrefix(name, "no-") {
				neg = true
				name = strings.TrimPrefix(name, "no-")
			}

			f, ok := longLkp[name]
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
			values[f.Name] = casted
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
			values[f.Name] = casted
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

func (p *Parser) appInfo() AppInfo {
	return AppInfo{
		Name:        p.cfg.App.Name,
		Description: p.cfg.App.Description,
		Version:     p.cfg.App.Version,
		Show:        p.cfg.ShowInfo,
		Colored:     p.cfg.Colored,
	}
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
