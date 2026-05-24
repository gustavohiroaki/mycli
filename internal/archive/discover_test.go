package archive

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectType(t *testing.T) {
	tests := []struct {
		path     string
		wantType Type
		wantOK   bool
	}{
		{"photos.zip", TypeZip, true},
		{"backup.tar", TypeTar, true},
		{"aulas.tar.gz", TypeTarGz, true},
		{"backup.tgz", TypeTarGz, true},
		{"notes.txt", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			gotType, gotOK := DetectType(tt.path)
			if gotType != tt.wantType || gotOK != tt.wantOK {
				t.Fatalf("DetectType(%q) = %q, %v; want %q, %v", tt.path, gotType, gotOK, tt.wantType, tt.wantOK)
			}
		})
	}
}

func TestBaseNameRemovesArchiveExtension(t *testing.T) {
	tests := map[string]string{
		"photos.zip":   "photos",
		"backup.tar":   "backup",
		"aulas.tar.gz": "aulas",
		"backup.tgz":   "backup",
	}
	for input, want := range tests {
		if got := BaseName(input); got != want {
			t.Fatalf("BaseName(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestDiscoverDirectoryNonRecursive(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.zip"), []byte("zip"), 0o644); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(root, "nested")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nested, "b.zip"), []byte("zip"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "notes.txt"), []byte("notes"), 0o644); err != nil {
		t.Fatal(err)
	}

	items, skipped, err := Discover(root, Options{Recursive: false})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if filepath.Base(items[0].SourcePath) != "a.zip" {
		t.Fatalf("SourcePath = %q", items[0].SourcePath)
	}
	if skipped != 1 {
		t.Fatalf("skipped = %d, want 1", skipped)
	}
}

func TestDiscoverDirectoryRecursive(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "nested")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "a.zip"), []byte("zip"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nested, "b.tgz"), []byte("tgz"), 0o644); err != nil {
		t.Fatal(err)
	}

	items, _, err := Discover(root, Options{Recursive: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}
}
