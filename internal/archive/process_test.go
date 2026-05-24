package archive

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProcessDeletesOriginalAfterVerifiedExtraction(t *testing.T) {
	root := t.TempDir()
	archivePath := filepath.Join(root, "files.zip")
	writeZip(t, archivePath, map[string]string{"a.txt": "hello"})

	item := Item{SourcePath: archivePath, Type: TypeZip, BaseName: "files", DestDir: filepath.Join(root, "files")}
	result := Process(item, Options{})
	if result.Status != StatusOK {
		t.Fatalf("result = %+v", result)
	}
	if _, err := os.Stat(archivePath); !os.IsNotExist(err) {
		t.Fatalf("archive still exists or unexpected error: %v", err)
	}
}

func TestProcessKeepPreservesOriginal(t *testing.T) {
	root := t.TempDir()
	archivePath := filepath.Join(root, "files.zip")
	writeZip(t, archivePath, map[string]string{"a.txt": "hello"})

	item := Item{SourcePath: archivePath, Type: TypeZip, BaseName: "files", DestDir: filepath.Join(root, "files")}
	result := Process(item, Options{Keep: true})
	if result.Status != StatusKeep {
		t.Fatalf("result = %+v", result)
	}
	if _, err := os.Stat(archivePath); err != nil {
		t.Fatalf("archive missing: %v", err)
	}
}

func TestProcessPreservesOriginalWhenExtractionFails(t *testing.T) {
	root := t.TempDir()
	archivePath := filepath.Join(root, "bad.zip")
	if err := os.WriteFile(archivePath, []byte("not a zip"), 0o644); err != nil {
		t.Fatal(err)
	}

	item := Item{SourcePath: archivePath, Type: TypeZip, BaseName: "bad", DestDir: filepath.Join(root, "bad")}
	result := Process(item, Options{})
	if result.Status != StatusFail {
		t.Fatalf("result = %+v", result)
	}
	if _, err := os.Stat(archivePath); err != nil {
		t.Fatalf("archive missing: %v", err)
	}
}

func TestProcessBatchContinuesAfterFailure(t *testing.T) {
	root := t.TempDir()
	good := filepath.Join(root, "good.zip")
	bad := filepath.Join(root, "bad.zip")
	writeZip(t, good, map[string]string{"a.txt": "hello"})
	if err := os.WriteFile(bad, []byte("not a zip"), 0o644); err != nil {
		t.Fatal(err)
	}

	results, summary := ProcessBatch([]Item{
		{SourcePath: bad, Type: TypeZip, BaseName: "bad", DestDir: filepath.Join(root, "bad")},
		{SourcePath: good, Type: TypeZip, BaseName: "good", DestDir: filepath.Join(root, "good")},
	}, Options{}, 0)

	if len(results) != 2 {
		t.Fatalf("len(results) = %d", len(results))
	}
	if summary.Failed != 1 || summary.Extracted != 1 || summary.DeletedOriginals != 1 {
		t.Fatalf("summary = %+v", summary)
	}
}
