package photo

import (
	"os"
	"path/filepath"
	"sync"
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

type blockingMetadataProvider struct {
	metadata Metadata
	blockAt  int

	mu        sync.Mutex
	active    int
	maxActive int
	calls     int
	release   chan struct{}
	once      sync.Once
}

func (p *blockingMetadataProvider) Available() bool {
	return true
}

func (p *blockingMetadataProvider) Read(path string) (Metadata, error) {
	p.once.Do(func() {
		p.release = make(chan struct{})
	})

	p.mu.Lock()
	p.active++
	p.calls++
	if p.active > p.maxActive {
		p.maxActive = p.active
	}
	if p.calls == p.blockAt {
		close(p.release)
	}
	release := p.release
	p.mu.Unlock()

	<-release

	p.mu.Lock()
	p.active--
	p.mu.Unlock()
	return p.metadata, nil
}

func (p *blockingMetadataProvider) releaseAll() {
	p.once.Do(func() {
		p.release = make(chan struct{})
	})
	defer func() {
		_ = recover()
	}()
	close(p.release)
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

func TestParseExiftoolMetadataDoesNotUseDateAsCamera(t *testing.T) {
	raw := []byte(`[{
		"SourceFile": "IMG_0001.JPG",
		"DateTimeOriginal": "2026-05-01 12:57:12",
		"CreateDate": "2026-05-01 12:57:12",
		"MediaCreateDate": "2026-05-01 12:57:12",
		"Model": "Canon EOS R6",
		"LensModel": "RF 35mm F1.8"
	}]`)

	got, err := parseExiftoolMetadata(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got.Date.Format("2006-01-02 15:04:05") != "2026-05-01 12:57:12" {
		t.Fatalf("Date = %s", got.Date.Format("2006-01-02 15:04:05"))
	}
	if got.Camera != "Canon EOS R6" {
		t.Fatalf("Camera = %q, want camera model", got.Camera)
	}
	if got.Lens != "RF 35mm F1.8" {
		t.Fatalf("Lens = %q", got.Lens)
	}
}
