package photo

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGlobalLibraryStoreSavesListsAndSetsDefault(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "mycli.db")

	if err := SaveGlobalLibrary(dbPath, Library{Name: "one", Path: "/photos/one", IsDefault: true}); err != nil {
		t.Fatal(err)
	}
	if err := SaveGlobalLibrary(dbPath, Library{Name: "two", Path: "/photos/two", IsDefault: false}); err != nil {
		t.Fatal(err)
	}
	if err := SetDefaultLibrary(dbPath, "two"); err != nil {
		t.Fatal(err)
	}

	got, err := DefaultLibrary(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "two" {
		t.Fatalf("default = %q, want two", got.Name)
	}
}

func TestLibraryConfigAndIndexRoundTrip(t *testing.T) {
	root := t.TempDir()
	config := ConfigFromOptions(Options{
		Destination:         root,
		Recursive:           true,
		Structure:           "{type}/{year}",
		Rename:              "grouped",
		Duplicates:          DuplicateSeparate,
		Report:              ReportText,
		BurstWindow:         2 * time.Second,
		SimilarityEnabled:   true,
		SimilarityThreshold: 8,
		FullPerformance:     true,
	})
	if err := SaveLibraryConfig(root, config); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadLibraryConfig(root)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Structure != "{type}/{year}" || loaded.BurstWindow != "2s" {
		t.Fatalf("loaded config = %+v", loaded)
	}

	dest := filepath.Join(root, "photos", "a.jpg")
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dest, []byte("photo"), 0o644); err != nil {
		t.Fatal(err)
	}
	plan := Plan{Actions: []PlannedAction{{
		Kind:       ActionCopy,
		SourcePath: "/card/a.jpg",
		DestPath:   dest,
		MediaType:  MediaTypePhoto,
		Hash:       "abc123",
		Metadata: Metadata{
			Date:   time.Date(2026, 5, 1, 13, 10, 59, 0, time.UTC),
			Camera: "Canon EOS RP",
			Lens:   "RF 35mm",
		},
		SourceSize: 5,
		Extension:  ".jpg",
	}}}
	if err := RecordImport(root, "/card", config, plan, Summary{Copied: 1}); err != nil {
		t.Fatal(err)
	}
	hashes, err := ExistingHashes(root)
	if err != nil {
		t.Fatal(err)
	}
	if hashes["abc123"] != dest {
		t.Fatalf("hash index = %#v", hashes)
	}
}
