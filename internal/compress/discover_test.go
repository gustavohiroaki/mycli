package compress

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverSingleVideo(t *testing.T) {
	root := t.TempDir()
	video := filepath.Join(root, "clip.mov")
	if err := os.WriteFile(video, []byte("video"), 0o644); err != nil {
		t.Fatal(err)
	}

	items, skipped, err := Discover(video, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if skipped != 0 || len(items) != 1 {
		t.Fatalf("items=%d skipped=%d", len(items), skipped)
	}
	want := filepath.Join(root, "clip_compressed.mp4")
	if items[0].DestPath != want {
		t.Fatalf("DestPath = %q, want %q", items[0].DestPath, want)
	}
}

func TestDiscoverDirectoryRecursive(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "nested")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "a.mp4"), []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nested, "b.mov"), []byte("b"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "photo.jpg"), []byte("p"), 0o644); err != nil {
		t.Fatal(err)
	}

	items, skipped, err := Discover(root, Options{Dest: filepath.Join(root, "out"), Recursive: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || skipped != 1 {
		t.Fatalf("items=%d skipped=%d", len(items), skipped)
	}
}
