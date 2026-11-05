package stencil

import (
	"bytes"
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
