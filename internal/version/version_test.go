package version

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitSeedsZero(t *testing.T) {
	root := t.TempDir()
	v, err := Init(root)
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	if got := v.String(); got != "0.0.0" {
		t.Fatalf("fresh init = %s, want 0.0.0", got)
	}
	if _, err := os.Stat(filepath.Join(root, Dir, filePatch)); err != nil {
		t.Fatalf("version_patch not created: %v", err)
	}
}

func TestInitIdempotent(t *testing.T) {
	root := t.TempDir()
	if _, err := Init(root); err != nil {
		t.Fatal(err)
	}
	if _, err := Bump(root, Patch); err != nil {
		t.Fatal(err)
	}
	// Re-init must not reset an existing version.
	v, err := Init(root)
	if err != nil {
		t.Fatal(err)
	}
	if v.String() != "0.0.1" {
		t.Fatalf("re-init reset version to %s, want 0.0.1", v)
	}
}

func TestBumpResetsLowerComponents(t *testing.T) {
	root := t.TempDir()
	if _, err := Init(root); err != nil {
		t.Fatal(err)
	}

	steps := []struct {
		part Part
		want string
	}{
		{Patch, "0.0.1"},
		{Patch, "0.0.2"},
		{Minor, "0.1.0"}, // patch resets
		{Patch, "0.1.1"},
		{Major, "1.0.0"}, // minor + patch reset
		{Patch, "1.0.1"},
	}
	for i, s := range steps {
		v, err := Bump(root, s.part)
		if err != nil {
			t.Fatalf("step %d: %v", i, err)
		}
		if v.String() != s.want {
			t.Fatalf("step %d: got %s, want %s", i, v, s.want)
		}
		// Bump must persist: a fresh Read agrees.
		got, err := Read(root)
		if err != nil {
			t.Fatalf("step %d read: %v", i, err)
		}
		if got.String() != s.want {
			t.Fatalf("step %d persisted %s, want %s", i, got, s.want)
		}
	}
}

func TestReadBeforeInit(t *testing.T) {
	if _, err := Read(t.TempDir()); err == nil {
		t.Fatal("expected error reading version before init")
	}
}

func TestReadRejectsGarbage(t *testing.T) {
	root := t.TempDir()
	if _, err := Init(root); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, Dir, filePatch), []byte("oops"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Read(root); err == nil {
		t.Fatal("expected error for non-numeric version file")
	}
}
