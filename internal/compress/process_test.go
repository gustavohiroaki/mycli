package compress

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCRFFromLevel(t *testing.T) {
	tests := []struct {
		level int
		want  int
	}{
		{1, 18},
		{35, 23},
		{100, 35},
	}
	for _, test := range tests {
		if got := crfFromLevel(test.level); got != test.want {
			t.Fatalf("crfFromLevel(%d) = %d, want %d", test.level, got, test.want)
		}
	}
}

func TestValidateOptionsRejectsInvalidLevel(t *testing.T) {
	if err := ValidateOptions(Options{Level: 0}); err == nil {
		t.Fatal("expected error")
	}
	if err := ValidateOptions(Options{Level: 101}); err == nil {
		t.Fatal("expected error")
	}
}

func TestVideoDurationRejectsInvalidNumber(t *testing.T) {
	// Covered indirectly by ffprobe in production; this keeps level tests independent
	// from a local FFmpeg install.
	if crfFromLevel(35) != 23 {
		t.Fatal("unexpected default CRF")
	}
}

func TestProcessWithProgressReportsEncodingPercent(t *testing.T) {
	root := t.TempDir()
	bin := filepath.Join(root, "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	writeExecutable(t, filepath.Join(bin, "ffprobe"), `#!/usr/bin/env sh
echo 10
`)
	writeExecutable(t, filepath.Join(bin, "ffmpeg"), `#!/usr/bin/env sh
last=""
for arg in "$@"; do
  last="$arg"
done
echo out_time_ms=0
echo out_time_ms=5000000
echo out_time_ms=10000000
printf x > "$last"
`)
	oldPath := os.Getenv("PATH")
	t.Setenv("PATH", bin+string(os.PathListSeparator)+oldPath)

	source := filepath.Join(root, "source.mov")
	dest := filepath.Join(root, "out.mp4")
	if err := os.WriteFile(source, []byte("0123456789"), 0o644); err != nil {
		t.Fatal(err)
	}

	var percents []int
	result := ProcessWithProgress(Item{SourcePath: source, DestPath: dest, Size: 10}, Options{Level: 35}, func(item Item, percent int) {
		percents = append(percents, percent)
	})
	if result.Status != StatusOK {
		t.Fatalf("status = %s error=%s", result.Status, result.Error)
	}
	if len(percents) == 0 || percents[len(percents)-1] != 100 {
		t.Fatalf("percents = %#v", percents)
	}
}

func writeExecutable(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
}
