# Batch Unpack Utility Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build `mycli unpack <file-or-directory>` to extract `.zip`, `.tar`, `.tar.gz`, and `.tgz` archives, verify extracted output, and delete each original archive only after that archive succeeds.

**Architecture:** Add a focused `internal/archive` package for archive discovery, destination planning, extraction, verification, and batch summaries. Add `cmd/unpack.go` as the Cobra command layer; keep user-facing printing there and keep destructive delete decisions inside the package after verification.

**Tech Stack:** Go 1.24.1, Cobra, standard library `archive/zip`, `archive/tar`, `compress/gzip`, filesystem APIs, Go `testing` package.

---

## Workspace Note

At plan creation time, `README.md` and `docs/photo-examples.md` may contain uncommitted documentation edits from previous photo-example work. Do not revert or overwrite those edits. When updating README for `unpack`, apply a narrow patch around the command examples and project tree.

## File Structure

- Create `internal/archive/types.go`: archive type constants, options, item result, extracted file metadata, batch summary.
- Create `internal/archive/discover.go`: supported archive detection, base-name calculation, file/directory discovery, destination planning.
- Create `internal/archive/discover_test.go`: type detection, base naming, non-recursive and recursive discovery tests.
- Create `internal/archive/extract.go`: safe zip/tar/tar.gz extraction and metadata collection.
- Create `internal/archive/extract_test.go`: extraction, traversal rejection, overwrite behavior, symlink skip behavior.
- Create `internal/archive/process.go`: process one archive, verify extracted metadata, delete original after success, process batches.
- Create `internal/archive/process_test.go`: delete-on-success, keep mode, failed archive preservation, multi-archive batch summary.
- Create `cmd/unpack.go`: Cobra command, flags, output formatting.
- Modify `README.md`: document `unpack` examples and destructive behavior.

---

### Task 1: Archive Types And Discovery

**Files:**
- Create: `internal/archive/types.go`
- Create: `internal/archive/discover.go`
- Create: `internal/archive/discover_test.go`

- [ ] **Step 1: Write failing discovery tests**

Create `internal/archive/discover_test.go`:

```go
package archive

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectType(t *testing.T) {
	tests := []struct {
		path     string
		wantType Type
		wantOK   bool
	}{
		{"photos.zip", TypeZip, true},
		{"backup.tar", TypeTar, true},
		{"aulas.tar.gz", TypeTarGz, true},
		{"backup.tgz", TypeTarGz, true},
		{"notes.txt", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			gotType, gotOK := DetectType(tt.path)
			if gotType != tt.wantType || gotOK != tt.wantOK {
				t.Fatalf("DetectType(%q) = %q, %v; want %q, %v", tt.path, gotType, gotOK, tt.wantType, tt.wantOK)
			}
		})
	}
}

func TestBaseNameRemovesArchiveExtension(t *testing.T) {
	tests := map[string]string{
		"photos.zip":   "photos",
		"backup.tar":   "backup",
		"aulas.tar.gz": "aulas",
		"backup.tgz":   "backup",
	}
	for input, want := range tests {
		if got := BaseName(input); got != want {
			t.Fatalf("BaseName(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestDiscoverDirectoryNonRecursive(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.zip"), []byte("zip"), 0o644); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(root, "nested")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nested, "b.zip"), []byte("zip"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "notes.txt"), []byte("notes"), 0o644); err != nil {
		t.Fatal(err)
	}

	items, skipped, err := Discover(root, Options{Recursive: false})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if filepath.Base(items[0].SourcePath) != "a.zip" {
		t.Fatalf("SourcePath = %q", items[0].SourcePath)
	}
	if skipped != 1 {
		t.Fatalf("skipped = %d, want 1", skipped)
	}
}

func TestDiscoverDirectoryRecursive(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "nested")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "a.zip"), []byte("zip"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nested, "b.tgz"), []byte("tgz"), 0o644); err != nil {
		t.Fatal(err)
	}

	items, _, err := Discover(root, Options{Recursive: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run:

```bash
go test ./internal/archive
```

Expected: FAIL because `internal/archive` symbols do not exist.

- [ ] **Step 3: Implement types**

Create `internal/archive/types.go`:

```go
package archive

