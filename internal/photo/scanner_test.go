package photo

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectMediaType(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		wantType  MediaType
		wantMatch bool
	}{
		{name: "jpeg photo", path: "IMG_0001.JPG", wantType: MediaTypePhoto, wantMatch: true},
		{name: "raw photo", path: "IMG_0001.CR3", wantType: MediaTypeRaw, wantMatch: true},
		{name: "video", path: "clip.MOV", wantType: MediaTypeVideo, wantMatch: true},
		{name: "other", path: "notes.txt", wantType: "", wantMatch: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotType, gotMatch := DetectMediaType(tt.path)
			if gotMatch != tt.wantMatch {
				t.Fatalf("match = %v, want %v", gotMatch, tt.wantMatch)
			}
			if gotType != tt.wantType {
				t.Fatalf("type = %q, want %q", gotType, tt.wantType)
			}
		})
	}
}

func TestValidateSourceDestinationRejectsDestinationInsideSource(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	dest := filepath.Join(source, "organized")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}

	err := ValidateSourceDestination(source, dest)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateSourceDestinationRequiresDirectorySource(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "file.jpg")
	dest := filepath.Join(root, "dest")
	if err := os.WriteFile(source, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := ValidateSourceDestination(source, dest)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestScanFindsNestedMediaByDefault(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	nested := filepath.Join(source, "DCIM", "100CANON")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nested, "IMG_0001.JPG"), []byte("photo"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "notes.txt"), []byte("notes"), 0o644); err != nil {
		t.Fatal(err)
	}

	files, err := Scan(source, ScanOptions{Recursive: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Fatalf("len(files) = %d, want 1", len(files))
	}
	if files[0].RelPath != filepath.Join("DCIM", "100CANON", "IMG_0001.JPG") {
		t.Fatalf("RelPath = %q", files[0].RelPath)
	}
}

func TestScanNoRecursiveSkipsNestedFiles(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	nested := filepath.Join(source, "nested")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "top.JPG"), []byte("photo"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nested, "nested.JPG"), []byte("photo"), 0o644); err != nil {
		t.Fatal(err)
	}

	files, err := Scan(source, ScanOptions{Recursive: false})
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Fatalf("len(files) = %d, want 1", len(files))
	}
	if filepath.Base(files[0].SourcePath) != "top.JPG" {
		t.Fatalf("SourcePath = %q", files[0].SourcePath)
	}
}

func TestScanUsesOnlyExplicitExcludes(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	exports := filepath.Join(source, "exports")
	if err := os.MkdirAll(exports, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(exports, "skip.JPG"), []byte("photo"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "keep.JPG"), []byte("photo"), 0o644); err != nil {
		t.Fatal(err)
	}

	files, err := Scan(source, ScanOptions{Recursive: true, Excludes: []string{"exports"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Fatalf("len(files) = %d, want 1", len(files))
	}
	if filepath.Base(files[0].SourcePath) != "keep.JPG" {
		t.Fatalf("SourcePath = %q", files[0].SourcePath)
	}
}
