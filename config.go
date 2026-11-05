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

	flags := collectFlags(path)
	var cfgFlag *Flag
	for _, f := range flags {
		if f.Name == app.configFlag {
			cfgFlag = f
			break
		}
	}
	if cfgFlag == nil {
		// The config flag is not reachable on this command path.
		return nil
	}

	cfgPath, explicit := resolveConfigPath(cfgFlag, argv)
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

	for _, f := range flags {
		if f.ConfigKey == "" {
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
// (stopping at "--"), then the flag's Env variable, then its default.
// explicit is true when the path came from argv or the environment.
func resolveConfigPath(f *Flag, argv []string) (path string, explicit bool) {
	for i := 0; i < len(argv); i++ {
		tok := argv[i]
		if tok == "--" {
			break
		}
		var body string
		var match bool
		switch {
		case strings.HasPrefix(tok, "--"):
			body = strings.TrimPrefix(tok, "--")
			name, val, hasEq := splitEq(body)
			if name != f.Name {
				continue
			}
			if hasEq {
				return val, true
			}
			match = true
		case f.Short != "" && strings.HasPrefix(tok, "-"):
			body = strings.TrimPrefix(tok, "-")
			name, val, hasEq := splitEq(body)
			if name != f.Short {
				continue
			}
			if hasEq {
				return val, true
			}
			match = true
		}
		if match && i+1 < len(argv) {
			return argv[i+1], true
		}
	}
	if f.Env != "" {
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
