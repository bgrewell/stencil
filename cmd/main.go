// Command stencil is the developer tool for projects built with the stencil
// library. It manages a project's semantic version (.stencil/version_*) and the
// git hook that auto-bumps it on commit.
//
// It is distinct from the library: an app built with stencil imports the
// library for its own CLI/UI, while this binary is run by that app's developer
// in their repository. The installed pre-commit hook calls back into this
// binary to bump the version.
package main

import (
	"fmt"
	"os"

	"github.com/bgrewell/stencil"
	"github.com/bgrewell/stencil/internal/hooker"
	"github.com/bgrewell/stencil/internal/version"
)

func main() {
	os.Exit(buildApp().Execute(os.Args[1:]))
}

func buildApp() *stencil.App {
	root := &stencil.Command{
		Name:            "stencil",
		Summary:         "Stencil developer tooling",
		Long:            "Manage project versioning and git hooks for apps built with stencil.",
		PersistentFlags: stencil.NewFlagSet(),
		Flags:           stencil.NewFlagSet(),
	}
	root.PersistentFlags.String("dir", "C", "project root directory", ".")
	root.Sub = []*stencil.Command{versionCmd(), hookCmd()}

	return stencil.NewApp(
		stencil.WithName("stencil"),
		stencil.WithDescription("Stencil developer tooling"),
		stencil.WithRootCommand(root),
	)
}

func versionCmd() *stencil.Command {
	return &stencil.Command{
		Name:    "version",
		Summary: "Manage the project's semantic version",
		Flags:   stencil.NewFlagSet(),
		Sub: []*stencil.Command{
			{
				Name:    "init",
				Summary: "Create .stencil/ with version files (0.0.0)",
				Flags:   stencil.NewFlagSet(),
				Run: func(ctx *stencil.Context) error {
					v, err := version.Init(ctx.Flags.String("dir"))
					if err != nil {
						return err
					}
					ctx.App.UI.Info("Initialized %s (version %s)", version.Dir, v)
					return nil
				},
			},
			{
				Name:    "show",
				Summary: "Print the current version",
				Flags:   stencil.NewFlagSet(),
				Run: func(ctx *stencil.Context) error {
					v, err := version.Read(ctx.Flags.String("dir"))
					if err != nil {
						return err
					}
					fmt.Fprintln(ctx.App.IO.Out, v)
					return nil
				},
			},
			bumpCmd(),
		},
	}
}

func bumpCmd() *stencil.Command {
	fs := stencil.NewFlagSet()
	fs.Bool("major", "", "bump the major version (resets minor & patch)", false)
	fs.Bool("minor", "", "bump the minor version (resets patch)", false)
	fs.Bool("patch", "", "bump the patch version (default)", false)
	return &stencil.Command{
		Name:    "bump",
		Summary: "Increment the version (patch by default)",
		Flags:   fs,
		Run: func(ctx *stencil.Context) error {
			part := version.Patch
			switch {
			case ctx.Flags.Bool("major"):
				part = version.Major
			case ctx.Flags.Bool("minor"):
				part = version.Minor
			}
			v, err := version.Bump(ctx.Flags.String("dir"), part)
			if err != nil {
				return err
			}
			ctx.App.UI.Info("Bumped version to %s", v)
			return nil
		},
	}
}

func hookCmd() *stencil.Command {
	return &stencil.Command{
		Name:    "hook",
		Summary: "Manage the git hook that auto-bumps the version",
		Flags:   stencil.NewFlagSet(),
		Sub: []*stencil.Command{
			{
				Name:    "install",
				Summary: "Install a pre-commit hook that bumps the patch version",
				Flags:   stencil.NewFlagSet(),
				Run: func(ctx *stencil.Context) error {
					root := ctx.Flags.String("dir")
					// Ensure version files exist so the hook has something to bump.
					if _, err := version.Init(root); err != nil {
						return err
					}
					// Bake in an absolute path to this binary so the hook works
					// regardless of the committer's PATH; fall back to "stencil".
					bin, err := os.Executable()
					if err != nil {
						bin = ""
					}
					if err := (hooker.VersionHook{Bin: bin}).Install(root); err != nil {
						return err
					}
					ctx.App.UI.Info("Installed %s hook (auto-bumps patch version on commit)", hooker.HookName)
					return nil
				},
			},
			{
				Name:    "uninstall",
				Summary: "Remove the stencil-managed git hook",
				Flags:   stencil.NewFlagSet(),
				Run: func(ctx *stencil.Context) error {
					if err := (hooker.VersionHook{}).Uninstall(ctx.Flags.String("dir")); err != nil {
						return err
					}
					ctx.App.UI.Info("Removed %s hook", hooker.HookName)
					return nil
				},
			},
		},
	}
}
