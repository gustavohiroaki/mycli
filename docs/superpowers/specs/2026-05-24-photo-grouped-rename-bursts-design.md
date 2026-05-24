# Photo Grouped Rename, Burst, And Similarity Detection Design

## Context

The photo workflow currently supports exact duplicate detection through SHA-256 hashes and optional renaming through a template such as:

```text
{date}_{time}_{camera}_{seq}{ext}
```

The next improvement is to make related photos easier to review by giving them similar names, detecting burst sequences based on capture time, and grouping visually similar files when the image format can be decoded.

## Goals

- Add a grouped rename mode for related photos.
- Keep grouped rename deterministic across repeated runs.
- Use short content hashes as the tie-breaker when multiple photos share the same timestamp.
- Detect burst groups from metadata timestamps.
- Detect visually similar image groups for decodable formats.
- Fall back to metadata-based grouping for formats that cannot be visually decoded.
- Reflect grouping through filenames only.
- Include burst information in the report.
- Include visual similarity information in the report.

## Non-Goals

- Do not move burst groups into separate folders in this iteration.
- Do not move similar groups into separate folders in this iteration.
- Do not delete or auto-select best photos from a burst.
- Do not delete or auto-select best photos from a similar group.
- Do not require RAW or HEIC visual decoding in this iteration.

## CLI Behavior

Grouped rename is enabled with:

```bash
mycli photo organize ./entrada ./biblioteca --rename grouped
```

Burst detection is enabled by passing a duration:

```bash
mycli photo organize ./entrada ./biblioteca --rename grouped --burst-window 2s
```

Visual similarity detection is enabled by passing a perceptual hash distance threshold:

```bash
mycli photo organize ./entrada ./biblioteca --rename grouped --similarity-threshold 8
```

The hybrid mode combines both:

```bash
mycli photo organize ./entrada ./biblioteca --rename grouped --burst-window 2s --similarity-threshold 8
```

The guided `mycli photo` menu should offer grouped rename as a rename option:

```text
Rename files?
1) Keep original names
2) Custom template
3) Grouped names
```

If grouped names are selected, the menu asks for burst detection:

```text
Detect bursts by time window? [y/N]
Burst window [2s]:
Detect visual similarity? [y/N]
Similarity threshold [8]:
```

## Grouped Rename Rules

The grouped rename base is:

```text
{date}_{time}_{camera}{ext}
```

Example:

```text
2026-05-24_14-32-10_canon-eos-r6.jpg
```

When burst detection is disabled, files are grouped by:

- capture date and time down to the second;
- camera model, or `unknown-camera` when unavailable;
- media type;
- extension.

When burst detection is enabled, files in the same burst group share the same grouped rename base, using the timestamp and camera of the first file in the burst.

When visual similarity detection is enabled, files in the same similar group share the same grouped rename base, using the timestamp and camera of the earliest file in the similar group.

When a file belongs to both a burst group and a similar group, the burst group takes precedence for the grouped rename base. Visual similarity is still reported.

Within a group:

- Files are sorted by capture timestamp.
- The first file by sorted order keeps the base name.
- Additional files with later timestamps get sequential suffixes:

```text
2026-05-24_14-32-10_canon-eos-r6.jpg
2026-05-24_14-32-10_canon-eos-r6_1.jpg
2026-05-24_14-32-10_canon-eos-r6_2.jpg
```

When multiple files in the same group have the exact same capture timestamp, those tied files use a short hash suffix instead of a numeric suffix:

```text
2026-05-24_14-32-10_canon-eos-r6_a91f.jpg
2026-05-24_14-32-10_canon-eos-r6_c204.jpg
```

The short hash is derived from the existing SHA-256 content hash and uses the first four hexadecimal characters. If the hash is unavailable, use a deterministic fallback from the original filename.

No burst or similarity group creates directories. Grouping is reflected only in filenames and report data.

## Burst Detection Rules

Burst detection groups files when:

- media type is `photos` or `raw`;
- files are from the same camera model, or the same `unknown-camera` fallback;
- files are sorted by capture timestamp;
- the difference between consecutive files is less than or equal to the configured `--burst-window`.

Default behavior:

