package photo

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func RenderTextReport(plan Plan, summary Summary) string {
	var builder strings.Builder
	builder.WriteString("Photo ingest report\n")
	builder.WriteString(fmt.Sprintf("Generated: %s\n", time.Now().Format(time.RFC3339)))
	builder.WriteString(fmt.Sprintf("Source: %s\n", plan.Options.Source))
	builder.WriteString(fmt.Sprintf("Destination: %s\n", plan.Options.Destination))
	builder.WriteString(fmt.Sprintf("Recursive: %v\n", plan.Options.Recursive))
	builder.WriteString(fmt.Sprintf("Excludes: %s\n", strings.Join(plan.Options.Excludes, ", ")))
	builder.WriteString(fmt.Sprintf("Move: %v\n", plan.Options.Move))
	builder.WriteString(fmt.Sprintf("Structure: %s\n", plan.Options.Structure))
	builder.WriteString(fmt.Sprintf("Rename: %s\n", plan.Options.Rename))
	builder.WriteString(fmt.Sprintf("Duplicate policy: %s\n", plan.Options.Duplicates))
	builder.WriteString(fmt.Sprintf("Media files: %d\n", summary.Media))
	builder.WriteString(fmt.Sprintf("Copied: %d\n", summary.Copied))
	builder.WriteString(fmt.Sprintf("Moved: %d\n", summary.Moved))
	builder.WriteString(fmt.Sprintf("Skipped: %d\n", summary.Skipped))
	builder.WriteString(fmt.Sprintf("Duplicates: %d\n", summary.Duplicates))
	builder.WriteString(fmt.Sprintf("Failed: %d\n", summary.Failed))
	builder.WriteString(fmt.Sprintf("Photos: %d\n", summary.Photos))
	builder.WriteString(fmt.Sprintf("Videos: %d\n", summary.Videos))
	builder.WriteString(fmt.Sprintf("Raw: %d\n", summary.Raw))
	builder.WriteString(fmt.Sprintf("Fallback metadata: %d\n", summary.FallbackDates))
	return builder.String()
}

func WriteReport(plan Plan, summary Summary) (string, error) {
	if plan.Options.Report == "" {
		plan.Options.Report = ReportText
	}
	if plan.Options.Report == ReportNone {
		return "", nil
	}
	if err := os.MkdirAll(plan.Options.Destination, 0o755); err != nil {
		return "", err
	}

	switch plan.Options.Report {
	case ReportJSON:
		path := filepath.Join(plan.Options.Destination, "photo-ingest-report.json")
		payload, err := json.MarshalIndent(struct {
			Options Options `json:"options"`
			Summary Summary `json:"summary"`
		}{Options: plan.Options, Summary: summary}, "", "  ")
		if err != nil {
			return "", err
		}
		return path, os.WriteFile(path, payload, 0o644)
	case ReportText:
		path := filepath.Join(plan.Options.Destination, "photo-ingest-report.txt")
		return path, os.WriteFile(path, []byte(RenderTextReport(plan, summary)), 0o644)
	default:
		return "", fmt.Errorf("invalid report format %q", plan.Options.Report)
	}
}
