package photo

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPlanIngestScansMetadataHashesAndPlans(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	dest := filepath.Join(root, "dest")
	if err := os.MkdirAll(filepath.Join(source, "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(source, "nested", "IMG_0001.JPG")
	if err := os.WriteFile(path, []byte("photo"), 0o644); err != nil {
		t.Fatal(err)
	}

	plan, summary, err := PlanIngest(Options{
		Source:      source,
		Destination: dest,
		Recursive:   true,
		Structure:   "{camera}/{year}/{month}/{day}/{type}",
		Duplicates:  DuplicateSkip,
		Report:      ReportNone,
	}, fakeMetadataProvider{metadata: Metadata{
		Date:   time.Date(2024, 5, 23, 14, 30, 15, 0, time.Local),
		Camera: "Canon EOS R6",
		Lens:   "RF 35mm",
	}})
	if err != nil {
		t.Fatal(err)
	}
	if summary.Scanned != 1 || summary.Media != 1 {
		t.Fatalf("summary = %+v", summary)
	}
	want := filepath.Join(dest, "canon-eos-r6", "2024", "05", "23", "photos", "IMG_0001.JPG")
	if plan.Actions[0].DestPath != want {
		t.Fatalf("DestPath = %q, want %q", plan.Actions[0].DestPath, want)
	}
}

func TestPlanIngestAddsBurstGrouping(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	dest := filepath.Join(root, "dest")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}
	a := filepath.Join(source, "a_20260524_100000.jpg")
	b := filepath.Join(source, "b_20260524_100001.jpg")
	if err := os.WriteFile(a, []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b, []byte("b"), 0o644); err != nil {
		t.Fatal(err)
	}

	plan, summary, err := PlanIngest(Options{
		Source:      source,
		Destination: dest,
		Recursive:   true,
		Structure:   DefaultStructure,
		Rename:      "grouped",
		Duplicates:  DuplicateSkip,
		Report:      ReportNone,
		BurstWindow: 2 * time.Second,
	}, fakeMetadataProvider{metadata: Metadata{
		Date:   time.Date(2026, 5, 24, 10, 0, 0, 0, time.Local),
		Camera: "Canon R6",
	}})
	if err != nil {
		t.Fatal(err)
	}
	if summary.BurstGroups != 1 {
		t.Fatalf("BurstGroups = %d, want 1", summary.BurstGroups)
	}
	if plan.Actions[0].DestPath == "" || plan.Actions[1].DestPath == "" {
		t.Fatalf("missing destinations: %+v", plan.Actions)
	}
}

func TestPlanIngestFullPerformanceReadsMetadataConcurrently(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	dest := filepath.Join(root, "dest")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"a.jpg", "b.jpg", "c.jpg", "d.jpg"} {
		if err := os.WriteFile(filepath.Join(source, name), []byte(name), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	provider := &blockingMetadataProvider{
		metadata: Metadata{
			Date:   time.Date(2026, 5, 24, 10, 0, 0, 0, time.Local),
			Camera: "Canon R6",
		},
		blockAt: 2,
	}
	defer provider.releaseAll()

	_, _, err := PlanIngest(Options{
		Source:          source,
		Destination:     dest,
		Recursive:       true,
		Structure:       DefaultStructure,
		Duplicates:      DuplicateSkip,
		Report:          ReportNone,
		FullPerformance: true,
	}, provider)
	if err != nil {
		t.Fatal(err)
	}
	if provider.maxActive < 2 {
		t.Fatalf("maxActive = %d, want concurrent metadata reads", provider.maxActive)
	}
}
