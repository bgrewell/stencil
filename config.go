package stencil

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// ConfigLoader reads the config file at path and returns its contents as a
// nested map. Nested sections are expected to decode as map[string]any.
// A loader must return an error wrapping fs.ErrNotExist when the file is
// missing so the parser can distinguish "absent default" from a bad path.
type ConfigLoader func(path string) (map[string]any, error)

// WithConfigFile enables the config-file layer. flagName names a String flag
// (persistent or local) that holds the path to the config file; load parses
// it. Flags that declare a ConfigKey are populated from the loaded file.
//
// Resolution order, lowest to highest: default < config < env < flag.
//
// The config path itself is resolved from the command line, then the flag's
// Env variable, then its default. A missing file is an error only when the
// path came from the command line or environment; a missing default path is
// silently ignored.
func WithConfigFile(flagName string, load ConfigLoader) Option {
	return func(a *App) {
		a.configFlag = flagName
		a.configLoader = load
	}
}

// YAMLLoader is a ConfigLoader for YAML files.
func YAMLLoader(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	out := map[string]any{}
	if err := yaml.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}
	return out, nil
}

// applyConfig loads the config file (if enabled and present) and applies
// values for flags that declare a ConfigKey.
func (p *Parser) applyConfig(path []*Command, argv []string, values map[string]any, sources map[string]Source) error {
	app := p.cfg.App
	if app.configFlag == "" || app.configLoader == nil {
		return nil
	}

	// Use the effective definitions so a leaf-local config flag that shadows
	// a persistent one wins, exactly as it does during flag resolution.
	longLkp, shortLkp := flagLookups(path)
	cfgFlag, ok := longLkp[app.configFlag]
	if !ok {
		// The config flag is not reachable on this command path.
		return nil
	}

	cfgPath, explicit := resolveConfigPath(cfgFlag, argv, longLkp, shortLkp)
	if cfgPath == "" {
		return nil
	}

	data, err := app.configLoader(cfgPath)
	if err != nil {
		if !explicit && errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return newUsagef("config: %v", err)
	}

	for _, f := range collectFlags(path) {
		// Only the effective definition may bind to the config file; a
		// shadowed persistent flag must not populate or invalidate the
		// leaf's value.
		if f.ConfigKey == "" || longLkp[f.Name] != f {
			continue
		}
		// The config flag never binds to its own file: the reported path
		// must always be the file that was actually loaded.
		if f.Name == app.configFlag {
			continue
		}
		raw, ok := lookupConfigKey(data, f.ConfigKey)
		if !ok {
			continue
		}
		casted, err := castValue(f, raw)
		if err != nil {
			return newUsagef("config %s: key %q: %v", cfgPath, f.ConfigKey, err)
		}
		values[f.Name] = casted
		sources[f.Name] = SourceConfig
	}
	return nil
}

// resolveConfigPath finds the config file path for flag f. It scans argv
// with the same token-consumption rules as parseFlags so that values
// belonging to other flags are never mistaken for options, and keeps the
// last occurrence to match parseFlags' last-wins semantics. It then falls
// back to the flag's Env variable (unless the flag is hidden, matching
// applyEnv), then its default. explicit is true when the path came from argv
// or the environment.
func resolveConfigPath(f *Flag, argv []string, longLkp, shortLkp map[string]*Flag) (path string, explicit bool) {
	var found bool
	for i := 0; i < len(argv); {
		tok := argv[i]
		if tok == "--" {
			break
		}
		var (
			def      *Flag
			sval     string
			hasEq    bool
			neg      bool
			consumed = 1
		)
		switch {
		case strings.HasPrefix(tok, "--"):
			var name string
			name, sval, hasEq = splitEq(strings.TrimPrefix(tok, "--"))
			def, _, neg, _ = lookupLong(longLkp, name)
		case strings.HasPrefix(tok, "-") && len(tok) > 1:
			body := strings.TrimPrefix(tok, "-")
			if !strings.Contains(body, "=") && len(body) > 1 {
				// Bundled short bools never take a value.
				i++
				continue
			}
			var name string
			name, sval, hasEq = splitEq(body)
			def = shortLkp[name]
		default:
			// Positional.
			i++
			continue
		}
		if def == nil {
			// Unknown flag; parseFlags will reject it.
			i++
			continue
		}
		var val string
		switch {
		case def.Type == FlagBool && (neg || !hasEq):
			// Bare or negated bool consumes nothing.
		case hasEq:
			val = sval
		case i+1 < len(argv):
			val = argv[i+1]
			consumed = 2
		default:
			// Missing value; parseFlags will reject it.
		}
		// Compare canonical names rather than pointers: an inherited short
		// form may still resolve to a shadowed definition, but parseFlags
		// stores it under the same name.
		if def.Name == f.Name && (hasEq || consumed == 2) {
			path, found = val, true
		}
		i += consumed
	}
	if found {
		return path, true
	}
	// Honor Env under the same conditions as applyEnv so the resolved flag
	// value and the loaded file never disagree.
	if f.Env != "" && !f.Hidden {
		if v, ok := os.LookupEnv(f.Env); ok && v != "" {
			return v, true
		}
	}
	if s, ok := f.Default.(string); ok {
		return s, false
	}
	return "", false
}

// lookupConfigKey walks a dotted key through nested maps.
func lookupConfigKey(data map[string]any, key string) (any, bool) {
	var cur any = data
	for _, part := range strings.Split(key, ".") {
		switch m := cur.(type) {
		case map[string]any:
			v, ok := m[part]
			if !ok {
				return nil, false
			}
			cur = v
		case map[any]any:
			v, ok := m[part]
			if !ok {
				return nil, false
			}
			cur = v
		default:
			return nil, false
		}
	}
	return cur, true
}
