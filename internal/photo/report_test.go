package photo

import (
	"strings"
	"testing"
)

func TestRenderTextReportIncludesSummary(t *testing.T) {
	report := RenderTextReport(Plan{Options: Options{
		Source: "/src", Destination: "/dest", Recursive: true, Structure: DefaultStructure, Duplicates: DuplicateSkip,
	}}, Summary{Media: 2, Copied: 1, Skipped: 1, Duplicates: 1})

	for _, want := range []string{"Source: /src", "Destination: /dest", "Media files: 2", "Copied: 1", "Duplicates: 1"} {
		if !strings.Contains(report, want) {
			t.Fatalf("report missing %q:\n%s", want, report)
		}
	}
}

func TestRenderTextReportIncludesGroupingSummary(t *testing.T) {
	report := RenderTextReport(Plan{Options: Options{
		Source: "/src", Destination: "/dest", Recursive: true, Structure: DefaultStructure, Duplicates: DuplicateSkip,
	}}, Summary{BurstGroups: 2, LargestBurst: 5, SimilarGroups: 1, LargestSimilar: 3, VisualSimilaritySkipped: 4})

	for _, want := range []string{"Burst groups: 2", "Largest burst: 5", "Similar groups: 1", "Largest similar group: 3", "Visual similarity skipped: 4"} {
		if !strings.Contains(report, want) {
			t.Fatalf("report missing %q:\n%s", want, report)
		}
	}
}
