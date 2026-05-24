# Photo Grouping And Similarity Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add grouped photo renaming, burst detection, and visual similarity grouping without creating extra group folders.

**Architecture:** Add focused grouping and similarity units inside `internal/photo`, keep exact duplicate SHA-256 detection unchanged, and pass grouping metadata into planning so filenames reflect burst/similar groups. CLI flags control burst and similarity detection; report/preview surface counts and skipped visual hash files.

**Tech Stack:** Go 1.24.1, Cobra, standard library `image`, `image/jpeg`, `image/png`, `math/bits`, Go `testing`.

---

## File Structure

- Create `internal/photo/groups.go`: burst grouping, hybrid group precedence, grouped filename assignment.
- Create `internal/photo/groups_test.go`: burst grouping and grouped rename tests.
- Create `internal/photo/similarity.go`: perceptual average hash and similarity grouping.
- Create `internal/photo/similarity_test.go`: image hash and similarity grouping tests.
- Modify `internal/photo/types.go`: add burst/similarity options, group/report fields, and plan group metadata.
- Modify `internal/photo/ingest.go`: compute visual hashes, detect groups, summarize group metadata.
- Modify `internal/photo/planner.go`: support `--rename grouped`.
- Modify `internal/photo/report.go`: include group metrics in text/JSON reports.
- Modify `cmd/photo.go`: add `--burst-window`, `--similarity-threshold`, guided prompts, and preview output.
- Modify `README.md` and `docs/photo-examples.md`: document grouped rename, burst, and similarity flags.

---

### Task 1: Types For Grouping

**Files:**
- Modify: `internal/photo/types.go`

- [ ] Add `BurstWindow time.Duration`, `SimilarityEnabled bool`, and `SimilarityThreshold int` to `Options`.
- [ ] Add `VisualHash uint64`, `HasVisualHash bool`, and `VisualHashSkipped bool` to `EnrichedFile`.
- [ ] Add `GroupType`, `FileGroup`, `GroupingResult`, and group-related fields to `Plan` and `Summary`.
- [ ] Run `go test ./internal/photo` and fix compile errors from new types only.
- [ ] Commit: `feat: add photo grouping types`.

### Task 2: Burst Groups And Grouped Names

**Files:**
- Create: `internal/photo/groups.go`
- Create: `internal/photo/groups_test.go`

- [ ] Write tests for burst grouping by camera and time window.
- [ ] Write tests proving videos are ignored by burst grouping.
- [ ] Write tests for grouped filenames: base name, numeric suffixes, short hash suffix for exact timestamp ties, and stable output when input order changes.
- [ ] Run `go test ./internal/photo` and observe RED failures for missing group functions.
- [ ] Implement burst grouping and grouped filename assignment.
- [ ] Run `go test ./internal/photo`; expected PASS.
- [ ] Commit: `feat: group photo bursts for renaming`.

### Task 3: Visual Similarity

**Files:**
- Create: `internal/photo/similarity.go`
- Create: `internal/photo/similarity_test.go`

- [ ] Write tests that generate small JPEG/PNG fixtures and verify stable perceptual hashes.
- [ ] Write tests that similar images group under threshold and different images do not.
- [ ] Write tests that RAW/HEIC/video/undecodable files are skipped without failing.
- [ ] Run `go test ./internal/photo` and observe RED failures for missing similarity functions.
- [ ] Implement `aHash`, Hamming distance, visual hash enrichment, and similar group detection.
- [ ] Run `go test ./internal/photo`; expected PASS.
- [ ] Commit: `feat: detect visually similar photos`.

### Task 4: Integrate Grouping Into Planning

**Files:**
- Modify: `internal/photo/ingest.go`
- Modify: `internal/photo/planner.go`
- Modify: `internal/photo/planner_test.go`
- Modify: `internal/photo/ingest_test.go`

- [ ] Write tests for `BuildPlanWithGrouping` using `Rename: "grouped"`.
- [ ] Write tests that burst group takes precedence over similar group for rename base.
- [ ] Run `go test ./internal/photo` and observe RED failures.
- [ ] Implement `BuildPlanWithGrouping`; keep existing `BuildPlan` as a wrapper for non-grouped callers.
- [ ] In `PlanIngest`, compute visual hashes when enabled, detect bursts/similar groups, merge group metadata, and pass it to planner.
- [ ] Run `go test ./internal/photo`; expected PASS.
- [ ] Commit: `feat: use photo groups in ingest planning`.

### Task 5: Report And CLI

**Files:**
- Modify: `internal/photo/report.go`
- Modify: `internal/photo/report_test.go`
- Modify: `cmd/photo.go`

- [ ] Write report tests for burst/similarity summary lines.
- [ ] Add `--burst-window` duration flag.
- [ ] Add `--similarity-threshold` int flag; use `cmd.Flags().Changed("similarity-threshold")` to distinguish omitted from zero.
- [ ] Add guided menu prompts for grouped rename, burst detection, and visual similarity detection.
- [ ] Update preview output with burst/similar/skipped counts.
- [ ] Run `go test ./...` and `go build ./...`; expected PASS.
- [ ] Verify help output includes both new flags.
- [ ] Commit: `feat: expose photo grouping options`.

### Task 6: Documentation And Verification

**Files:**
- Modify: `README.md`
- Modify: `docs/photo-examples.md`

- [ ] Document `--rename grouped`, `--burst-window 2s`, and `--similarity-threshold 8`.
- [ ] Explain that grouping changes names and reports only; it does not create group folders or delete similar photos.
- [ ] Run `gofmt -w cmd/*.go internal/photo/*.go`.
- [ ] Run `go test ./...`.
- [ ] Run `go vet ./...`.
- [ ] Run `go build -o mycli`.
- [ ] Smoke test with generated JPEGs and `--rename grouped --similarity-threshold 8 --allow-fallback`.
- [ ] Commit: `docs: document photo grouping options`.

---

## Self-Review

- Spec coverage: plan covers grouped rename, burst detection, visual similarity for JPEG/PNG, skipped visual hashes, no group folders, report/preview fields, CLI flags, and documentation.
- Placeholder scan: no placeholder tasks remain; each task has exact files, verification commands, and commit scope.
- Type consistency: group and similarity types are introduced before use in ingest/planner/report/CLI.
