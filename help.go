package stencil

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/fatih/color"
)

type AppShow struct {
	Version    bool
	BuildDate  bool
	CommitHash bool
	Branch     bool
}

type AppInfo struct {
	Name        string
	Description string
	Version     VersionInfo
	Show        AppShow
	Colored     bool
}

func renderHelp(out io.Writer, info AppInfo, path []*Command) {
	color.NoColor = !info.Colored
	y := color.New(color.FgYellow).SprintFunc()
	b := color.New(color.FgBlue).SprintFunc()
	g := color.New(color.FgGreen).SprintFunc()

	cur := path[len(path)-1]

	fullPath := make([]string, len(path))
	for i, c := range path {
		fullPath[i] = c.Name
	}

	// Header
	title := strings.Join(fullPath, " ")
	fmt.Fprintf(out, "%s: %s\n\n", b(title), cur.Summary)
	if cur.Long != "" {
		fmt.Fprintf(out, "%s\n\n", cur.Long)
	}
	if len(path) == 1 {
		// Version block
		if info.Show.Version || info.Show.BuildDate || info.Show.CommitHash || info.Show.Branch {
			fmt.Fprintln(out, y("Version:"))
			if info.Show.Version {
				fmt.Fprintf(out, "  %s\n", info.Version.Version)
			}
			if info.Show.BuildDate {
				fmt.Fprintf(out, "  built %s\n", info.Version.BuildDate)
			}
			if info.Show.CommitHash {
				fmt.Fprintf(out, "  commit %s\n", info.Version.CommitHash)
			}
			if info.Show.Branch {
				fmt.Fprintf(out, "  branch %s\n", info.Version.Branch)
			}
			fmt.Fprintln(out)
		}
		if info.Description != "" {
			fmt.Fprintf(out, "%s %s\n\n", g("Description:"), info.Description)
		}
	}

	// Usage
	fmt.Fprintln(out, y("Usage:"))
	fmt.Fprintf(out, "  %s [FLAGS] [COMMAND] [ARGS]\n\n", fullPath[0])

	// Flags (merged)
	merged := collectFlags(path)
	if len(merged) > 0 {
		fmt.Fprintln(out, y("Flags:"))
		for _, f := range merged {
			if f.Hidden {
				continue
			}
			short := ""
			if f.Short != "" {
				short = fmt.Sprintf("-%s, ", f.Short)
			}
			def := fmt.Sprintf(" (default: %v)", f.Default)
			if len(f.Enum) > 0 {
				def = fmt.Sprintf(" (one of: %s; default: %v)", strings.Join(f.Enum, ","), f.Default)
			}
			fmt.Fprintf(out, "  %s--%s\t%s%s\n", short, f.Name, f.Usage, def)
		}
		fmt.Fprintln(out)
	}

	// Subcommands
	if len(cur.Sub) > 0 {
		fmt.Fprintln(out, y("Commands:"))
		// sort for stability
		subs := make([]*Command, 0, len(cur.Sub))
		for _, c := range cur.Sub {
			if c.Hidden {
				continue
			}
			subs = append(subs, c)
		}
		sort.Slice(subs, func(i, j int) bool { return subs[i].Name < subs[j].Name })
		for _, c := range subs {
			line := c.Name
			if c.Deprecated != "" {
				line += " (deprecated)"
			}
			fmt.Fprintf(out, "  %-16s %s\n", line, c.Summary)
		}
		fmt.Fprintln(out)
	}

	// Args
	if len(cur.Args.Names) > 0 {
		fmt.Fprintln(out, y("Arguments:"))
		fmt.Fprintf(out, "  %s\n\n", strings.Join(cur.Args.Names, " "))
	}
}

func collectFlags(path []*Command) []*Flag {
	var out []*Flag
	for _, c := range path {
		if c.PersistentFlags != nil {
			out = append(out, c.PersistentFlags.list()...)
		}
	}
	leaf := path[len(path)-1]
	if leaf.Flags != nil {
		out = append(out, leaf.Flags.list()...)
	}
	return out
}

func renderVersion(out io.Writer, info AppInfo) {
	fmt.Fprintf(out, "%s %s\n", info.Name, info.Version.Version)
	if info.Version.CommitHash != "" || info.Version.BuildDate != "" || info.Version.Branch != "" {
		fmt.Fprintf(out, "commit: %s  built: %s  branch: %s\n",
			info.Version.CommitHash, info.Version.BuildDate, info.Version.Branch)
	}
}
