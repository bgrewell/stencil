package stencil

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type captured struct {
	flags *ResolvedFlags
}

func newTestApp(t *testing.T, withConfig bool) (*App, *captured, *bytes.Buffer) {
	t.Helper()
	cap := &captured{}

	persistent := NewFlagSet()
	cfg := persistent.String("config", "c", "config file", "")
	cfg.Env = "STENCIL_TEST_CONFIG"

	local := NewFlagSet()
	w := local.Int("workers", "w", "worker count", 1)
	w.ConfigKey = "upf.workers"
	w.Env = "STENCIL_TEST_WORKERS"
	m := local.String("mode", "m", "datapath mode", "af_packet")
	m.ConfigKey = "upf.mode"
	d := local.Duration("duration", "d", "test duration", 10*time.Second)
	d.ConfigKey = "traffic.duration"
	s := local.IntSlice("sizes", "", "packet sizes", []int{64})
	s.ConfigKey = "traffic.sizes"
	g := local.StringSlice("gens", "", "generators", nil)
	g.ConfigKey = "traffic.generators"
	local.Bool("verbose", "v", "verbose", false)

	run := &Command{
		Name:  "run",
		Flags: local,
		Run: func(ctx *Context) error {
			cap.flags = ctx.Flags
			return nil
		},
	}
	root := &Command{Name: "app", PersistentFlags: persistent, Sub: []*Command{run}}

	var out, errb bytes.Buffer
	opts := []Option{WithName("app"), WithRootCommand(root), WithIO(nil, &out, &errb), WithColorMode(ColorOff)}
	if withConfig {
		opts = append(opts, WithConfigFile("config", YAMLLoader))
	}
	return NewApp(opts...), cap, &errb
}

func writeConfig(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "cfg.yaml")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

const sampleConfig = `
upf:
  workers: 4
  mode: af_xdp
traffic:
  duration: 30s
  sizes: [64, 512, 1400]
  generators: [loom, iperf3]
`

func TestProvenanceDefaultsAndFlags(t *testing.T) {
	app, cap, errb := newTestApp(t, false)
	if code := app.Execute([]string{"run", "--workers", "2", "-v"}); code != int(ExitOK) {
		t.Fatalf("exit %d: %s", code, errb.String())
	}
	f := cap.flags
	if f.Int("workers") != 2 || f.Source("workers") != SourceFlag || !f.Changed("workers") {
		t.Fatalf("workers: got %d from %v", f.Int("workers"), f.Source("workers"))
	}
	if !f.Bool("verbose") || f.Source("verbose") != SourceFlag {
		t.Fatalf("verbose: got %v from %v", f.Bool("verbose"), f.Source("verbose"))
	}
	if f.String("mode") != "af_packet" || f.Changed("mode") {
		t.Fatalf("mode: got %q from %v", f.String("mode"), f.Source("mode"))
	}
	if f.Source("nonexistent") != SourceDefault {
		t.Fatalf("unknown flag should report default source")
	}
}

func TestProvenanceBundledShortBools(t *testing.T) {
	app, cap, errb := newTestApp(t, false)
	if code := app.Execute([]string{"run", "-v"}); code != int(ExitOK) {
		t.Fatalf("exit %d: %s", code, errb.String())
	}
	if cap.flags.Source("verbose") != SourceFlag {
		t.Fatalf("verbose source = %v", cap.flags.Source("verbose"))
	}
}

func TestProvenanceEnv(t *testing.T) {
	t.Setenv("STENCIL_TEST_WORKERS", "8")
	app, cap, errb := newTestApp(t, false)
	if code := app.Execute([]string{"run"}); code != int(ExitOK) {
		t.Fatalf("exit %d: %s", code, errb.String())
	}
	if cap.flags.Int("workers") != 8 || cap.flags.Source("workers") != SourceEnv {
		t.Fatalf("workers: got %d from %v", cap.flags.Int("workers"), cap.flags.Source("workers"))
	}
}

