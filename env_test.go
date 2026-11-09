package stencil

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"
)

func newEnvApp() (*App, *captured, *bytes.Buffer) {
	cap := &captured{}
	fs := NewFlagSet()
	fs.Int("port", "p", "port", 1).Env = "STENCIL_ENVT_PORT"
	fs.Duration("wait", "", "wait", time.Second).Env = "STENCIL_ENVT_WAIT"
	fs.Bool("on", "", "on", false).Env = "STENCIL_ENVT_ON"
	fs.String("mode", "", "mode", "a").Env = "STENCIL_ENVT_MODE"
	e := fs.String("enum", "", "enum", "x")
	e.Env = "STENCIL_ENVT_ENUM"
	e.Enum = []string{"x", "y"}
	v := fs.String("val", "", "val", "ok")
	v.Env = "STENCIL_ENVT_VAL"
	v.Validate = func(a any) error {
		if a.(string) == "bad" {
			return fmt.Errorf("val is bad")
		}
		return nil
	}
	root := &Command{Name: "app", Flags: fs, Run: func(ctx *Context) error {
		cap.flags = ctx.Flags
		return nil
	}}
	var out, errb bytes.Buffer
	return NewApp(WithName("app"), WithRootCommand(root), WithIO(nil, &out, &errb), WithColorMode(ColorOff)), cap, &errb
}

func TestEnvInvalidIsUsageError(t *testing.T) {
	cases := []struct{ env, val, want string }{
		{"STENCIL_ENVT_PORT", "abc", "invalid env STENCIL_ENVT_PORT for --port"},
		{"STENCIL_ENVT_WAIT", "soon", "invalid env STENCIL_ENVT_WAIT for --wait"},
		{"STENCIL_ENVT_ON", "maybe", "invalid env STENCIL_ENVT_ON for --on"},
		{"STENCIL_ENVT_ENUM", "z", "invalid env STENCIL_ENVT_ENUM for --enum"},
		{"STENCIL_ENVT_VAL", "bad", "val is bad"},
	}
	for _, c := range cases {
		t.Run(c.env, func(t *testing.T) {
			t.Setenv(c.env, c.val)
			app, cap, errb := newEnvApp()
			if code := app.Execute(nil); code != int(ExitUsage) {
				t.Fatalf("exit %d, want usage", code)
			}
			if cap.flags != nil {
				t.Fatal("Run should not have been called")
			}
			if !strings.Contains(errb.String(), c.want) {
				t.Fatalf("stderr %q missing %q", errb.String(), c.want)
			}
		})
	}
}

func TestEnvBadEvenWhenArgvOverrides(t *testing.T) {
	t.Setenv("STENCIL_ENVT_PORT", "abc")
	app, _, _ := newEnvApp()
	if code := app.Execute([]string{"--port", "5"}); code != int(ExitUsage) {
		t.Fatalf("exit %d, want usage", code)
	}
}

func TestEnvValidCast(t *testing.T) {
	t.Setenv("STENCIL_ENVT_PORT", "8080")
	t.Setenv("STENCIL_ENVT_WAIT", "2s")
	t.Setenv("STENCIL_ENVT_ON", "true")
	app, cap, errb := newEnvApp()
	if code := app.Execute(nil); code != int(ExitOK) {
		t.Fatalf("exit %d: %s", code, errb.String())
	}
	f := cap.flags
	if f.Int("port") != 8080 || f.Duration("wait") != 2*time.Second || !f.Bool("on") {
		t.Fatalf("bad cast: %d %v %v", f.Int("port"), f.Duration("wait"), f.Bool("on"))
	}
	for _, n := range []string{"port", "wait", "on"} {
		if f.Source(n) != SourceEnv {
			t.Fatalf("%s source %v", n, f.Source(n))
		}
	}
}

func TestEnvEmptyIgnoredAndArgvOverrides(t *testing.T) {
	t.Setenv("STENCIL_ENVT_PORT", "")
	t.Setenv("STENCIL_ENVT_WAIT", "2s")
	app, cap, errb := newEnvApp()
	if code := app.Execute([]string{"--wait", "9s"}); code != int(ExitOK) {
		t.Fatalf("exit %d: %s", code, errb.String())
	}
	if cap.flags.Int("port") != 1 || cap.flags.Source("port") == SourceEnv {
		t.Fatal("empty env should be ignored")
	}
	if cap.flags.Duration("wait") != 9*time.Second {
		t.Fatalf("argv should override env: %v", cap.flags.Duration("wait"))
	}
}
