package initializer

import (
	"os"
	"path"
)

type Initializer interface {
	Initialize() error
}

func NewInitializer(rootDir string) Initializer {
	return DefaultInitializer{rootDir: rootDir}
}

type DefaultInitializer struct {
	rootDir string
}

func (d DefaultInitializer) Initialize() error {
	// Create the .stencil directory if it doesn't exist
	if err := CreateStencilDirectory(d.rootDir); err != nil {
		return err
	}

	// Ensure default stencils are created
	return EnsureDefaultStencils(d.rootDir)
}

func CreateStencilDirectory(rootDir string) error {
	dir := path.Join(rootDir, ".stencil")
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		// Create the directory
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	return nil
}

func EnsureDefaultStencils(rootDir string) error {
	dir := path.Join(rootDir, ".stencil")
	// Ensure the default stencils exist
	defaultStencils := []string{"version_major", "version_minor", "version_patch"}
	for _, stencil := range defaultStencils {
		stencilPath := path.Join(dir, stencil)
		if _, err := os.Stat(stencilPath); os.IsNotExist(err) {
			// Create the default stencil file
			if err := os.WriteFile(stencilPath, []byte("0"), 0644); err != nil {
				return err
			}
		}
	}
	return nil
}
