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
