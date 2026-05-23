package photo

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExecutePlanCopiesFiles(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source.jpg")
	dest := filepath.Join(root, "out", "source.jpg")
	if err := os.WriteFile(source, []byte("photo"), 0o644); err != nil {
		t.Fatal(err)
	}

	summary := ExecutePlan(Plan{Actions: []PlannedAction{{
		Kind:       ActionCopy,
		SourcePath: source,
		DestPath:   dest,
		MediaType:  MediaTypePhoto,
	}}})

	if summary.Copied != 1 || summary.Failed != 0 {
		t.Fatalf("summary = %+v", summary)
	}
	if _, err := os.Stat(dest); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(source); err != nil {
		t.Fatal(err)
	}
}

func TestExecutePlanMovesFiles(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source.jpg")
	dest := filepath.Join(root, "out", "source.jpg")
	if err := os.WriteFile(source, []byte("photo"), 0o644); err != nil {
		t.Fatal(err)
	}

	summary := ExecutePlan(Plan{Actions: []PlannedAction{{
		Kind:       ActionMove,
		SourcePath: source,
		DestPath:   dest,
		MediaType:  MediaTypePhoto,
	}}})

	if summary.Moved != 1 || summary.Failed != 0 {
		t.Fatalf("summary = %+v", summary)
	}
	if _, err := os.Stat(dest); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(source); !os.IsNotExist(err) {
		t.Fatalf("source still exists or unexpected error: %v", err)
	}
}
