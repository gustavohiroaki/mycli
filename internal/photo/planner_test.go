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

func TestBuildPlanWithGroupingUsesGroupedRename(t *testing.T) {
	files := []EnrichedFile{
		{
			File:     MediaFile{SourcePath: "/src/a.jpg", Type: MediaTypePhoto, Extension: ".jpg"},
			Metadata: Metadata{Date: time.Date(2026, 5, 24, 10, 0, 0, 0, time.Local), Camera: "Canon R6"},
			Hash:     "aaaa",
		},
		{
			File:     MediaFile{SourcePath: "/src/b.jpg", Type: MediaTypePhoto, Extension: ".jpg"},
			Metadata: Metadata{Date: time.Date(2026, 5, 24, 10, 0, 1, 0, time.Local), Camera: "Canon R6"},
			Hash:     "bbbb",
		},
	}
	grouping := GroupingResult{
		BurstGroups: []FileGroup{{ID: "burst-001", Type: GroupBurst, Files: []string{"/src/a.jpg", "/src/b.jpg"}}},
		PreferredGroupByFile: map[string]string{
			"/src/a.jpg": "burst-001",
			"/src/b.jpg": "burst-001",
		},
	}

	plan, err := BuildPlanWithGrouping(files, map[string]bool{}, Options{
		Destination: "/dest",
		Structure:   DefaultStructure,
		Rename:      "grouped",
		Duplicates:  DuplicateSkip,
	}, grouping)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(plan.Actions[0].DestPath) != "2026-05-24_10-00-00_canon-r6.jpg" {
		t.Fatalf("first DestPath = %q", plan.Actions[0].DestPath)
	}
	if filepath.Base(plan.Actions[1].DestPath) != "2026-05-24_10-00-00_canon-r6_1.jpg" {
		t.Fatalf("second DestPath = %q", plan.Actions[1].DestPath)
	}
}
