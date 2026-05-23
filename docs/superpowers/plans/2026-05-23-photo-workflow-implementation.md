# Photo Workflow Utility Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build `mycli photo` as a guided photography workflow and migrate the current root `organize` behavior into `mycli photo organize`.

**Architecture:** Move organization logic out of `cmd/organize.go` into focused `internal/photo` package units. Keep Cobra/menu code in `cmd/photo.go`, while scanner, metadata, templates, duplicates, planner, executor, report, and ingest orchestration remain testable without terminal interaction.

**Tech Stack:** Go 1.24.1, Cobra, standard library filesystem APIs, `exiftool` as external metadata provider, Go `testing` package.

---

## File Structure

- Create `internal/photo/types.go`: shared option, metadata, file, action, summary, and report types.
- Create `internal/photo/scanner.go`: recursive scan, media extension classification, explicit exclude matching, source/destination safety validation.
- Create `internal/photo/scanner_test.go`: scanner, exclude, media type, and destination safety tests.
- Create `internal/photo/metadata.go`: `exiftool` abstraction, fallback date parsing from filenames and modtime.
- Create `internal/photo/metadata_test.go`: filename date and fallback metadata tests with fake provider.
- Create `internal/photo/templates.go`: folder and rename presets, token rendering, token validation, path segment sanitization.
- Create `internal/photo/templates_test.go`: preset, custom template, unknown token, and sanitization tests.
- Create `internal/photo/duplicates.go`: SHA-256 hashing and duplicate policy classification.
- Create `internal/photo/duplicates_test.go`: duplicate policy tests.
- Create `internal/photo/planner.go`: convert scanned/enriched files into copy/move/skip actions with destination paths.
- Create `internal/photo/planner_test.go`: folder structure, rename, duplicate policy, and collision tests.
- Create `internal/photo/executor.go`: copy/move execution and failure recording.
- Create `internal/photo/executor_test.go`: temp-directory copy/move integration tests.
- Create `internal/photo/report.go`: text and JSON report writers.
- Create `internal/photo/report_test.go`: report content tests.
- Create `internal/photo/ingest.go`: orchestration API used by CLI.
- Create `internal/photo/ingest_test.go`: end-to-end package tests with fake metadata provider.
- Create `cmd/photo.go`: Cobra `photo` command, guided menu, direct `photo organize` command and flags.
- Delete `cmd/organize.go`: old root command is removed from public command tree after behavior migrates.
- Modify `README.md`: document `mycli photo`, `mycli photo organize`, recursive scanning, `exiftool`, flags, and examples.

---

### Task 1: Shared Types And Safety Helpers

**Files:**
- Create: `internal/photo/types.go`
- Create: `internal/photo/scanner.go`
- Create: `internal/photo/scanner_test.go`

- [ ] **Step 1: Write failing tests for media classification and destination safety**

Create `internal/photo/scanner_test.go`:

```go
package photo

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectMediaType(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		wantType  MediaType
		wantMatch bool
	}{
		{name: "jpeg photo", path: "IMG_0001.JPG", wantType: MediaTypePhoto, wantMatch: true},
		{name: "raw photo", path: "IMG_0001.CR3", wantType: MediaTypeRaw, wantMatch: true},
		{name: "video", path: "clip.MOV", wantType: MediaTypeVideo, wantMatch: true},
		{name: "other", path: "notes.txt", wantType: "", wantMatch: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotType, gotMatch := DetectMediaType(tt.path)
			if gotMatch != tt.wantMatch {
				t.Fatalf("match = %v, want %v", gotMatch, tt.wantMatch)
			}
			if gotType != tt.wantType {
				t.Fatalf("type = %q, want %q", gotType, tt.wantType)
			}
		})
	}
}

func TestValidateSourceDestinationRejectsDestinationInsideSource(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	dest := filepath.Join(source, "organized")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}

	err := ValidateSourceDestination(source, dest)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateSourceDestinationRequiresDirectorySource(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "file.jpg")
	dest := filepath.Join(root, "dest")
	if err := os.WriteFile(source, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := ValidateSourceDestination(source, dest)
	if err == nil {
		t.Fatal("expected error")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run:

```bash
go test ./internal/photo
```

Expected: FAIL because `internal/photo` package and symbols do not exist.

- [ ] **Step 3: Add shared types**

Create `internal/photo/types.go`:

```go
package photo

import "time"

type MediaType string

const (
	MediaTypePhoto MediaType = "photos"
	MediaTypeVideo MediaType = "videos"
	MediaTypeRaw   MediaType = "raw"
)

type DuplicatePolicy string

const (
	DuplicateSkip     DuplicatePolicy = "skip"
	DuplicateSeparate DuplicatePolicy = "separate"
	DuplicateSuffix   DuplicatePolicy = "suffix"
)

type ReportFormat string

const (
	ReportText ReportFormat = "txt"
	ReportJSON ReportFormat = "json"
	ReportNone ReportFormat = "none"
)

type Options struct {
	Source        string
	Destination   string
	Recursive     bool
	Excludes      []string
	Move          bool
	Structure     string
	Rename        string
	Duplicates    DuplicatePolicy
	AllowFallback bool
	Report        ReportFormat
}

type MediaFile struct {
	SourcePath string
	RelPath    string
	Type       MediaType
	Extension  string
	Size       int64
	ModTime    time.Time
}

type Metadata struct {
	Date         time.Time
	Camera       string
	Lens         string
	UsedFallback bool
}

type EnrichedFile struct {
	File     MediaFile
	Metadata Metadata
	Hash     string
}

type ActionKind string

const (
	ActionCopy ActionKind = "copy"
	ActionMove ActionKind = "move"
	ActionSkip ActionKind = "skip"
)

type PlannedAction struct {
	Kind        ActionKind
	SourcePath  string
	DestPath    string
	MediaType   MediaType
	Duplicate   bool
	UsedFallback bool
	Error       string
}

type Plan struct {
	Options Options
	Actions []PlannedAction
}