func TestConfigFileApplied(t *testing.T) {
	p := writeConfig(t, sampleConfig)
	app, cap, errb := newTestApp(t, true)
	if code := app.Execute([]string{"run", "-c", p}); code != int(ExitOK) {
		t.Fatalf("exit %d: %s", code, errb.String())
	}
	f := cap.flags
	if f.Int("workers") != 4 || f.Source("workers") != SourceConfig {
		t.Fatalf("workers: got %d from %v", f.Int("workers"), f.Source("workers"))
	}
	if f.String("mode") != "af_xdp" || f.Source("mode") != SourceConfig {
		t.Fatalf("mode: got %q from %v", f.String("mode"), f.Source("mode"))
	}
	if f.Duration("duration") != 30*time.Second {
		t.Fatalf("duration: got %v", f.Duration("duration"))
	}
	if got := f.IntSlice("sizes"); len(got) != 3 || got[2] != 1400 {
		t.Fatalf("sizes: got %v", got)
	}
	if got := f.StringSlice("gens"); len(got) != 2 || got[1] != "iperf3" {
		t.Fatalf("gens: got %v", got)
	}
	if f.Source("config") != SourceFlag {
		t.Fatalf("config flag source = %v", f.Source("config"))
	}
}

func TestConfigPrecedence(t *testing.T) {
	p := writeConfig(t, sampleConfig)

	// env beats config
	t.Setenv("STENCIL_TEST_WORKERS", "6")
	app, cap, errb := newTestApp(t, true)
	if code := app.Execute([]string{"run", "--config=" + p}); code != int(ExitOK) {
		t.Fatalf("exit %d: %s", code, errb.String())
	}
	if cap.flags.Int("workers") != 6 || cap.flags.Source("workers") != SourceEnv {
		t.Fatalf("env should beat config: got %d from %v", cap.flags.Int("workers"), cap.flags.Source("workers"))
	}

	// flag beats env and config
	app, cap, errb = newTestApp(t, true)
	if code := app.Execute([]string{"run", "-c", p, "-w", "12"}); code != int(ExitOK) {
		t.Fatalf("exit %d: %s", code, errb.String())
	}
	if cap.flags.Int("workers") != 12 || cap.flags.Source("workers") != SourceFlag {
		t.Fatalf("flag should beat env: got %d from %v", cap.flags.Int("workers"), cap.flags.Source("workers"))
	}
	// untouched config-backed flag still comes from config
	if cap.flags.Source("mode") != SourceConfig {
		t.Fatalf("mode source = %v", cap.flags.Source("mode"))
	}
}

func TestConfigPathFromEnv(t *testing.T) {
	p := writeConfig(t, sampleConfig)
	t.Setenv("STENCIL_TEST_CONFIG", p)
	app, cap, errb := newTestApp(t, true)
	if code := app.Execute([]string{"run"}); code != int(ExitOK) {
		t.Fatalf("exit %d: %s", code, errb.String())
	}
	if cap.flags.Source("workers") != SourceConfig {
		t.Fatalf("workers source = %v", cap.flags.Source("workers"))
	}
}

func TestConfigMissingExplicitIsError(t *testing.T) {
	app, _, errb := newTestApp(t, true)
	code := app.Execute([]string{"run", "-c", filepath.Join(t.TempDir(), "nope.yaml")})
	if code != int(ExitUsage) {
		t.Fatalf("expected usage exit, got %d", code)
	}
	if !strings.Contains(errb.String(), "config:") {
		t.Fatalf("expected config error, got %q", errb.String())
	}
}

func TestConfigMissingDefaultIgnored(t *testing.T) {
	app, cap, errb := newTestApp(t, true)
	// Point the config flag's default at a file that does not exist.
	app.Root.PersistentFlags.list()[0].Default = filepath.Join(t.TempDir(), "absent.yaml")
	if code := app.Execute([]string{"run"}); code != int(ExitOK) {
		t.Fatalf("exit %d: %s", code, errb.String())
	}
	if cap.flags.Changed("workers") {
		t.Fatalf("workers should be default")
	}
}

