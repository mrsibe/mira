package fonts

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"golang.org/x/image/font/gofont/goregular"
)

func TestListFromDirsFindsFamilies(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Go-Regular.ttf"), goregular.TTF, 0o644); err != nil {
		t.Fatal(err)
	}

	families := listFromDirs([]string{dir})
	if !slices.Contains(families, "Go") {
		t.Fatalf("families = %q, want Go", families)
	}
}

func TestCleanFamily(t *testing.T) {
	if got := cleanFamily("\x00  Noto\tSans  \x7f"); got != "Noto Sans" {
		t.Fatalf("cleanFamily = %q", got)
	}
}

func TestListFromDirsIgnoresMissingDirectoriesAndOtherFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("not a font"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := listFromDirs([]string{dir, filepath.Join(dir, "missing")}); len(got) != 0 {
		t.Fatalf("families = %q, want none", got)
	}
}
