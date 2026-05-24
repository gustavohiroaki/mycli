package archive

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

func TestExtractZipWritesFiles(t *testing.T) {
	root := t.TempDir()
	archivePath := filepath.Join(root, "files.zip")
	writeZip(t, archivePath, map[string]string{"dir/a.txt": "hello"})

	item := Item{SourcePath: archivePath, Type: TypeZip, DestDir: filepath.Join(root, "files")}
	files, skipped, err := Extract(item, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if skipped != 0 || len(files) != 1 {
		t.Fatalf("files = %d skipped = %d", len(files), skipped)
	}
	content, err := os.ReadFile(filepath.Join(item.DestDir, "dir", "a.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "hello" {
		t.Fatalf("content = %q", content)
	}
}

func TestExtractTarGzWritesFiles(t *testing.T) {
	root := t.TempDir()
	archivePath := filepath.Join(root, "files.tar.gz")
	writeTarGz(t, archivePath, map[string]string{"a.txt": "hello"})

	item := Item{SourcePath: archivePath, Type: TypeTarGz, DestDir: filepath.Join(root, "files")}
	files, skipped, err := Extract(item, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if skipped != 0 || len(files) != 1 {
		t.Fatalf("files = %d skipped = %d", len(files), skipped)
	}
}

func TestExtractRejectsTraversal(t *testing.T) {
	root := t.TempDir()
	archivePath := filepath.Join(root, "bad.zip")
	writeZip(t, archivePath, map[string]string{"../evil.txt": "no"})

	item := Item{SourcePath: archivePath, Type: TypeZip, DestDir: filepath.Join(root, "bad")}
	_, _, err := Extract(item, Options{})
	if err == nil {
		t.Fatal("expected traversal error")
	}
	if _, err := os.Stat(filepath.Join(root, "evil.txt")); !os.IsNotExist(err) {
		t.Fatalf("evil file exists or unexpected error: %v", err)
	}
}

func TestExtractExistingNonEmptyDestinationRequiresOverwrite(t *testing.T) {
	root := t.TempDir()
	archivePath := filepath.Join(root, "files.zip")
	writeZip(t, archivePath, map[string]string{"a.txt": "hello"})
	dest := filepath.Join(root, "files")
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dest, "existing.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, _, err := Extract(Item{SourcePath: archivePath, Type: TypeZip, DestDir: dest}, Options{})
	if err == nil {
		t.Fatal("expected non-empty destination error")
	}
}

func writeZip(t *testing.T, path string, files map[string]string) {
	t.Helper()
	out, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	writer := zip.NewWriter(out)
	for name, content := range files {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
}

func writeTarGz(t *testing.T, path string, files map[string]string) {
	t.Helper()
	var buffer bytes.Buffer
	gzipWriter := gzip.NewWriter(&buffer)
	tarWriter := tar.NewWriter(gzipWriter)
	for name, content := range files {
		header := &tar.Header{Name: name, Mode: 0o644, Size: int64(len(content))}
		if err := tarWriter.WriteHeader(header); err != nil {
			t.Fatal(err)
		}
		if _, err := tarWriter.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, buffer.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}