type Type string

const (
	TypeZip   Type = "zip"
	TypeTar   Type = "tar"
	TypeTarGz Type = "tar.gz"
)

type Options struct {
	Input     string
	Dest      string
	Recursive bool
	Keep      bool
	Overwrite bool
}

type Item struct {
	SourcePath string
	Type       Type
	BaseName   string
	DestDir    string
}

type ExtractedFile struct {
	Path string
	Size int64
	CRC32 uint32
	HasCRC32 bool
}

type ResultStatus string

const (
	StatusOK   ResultStatus = "OK"
	StatusKeep ResultStatus = "KEEP"
	StatusFail ResultStatus = "FAIL"
)

type Result struct {
	Item             Item
	Status           ResultStatus
	FilesExtracted   int
	SkippedLinks     int
	OriginalDeleted  bool
	OriginalKept     bool
	Error            string
}

type Summary struct {
	ArchivesFound      int
	Extracted          int
	DeletedOriginals   int
	KeptOriginals      int
	Failed             int
	SkippedUnsupported int
}
```

- [ ] **Step 4: Implement discovery**

Create `internal/archive/discover.go`:

```go
package archive

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func DetectType(path string) (Type, bool) {
	lower := strings.ToLower(path)
	switch {
	case strings.HasSuffix(lower, ".tar.gz"):
		return TypeTarGz, true
	case strings.HasSuffix(lower, ".tgz"):
		return TypeTarGz, true
	case strings.HasSuffix(lower, ".zip"):
		return TypeZip, true
	case strings.HasSuffix(lower, ".tar"):
		return TypeTar, true
	default:
		return "", false
	}
}

func BaseName(path string) string {
	name := filepath.Base(path)
	lower := strings.ToLower(name)
	switch {
	case strings.HasSuffix(lower, ".tar.gz"):
		return name[:len(name)-len(".tar.gz")]
	case strings.HasSuffix(lower, ".tgz"):
		return name[:len(name)-len(".tgz")]
	case strings.HasSuffix(lower, ".zip"):
		return name[:len(name)-len(".zip")]
	case strings.HasSuffix(lower, ".tar"):
		return name[:len(name)-len(".tar")]
	default:
		return strings.TrimSuffix(name, filepath.Ext(name))
	}
}

func Discover(input string, options Options) ([]Item, int, error) {
	info, err := os.Stat(input)
	if err != nil {
		return nil, 0, err
	}
	if !info.IsDir() {
		archiveType, ok := DetectType(input)
		if !ok {
			return nil, 0, fmt.Errorf("unsupported archive: %s", input)
		}
		return []Item{buildItem(input, archiveType, options)}, 0, nil
	}

	var items []Item
	skipped := 0
	walkErr := filepath.WalkDir(input, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == input {
			return nil
		}
		if entry.IsDir() {
			if !options.Recursive {
				return filepath.SkipDir
			}
			return nil
		}
		archiveType, ok := DetectType(path)
		if !ok {
			skipped++
			return nil
		}
		items = append(items, buildItem(path, archiveType, options))
		return nil
	})
	if walkErr != nil {
		return nil, 0, walkErr
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].SourcePath < items[j].SourcePath
	})
	return items, skipped, nil
}

func buildItem(sourcePath string, archiveType Type, options Options) Item {
	baseName := BaseName(sourcePath)
	destRoot := filepath.Dir(sourcePath)
	if options.Dest != "" {
		destRoot = options.Dest
	}
	return Item{
		SourcePath: sourcePath,
		Type:       archiveType,
		BaseName:   baseName,
		DestDir:    filepath.Join(destRoot, baseName),
	}
}
```

- [ ] **Step 5: Run tests**

Run:

```bash
go test ./internal/archive
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/archive/types.go internal/archive/discover.go internal/archive/discover_test.go
git commit -m "feat: discover archives for unpack"
```

---

### Task 2: Safe Extraction

**Files:**
- Create: `internal/archive/extract.go`
- Create: `internal/archive/extract_test.go`

- [ ] **Step 1: Write failing extraction tests**

Create `internal/archive/extract_test.go`:

```go
package archive

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

