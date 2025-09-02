package stencil

import (
	"bytes"
	"fmt"
	"testing"
)

// Override the versioning variables for predictable testing
func init() {
	appVersion = "test-version"
	appBuildDate = "test-build-date"
	appCommitHash = "test-commit-hash"
	appBranch = "test-branch"
}

func TestShowHelp_DefaultOutput(t *testing.T) {
	// Create a buffer to capture output
	var buf bytes.Buffer

	// Create a stencil with output directed to buffer
	s := &Stencil{
		AppName:     "TestApp",
		AppDesc:     "Description of TestApp",
		ShowVersion: true,
		Output:      &buf,
	}

	// Call ShowHelp
	s.ShowHelp()

	// Verify the output
	expectedOutput := fmt.Sprintf(`Usage: TestApp [OPTIONS]

Description: Description of TestApp
Version: %s
`, appVersion)

	if buf.String() != expectedOutput {
		t.Errorf("Unexpected output: got %q, want %q", buf.String(), expectedOutput)
	}
}
func TestShowHelp_CustomOutput(t *testing.T) {
	// Create a buffer to capture output
	var buf bytes.Buffer

	// Create stencil with various flags and output to buffer
	s := &Stencil{
		AppName:        "AdvancedApp",
		AppDesc:        "Advanced features description",
		ShowVersion:    true,
		ShowBuildDate:  true,
		ShowCommitHash: true,
		ShowBranch:     true,
		ColoredOutput:  true,
		Output:         &buf,
	}

	// Call ShowHelp
	s.ShowHelp()

	// Verify the output
	expectedOutput := fmt.Sprintf(`Usage: AdvancedApp [OPTIONS]

Description: Advanced features description
Version: %s
Build Date: %s
Commit Hash: %s
Branch: %s
`, appVersion, appBuildDate, appCommitHash, appBranch)

	if buf.String() != expectedOutput {
		t.Errorf("Unexpected output: got %q, want %q", buf.String(), expectedOutput)
	}
}