func TestConfigBadValueIsError(t *testing.T) {
	p := writeConfig(t, "upf:\n  workers: lots\n")
	app, _, errb := newTestApp(t, true)
	if code := app.Execute([]string{"run", "-c", p}); code != int(ExitUsage) {
		t.Fatalf("expected usage exit, got %d", code)
	}
	if !strings.Contains(errb.String(), "upf.workers") {
		t.Fatalf("error should name the key, got %q", errb.String())
	}
}

func TestConfigDisabledIgnoresKeys(t *testing.T) {
	p := writeConfig(t, sampleConfig)
	app, cap, errb := newTestApp(t, false)
	if code := app.Execute([]string{"run", "-c", p}); code != int(ExitOK) {
		t.Fatalf("exit %d: %s", code, errb.String())
	}
	if cap.flags.Changed("workers") {
		t.Fatalf("config layer should be inactive without WithConfigFile")
	}
}

func TestLookupConfigKey(t *testing.T) {
	data := map[string]any{"a": map[string]any{"b": map[any]any{"c": 3}}, "x": 1}
	if v, ok := lookupConfigKey(data, "a.b.c"); !ok || v != 3 {
		t.Fatalf("a.b.c = %v %v", v, ok)
	}
	if _, ok := lookupConfigKey(data, "x.y"); ok {
		t.Fatalf("x.y should not resolve through a scalar")
	}
	if _, ok := lookupConfigKey(data, "a.missing"); ok {
		t.Fatalf("missing key should not resolve")
	}
}

func newArrayApp(t *testing.T) (*App, *captured, *bytes.Buffer) {
	t.Helper()
	cap := &captured{}
	fs := NewFlagSet()
	fs.StringArray("set", "s", "key=value override (repeatable)", nil).Env = "STENCIL_TEST_SET"
	run := &Command{Name: "run", Flags: fs, Run: func(ctx *Context) error { cap.flags = ctx.Flags; return nil }}
	root := &Command{Name: "app", Sub: []*Command{run}}
	var out, errb bytes.Buffer
	return NewApp(WithName("app"), WithRootCommand(root), WithIO(nil, &out, &errb), WithColorMode(ColorOff)), cap, &errb
}

func TestStringArrayAccumulatesWithoutSplitting(t *testing.T) {
	app, cap, errb := newArrayApp(t)
	argv := []string{"run", "--set", "a.b=[1,2]", "-s", "c=x,y", "--set=d=e"}
	if code := app.Execute(argv); code != int(ExitOK) {
		t.Fatalf("exit %d: %s", code, errb.String())
	}
	got := cap.flags.StringArray("set")
	want := []string{"a.b=[1,2]", "c=x,y", "d=e"}
	if len(got) != len(want) {
		t.Fatalf("got %q, want %q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %q, want %q", got, want)
		}
	}
	if cap.flags.Source("set") != SourceFlag {
		t.Fatalf("source = %v", cap.flags.Source("set"))
	}
}

func TestStringArrayFlagReplacesEnv(t *testing.T) {
	t.Setenv("STENCIL_TEST_SET", "from=env")
	app, cap, errb := newArrayApp(t)
	if code := app.Execute([]string{"run"}); code != int(ExitOK) {
		t.Fatalf("exit %d: %s", code, errb.String())
	}
	if got := cap.flags.StringArray("set"); len(got) != 1 || got[0] != "from=env" {
		t.Fatalf("env value: got %q", got)
	}

	app, cap, errb = newArrayApp(t)
	if code := app.Execute([]string{"run", "--set", "x=1", "--set", "y=2"}); code != int(ExitOK) {
		t.Fatalf("exit %d: %s", code, errb.String())
	}
	if got := cap.flags.StringArray("set"); len(got) != 2 || got[0] != "x=1" {
		t.Fatalf("flags should replace env, then accumulate: got %q", got)
	}
}