- no burst detection unless `--burst-window` is provided or the menu option is enabled;
- menu default window is `2s`;
- direct command accepts any Go duration string such as `500ms`, `2s`, or `1m`.

Burst groups with a single file are not counted as bursts.

## Visual Similarity Rules

Visual similarity detection uses a perceptual hash for decodable image formats.

Initial decodable formats:

```text
.jpg
.jpeg
.png
```

The first implementation should use a deterministic average hash (`aHash`) based on an 8x8 grayscale sample:

- decode image;
- resize/sample to 8x8;
- compute average brightness;
- set one bit per pixel based on whether that pixel is above or equal to the average;
- compare hashes by Hamming distance.

Files are considered visually similar when:

- both files have a visual hash;
- both files are media type `photos`;
- Hamming distance is less than or equal to `--similarity-threshold`.

Default behavior:

- no visual similarity detection unless `--similarity-threshold` is provided or the menu option is enabled;
- menu default threshold is `8`;
- `--similarity-threshold 0` detects only identical perceptual hashes;
- negative thresholds are invalid.

Files that cannot be decoded for visual hashing do not fail the workflow. They are counted as visual-similarity skipped and may still participate in burst grouping.

Similar groups with a single file are not counted as similar groups.

## Reporting

The text and JSON reports should include:

- total burst groups;
- largest burst size;
- files that belong to each burst group;
- burst window used;
- total similar groups;
- largest similar group size;
- files that belong to each similar group;
- similarity threshold used;
- count of files skipped for visual similarity because they could not be decoded.

The preview should include a compact summary:

```text
Burst groups: 3
Largest burst: 12 files
Similar groups: 2
Visual similarity skipped: 17
```

## Architecture

Add focused functionality to `internal/photo`:

```text
internal/photo/
  groups.go
  groups_test.go
  similarity.go
  similarity_test.go
```

Responsibilities:

- build grouped rename keys;
- sort files deterministically by date, camera, type, extension, hash, and original path;
- assign grouped filenames;
- detect burst groups from enriched files;
- compute perceptual hashes for decodable image files;
- detect similar groups by perceptual hash distance.

Existing files to update:

- `types.go`: add grouped rename and burst-related fields.
- `duplicates.go`: keep exact SHA-256 duplicate detection; do not replace it with perceptual similarity.
- `planner.go`: use grouped rename mode before assigning destination names.
- `ingest.go`: pass enriched files through burst and visual similarity detection when enabled.
- `report.go`: include burst and similarity counts and group details.
- `cmd/photo.go`: add `--burst-window`, `--similarity-threshold`, and guided menu prompts.
- `README.md`: document grouped rename, burst detection, and visual similarity detection.

## Error Handling

- Invalid `--burst-window` duration fails before planning.
- `--burst-window 0s` disables burst detection.
- Invalid `--similarity-threshold` fails before planning.
- Omitted `--similarity-threshold` disables visual similarity detection.
- `--rename grouped` works without `--burst-window`.
- `--rename grouped` works without `--similarity-threshold`.
- Missing metadata uses fallback date and `unknown-camera`, consistent with the existing workflow.
- Hashing failures do not abort planning; they use original filename as deterministic tie-breaker.
- Visual decoding failures do not abort planning; they are counted and reported.

## Testing Strategy

Unit tests:

- grouped rename keeps the first file as base and suffixes later files;
- equal timestamps use stable short-hash tie-breaking;
- grouped rename is stable when input order changes;
- burst detection groups files within the configured time window;
- burst detection does not group files across different cameras;
- burst detection ignores videos;
- visual similarity computes stable hashes for small generated JPEG/PNG fixtures;
- visual similarity groups images within threshold;
- visual similarity does not group images above threshold;
- visual similarity ignores RAW, HEIC, videos, and undecodable files;
- burst grouping takes precedence over similar grouping for rename base;
- invalid burst duration fails in command validation;
- invalid similarity threshold fails in command validation.

Integration-style tests:

- direct organize command with `--rename grouped`;
- direct organize command with `--rename grouped --burst-window 2s`;
- direct organize command with `--rename grouped --similarity-threshold 8`;
- report includes burst and similarity summary.