func TestExtractZipWritesFiles(t *testing.T) {
	root := t.TempDir()
	archivePath := filepath.Join(root, "files.zip")
	writeZip(t, archivePath, map[string]string{"dir/a.txt": "hello"})

	item := Item{SourcePath: archivePath, Type: TypeZip, DestDir: filepath.Join(root, "files")}
	files, skipped, err := Extract(item, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if skipped != 0 || len(files) != 1 {
		t.Fatalf("files = %d skipped = %d", len(files), skipped)
	}
	content, err := os.ReadFile(filepath.Join(item.DestDir, "dir", "a.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "hello" {
		t.Fatalf("content = %q", content)
	}
}

func TestExtractTarGzWritesFiles(t *testing.T) {
	root := t.TempDir()
	archivePath := filepath.Join(root, "files.tar.gz")
	writeTarGz(t, archivePath, map[string]string{"a.txt": "hello"})

	item := Item{SourcePath: archivePath, Type: TypeTarGz, DestDir: filepath.Join(root, "files")}
	files, skipped, err := Extract(item, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if skipped != 0 || len(files) != 1 {
		t.Fatalf("files = %d skipped = %d", len(files), skipped)
	}
}

func TestExtractRejectsTraversal(t *testing.T) {
	root := t.TempDir()
	archivePath := filepath.Join(root, "bad.zip")
	writeZip(t, archivePath, map[string]string{"../evil.txt": "no"})

	item := Item{SourcePath: archivePath, Type: TypeZip, DestDir: filepath.Join(root, "bad")}
	_, _, err := Extract(item, Options{})
	if err == nil {
		t.Fatal("expected traversal error")
	}
	if _, err := os.Stat(filepath.Join(root, "evil.txt")); !os.IsNotExist(err) {
		t.Fatalf("evil file exists or unexpected error: %v", err)
	}
}

func TestExtractExistingNonEmptyDestinationRequiresOverwrite(t *testing.T) {
	root := t.TempDir()
	archivePath := filepath.Join(root, "files.zip")
	writeZip(t, archivePath, map[string]string{"a.txt": "hello"})
	dest := filepath.Join(root, "files")
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dest, "existing.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, _, err := Extract(Item{SourcePath: archivePath, Type: TypeZip, DestDir: dest}, Options{})
	if err == nil {
		t.Fatal("expected non-empty destination error")
	}
}

func writeZip(t *testing.T, path string, files map[string]string) {
	t.Helper()
	out, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	writer := zip.NewWriter(out)
	for name, content := range files {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
}

func writeTarGz(t *testing.T, path string, files map[string]string) {
	t.Helper()
	var buffer bytes.Buffer
	gzipWriter := gzip.NewWriter(&buffer)
	tarWriter := tar.NewWriter(gzipWriter)
	for name, content := range files {
		header := &tar.Header{Name: name, Mode: 0o644, Size: int64(len(content))}
		if err := tarWriter.WriteHeader(header); err != nil {
			t.Fatal(err)
		}
		if _, err := tarWriter.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, buffer.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run:

```bash
go test ./internal/archive
```

Expected: FAIL because `Extract` does not exist.

- [ ] **Step 3: Implement safe extraction**

Create `internal/archive/extract.go`:

```go
package archive

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func Extract(item Item, options Options) ([]ExtractedFile, int, error) {
	if err := prepareDestination(item.DestDir, options.Overwrite); err != nil {
		return nil, 0, err
	}
	switch item.Type {
	case TypeZip:
		return extractZip(item)
	case TypeTar:
		return extractTarFile(item)
	case TypeTarGz:
		return extractTarGz(item)
	default:
		return nil, 0, fmt.Errorf("unsupported archive type %q", item.Type)
	}
}

func prepareDestination(dest string, overwrite bool) error {
	if entries, err := os.ReadDir(dest); err == nil && len(entries) > 0 && !overwrite {
		return fmt.Errorf("destination exists and is non-empty: %s", dest)
	}
	return os.MkdirAll(dest, 0o755)
}

func extractZip(item Item) ([]ExtractedFile, int, error) {
	reader, err := zip.OpenReader(item.SourcePath)
	if err != nil {
		return nil, 0, err
	}
	defer reader.Close()

	var files []ExtractedFile
	skipped := 0
	for _, entry := range reader.File {
		target, err := safeJoin(item.DestDir, entry.Name)
		if err != nil {
			return nil, skipped, err
		}
		if entry.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return nil, skipped, err
			}
			continue
		}
		mode := entry.FileInfo().Mode()
		if mode&os.ModeSymlink != 0 {
			skipped++
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return nil, skipped, err
		}
		source, err := entry.Open()
		if err != nil {
			return nil, skipped, err
		}
		size, crc, copyErr := copyWithCRC(target, source, mode.Perm())
		closeErr := source.Close()
		if copyErr != nil {
			return nil, skipped, copyErr
		}
		if closeErr != nil {
			return nil, skipped, closeErr
		}
		files = append(files, ExtractedFile{Path: target, Size: size, CRC32: crc, HasCRC32: true})
	}
	return files, skipped, nil
}

func extractTarFile(item Item) ([]ExtractedFile, int, error) {
	file, err := os.Open(item.SourcePath)
	if err != nil {
		return nil, 0, err
	}
	defer file.Close()
	return extractTar(item.DestDir, tar.NewReader(file))
}

func extractTarGz(item Item) ([]ExtractedFile, int, error) {
	file, err := os.Open(item.SourcePath)
	if err != nil {
		return nil, 0, err
	}
	defer file.Close()
	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return nil, 0, err
	}
	defer gzipReader.Close()
	return extractTar(item.DestDir, tar.NewReader(gzipReader))
}

func extractTar(dest string, reader *tar.Reader) ([]ExtractedFile, int, error) {
	var files []ExtractedFile
	skipped := 0
	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, skipped, err
		}
		target, err := safeJoin(dest, header.Name)
		if err != nil {
			return nil, skipped, err
		}
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return nil, skipped, err
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return nil, skipped, err
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode).Perm())
			if err != nil {
				return nil, skipped, err
			}
			size, copyErr := io.Copy(out, reader)
			closeErr := out.Close()
			if copyErr != nil {
				return nil, skipped, copyErr
			}
			if closeErr != nil {
				return nil, skipped, closeErr
			}
			if size != header.Size {
				return nil, skipped, fmt.Errorf("size mismatch for %s", header.Name)
			}
			files = append(files, ExtractedFile{Path: target, Size: size})
		case tar.TypeSymlink, tar.TypeLink:
			skipped++
		default:
			skipped++
		}
	}
	return files, skipped, nil
}

func safeJoin(dest string, entryName string) (string, error) {
	if filepath.IsAbs(entryName) || hasWindowsDrive(entryName) {
		return "", fmt.Errorf("unsafe archive path: %s", entryName)
	}
	clean := filepath.Clean(entryName)
	if clean == "." || strings.HasPrefix(clean, ".."+string(os.PathSeparator)) || clean == ".." {
		return "", fmt.Errorf("unsafe archive path: %s", entryName)
	}
	target := filepath.Join(dest, clean)
	rel, err := filepath.Rel(dest, target)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("unsafe archive path: %s", entryName)
	}
	return target, nil
}

func hasWindowsDrive(path string) bool {
	if len(path) >= 2 && path[1] == ':' {
		return true
	}
	return runtime.GOOS != "windows" && strings.Contains(path, ":\\")
}

func copyWithCRC(target string, source io.Reader, perm os.FileMode) (int64, uint32, error) {
	out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return 0, 0, err
	}
	defer out.Close()
	hash := crc32.NewIEEE()
	written, err := io.Copy(io.MultiWriter(out, hash), source)
	if err != nil {
		return 0, 0, err
	}
	return written, hash.Sum32(), nil
}
```

- [ ] **Step 4: Run tests**

Run:

```bash
go test ./internal/archive
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/archive/extract.go internal/archive/extract_test.go
git commit -m "feat: safely extract archives"
```

---

### Task 3: Verification, Deletion Gate, And Batch Processing

**Files:**
- Create: `internal/archive/process.go`
- Create: `internal/archive/process_test.go`

- [ ] **Step 1: Write failing process tests**

Create `internal/archive/process_test.go`:

```go
package archive

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProcessDeletesOriginalAfterVerifiedExtraction(t *testing.T) {
	root := t.TempDir()
	archivePath := filepath.Join(root, "files.zip")
	writeZip(t, archivePath, map[string]string{"a.txt": "hello"})

	item := Item{SourcePath: archivePath, Type: TypeZip, BaseName: "files", DestDir: filepath.Join(root, "files")}
	result := Process(item, Options{})
	if result.Status != StatusOK {
		t.Fatalf("result = %+v", result)
	}
	if _, err := os.Stat(archivePath); !os.IsNotExist(err) {
		t.Fatalf("archive still exists or unexpected error: %v", err)
	}
}

func TestProcessKeepPreservesOriginal(t *testing.T) {
	root := t.TempDir()
	archivePath := filepath.Join(root, "files.zip")
	writeZip(t, archivePath, map[string]string{"a.txt": "hello"})

	item := Item{SourcePath: archivePath, Type: TypeZip, BaseName: "files", DestDir: filepath.Join(root, "files")}
	result := Process(item, Options{Keep: true})
	if result.Status != StatusKeep {
		t.Fatalf("result = %+v", result)
	}
	if _, err := os.Stat(archivePath); err != nil {
		t.Fatalf("archive missing: %v", err)
	}
}

func TestProcessPreservesOriginalWhenExtractionFails(t *testing.T) {
	root := t.TempDir()
	archivePath := filepath.Join(root, "bad.zip")
	if err := os.WriteFile(archivePath, []byte("not a zip"), 0o644); err != nil {
		t.Fatal(err)
	}

	item := Item{SourcePath: archivePath, Type: TypeZip, BaseName: "bad", DestDir: filepath.Join(root, "bad")}
	result := Process(item, Options{})
	if result.Status != StatusFail {
		t.Fatalf("result = %+v", result)
	}
	if _, err := os.Stat(archivePath); err != nil {
		t.Fatalf("archive missing: %v", err)
	}
}

func TestProcessBatchContinuesAfterFailure(t *testing.T) {
	root := t.TempDir()
	good := filepath.Join(root, "good.zip")
	bad := filepath.Join(root, "bad.zip")
	writeZip(t, good, map[string]string{"a.txt": "hello"})
	if err := os.WriteFile(bad, []byte("not a zip"), 0o644); err != nil {
		t.Fatal(err)
	}

	results, summary := ProcessBatch([]Item{
		{SourcePath: bad, Type: TypeZip, BaseName: "bad", DestDir: filepath.Join(root, "bad")},
		{SourcePath: good, Type: TypeZip, BaseName: "good", DestDir: filepath.Join(root, "good")},
	}, Options{}, 0)

	if len(results) != 2 {
		t.Fatalf("len(results) = %d", len(results))
	}
	if summary.Failed != 1 || summary.Extracted != 1 || summary.DeletedOriginals != 1 {
		t.Fatalf("summary = %+v", summary)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run:

```bash
go test ./internal/archive
```

Expected: FAIL because `Process` and `ProcessBatch` do not exist.

- [ ] **Step 3: Implement processing**

Create `internal/archive/process.go`:

```go
package archive

import (
	"fmt"
	"os"
)

func Process(item Item, options Options) Result {
	files, skipped, err := Extract(item, options)
	if err != nil {
		return Result{Item: item, Status: StatusFail, SkippedLinks: skipped, OriginalKept: true, Error: err.Error()}
	}
	if err := Verify(files); err != nil {
		return Result{Item: item, Status: StatusFail, FilesExtracted: len(files), SkippedLinks: skipped, OriginalKept: true, Error: err.Error()}
	}
	if options.Keep {
		return Result{Item: item, Status: StatusKeep, FilesExtracted: len(files), SkippedLinks: skipped, OriginalKept: true}
	}
	if err := os.Remove(item.SourcePath); err != nil {
		return Result{Item: item, Status: StatusFail, FilesExtracted: len(files), SkippedLinks: skipped, OriginalKept: true, Error: fmt.Sprintf("delete verified archive: %v", err)}
	}
	return Result{Item: item, Status: StatusOK, FilesExtracted: len(files), SkippedLinks: skipped, OriginalDeleted: true}
}

func Verify(files []ExtractedFile) error {
	for _, file := range files {
		info, err := os.Stat(file.Path)
		if err != nil {
			return err
		}
		if info.Size() != file.Size {
			return fmt.Errorf("size mismatch: %s", file.Path)
		}
		if file.HasCRC32 {
			hash, err := fileCRC32(file.Path)
			if err != nil {
				return err
			}
			if hash != file.CRC32 {
				return fmt.Errorf("checksum mismatch: %s", file.Path)
			}
		}
	}
	return nil
}

func ProcessBatch(items []Item, options Options, skippedUnsupported int) ([]Result, Summary) {
	summary := Summary{ArchivesFound: len(items), SkippedUnsupported: skippedUnsupported}
	results := make([]Result, 0, len(items))
	for _, item := range items {
		result := Process(item, options)
		results = append(results, result)
		switch result.Status {
		case StatusOK:
			summary.Extracted++
			summary.DeletedOriginals++
		case StatusKeep:
			summary.Extracted++
			summary.KeptOriginals++
		case StatusFail:
			summary.Failed++
			if result.OriginalKept {
				summary.KeptOriginals++
			}
		}
	}
	return results, summary
}
```

Append this helper to `internal/archive/extract.go`:

```go
func fileCRC32(path string) (uint32, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()
	hash := crc32.NewIEEE()
	if _, err := io.Copy(hash, file); err != nil {
		return 0, err
	}
	return hash.Sum32(), nil
}
```

- [ ] **Step 4: Run tests**

Run:

```bash
go test ./internal/archive
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/archive/process.go internal/archive/process_test.go internal/archive/extract.go
git commit -m "feat: verify and delete extracted archives"
```

---

### Task 4: Cobra Command

**Files:**
- Create: `cmd/unpack.go`

- [ ] **Step 1: Add command implementation**

Create `cmd/unpack.go`:

```go
package cmd

import (
	"fmt"

	"mycli/internal/archive"

	"github.com/spf13/cobra"
)

var unpackOptions archive.Options

var unpackCmd = &cobra.Command{
	Use:   "unpack <file-or-directory>",
	Short: "Extract archives and delete originals after verification",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		options := unpackOptions
		options.Input = args[0]
		items, skipped, err := archive.Discover(options.Input, options)
		if err != nil {
			return err
		}
		results, summary := archive.ProcessBatch(items, options, skipped)
		for _, result := range results {
			printUnpackResult(result)
		}
		printUnpackSummary(summary)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(unpackCmd)
	unpackCmd.Flags().StringVar(&unpackOptions.Dest, "dest", "", "Extraction root directory")
	unpackCmd.Flags().BoolVar(&unpackOptions.Recursive, "recursive", false, "Process archives in subdirectories")
	unpackCmd.Flags().BoolVar(&unpackOptions.Keep, "keep", false, "Keep original archives after successful verification")
	unpackCmd.Flags().BoolVar(&unpackOptions.Overwrite, "overwrite", false, "Allow overwriting files inside extraction directories")
}

func printUnpackResult(result archive.Result) {
	switch result.Status {
	case archive.StatusOK:
		fmt.Printf("OK    %s -> %s (%d files, original deleted)\n", result.Item.SourcePath, result.Item.DestDir, result.FilesExtracted)
	case archive.StatusKeep:
		fmt.Printf("KEEP  %s -> %s (%d files, original kept)\n", result.Item.SourcePath, result.Item.DestDir, result.FilesExtracted)
	case archive.StatusFail:
		fmt.Printf("FAIL  %s -> %s\n", result.Item.SourcePath, result.Error)
	}
}

func printUnpackSummary(summary archive.Summary) {
	fmt.Printf("Archives found: %d\n", summary.ArchivesFound)
	fmt.Printf("Extracted: %d\n", summary.Extracted)
	fmt.Printf("Deleted originals: %d\n", summary.DeletedOriginals)
	fmt.Printf("Kept originals: %d\n", summary.KeptOriginals)
	fmt.Printf("Failed: %d\n", summary.Failed)
	fmt.Printf("Skipped unsupported: %d\n", summary.SkippedUnsupported)
}
```

- [ ] **Step 2: Run formatting, tests, and build**

Run:

```bash
gofmt -w cmd/unpack.go internal/archive/*.go
go test ./...
go build ./...
```

Expected: all commands PASS.

- [ ] **Step 3: Verify help output**

Run:

```bash
go run . unpack --help
```

Expected output includes `--dest`, `--recursive`, `--keep`, and `--overwrite`.

- [ ] **Step 4: Commit**

```bash
git add cmd/unpack.go
git commit -m "feat: add unpack command"
```

---

### Task 5: README And Smoke Tests

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Update README**

Add `unpack` to the tools list:

```markdown
### `unpack` - Descompactador seguro em lote
Descompacta arquivos `.zip`, `.tar`, `.tar.gz` e `.tgz`, verifica os arquivos extraidos e apaga cada compactado original somente depois de uma verificacao bem-sucedida.
```

Add examples near the command examples:

```markdown
# Descompactar um arquivo e apagar o original depois da verificacao
./mycli unpack arquivo.zip

# Descompactar compactados de uma pasta
./mycli unpack ./downloads

# Procurar tambem em subpastas
./mycli unpack ./downloads --recursive

# Preservar os originais
./mycli unpack ./downloads --keep

# Extrair em outro destino
./mycli unpack ./downloads --dest ./extraidos
```

Mention:

```markdown
O `unpack` apaga cada arquivo compactado automaticamente apenas depois de extrair e verificar aquele arquivo. Use `--keep` para preservar os compactados originais.
```

- [ ] **Step 2: Run full verification**

Run:

```bash
gofmt -w cmd/*.go internal/archive/*.go
go test ./...
go vet ./...
go build -o mycli
```

Expected: all commands PASS.

- [ ] **Step 3: Smoke test zip delete behavior**

Run:

```bash
tmp="$(mktemp -d)"
mkdir -p "$tmp/source"
printf hello > "$tmp/source/a.txt"
(cd "$tmp/source" && zip -q "$tmp/files.zip" a.txt)
./mycli unpack "$tmp/files.zip"
test -f "$tmp/files/a.txt"
test ! -f "$tmp/files.zip"
```

Expected: exit code 0.

- [ ] **Step 4: Smoke test keep behavior**

Run:

```bash
tmp="$(mktemp -d)"
mkdir -p "$tmp/source"
printf hello > "$tmp/source/a.txt"
(cd "$tmp/source" && zip -q "$tmp/files.zip" a.txt)
./mycli unpack "$tmp/files.zip" --keep
test -f "$tmp/files/a.txt"
test -f "$tmp/files.zip"
```

Expected: exit code 0.

- [ ] **Step 5: Commit**

```bash
git add README.md
git commit -m "docs: document unpack command"
```

---

## Self-Review

- Spec coverage: plan covers root `unpack`, `.zip`, `.tar`, `.tar.gz`, `.tgz`, single-file and directory batch input, non-recursive default, `--recursive`, `--dest`, `--keep`, `--overwrite`, verification before delete, failed archive preservation, unsupported skipped files, path traversal rejection, symlink/hardlink skipping, per-archive output, and summary output.
- Placeholder scan: no incomplete implementation steps remain; every task includes concrete files, commands, expected results, and code snippets.
- Type consistency: `Type`, `Options`, `Item`, `ExtractedFile`, `Result`, and `Summary` are introduced before use and referenced consistently across tasks.
