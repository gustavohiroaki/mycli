package photo

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHashFileMatchesSameContent(t *testing.T) {
	root := t.TempDir()
	a := filepath.Join(root, "a.jpg")
	b := filepath.Join(root, "b.jpg")
	if err := os.WriteFile(a, []byte("same"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b, []byte("same"), 0o644); err != nil {
		t.Fatal(err)
	}

	hashA, err := HashFile(a)
	if err != nil {
		t.Fatal(err)
	}
	hashB, err := HashFile(b)
	if err != nil {
		t.Fatal(err)
	}
	if hashA != hashB {
		t.Fatalf("hashes differ")
	}
}

func TestMarkDuplicatesMarksSecondFile(t *testing.T) {
	files := []EnrichedFile{
		{File: MediaFile{SourcePath: "a.jpg"}, Hash: "abc"},
		{File: MediaFile{SourcePath: "b.jpg"}, Hash: "abc"},
		{File: MediaFile{SourcePath: "c.jpg"}, Hash: "def"},
	}

	got := MarkDuplicates(files)
	if got["a.jpg"] {
		t.Fatal("first file should not be duplicate")
	}
	if !got["b.jpg"] {
		t.Fatal("second file should be duplicate")
	}
	if got["c.jpg"] {
		t.Fatal("unique file should not be duplicate")
	}
}
