package stencil

type RunFunc func(ctx *Context) error

type Command struct {
	Name       string
	Summary    string
	Long       string
	Aliases    []string
	Hidden     bool
	Deprecated string

	// Execution
	Run               RunFunc
	PreRun            RunFunc
	PostRun           RunFunc
	PersistentPreRun  RunFunc
	PersistentPostRun RunFunc

	// Flags
	PersistentFlags *FlagSet // inherited by descendants
	Flags           *FlagSet // local to this command

	// Positionals
	Args ArgSpec

	// Children
	Sub []*Command
}

type ArgSpec struct {
	Min, Max int      // Max=0 => unlimited
	Names    []string // for help
	Validate func([]string) error
}

type Context struct {
	App   *App
	Path  []*Command // root→leaf
	Args  []string
	Flags *ResolvedFlags
}