type Summary struct {
	Scanned       int
	Media         int
	Copied        int
	Moved         int
	Skipped       int
	Duplicates    int
	Failed        int
	Photos        int
	Videos        int
	Raw           int
	FallbackDates int
}
```

- [ ] **Step 4: Add media classification and destination validation**

Create `internal/photo/scanner.go`:

```go
package photo

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var photoExtensions = map[string]struct{}{
	".jpg": {}, ".jpeg": {}, ".png": {}, ".gif": {}, ".webp": {},
	".bmp": {}, ".tif": {}, ".tiff": {}, ".heic": {}, ".heif": {},
}

var rawExtensions = map[string]struct{}{
	".dng": {}, ".raw": {}, ".cr2": {}, ".cr3": {}, ".crw": {},
	".nef": {}, ".nrw": {}, ".arw": {}, ".srf": {}, ".sr2": {},
	".raf": {}, ".rw2": {}, ".orf": {}, ".pef": {}, ".x3f": {},
}

var videoExtensions = map[string]struct{}{
	".mp4": {}, ".mov": {}, ".avi": {}, ".mkv": {}, ".m4v": {},
	".3gp": {}, ".mts": {}, ".m2ts": {}, ".mpg": {}, ".mpeg": {},
	".wmv": {}, ".flv": {}, ".webm": {},
}

func DetectMediaType(path string) (MediaType, bool) {
	ext := strings.ToLower(filepath.Ext(path))
	if _, ok := photoExtensions[ext]; ok {
		return MediaTypePhoto, true
	}
	if _, ok := rawExtensions[ext]; ok {
		return MediaTypeRaw, true
	}
	if _, ok := videoExtensions[ext]; ok {
		return MediaTypeVideo, true
	}
	return "", false
}

func ValidateSourceDestination(source string, destination string) error {
	sourceAbs, err := filepath.Abs(source)
	if err != nil {
		return fmt.Errorf("resolve source: %w", err)
	}
	destinationAbs, err := filepath.Abs(destination)
	if err != nil {
		return fmt.Errorf("resolve destination: %w", err)
	}
	if sourceAbs == destinationAbs {
		return errors.New("source and destination cannot be the same")
	}
	info, err := os.Stat(sourceAbs)
	if err != nil {
		return fmt.Errorf("access source: %w", err)
	}
	if !info.IsDir() {
		return errors.New("source must be a directory")
	}
	if isSubpath(destinationAbs, sourceAbs) {
		return errors.New("destination cannot be inside source")
	}
	return nil
}

func isSubpath(candidate string, parent string) bool {
	relative, err := filepath.Rel(parent, candidate)
	if err != nil {
		return false
	}
	return relative != "." && relative != ".." && !strings.HasPrefix(relative, ".."+string(os.PathSeparator))
}
```

- [ ] **Step 5: Run tests**

Run:

```bash
go test ./internal/photo
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/photo/types.go internal/photo/scanner.go internal/photo/scanner_test.go
git commit -m "feat: add photo workflow base types"
```

---

### Task 2: Recursive Scanner With Explicit Excludes

**Files:**
- Modify: `internal/photo/scanner.go`
- Modify: `internal/photo/scanner_test.go`

- [ ] **Step 1: Write failing scanner tests**

Append to `internal/photo/scanner_test.go`:

```go
func TestScanFindsNestedMediaByDefault(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	nested := filepath.Join(source, "DCIM", "100CANON")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nested, "IMG_0001.JPG"), []byte("photo"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "notes.txt"), []byte("notes"), 0o644); err != nil {
		t.Fatal(err)
	}

	files, err := Scan(source, ScanOptions{Recursive: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Fatalf("len(files) = %d, want 1", len(files))
	}
	if files[0].RelPath != filepath.Join("DCIM", "100CANON", "IMG_0001.JPG") {
		t.Fatalf("RelPath = %q", files[0].RelPath)
	}
}

func TestScanNoRecursiveSkipsNestedFiles(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	nested := filepath.Join(source, "nested")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "top.JPG"), []byte("photo"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nested, "nested.JPG"), []byte("photo"), 0o644); err != nil {
		t.Fatal(err)
	}

	files, err := Scan(source, ScanOptions{Recursive: false})
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Fatalf("len(files) = %d, want 1", len(files))
	}
	if filepath.Base(files[0].SourcePath) != "top.JPG" {
		t.Fatalf("SourcePath = %q", files[0].SourcePath)
	}
}