func TestConfigRepeatedFlagKeepsLast(t *testing.T) {
	first := writeConfig(t, "upf:\n  workers: 1\n")
	second := filepath.Join(t.TempDir(), "second.yaml")
	if err := os.WriteFile(second, []byte("upf:\n  workers: 2\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name string
		argv []string
	}{
		{"long-eq", []string{"run", "--config=" + first, "--config=" + second}},
		{"long-space", []string{"run", "--config", first, "--config", second}},
		{"short", []string{"run", "-c", first, "-c", second}},
		{"mixed", []string{"run", "-c=" + first, "--config", second}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app, cap, errb := newTestApp(t, true)
			if code := app.Execute(tc.argv); code != int(ExitOK) {
				t.Fatalf("exit %d: %s", code, errb.String())
			}
			if cap.flags.String("config") != second {
				t.Fatalf("config flag = %q, want %q", cap.flags.String("config"), second)
			}
			if got := cap.flags.Int("workers"); got != 2 || cap.flags.Source("workers") != SourceConfig {
				t.Fatalf("workers = %d from %v, want 2 from config", got, cap.flags.Source("workers"))
			}
		})
	}
}

func TestConfigScanSkipsConsumedValues(t *testing.T) {
	real := writeConfig(t, sampleConfig)

	t.Run("value of another flag is not an option", func(t *testing.T) {
		app, cap, errb := newTestApp(t, true)
		argv := []string{"run", "--mode", "--config=phantom.yaml", "-c", real}
		if code := app.Execute(argv); code != int(ExitOK) {
			t.Fatalf("exit %d: %s", code, errb.String())
		}
		if got := cap.flags.String("mode"); got != "--config=phantom.yaml" {
			t.Fatalf("mode = %q", got)
		}
		if cap.flags.Source("workers") != SourceConfig {
			t.Fatalf("real config should have loaded: workers from %v", cap.flags.Source("workers"))
		}
	})

	t.Run("short form value is not an option", func(t *testing.T) {
		app, cap, errb := newTestApp(t, true)
		if code := app.Execute([]string{"run", "-m", "-c"}); code != int(ExitOK) {
			t.Fatalf("exit %d: %s", code, errb.String())
		}
		if got := cap.flags.String("mode"); got != "-c" {
			t.Fatalf("mode = %q", got)
		}
		if cap.flags.Changed("workers") {
			t.Fatalf("no config should have loaded")
		}
	})

	t.Run("consumed double-dash does not end scan", func(t *testing.T) {
		app, cap, errb := newTestApp(t, true)
		argv := []string{"run", "--mode", "--", "-c", real}
		if code := app.Execute(argv); code != int(ExitOK) {
			t.Fatalf("exit %d: %s", code, errb.String())
		}
		if got := cap.flags.String("mode"); got != "--" {
			t.Fatalf("mode = %q", got)
		}
		if cap.flags.Source("workers") != SourceConfig {
			t.Fatalf("config after consumed -- should load: workers from %v", cap.flags.Source("workers"))
		}
	})

	t.Run("unconsumed double-dash ends scan", func(t *testing.T) {
		app, cap, errb := newTestApp(t, true)
		argv := []string{"run", "-v", "--", "-c", real}
		if code := app.Execute(argv); code != int(ExitOK) {
			t.Fatalf("exit %d: %s", code, errb.String())
		}
		if cap.flags.Changed("workers") {
			t.Fatalf("config after bare -- must not load")
		}
		if len(cap.flags.values) == 0 {
			t.Fatal("no flags captured")
		}
	})

	t.Run("bool forms consume nothing", func(t *testing.T) {
		app, cap, errb := newTestApp(t, true)
		argv := []string{"run", "--no-verbose", "-v", "--verbose", "-c", real}
		if code := app.Execute(argv); code != int(ExitOK) {
			t.Fatalf("exit %d: %s", code, errb.String())
		}
		if cap.flags.Source("workers") != SourceConfig {
			t.Fatalf("config should load after bool flags: workers from %v", cap.flags.Source("workers"))
		}
	})
}

