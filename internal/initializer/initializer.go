package initializer

import "github.com/bgrewell/stencil/internal/version"

// Initializer prepares a project's .stencil directory.
type Initializer interface {
	Initialize() error
}

func NewInitializer(rootDir string) Initializer {
	return DefaultInitializer{rootDir: rootDir}
}

type DefaultInitializer struct {
	rootDir string
}

// Initialize creates the .stencil directory and seeds the default version
// files. The version package owns the file layout so the two never diverge.
func (d DefaultInitializer) Initialize() error {
	_, err := version.Init(d.rootDir)
	return err
}