func TestScanUsesOnlyExplicitExcludes(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	exports := filepath.Join(source, "exports")
	if err := os.MkdirAll(exports, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(exports, "skip.JPG"), []byte("photo"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "keep.JPG"), []byte("photo"), 0o644); err != nil {
		t.Fatal(err)
	}

	files, err := Scan(source, ScanOptions{Recursive: true, Excludes: []string{"exports"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Fatalf("len(files) = %d, want 1", len(files))
	}
	if filepath.Base(files[0].SourcePath) != "keep.JPG" {
		t.Fatalf("SourcePath = %q", files[0].SourcePath)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run:

```bash
go test ./internal/photo
```

Expected: FAIL because `Scan` and `ScanOptions` do not exist.

- [ ] **Step 3: Implement scanner**

Append to `internal/photo/scanner.go`:

```go
type ScanOptions struct {
	Recursive bool
	Excludes  []string
}

func Scan(source string, options ScanOptions) ([]MediaFile, error) {
	sourceAbs, err := filepath.Abs(source)
	if err != nil {
		return nil, fmt.Errorf("resolve source: %w", err)
	}

	var files []MediaFile
	err = filepath.WalkDir(sourceAbs, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if path == sourceAbs {
			return nil
		}

		rel, err := filepath.Rel(sourceAbs, path)
		if err != nil {
			return err
		}

		if matchesExclude(rel, options.Excludes) {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if entry.IsDir() {
			if !options.Recursive {
				return filepath.SkipDir
			}
			return nil
		}

		mediaType, ok := DetectMediaType(path)
		if !ok {
			return nil
		}

		info, err := entry.Info()
		if err != nil {
			return err
		}

		files = append(files, MediaFile{
			SourcePath: path,
			RelPath:    rel,
			Type:       mediaType,
			Extension:  strings.ToLower(filepath.Ext(path)),
			Size:       info.Size(),
			ModTime:    info.ModTime(),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}

func matchesExclude(rel string, excludes []string) bool {
	cleanRel := filepath.ToSlash(filepath.Clean(rel))
	for _, exclude := range excludes {
		cleanExclude := filepath.ToSlash(filepath.Clean(strings.TrimSpace(exclude)))
		if cleanExclude == "." || cleanExclude == "" {
			continue
		}
		if cleanRel == cleanExclude || strings.HasPrefix(cleanRel, cleanExclude+"/") {
			return true
		}
		matched, err := filepath.Match(cleanExclude, filepath.Base(cleanRel))
		if err == nil && matched {
			return true
		}
	}
	return false
}
```

- [ ] **Step 4: Run tests**

Run:

```bash
go test ./internal/photo
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/photo/scanner.go internal/photo/scanner_test.go
git commit -m "feat: scan photo sources recursively"
```

---

### Task 3: Metadata Reader With Exiftool Abstraction And Fallback

**Files:**
- Create: `internal/photo/metadata.go`
- Create: `internal/photo/metadata_test.go`

- [ ] **Step 1: Write failing metadata tests**

Create `internal/photo/metadata_test.go`:

```go
package photo

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

type fakeMetadataProvider struct {
	metadata Metadata
	err      error
}

func (f fakeMetadataProvider) Available() bool {
	return f.err == nil
}

func (f fakeMetadataProvider) Read(path string) (Metadata, error) {
	if f.err != nil {
		return Metadata{}, f.err
	}
	return f.metadata, nil
}

func TestDateFromFilenameParsesCompactDateTime(t *testing.T) {
	got, ok := DateFromFilename("IMG_20240523_143015.CR3")
	if !ok {
		t.Fatal("expected date")
	}
	if got.Format("2006-01-02 15:04:05") != "2024-05-23 14:30:15" {
		t.Fatalf("date = %s", got.Format("2006-01-02 15:04:05"))
	}
}

func TestResolveMetadataUsesProvider(t *testing.T) {
	wantDate := time.Date(2024, 5, 23, 14, 30, 0, 0, time.Local)
	file := MediaFile{SourcePath: "IMG_0001.JPG", ModTime: time.Now()}
	got := ResolveMetadata(file, fakeMetadataProvider{metadata: Metadata{
		Date:   wantDate,
		Camera: "Canon EOS R6",
		Lens:   "RF 35mm",
	}})

	if !got.Date.Equal(wantDate) {
		t.Fatalf("Date = %v, want %v", got.Date, wantDate)
	}
	if got.Camera != "Canon EOS R6" {
		t.Fatalf("Camera = %q", got.Camera)
	}
	if got.UsedFallback {
		t.Fatal("UsedFallback = true, want false")
	}
}

func TestResolveMetadataFallsBackToFilenameThenModTime(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "IMG_20240523_143015.JPG")
	if err := os.WriteFile(path, []byte("photo"), 0o644); err != nil {
		t.Fatal(err)
	}
	file := MediaFile{SourcePath: path, ModTime: time.Date(2020, 1, 2, 3, 4, 5, 0, time.Local)}

	got := ResolveMetadata(file, fakeMetadataProvider{err: os.ErrNotExist})
	if got.Date.Format("2006-01-02 15:04:05") != "2024-05-23 14:30:15" {
		t.Fatalf("Date = %s", got.Date.Format("2006-01-02 15:04:05"))
	}
	if got.Camera != "unknown-camera" {
		t.Fatalf("Camera = %q", got.Camera)
	}
	if !got.UsedFallback {
		t.Fatal("UsedFallback = false, want true")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run:

```bash
go test ./internal/photo
```

Expected: FAIL because metadata symbols do not exist.

- [ ] **Step 3: Implement metadata reader**

Create `internal/photo/metadata.go`:

```go
package photo

import (
	"errors"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type MetadataProvider interface {
	Available() bool
	Read(path string) (Metadata, error)
}

type ExiftoolProvider struct{}

func (ExiftoolProvider) Available() bool {
	_, err := exec.LookPath("exiftool")
	return err == nil
}

func (ExiftoolProvider) Read(path string) (Metadata, error) {
	if !ExiftoolProvider{}.Available() {
		return Metadata{}, errors.New("exiftool not found")
	}
	output, err := exec.Command(
		"exiftool",
		"-s3",
		"-d", "%Y-%m-%d %H:%M:%S",
		"-DateTimeOriginal",
		"-CreateDate",
		"-MediaCreateDate",
		"-TrackCreateDate",
		"-Model",
		"-LensModel",
		path,
	).Output()
	if err != nil {
		return Metadata{}, err
	}

	lines := strings.Split(string(output), "\n")
	var metadata Metadata
	for _, line := range lines {
		value := strings.TrimSpace(line)
		if value == "" {
			continue
		}
		if metadata.Date.IsZero() {
			if parsed, err := time.ParseInLocation("2006-01-02 15:04:05", value, time.Local); err == nil {
				metadata.Date = parsed
				continue
			}
		}
		if metadata.Camera == "" {
			metadata.Camera = value
			continue
		}
		if metadata.Lens == "" {
			metadata.Lens = value
		}
	}
	if metadata.Date.IsZero() {
		return Metadata{}, errors.New("metadata date not found")
	}
	if metadata.Camera == "" {
		metadata.Camera = "unknown-camera"
	}
	if metadata.Lens == "" {
		metadata.Lens = "unknown-lens"
	}
	return metadata, nil
}

var datePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(20\d{2})[-_]?([01]\d)[-_]?([0-3]\d)[T _-]?([0-2]\d)?([0-5]\d)?([0-5]\d)?`),
	regexp.MustCompile(`(?i)(\d{2})[-_]?([01]\d)[-_]?(20\d{2})`),
}

func ResolveMetadata(file MediaFile, provider MetadataProvider) Metadata {
	if provider != nil && provider.Available() {
		metadata, err := provider.Read(file.SourcePath)
		if err == nil {
			metadata.UsedFallback = false
			if metadata.Camera == "" {
				metadata.Camera = "unknown-camera"
			}
			if metadata.Lens == "" {
				metadata.Lens = "unknown-lens"
			}
			return metadata
		}
	}

	metadata := Metadata{
		Date:         file.ModTime,
		Camera:       "unknown-camera",
		Lens:         "unknown-lens",
		UsedFallback: true,
	}
	if date, ok := DateFromFilename(filepath.Base(file.SourcePath)); ok {
		metadata.Date = date
	}
	return metadata
}

func DateFromFilename(name string) (time.Time, bool) {
	for index, pattern := range datePatterns {
		matches := pattern.FindStringSubmatch(name)
		if len(matches) == 0 {
			continue
		}

		var year, month, day int
		var hour, minute, second int
		var err error

		if index == 0 {
			year, err = strconv.Atoi(matches[1])
			if err != nil {
				continue
			}
			month, err = strconv.Atoi(matches[2])
			if err != nil {
				continue
			}
			day, err = strconv.Atoi(matches[3])
			if err != nil {
				continue
			}
			hour = atoiDefault(matches[4])
			minute = atoiDefault(matches[5])
			second = atoiDefault(matches[6])
		} else {
			day, err = strconv.Atoi(matches[1])
			if err != nil {
				continue
			}
			month, err = strconv.Atoi(matches[2])
			if err != nil {
				continue
			}
			year, err = strconv.Atoi(matches[3])
			if err != nil {
				continue
			}
		}

		mediaDate := time.Date(year, time.Month(month), day, hour, minute, second, 0, time.Local)
		if mediaDate.Year() == year && int(mediaDate.Month()) == month && mediaDate.Day() == day {
			return mediaDate, true
		}
	}
	return time.Time{}, false
}

func atoiDefault(value string) int {
	if value == "" {
		return 0
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return parsed
}
```

- [ ] **Step 4: Run tests**

Run:

```bash
go test ./internal/photo
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/photo/metadata.go internal/photo/metadata_test.go
git commit -m "feat: read photo metadata with fallback"
```

---

### Task 4: Template Rendering For Folders And Names

**Files:**
- Create: `internal/photo/templates.go`
- Create: `internal/photo/templates_test.go`

- [ ] **Step 1: Write failing template tests**

Create `internal/photo/templates_test.go`:

```go
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run:

```bash
go test ./internal/photo
```

Expected: FAIL because template functions do not exist.

- [ ] **Step 3: Implement templates**

Create `internal/photo/templates.go`:

```go
package photo

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	DefaultStructure = "{year}/{month}/{day}/{type}"
	DefaultRename    = "{date}_{time}_{camera}_{seq}{ext}"
)

var structurePresets = map[string]string{
	"date":        "{year}/{month}/{day}/{type}",
	"date-folder": "{year}/{year}-{month}-{day}/{type}",
	"camera-date": "{camera}/{year}/{month}/{day}/{type}",
	"year-camera": "{year}/{camera}/{year}-{month}-{day}/{type}",
	"legacy-pt":   "{year}/{month}/{day}/{type}",
}

var tokenPattern = regexp.MustCompile(`\{([a-z]+)\}`)
var unsafeSegmentChars = regexp.MustCompile(`[^a-z0-9._-]+`)

func ResolveStructure(value string) (string, error) {
	if strings.TrimSpace(value) == "" {
		return DefaultStructure, nil
	}
	if preset, ok := structurePresets[value]; ok {
		return preset, nil
	}
	if err := ValidateTemplate(value); err != nil {
		return "", err
	}
	return value, nil
}

func ValidateTemplate(template string) error {
	tokens := tokenPattern.FindAllStringSubmatch(template, -1)
	for _, token := range tokens {
		if _, ok := allowedTokenSet()[token[1]]; !ok {
			return fmt.Errorf("unknown template token %q", token[1])
		}
	}
	return nil
}

func RenderTemplate(template string, file EnrichedFile, seq int) (string, error) {
	if err := ValidateTemplate(template); err != nil {
		return "", err
	}

	values := map[string]string{
		"year":      file.Metadata.Date.Format("2006"),
		"month":     file.Metadata.Date.Format("01"),
		"day":       file.Metadata.Date.Format("02"),
		"date":      file.Metadata.Date.Format("2006-01-02"),
		"time":      file.Metadata.Date.Format("15-04-05"),
		"camera":    sanitizeTokenValue(defaultString(file.Metadata.Camera, "unknown-camera")),
		"lens":      sanitizeTokenValue(defaultString(file.Metadata.Lens, "unknown-lens")),
		"type":      string(file.File.Type),
		"extension": strings.TrimPrefix(strings.ToLower(file.File.Extension), "."),
		"ext":       strings.ToLower(file.File.Extension),
		"seq":       fmt.Sprintf("%03d", seq),
	}

	return tokenPattern.ReplaceAllStringFunc(template, func(token string) string {
		key := strings.TrimSuffix(strings.TrimPrefix(token, "{"), "}")
		return values[key]
	}), nil
}

func allowedTokenSet() map[string]struct{} {
	return map[string]struct{}{
		"year": {}, "month": {}, "day": {}, "date": {}, "time": {},
		"camera": {}, "lens": {}, "type": {}, "extension": {}, "ext": {}, "seq": {},
	}
}

func sanitizeTokenValue(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = unsafeSegmentChars.ReplaceAllString(value, "-")
	value = strings.Trim(value, "-")
	if value == "" {
		return "unknown"
	}
	return value
}

func defaultString(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
```

- [ ] **Step 4: Run tests**

Run:

```bash
go test ./internal/photo
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/photo/templates.go internal/photo/templates_test.go
git commit -m "feat: render photo organization templates"
```

---

### Task 5: Duplicate Detection

**Files:**
- Create: `internal/photo/duplicates.go`
- Create: `internal/photo/duplicates_test.go`

- [ ] **Step 1: Write failing duplicate tests**

Create `internal/photo/duplicates_test.go`:

```go
package photo

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHashFileMatchesSameContent(t *testing.T) {
	root := t.TempDir()
	a := filepath.Join(root, "a.jpg")
	b := filepath.Join(root, "b.jpg")
	if err := os.WriteFile(a, []byte("same"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b, []byte("same"), 0o644); err != nil {
		t.Fatal(err)
	}

	hashA, err := HashFile(a)
	if err != nil {
		t.Fatal(err)
	}
	hashB, err := HashFile(b)
	if err != nil {
		t.Fatal(err)
	}
	if hashA != hashB {
		t.Fatalf("hashes differ")
	}
}

func TestMarkDuplicatesMarksSecondFile(t *testing.T) {
	files := []EnrichedFile{
		{File: MediaFile{SourcePath: "a.jpg"}, Hash: "abc"},
		{File: MediaFile{SourcePath: "b.jpg"}, Hash: "abc"},
		{File: MediaFile{SourcePath: "c.jpg"}, Hash: "def"},
	}

	got := MarkDuplicates(files)
	if got["a.jpg"] {
		t.Fatal("first file should not be duplicate")
	}
	if !got["b.jpg"] {
		t.Fatal("second file should be duplicate")
	}
	if got["c.jpg"] {
		t.Fatal("unique file should not be duplicate")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run:

```bash
go test ./internal/photo
```

Expected: FAIL because duplicate functions do not exist.

- [ ] **Step 3: Implement duplicate helpers**

Create `internal/photo/duplicates.go`:

```go
package photo

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
)

func HashFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func MarkDuplicates(files []EnrichedFile) map[string]bool {
	seen := map[string]string{}
	duplicates := map[string]bool{}
	for _, file := range files {
		if file.Hash == "" {
			continue
		}
		if _, ok := seen[file.Hash]; ok {
			duplicates[file.File.SourcePath] = true
			continue
		}
		seen[file.Hash] = file.File.SourcePath
		duplicates[file.File.SourcePath] = false
	}
	return duplicates
}
```

- [ ] **Step 4: Run tests**

Run:

```bash
go test ./internal/photo
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/photo/duplicates.go internal/photo/duplicates_test.go
git commit -m "feat: detect duplicate photo files"
```

---

### Task 6: Planner

**Files:**
- Create: `internal/photo/planner.go`
- Create: `internal/photo/planner_test.go`

- [ ] **Step 1: Write failing planner tests**

Create `internal/photo/planner_test.go`:

```go
package photo

import (
	"path/filepath"
	"testing"
	"time"
)

func TestBuildPlanCreatesDestinationFromTemplate(t *testing.T) {
	file := EnrichedFile{
		File: MediaFile{SourcePath: "/src/IMG_0001.CR3", Type: MediaTypeRaw, Extension: ".cr3"},
		Metadata: Metadata{Date: time.Date(2024, 5, 23, 14, 30, 15, 0, time.Local), Camera: "Canon EOS R6"},
		Hash: "abc",
	}

	plan, err := BuildPlan([]EnrichedFile{file}, map[string]bool{}, Options{
		Destination: "/dest",
		Recursive: true,
		Structure: "{camera}/{year}/{month}/{day}/{type}",
		Duplicates: DuplicateSkip,
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
		File: MediaFile{SourcePath: "/src/IMG_0001.JPG", Type: MediaTypePhoto, Extension: ".jpg"},
		Metadata: Metadata{Date: time.Date(2024, 5, 23, 14, 30, 15, 0, time.Local)},
		Hash: "abc",
	}

	plan, err := BuildPlan([]EnrichedFile{file}, map[string]bool{"/src/IMG_0001.JPG": true}, Options{
		Destination: "/dest",
		Structure: DefaultStructure,
		Duplicates: DuplicateSkip,
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
		File: MediaFile{SourcePath: "/src/IMG_0001.JPG", Type: MediaTypePhoto, Extension: ".jpg"},
		Metadata: Metadata{Date: time.Date(2024, 5, 23, 14, 30, 15, 0, time.Local), Camera: "Canon EOS R6"},
		Hash: "abc",
	}

	plan, err := BuildPlan([]EnrichedFile{file}, map[string]bool{}, Options{
		Destination: "/dest",
		Structure: DefaultStructure,
		Rename: DefaultRename,
		Duplicates: DuplicateSkip,
	})
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(plan.Actions[0].DestPath) != "2024-05-23_14-30-15_canon-eos-r6_001.jpg" {
		t.Fatalf("DestPath = %q", plan.Actions[0].DestPath)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run:

```bash
go test ./internal/photo
```

Expected: FAIL because `BuildPlan` does not exist.

- [ ] **Step 3: Implement planner**

Create `internal/photo/planner.go`:

```go
package photo

import (
	"path/filepath"
)

func BuildPlan(files []EnrichedFile, duplicates map[string]bool, options Options) (Plan, error) {
	structure, err := ResolveStructure(options.Structure)
	if err != nil {
		return Plan{}, err
	}
	if options.Duplicates == "" {
		options.Duplicates = DuplicateSkip
	}
	if !options.Move {
		options.Move = false
	}

	planned := Plan{Options: options}
	usedDestinations := map[string]int{}
	for index, file := range files {
		isDuplicate := duplicates[file.File.SourcePath]
		action := PlannedAction{
			Kind:         ActionCopy,
			SourcePath:   file.File.SourcePath,
			MediaType:    file.File.Type,
			Duplicate:    isDuplicate,
			UsedFallback: file.Metadata.UsedFallback,
		}
		if options.Move {
			action.Kind = ActionMove
		}

		destinationRoot := options.Destination
		if isDuplicate {
			switch options.Duplicates {
			case DuplicateSkip:
				action.Kind = ActionSkip
				planned.Actions = append(planned.Actions, action)
				continue
			case DuplicateSeparate:
				destinationRoot = filepath.Join(options.Destination, "duplicates")
			case DuplicateSuffix:
			}
		}

		relativeDir, err := RenderTemplate(structure, file, index+1)
		if err != nil {
			return Plan{}, err
		}

		fileName := filepath.Base(file.File.SourcePath)
		if options.Rename != "" {
			fileName, err = RenderTemplate(options.Rename, file, index+1)
			if err != nil {
				return Plan{}, err
			}
		}

		destination := filepath.Join(destinationRoot, filepath.FromSlash(relativeDir), fileName)
		destination = uniquePlannedDestination(destination, usedDestinations)
		action.DestPath = destination
		planned.Actions = append(planned.Actions, action)
	}
	return planned, nil
}

func uniquePlannedDestination(path string, used map[string]int) string {
	if used[path] == 0 {
		used[path] = 1
		return path
	}
	used[path]++
	ext := filepath.Ext(path)
	base := path[:len(path)-len(ext)]
	return base + "_" + stringFromInt(used[path]) + ext
}

func stringFromInt(value int) string {
	return fmt.Sprintf("%d", value)
}
```

After adding the file, add `fmt` to the import block:

```go
import (
	"fmt"
	"path/filepath"
)
```

- [ ] **Step 4: Run tests**

Run:

```bash
go test ./internal/photo
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/photo/planner.go internal/photo/planner_test.go
git commit -m "feat: plan photo ingest actions"
```

---

### Task 7: Executor And Reports

**Files:**
- Create: `internal/photo/executor.go`
- Create: `internal/photo/executor_test.go`
- Create: `internal/photo/report.go`
- Create: `internal/photo/report_test.go`

- [ ] **Step 1: Write failing executor and report tests**

Create `internal/photo/executor_test.go`:

```go
package photo

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExecutePlanCopiesFiles(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source.jpg")
	dest := filepath.Join(root, "out", "source.jpg")
	if err := os.WriteFile(source, []byte("photo"), 0o644); err != nil {
		t.Fatal(err)
	}

	summary := ExecutePlan(Plan{Actions: []PlannedAction{{
		Kind:       ActionCopy,
		SourcePath: source,
		DestPath:   dest,
		MediaType:  MediaTypePhoto,
	}}})

	if summary.Copied != 1 || summary.Failed != 0 {
		t.Fatalf("summary = %+v", summary)
	}
	if _, err := os.Stat(dest); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(source); err != nil {
		t.Fatal(err)
	}
}

func TestExecutePlanMovesFiles(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source.jpg")
	dest := filepath.Join(root, "out", "source.jpg")
	if err := os.WriteFile(source, []byte("photo"), 0o644); err != nil {
		t.Fatal(err)
	}

	summary := ExecutePlan(Plan{Actions: []PlannedAction{{
		Kind:       ActionMove,
		SourcePath: source,
		DestPath:   dest,
		MediaType:  MediaTypePhoto,
	}}})

	if summary.Moved != 1 || summary.Failed != 0 {
		t.Fatalf("summary = %+v", summary)
	}
	if _, err := os.Stat(dest); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(source); !os.IsNotExist(err) {
		t.Fatalf("source still exists or unexpected error: %v", err)
	}
}
```

Create `internal/photo/report_test.go`:

```go
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run:

```bash
go test ./internal/photo
```

Expected: FAIL because executor and report functions do not exist.

- [ ] **Step 3: Implement executor**

Create `internal/photo/executor.go`:

```go
package photo

import (
	"io"
	"os"
	"path/filepath"
	"time"
)

func ExecutePlan(plan Plan) Summary {
	summary := Summary{Media: len(plan.Actions)}
	for _, action := range plan.Actions {
		countAction(action, &summary)
		if action.Kind == ActionSkip {
			summary.Skipped++
			continue
		}
		if err := os.MkdirAll(filepath.Dir(action.DestPath), 0o755); err != nil {
			summary.Failed++
			continue
		}
		var err error
		if action.Kind == ActionMove {
			err = moveFile(action.SourcePath, action.DestPath)
		} else {
			err = copyFile(action.SourcePath, action.DestPath)
		}
		if err != nil {
			summary.Failed++
			continue
		}
		if action.Kind == ActionMove {
			summary.Moved++
		} else {
			summary.Copied++
		}
	}
	return summary
}

func countAction(action PlannedAction, summary *Summary) {
	if action.Duplicate {
		summary.Duplicates++
	}
	if action.UsedFallback {
		summary.FallbackDates++
	}
	switch action.MediaType {
	case MediaTypePhoto:
		summary.Photos++
	case MediaTypeVideo:
		summary.Videos++
	case MediaTypeRaw:
		summary.Raw++
	}
}

func moveFile(source string, target string) error {
	if err := os.Rename(source, target); err == nil {
		return nil
	}
	if err := copyFile(source, target); err != nil {
		return err
	}
	return os.Remove(source)
}

func copyFile(source string, target string) error {
	sourceFile, err := os.Open(source)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	info, err := sourceFile.Stat()
	if err != nil {
		return err
	}

	targetFile, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode().Perm())
	if err != nil {
		return err
	}

	_, copyErr := io.Copy(targetFile, sourceFile)
	closeErr := targetFile.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Chtimes(target, time.Now(), info.ModTime())
}
```

- [ ] **Step 4: Implement report rendering**

Create `internal/photo/report.go`:

```go
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
	default:
		path := filepath.Join(plan.Options.Destination, "photo-ingest-report.txt")
		return path, os.WriteFile(path, []byte(RenderTextReport(plan, summary)), 0o644)
	}
}
```

- [ ] **Step 5: Run tests**

Run:

```bash
go test ./internal/photo
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/photo/executor.go internal/photo/executor_test.go internal/photo/report.go internal/photo/report_test.go
git commit -m "feat: execute and report photo ingest"
```

---

### Task 8: Ingest Orchestration

**Files:**
- Create: `internal/photo/ingest.go`
- Create: `internal/photo/ingest_test.go`

- [ ] **Step 1: Write failing ingest test**

Create `internal/photo/ingest_test.go`:

```go
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run:

```bash
go test ./internal/photo
```

Expected: FAIL because `PlanIngest` does not exist.

- [ ] **Step 3: Implement ingest orchestration**

Create `internal/photo/ingest.go`:

```go
package photo

func PlanIngest(options Options, provider MetadataProvider) (Plan, Summary, error) {
	if !options.Recursive {
		options.Recursive = false
	}
	if options.Structure == "" {
		options.Structure = DefaultStructure
	}
	if options.Duplicates == "" {
		options.Duplicates = DuplicateSkip
	}
	if options.Report == "" {
		options.Report = ReportText
	}

	if err := ValidateSourceDestination(options.Source, options.Destination); err != nil {
		return Plan{}, Summary{}, err
	}

	files, err := Scan(options.Source, ScanOptions{Recursive: options.Recursive, Excludes: options.Excludes})
	if err != nil {
		return Plan{}, Summary{}, err
	}

	enriched := make([]EnrichedFile, 0, len(files))
	for _, file := range files {
		metadata := ResolveMetadata(file, provider)
		hash, err := HashFile(file.SourcePath)
		if err != nil {
			hash = ""
		}
		enriched = append(enriched, EnrichedFile{File: file, Metadata: metadata, Hash: hash})
	}

	duplicates := MarkDuplicates(enriched)
	plan, err := BuildPlan(enriched, duplicates, options)
	if err != nil {
		return Plan{}, Summary{}, err
	}
	summary := SummarizePlan(plan)
	summary.Scanned = len(files)
	summary.Media = len(files)
	return plan, summary, nil
}

func ExecuteIngest(options Options, provider MetadataProvider) (Plan, Summary, string, error) {
	plan, _, err := PlanIngest(options, provider)
	if err != nil {
		return Plan{}, Summary{}, "", err
	}
	summary := ExecutePlan(plan)
	summary.Scanned = len(plan.Actions)
	summary.Media = len(plan.Actions)
	reportPath, err := WriteReport(plan, summary)
	return plan, summary, reportPath, err
}

func SummarizePlan(plan Plan) Summary {
	summary := Summary{Media: len(plan.Actions)}
	for _, action := range plan.Actions {
		countAction(action, &summary)
		if action.Kind == ActionSkip {
			summary.Skipped++
		}
	}
	return summary
}
```

- [ ] **Step 4: Run tests**

Run:

```bash
go test ./internal/photo
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/photo/ingest.go internal/photo/ingest_test.go
git commit -m "feat: orchestrate photo ingest workflow"
```

---

### Task 9: Cobra `photo` Command And Guided Menu

**Files:**
- Create: `cmd/photo.go`
- Delete: `cmd/organize.go`

- [ ] **Step 1: Add `photo` Cobra command**

Create `cmd/photo.go`:

```go
package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"mycli/internal/photo"

	"github.com/spf13/cobra"
)

var photoOptions = photo.Options{
	Recursive:  true,
	Structure:  photo.DefaultStructure,
	Duplicates: photo.DuplicateSkip,
	Report:     photo.ReportText,
}

var photoCmd = &cobra.Command{
	Use:   "photo",
	Short: "Photography workflow utilities",
	Long:  "Guided and scriptable photography ingest workflows.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPhotoMenu()
	},
}

var photoOrganizeCmd = &cobra.Command{
	Use:   "organize <source> <destination>",
	Short: "Organize photos and videos into a photography library",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		options := photoOptions
		options.Source = args[0]
		options.Destination = args[1]
		if !options.AllowFallback && !photo.ExiftoolProvider{}.Available() {
			return fmt.Errorf("exiftool not found; install exiftool or pass --allow-fallback")
		}
		plan, summary, err := photo.PlanIngest(options, photo.ExiftoolProvider{})
		if err != nil {
			return err
		}
		printPlanPreview(plan, summary)
		_, finalSummary, reportPath, err := photo.ExecuteIngest(options, photo.ExiftoolProvider{})
		if err != nil {
			return err
		}
		printFinalSummary(finalSummary, reportPath)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(photoCmd)
	photoCmd.AddCommand(photoOrganizeCmd)

	photoOrganizeCmd.Flags().BoolVar(&photoOptions.Move, "move", false, "Move files instead of copying them")
	photoOrganizeCmd.Flags().BoolVar(&photoOptions.Recursive, "recursive", true, "Scan source recursively")
	photoOrganizeCmd.Flags().BoolFunc("no-recursive", "Disable recursive scan", func(value string) error {
		photoOptions.Recursive = false
		return nil
	})
	photoOrganizeCmd.Flags().StringArrayVar(&photoOptions.Excludes, "exclude", nil, "Relative path or basename pattern to exclude")
	photoOrganizeCmd.Flags().StringVar(&photoOptions.Structure, "structure", photo.DefaultStructure, "Folder structure preset or template")
	photoOrganizeCmd.Flags().StringVar(&photoOptions.Rename, "rename", "", "Optional rename template")
	photoOrganizeCmd.Flags().Var((*duplicatePolicyValue)(&photoOptions.Duplicates), "duplicates", "Duplicate policy: skip, separate, suffix")
	photoOrganizeCmd.Flags().BoolVar(&photoOptions.AllowFallback, "allow-fallback", false, "Continue without exiftool using filename/modtime fallback")
	photoOrganizeCmd.Flags().Var((*reportFormatValue)(&photoOptions.Report), "report", "Report format: txt, json, none")
}
```

- [ ] **Step 2: Add menu helpers and flag values**

Append to `cmd/photo.go`:

```go
func runPhotoMenu() error {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Photo workflow")
	fmt.Println("1) Complete ingest")
	choice := promptLine(reader, "Choose an option [1]: ")
	if choice != "" && choice != "1" {
		return fmt.Errorf("invalid option %q", choice)
	}

	options := photo.Options{
		Recursive:  true,
		Structure:  photo.DefaultStructure,
		Duplicates: photo.DuplicateSkip,
		Report:     photo.ReportText,
	}
	options.Source = promptRequired(reader, "Source directory: ")
	options.Destination = promptRequired(reader, "Destination directory: ")
	options.Recursive = promptYesNo(reader, "Scan subfolders? [Y/n]: ", true)
	excludes := promptLine(reader, "Exclude paths, comma-separated: ")
	if excludes != "" {
		for _, item := range strings.Split(excludes, ",") {
			item = strings.TrimSpace(item)
			if item != "" {
				options.Excludes = append(options.Excludes, item)
			}
		}
	}
	options.Move = promptYesNo(reader, "Move files instead of copying? [y/N]: ", false)
	options.Structure = promptLine(reader, "Structure preset/template [{year}/{month}/{day}/{type}]: ")
	if options.Structure == "" {
		options.Structure = photo.DefaultStructure
	}
	if promptYesNo(reader, "Rename files? [y/N]: ", false) {
		options.Rename = promptLine(reader, "Rename template [{date}_{time}_{camera}_{seq}{ext}]: ")
		if options.Rename == "" {
			options.Rename = photo.DefaultRename
		}
	}
	duplicates := promptLine(reader, "Duplicates [skip|separate|suffix] (skip): ")
	if duplicates != "" {
		options.Duplicates = photo.DuplicatePolicy(duplicates)
	}

	provider := photo.ExiftoolProvider{}
	if !provider.Available() {
		if !promptYesNo(reader, "exiftool not found. Continue with fallback metadata? [y/N]: ", false) {
			return fmt.Errorf("exiftool not found")
		}
		options.AllowFallback = true
	}

	plan, summary, err := photo.PlanIngest(options, provider)
	if err != nil {
		return err
	}
	printPlanPreview(plan, summary)
	if !promptYesNo(reader, "Execute this plan? [y/N]: ", false) {
		return nil
	}
	_, finalSummary, reportPath, err := photo.ExecuteIngest(options, provider)
	if err != nil {
		return err
	}
	printFinalSummary(finalSummary, reportPath)
	return nil
}

func promptLine(reader *bufio.Reader, label string) string {
	fmt.Print(label)
	value, _ := reader.ReadString('\n')
	return strings.TrimSpace(value)
}

func promptRequired(reader *bufio.Reader, label string) string {
	for {
		value := promptLine(reader, label)
		if value != "" {
			return value
		}
		fmt.Println("Value required.")
	}
}

func promptYesNo(reader *bufio.Reader, label string, defaultValue bool) bool {
	value := strings.ToLower(promptLine(reader, label))
	if value == "" {
		return defaultValue
	}
	return value == "y" || value == "yes" || value == "s" || value == "sim"
}

func printPlanPreview(plan photo.Plan, summary photo.Summary) {
	fmt.Printf("Found %d media files\n", summary.Media)
	fmt.Printf("Photos: %d, Raw: %d, Videos: %d\n", summary.Photos, summary.Raw, summary.Videos)
	fmt.Printf("Duplicates: %d, fallback metadata: %d\n", summary.Duplicates, summary.FallbackDates)
	for i, action := range plan.Actions {
		if i >= 5 {
			break
		}
		fmt.Printf("%s -> %s\n", action.SourcePath, action.DestPath)
	}
}

func printFinalSummary(summary photo.Summary, reportPath string) {
	fmt.Printf("Copied: %d, moved: %d, skipped: %d, failed: %d\n", summary.Copied, summary.Moved, summary.Skipped, summary.Failed)
	if reportPath != "" {
		fmt.Printf("Report: %s\n", reportPath)
	}
}

type duplicatePolicyValue photo.DuplicatePolicy

func (v *duplicatePolicyValue) String() string {
	return string(*v)
}

func (v *duplicatePolicyValue) Set(value string) error {
	switch photo.DuplicatePolicy(value) {
	case photo.DuplicateSkip, photo.DuplicateSeparate, photo.DuplicateSuffix:
		*v = duplicatePolicyValue(value)
		return nil
	default:
		return fmt.Errorf("invalid duplicate policy %q", value)
	}
}

func (v *duplicatePolicyValue) Type() string {
	return "duplicate-policy"
}

type reportFormatValue photo.ReportFormat

func (v *reportFormatValue) String() string {
	return string(*v)
}

func (v *reportFormatValue) Set(value string) error {
	switch photo.ReportFormat(value) {
	case photo.ReportText, photo.ReportJSON, photo.ReportNone:
		*v = reportFormatValue(value)
		return nil
	default:
		return fmt.Errorf("invalid report format %q", value)
	}
}

func (v *reportFormatValue) Type() string {
	return "report-format"
}
```

- [ ] **Step 3: Delete old root organize command**

Delete:

```bash
rm cmd/organize.go
```

- [ ] **Step 4: Run build and tests**

Run:

```bash
go test ./...
go build ./...
```

Expected: PASS.

- [ ] **Step 5: Manually verify help output**

Run:

```bash
go run . help
go run . photo --help
go run . photo organize --help
```

Expected:

- Root help lists `photo` and does not list root `organize`.
- `photo --help` lists `organize`.
- `photo organize --help` lists workflow flags.

- [ ] **Step 6: Commit**

```bash
git add cmd/photo.go cmd/organize.go
git commit -m "feat: add guided photo workflow command"
```

---

### Task 10: README Documentation And Final Verification

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Update README photo sections**

Modify `README.md` so the tools section includes:

```markdown
### `photo` - Workflow de Fotografia
Executa um menu guiado para ingestao de fotos e videos, com organizacao por metadados, estruturas de pastas customizaveis, tratamento de duplicados e relatorio final.

### `photo organize` - Organizacao direta de midia
Organiza fotos e videos por linha de comando, usando scan recursivo por padrao.
```

Add usage examples:

```markdown
### Exemplo de uso - Workflow de Fotografia

```bash
./mycli photo
```

### Exemplo de uso - Organizacao direta

```bash
./mycli photo organize ./entrada ./biblioteca
./mycli photo organize ./entrada ./biblioteca --structure "camera-date" --duplicates skip
./mycli photo organize ./entrada ./biblioteca --no-recursive --exclude exports
```

O comando usa `exiftool` para ler data, camera e lente. Sem `exiftool`, o modo interativo pergunta se deve continuar com fallback limitado; o modo direto exige `--allow-fallback`.
```

- [ ] **Step 2: Run format, tests, vet, and build**

Run:

```bash
go fmt ./...
go test ./...
go vet ./...
go build -o mycli
```

Expected: all commands PASS.

- [ ] **Step 3: Smoke test direct organize with fallback**

Run:

```bash
tmp="$(mktemp -d)"
mkdir -p "$tmp/in/nested" "$tmp/out"
printf photo > "$tmp/in/nested/IMG_20240523_143015.JPG"
go run . photo organize "$tmp/in" "$tmp/out" --allow-fallback --report txt
find "$tmp/out" -type f | sort
```

Expected output includes:

```text
photo-ingest-report.txt
IMG_20240523_143015.JPG
```

The media file should be under:

```text
$tmp/out/2024/05/23/photos/IMG_20240523_143015.JPG
```

- [ ] **Step 4: Check git status**

Run:

```bash
git status --short
```

Expected: README and implementation files are modified or staged only from this work.

- [ ] **Step 5: Commit**

```bash
git add README.md
git commit -m "docs: document photo workflow"
```

---

## Self-Review

- Spec coverage: plan covers `mycli photo`, guided menu, `photo organize`, recursive default scanning, explicit excludes only, structure presets/custom templates, camera metadata, `exiftool`, duplicate policy, optional renaming, reports, old root command removal, tests, and README updates.
- Placeholder scan: no placeholder tasks remain; each task has exact files, code or command steps, and expected results.
- Type consistency: shared types are introduced before use. Later tasks consistently use `Options`, `MediaFile`, `Metadata`, `EnrichedFile`, `Plan`, `Summary`, `DuplicatePolicy`, and `ReportFormat`.
