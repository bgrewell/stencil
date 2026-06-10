package stencil

import (
	"bytes"
	"strings"
	"testing"
)

func TestRenderHelpEnvVarsSection(t *testing.T) {
	fs := NewFlagSet()
	f := fs.String("token", "t", "API token", "")
	f.Env = "APP_TOKEN"
	fs.Bool("verbose", "v", "verbose output", false) // no Env: must not appear

	root := &Command{Name: "app", Summary: "demo", Flags: fs}

	var buf bytes.Buffer
	renderHelp(&buf, AppInfo{Name: "app"}, []*Command{root})
	out := buf.String()

	if !strings.Contains(out, "Environment Variables:") {
		t.Fatalf("missing env section:\n%s", out)
	}
	if !strings.Contains(out, "APP_TOKEN") || !strings.Contains(out, "(sets --token)") {
		t.Fatalf("env var not rendered correctly:\n%s", out)
	}
	if strings.Contains(out, "verbose output (sets") {
		t.Fatalf("flag without Env should not appear in env section:\n%s", out)
	}
}

func TestRenderHelpNoEnvSectionWhenNone(t *testing.T) {
	fs := NewFlagSet()
	fs.Bool("verbose", "v", "verbose output", false)
	root := &Command{Name: "app", Summary: "demo", Flags: fs}

	var buf bytes.Buffer
	renderHelp(&buf, AppInfo{Name: "app"}, []*Command{root})
	if strings.Contains(buf.String(), "Environment Variables:") {
		t.Fatalf("env section should be omitted when no flags bind env vars:\n%s", buf.String())
	}
}
