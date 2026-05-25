package photo

import (
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestVisualHashStableForJPEGAndPNG(t *testing.T) {
	root := t.TempDir()
	jpg := filepath.Join(root, "white.jpg")
	pngPath := filepath.Join(root, "white.png")
	writeSolidJPEG(t, jpg, color.RGBA{R: 240, G: 240, B: 240, A: 255})
	writeSolidPNG(t, pngPath, color.RGBA{R: 240, G: 240, B: 240, A: 255})

	jpgHash, jpgOK, err := VisualHashFile(jpg)
	if err != nil {
		t.Fatal(err)
	}
	pngHash, pngOK, err := VisualHashFile(pngPath)
	if err != nil {
		t.Fatal(err)
	}
	if !jpgOK || !pngOK {
		t.Fatal("expected hashes")
	}
	if jpgHash != pngHash {
		t.Fatalf("hashes differ: jpg=%064b png=%064b", jpgHash, pngHash)
	}
}

func TestDetectSimilarGroupsWithinThreshold(t *testing.T) {
	files := []EnrichedFile{
		similarFile("/src/a.jpg", 0b1111, true),
		similarFile("/src/b.jpg", 0b1110, true),
		similarFile("/src/c.jpg", 0b0000, true),
	}

	groups := DetectSimilarGroups(files, 1)
	if len(groups) != 1 {
		t.Fatalf("len(groups) = %d, want 1", len(groups))
	}
	if len(groups[0].Files) != 2 {
		t.Fatalf("group size = %d, want 2", len(groups[0].Files))
	}
}

func TestDetectSimilarGroupsIgnoresVideosRawAndMissingHashes(t *testing.T) {
	files := []EnrichedFile{
		similarFile("/src/a.jpg", 0b1111, true),
		{File: MediaFile{SourcePath: "/src/b.mov", Type: MediaTypeVideo, Extension: ".mov"}, VisualHash: 0b1111, HasVisualHash: true},
		{File: MediaFile{SourcePath: "/src/c.cr3", Type: MediaTypeRaw, Extension: ".cr3"}, VisualHash: 0b1111, HasVisualHash: true},
		similarFile("/src/d.jpg", 0, false),
	}

	groups := DetectSimilarGroups(files, 1)
	if len(groups) != 0 {
		t.Fatalf("len(groups) = %d, want 0", len(groups))
	}
}

func TestDetectSimilarGroupsAgainstKnown(t *testing.T) {
	files := []EnrichedFile{
		similarFile("/new/a.jpg", 0b1111, true),
		similarFile("/new/b.jpg", 0b0000, true),
	}
	known := []KnownVisualHash{{Path: "/library/old.jpg", Hash: 0b1110}}

	groups := DetectSimilarGroupsAgainstKnown(files, known, 1)
	if len(groups) != 1 {
		t.Fatalf("len(groups) = %d, want 1", len(groups))
	}
	if groups[0].Type != GroupSimilar || len(groups[0].Files) != 1 || groups[0].Files[0] != "/new/a.jpg" {
		t.Fatalf("groups = %+v", groups)
	}
}

func TestEnrichVisualHashesSkipsUndecodableFiles(t *testing.T) {
	root := t.TempDir()
	bad := filepath.Join(root, "bad.jpg")
	if err := os.WriteFile(bad, []byte("not an image"), 0o644); err != nil {
		t.Fatal(err)
	}
	files := []EnrichedFile{{File: MediaFile{SourcePath: bad, Type: MediaTypePhoto, Extension: ".jpg"}}}

	got, skipped := EnrichVisualHashes(files)
	if skipped != 1 {
		t.Fatalf("skipped = %d, want 1", skipped)
	}
	if got[0].HasVisualHash {
		t.Fatal("expected no hash")
	}
	if !got[0].VisualHashSkipped {
		t.Fatal("expected skipped flag")
	}
}

func similarFile(path string, hash uint64, hasHash bool) EnrichedFile {
	return EnrichedFile{
		File:          MediaFile{SourcePath: path, Type: MediaTypePhoto, Extension: filepath.Ext(path)},
		VisualHash:    hash,
		HasVisualHash: hasHash,
	}
}

func writeSolidJPEG(t *testing.T, path string, c color.Color) {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if err := jpeg.Encode(file, solidImage(c), nil); err != nil {
		t.Fatal(err)
	}
}

func writeSolidPNG(t *testing.T, path string, c color.Color) {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if err := png.Encode(file, solidImage(c)); err != nil {
		t.Fatal(err)
	}
}

func solidImage(c color.Color) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			img.Set(x, y, c)
		}
	}
	return img
}
