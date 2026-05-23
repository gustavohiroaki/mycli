package photo

import (
	"path/filepath"
	"testing"
	"time"
)

func TestBuildPlanCreatesDestinationFromTemplate(t *testing.T) {
	file := EnrichedFile{
		File:     MediaFile{SourcePath: "/src/IMG_0001.CR3", Type: MediaTypeRaw, Extension: ".cr3"},
		Metadata: Metadata{Date: time.Date(2024, 5, 23, 14, 30, 15, 0, time.Local), Camera: "Canon EOS R6"},
		Hash:     "abc",
	}

	plan, err := BuildPlan([]EnrichedFile{file}, map[string]bool{}, Options{
		Destination: "/dest",
		Recursive:   true,
		Structure:   "{camera}/{year}/{month}/{day}/{type}",
		Duplicates:  DuplicateSkip,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Actions) != 1 {
		t.Fatalf("actions = %d", len(plan.Actions))
	}
	want := filepath.Join("/dest", "canon-eos-r6", "2024", "05", "23", "raw", "IMG_0001.CR3")
	if plan.Actions[0].DestPath != want {
		t.Fatalf("DestPath = %q, want %q", plan.Actions[0].DestPath, want)
	}
	if plan.Actions[0].Kind != ActionCopy {
		t.Fatalf("Kind = %q", plan.Actions[0].Kind)
	}
}

func TestBuildPlanSkipsDuplicatesByDefault(t *testing.T) {
	file := EnrichedFile{
		File:     MediaFile{SourcePath: "/src/IMG_0001.JPG", Type: MediaTypePhoto, Extension: ".jpg"},
		Metadata: Metadata{Date: time.Date(2024, 5, 23, 14, 30, 15, 0, time.Local)},
		Hash:     "abc",
	}

	plan, err := BuildPlan([]EnrichedFile{file}, map[string]bool{"/src/IMG_0001.JPG": true}, Options{
		Destination: "/dest",
		Structure:   DefaultStructure,
		Duplicates:  DuplicateSkip,
	})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Actions[0].Kind != ActionSkip {
		t.Fatalf("Kind = %q", plan.Actions[0].Kind)
	}
}

func TestBuildPlanUsesRenameTemplate(t *testing.T) {
	file := EnrichedFile{
		File:     MediaFile{SourcePath: "/src/IMG_0001.JPG", Type: MediaTypePhoto, Extension: ".jpg"},
		Metadata: Metadata{Date: time.Date(2024, 5, 23, 14, 30, 15, 0, time.Local), Camera: "Canon EOS R6"},
		Hash:     "abc",
	}

	plan, err := BuildPlan([]EnrichedFile{file}, map[string]bool{}, Options{
		Destination: "/dest",
		Structure:   DefaultStructure,
		Rename:      DefaultRename,
		Duplicates:  DuplicateSkip,
	})
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(plan.Actions[0].DestPath) != "2024-05-23_14-30-15_canon-eos-r6_001.jpg" {
		t.Fatalf("DestPath = %q", plan.Actions[0].DestPath)
	}
}

func TestBuildPlanSeparatesDuplicates(t *testing.T) {
	file := EnrichedFile{
		File:     MediaFile{SourcePath: "/src/IMG_0001.JPG", Type: MediaTypePhoto, Extension: ".jpg"},
		Metadata: Metadata{Date: time.Date(2024, 5, 23, 14, 30, 15, 0, time.Local)},
		Hash:     "abc",
	}

	plan, err := BuildPlan([]EnrichedFile{file}, map[string]bool{"/src/IMG_0001.JPG": true}, Options{
		Destination: "/dest",
		Structure:   DefaultStructure,
		Duplicates:  DuplicateSeparate,
	})
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join("/dest", "duplicates", "2024", "05", "23", "photos", "IMG_0001.JPG")
	if plan.Actions[0].DestPath != want {
		t.Fatalf("DestPath = %q, want %q", plan.Actions[0].DestPath, want)
	}
}
