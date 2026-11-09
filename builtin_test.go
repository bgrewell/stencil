package stencil

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
)

func runBuiltin(t *testing.T, setup func(root, sub *Command), argv ...string) (code int, out string, got []string, ran bool) {
	t.Helper()
	var ob, eb bytes.Buffer
	app := NewApp(WithName("app"), WithIO(nil, &ob, &eb), WithVersionInfo(VersionInfo{Version: "9.9.9"}))
	root := app.Root
	sub := &Command{Name: "run", Summary: "run it", Flags: NewFlagSet(), Run: func(c *Context) error {
		ran = true
		got = c.Args
		return nil
	}}
	root.Sub = append(root.Sub, sub)
	root.PersistentFlags.String("name", "n", "name", "")
	root.PersistentFlags.Bool("loud", "l", "loud", false)
	if setup != nil {
		setup(root, sub)
	}
	code = app.Execute(argv)
	return code, ob.String(), got, ran
}

func isVersion(out string) bool {
	return strings.Contains(out, "9.9.9") && !strings.Contains(out, "Usage:")
}
func isHelp(out string) bool {
	return strings.Contains(out, "Usage") || strings.Contains(out, "run it")
}

func TestBuiltinBeforeTerminator(t *testing.T) {
	for _, argv := range [][]string{
		{"--version"}, {"-V"}, {"run", "--version"}, {"run", "x", "-V"}, {"--loud", "run", "-V"}, {"run", "--name", "a", "-V"},
	} {
		code, out, _, ran := runBuiltin(t, nil, argv...)
		if code != 0 || !isVersion(out) || ran {
			t.Errorf("%v: code=%d ran=%v out=%q", argv, code, ran, out)
		}
	}
	for _, argv := range [][]string{{"-h"}, {"--help"}, {"run", "-h"}, {"run", "x", "--help"}} {
		code, out, _, ran := runBuiltin(t, nil, argv...)
		if code != 0 || !isHelp(out) || ran {
			t.Errorf("%v: code=%d ran=%v out=%q", argv, code, ran, out)
		}
	}
}

func TestBuiltinAfterTerminatorIsPositional(t *testing.T) {
	for _, tok := range []string{"--version", "-h", "-V", "--help"} {
		code, out, got, ran := runBuiltin(t, nil, "run", "--", tok)
		if code != 0 || !ran || !reflect.DeepEqual(got, []string{tok}) || out != "" {
			t.Errorf("%s: code=%d ran=%v got=%v out=%q", tok, code, ran, got, out)
		}
	}
}

func TestBuiltinAsFlagValue(t *testing.T) {
	for _, argv := range [][]string{
		{"run", "--name", "--version"}, {"run", "-n", "-h"}, {"run", "--name=--version"}, {"run", "-n=-h"},
	} {
		code, out, _, ran := runBuiltin(t, nil, argv...)
		if code != 0 || !ran || out != "" {
			t.Errorf("%v: code=%d ran=%v out=%q", argv, code, ran, out)
		}
	}
}

func TestBuiltinBoolFlagDoesNotSwallow(t *testing.T) {
	code, out, _, ran := runBuiltin(t, nil, "run", "--loud", "--version")
	if code != 0 || !isVersion(out) || ran {
		t.Errorf("code=%d ran=%v out=%q", code, ran, out)
	}
	code, out, _, ran = runBuiltin(t, nil, "run", "-l", "-h")
	if code != 0 || !isHelp(out) || ran {
		t.Errorf("code=%d ran=%v out=%q", code, ran, out)
	}
}

func TestBuiltinVersionWinsOverHelp(t *testing.T) {
	for _, argv := range [][]string{{"-h", "--version"}, {"--version", "-h"}, {"run", "--help", "-V"}} {
		_, out, _, _ := runBuiltin(t, nil, argv...)
		if !isVersion(out) {
			t.Errorf("%v: out=%q", argv, out)
		}
	}
}

func TestBuiltinUserShortVOverrides(t *testing.T) {
	var seen string
	setup := func(root, sub *Command) {
		sub.Flags.String("verb", "V", "verbosity", "")
		sub.Run = func(c *Context) error { seen, _ = c.Flags.values["verb"].(string); return nil }
	}
	code, out, _, _ := runBuiltin(t, setup, "run", "-V", "high")
	if code != 0 || out != "" || seen != "high" {
		t.Errorf("code=%d out=%q seen=%q", code, out, seen)
	}
}

func TestBuiltinUserHelpFlagOverrides(t *testing.T) {
	setup := func(root, sub *Command) {
		sub.Flags.Bool("help", "h", "custom help", false)
	}
	code, out, _, ran := runBuiltin(t, setup, "run", "-h")
	if code != 0 || out != "" || !ran {
		t.Errorf("-h: code=%d ran=%v out=%q", code, ran, out)
	}
	code, out, _, ran = runBuiltin(t, setup, "run", "--help")
	if code != 0 || out != "" || !ran {
		t.Errorf("--help: code=%d ran=%v out=%q", code, ran, out)
	}
}