func TestConfigShadowedFlagUsesLeafDefinition(t *testing.T) {
	dir := t.TempDir()
	rootCfg := filepath.Join(dir, "root.yaml")
	leafCfg := filepath.Join(dir, "leaf.yaml")
	if err := os.WriteFile(rootCfg, []byte("upf:\n  workers: 10\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(leafCfg, []byte("upf:\n  workers: 20\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	cap := &captured{}
	persistent := NewFlagSet()
	persistent.String("config", "c", "config file", rootCfg)
	local := NewFlagSet()
	local.String("config", "c", "config file", leafCfg)
	local.Int("workers", "w", "worker count", 1).ConfigKey = "upf.workers"
	run := &Command{Name: "run", Flags: local, Run: func(ctx *Context) error { cap.flags = ctx.Flags; return nil }}
	root := &Command{Name: "app", PersistentFlags: persistent, Sub: []*Command{run}}
	var out, errb bytes.Buffer
	app := NewApp(WithName("app"), WithRootCommand(root), WithIO(nil, &out, &errb),
		WithColorMode(ColorOff), WithConfigFile("config", YAMLLoader))

	if code := app.Execute([]string{"run"}); code != int(ExitOK) {
		t.Fatalf("exit %d: %s", code, errb.String())
	}
	if got := cap.flags.String("config"); got != leafCfg {
		t.Fatalf("resolved config = %q, want leaf default %q", got, leafCfg)
	}
	if got := cap.flags.Int("workers"); got != 20 {
		t.Fatalf("workers = %d, want 20 (from leaf.yaml)", got)
	}
}

func TestConfigShadowedBindingIgnored(t *testing.T) {
	p := writeConfig(t, "root:\n  workers: 4\nleaf:\n  workers: lots\n")

	cap := &captured{}
	persistent := NewFlagSet()
	persistent.String("config", "c", "config file", "")
	persistent.Int("workers", "", "root workers", 2).ConfigKey = "root.workers"
	local := NewFlagSet()
	local.Int("workers", "w", "leaf workers", 1)
	run := &Command{Name: "run", Flags: local, Run: func(ctx *Context) error { cap.flags = ctx.Flags; return nil }}
	root := &Command{Name: "app", PersistentFlags: persistent, Sub: []*Command{run}}
	var out, errb bytes.Buffer
	app := NewApp(WithName("app"), WithRootCommand(root), WithIO(nil, &out, &errb),
		WithColorMode(ColorOff), WithConfigFile("config", YAMLLoader))

	// The leaf definition has no ConfigKey, so the shadowed persistent
	// binding must not populate it.
	if code := app.Execute([]string{"run", "-c", p}); code != int(ExitOK) {
		t.Fatalf("exit %d: %s", code, errb.String())
	}
	if got := cap.flags.Int("workers"); got != 1 || cap.flags.Source("workers") != SourceDefault {
		t.Fatalf("workers = %d from %v, want 1 from default", got, cap.flags.Source("workers"))
	}

	// A shadowed binding with an invalid value must not invalidate the
	// invocation either.
	local.list()[0].ConfigKey = ""
	persistent.list()[1].ConfigKey = "leaf.workers"
	app = NewApp(WithName("app"), WithRootCommand(root), WithIO(nil, &out, &errb),
		WithColorMode(ColorOff), WithConfigFile("config", YAMLLoader))
	if code := app.Execute([]string{"run", "-c", p}); code != int(ExitOK) {
		t.Fatalf("shadowed bad value should be ignored: exit %d: %s", code, errb.String())
	}
	if got := cap.flags.Int("workers"); got != 1 {
		t.Fatalf("workers = %d, want 1", got)
	}
}

func TestConfigScanNormalizesNoPrefix(t *testing.T) {
	real := writeConfig(t, sampleConfig)
	app, cap, errb := newTestApp(t, true)
	// parseFlags strips "no-" for every long flag, so --no-mode is the
	// string flag "mode" and consumes the following token.
	argv := []string{"run", "--config=" + real, "--no-mode", "--config=phantom.yaml"}
	if code := app.Execute(argv); code != int(ExitOK) {
		t.Fatalf("exit %d: %s", code, errb.String())
	}
	if got := cap.flags.String("mode"); got != "--config=phantom.yaml" {
		t.Fatalf("mode = %q", got)
	}
	if got := cap.flags.String("config"); got != real {
		t.Fatalf("config = %q, want %q", got, real)
	}
	if cap.flags.Source("workers") != SourceConfig {
		t.Fatalf("real config should have loaded: workers from %v", cap.flags.Source("workers"))
	}
}

// newShadowedConfigApp builds an app whose leaf redefines the persistent
// "config" flag. leafShort and the Env bindings let tests vary the shape of
// the shadowing.
func newShadowedConfigApp(t *testing.T, rootDef, leafDef, leafShort, rootEnv, leafEnv string) (*App, *captured, *bytes.Buffer) {
	t.Helper()
	cap := &captured{}
	persistent := NewFlagSet()
	persistent.String("config", "c", "config file", rootDef).Env = rootEnv
	local := NewFlagSet()
	local.String("config", leafShort, "config file", leafDef).Env = leafEnv
	local.Int("workers", "w", "worker count", 1).ConfigKey = "upf.workers"
	run := &Command{Name: "run", Flags: local, Run: func(ctx *Context) error { cap.flags = ctx.Flags; return nil }}
	root := &Command{Name: "app", PersistentFlags: persistent, Sub: []*Command{run}}
	var out, errb bytes.Buffer
	app := NewApp(WithName("app"), WithRootCommand(root), WithIO(nil, &out, &errb),
		WithColorMode(ColorOff), WithConfigFile("config", YAMLLoader))
	return app, cap, &errb
}

func TestConfigInheritedShortFormSelectsLeafFlag(t *testing.T) {
	dir := t.TempDir()
	leafDef := filepath.Join(dir, "leaf.yaml")
	chosen := filepath.Join(dir, "chosen.yaml")
	if err := os.WriteFile(leafDef, []byte("upf:\n  workers: 20\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(chosen, []byte("upf:\n  workers: 30\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	// The leaf redefines config without a short form; the inherited -c
	// still sets "config" during parsing, so it must select the file too.
	app, cap, errb := newShadowedConfigApp(t, "", leafDef, "", "", "")
	if code := app.Execute([]string{"run", "-c", chosen}); code != int(ExitOK) {
		t.Fatalf("exit %d: %s", code, errb.String())
	}
	if got := cap.flags.String("config"); got != chosen {
		t.Fatalf("config = %q, want %q", got, chosen)
	}
	if got := cap.flags.Int("workers"); got != 30 {
		t.Fatalf("workers = %d, want 30 (from chosen.yaml)", got)
	}

	// An explicitly requested missing file via the inherited -c is an error,
	// not a silent fall back to the leaf default.
	app, _, errb = newShadowedConfigApp(t, "", leafDef, "", "", "")
	if code := app.Execute([]string{"run", "-c", filepath.Join(dir, "nope.yaml")}); code != int(ExitUsage) {
		t.Fatalf("expected usage exit for missing explicit config, got %d: %s", code, errb.String())
	}
}

func TestEnvUsesEffectiveDefinition(t *testing.T) {
	dir := t.TempDir()
	rootCfg := filepath.Join(dir, "root.yaml")
	leafCfg := filepath.Join(dir, "leaf.yaml")
	if err := os.WriteFile(rootCfg, []byte("upf:\n  workers: 10\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(leafCfg, []byte("upf:\n  workers: 20\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("STENCIL_TEST_ROOT_CONFIG", rootCfg)
	t.Setenv("STENCIL_TEST_LEAF_CONFIG", leafCfg)

	app, cap, errb := newShadowedConfigApp(t, "", "", "c", "STENCIL_TEST_ROOT_CONFIG", "STENCIL_TEST_LEAF_CONFIG")
	if code := app.Execute([]string{"run"}); code != int(ExitOK) {
		t.Fatalf("exit %d: %s", code, errb.String())
	}
	// The resolved value, its provenance, and the file actually loaded must
	// all come from the leaf definition.
	if got := cap.flags.String("config"); got != leafCfg || cap.flags.Source("config") != SourceEnv {
		t.Fatalf("config = %q from %v, want %q from env", got, cap.flags.Source("config"), leafCfg)
	}
	if got := cap.flags.Int("workers"); got != 20 {
		t.Fatalf("workers = %d, want 20 (from leaf.yaml)", got)
	}

	// A shadowed persistent Env binding must not apply when the leaf has none.
	app, cap, errb = newShadowedConfigApp(t, "", "", "c", "STENCIL_TEST_ROOT_CONFIG", "")
	if code := app.Execute([]string{"run"}); code != int(ExitOK) {
		t.Fatalf("exit %d: %s", code, errb.String())
	}
	if cap.flags.Changed("config") || cap.flags.Changed("workers") {
		t.Fatalf("shadowed env binding applied: config=%q workers=%d", cap.flags.String("config"), cap.flags.Int("workers"))
	}
}

func TestStringArrayManyOccurrences(t *testing.T) {
	app, cap, errb := newArrayApp(t)
	argv := []string{"run"}
	want := make([]string, 0, 200)
	for i := 0; i < 200; i++ {
		v := fmt.Sprintf("k%d=v%d", i, i)
		argv = append(argv, "--set", v)
		want = append(want, v)
	}
	if code := app.Execute(argv); code != int(ExitOK) {
		t.Fatalf("exit %d: %s", code, errb.String())
	}
	got := cap.flags.StringArray("set")
	if len(got) != len(want) {
		t.Fatalf("got %d values, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("index %d: got %q, want %q", i, got[i], want[i])
		}
	}
}

func TestConfigHiddenFlagIgnoresEnv(t *testing.T) {
	p := writeConfig(t, sampleConfig)
	t.Setenv("STENCIL_TEST_CONFIG", p)
	app, cap, errb := newTestApp(t, true)
	// applyEnv skips hidden flags, so the path scan must too; otherwise the
	// file would load while the flag still reports its default.
	app.Root.PersistentFlags.list()[0].Hidden = true
	if code := app.Execute([]string{"run"}); code != int(ExitOK) {
		t.Fatalf("exit %d: %s", code, errb.String())
	}
	if cap.flags.Changed("config") || cap.flags.Changed("workers") {
		t.Fatalf("hidden config flag must ignore env: config=%q workers from %v",
			cap.flags.String("config"), cap.flags.Source("workers"))
	}

	// Explicit command-line values still work for a hidden flag.
	app, cap, errb = newTestApp(t, true)
	app.Root.PersistentFlags.list()[0].Hidden = true
	if code := app.Execute([]string{"run", "-c", p}); code != int(ExitOK) {
		t.Fatalf("exit %d: %s", code, errb.String())
	}
	if cap.flags.String("config") != p || cap.flags.Source("workers") != SourceConfig {
		t.Fatalf("hidden config flag via argv: config=%q workers from %v",
			cap.flags.String("config"), cap.flags.Source("workers"))
	}
}

func TestConfigFlagNeverBindsToItself(t *testing.T) {
	p := writeConfig(t, "config: other.yaml\nupf:\n  workers: 4\n")
	app, cap, errb := newTestApp(t, true)
	cfg := app.Root.PersistentFlags.list()[0]
	cfg.ConfigKey = "config"
	// Resolve the path from the default so no later layer masks a
	// self-binding overwrite.
	cfg.Default = p
	if code := app.Execute([]string{"run"}); code != int(ExitOK) {
		t.Fatalf("exit %d: %s", code, errb.String())
	}
	// The reported path and provenance must describe the file actually
	// loaded, not a value found inside it.
	if got := cap.flags.String("config"); got != p || cap.flags.Source("config") != SourceDefault {
		t.Fatalf("config = %q from %v, want %q from default", got, cap.flags.Source("config"), p)
	}
	if cap.flags.Int("workers") != 4 {
		t.Fatalf("workers = %d, want 4", cap.flags.Int("workers"))
	}
}
