package photo

import (
	"testing"
	"time"
)

func TestRenderTemplateWithCameraPreset(t *testing.T) {
	file := EnrichedFile{
		File: MediaFile{Type: MediaTypeRaw, Extension: ".cr3"},
		Metadata: Metadata{
			Date:   time.Date(2024, 5, 23, 14, 30, 15, 0, time.Local),
			Camera: "Canon EOS R6",
			Lens:   "RF 35mm F1.8",
		},
	}

	got, err := RenderTemplate("{camera}/{year}/{month}/{day}/{type}", file, 1)
	if err != nil {
		t.Fatal(err)
	}
	want := "canon-eos-r6/2024/05/23/raw"
	if got != want {
		t.Fatalf("template = %q, want %q", got, want)
	}
}

func TestRenderTemplateRejectsUnknownToken(t *testing.T) {
	_, err := RenderTemplate("{unknown}/{year}", EnrichedFile{}, 1)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestResolveStructurePreset(t *testing.T) {
	got, err := ResolveStructure("camera-date")
	if err != nil {
		t.Fatal(err)
	}
	if got != "{camera}/{year}/{month}/{day}/{type}" {
		t.Fatalf("preset = %q", got)
	}
}

func TestRenderRenameTemplate(t *testing.T) {
	file := EnrichedFile{
		File: MediaFile{Type: MediaTypePhoto, Extension: ".jpg"},
		Metadata: Metadata{
			Date:   time.Date(2024, 5, 23, 14, 30, 15, 0, time.Local),
			Camera: "Canon EOS R6",
		},
	}

	got, err := RenderTemplate("{date}_{time}_{camera}_{seq}{ext}", file, 7)
	if err != nil {
		t.Fatal(err)
	}
	want := "2024-05-23_14-30-15_canon-eos-r6_007.jpg"
	if got != want {
		t.Fatalf("rename = %q, want %q", got, want)
	}
}
