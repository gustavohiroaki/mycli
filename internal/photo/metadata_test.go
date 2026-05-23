package photo

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

type fakeMetadataProvider struct {
	metadata Metadata
	err      error
}

func (f fakeMetadataProvider) Available() bool {
	return f.err == nil
}

func (f fakeMetadataProvider) Read(path string) (Metadata, error) {
	if f.err != nil {
		return Metadata{}, f.err
	}
	return f.metadata, nil
}

func TestDateFromFilenameParsesCompactDateTime(t *testing.T) {
	got, ok := DateFromFilename("IMG_20240523_143015.CR3")
	if !ok {
		t.Fatal("expected date")
	}
	if got.Format("2006-01-02 15:04:05") != "2024-05-23 14:30:15" {
		t.Fatalf("date = %s", got.Format("2006-01-02 15:04:05"))
	}
}

func TestResolveMetadataUsesProvider(t *testing.T) {
	wantDate := time.Date(2024, 5, 23, 14, 30, 0, 0, time.Local)
	file := MediaFile{SourcePath: "IMG_0001.JPG", ModTime: time.Now()}
	got := ResolveMetadata(file, fakeMetadataProvider{metadata: Metadata{
		Date:   wantDate,
		Camera: "Canon EOS R6",
		Lens:   "RF 35mm",
	}})

	if !got.Date.Equal(wantDate) {
		t.Fatalf("Date = %v, want %v", got.Date, wantDate)
	}
	if got.Camera != "Canon EOS R6" {
		t.Fatalf("Camera = %q", got.Camera)
	}
	if got.UsedFallback {
		t.Fatal("UsedFallback = true, want false")
	}
}

func TestResolveMetadataFallsBackToFilenameThenModTime(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "IMG_20240523_143015.JPG")
	if err := os.WriteFile(path, []byte("photo"), 0o644); err != nil {
		t.Fatal(err)
	}
	file := MediaFile{SourcePath: path, ModTime: time.Date(2020, 1, 2, 3, 4, 5, 0, time.Local)}

	got := ResolveMetadata(file, fakeMetadataProvider{err: os.ErrNotExist})
	if got.Date.Format("2006-01-02 15:04:05") != "2024-05-23 14:30:15" {
		t.Fatalf("Date = %s", got.Date.Format("2006-01-02 15:04:05"))
	}
	if got.Camera != "unknown-camera" {
		t.Fatalf("Camera = %q", got.Camera)
	}
	if !got.UsedFallback {
		t.Fatal("UsedFallback = false, want true")
	}
}
