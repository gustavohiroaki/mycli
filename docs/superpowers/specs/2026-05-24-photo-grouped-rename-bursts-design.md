# Photo Grouped Rename And Burst Detection Design

## Context

The photo workflow currently supports exact duplicate detection through SHA-256 hashes and optional renaming through a template such as:

```text
{date}_{time}_{camera}_{seq}{ext}
```

The next improvement is to make related photos easier to review by giving them similar names and detecting burst sequences based on capture time.

## Goals

- Add a grouped rename mode for related photos.
- Keep grouped rename deterministic across repeated runs.
- Use short content hashes as the tie-breaker when multiple photos share the same timestamp.
- Detect burst groups from metadata timestamps.
- Include burst information in the report.
- Keep this feature separate from perceptual/visual similarity detection.

## Non-Goals

- Do not implement perceptual hash or visual-near-duplicate detection in this iteration.
- Do not move burst groups into separate folders in this iteration.
- Do not include burst identifiers in filenames in this iteration.
- Do not delete or auto-select best photos from a burst.

## CLI Behavior

Grouped rename is enabled with:

```bash
mycli photo organize ./entrada ./biblioteca --rename grouped
```

Burst detection is enabled by passing a duration:

```bash
mycli photo organize ./entrada ./biblioteca --rename grouped --burst-window 2s
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

## Reporting

The text and JSON reports should include:

- total burst groups;
- largest burst size;
- files that belong to each burst group;
- burst window used.

The preview should include a compact summary:

```text
Burst groups: 3
Largest burst: 12 files
```

## Architecture

Add focused functionality to `internal/photo`:

```text
internal/photo/
  groups.go
  groups_test.go
```

Responsibilities:

- build grouped rename keys;
- sort files deterministically by date, camera, type, extension, hash, and original path;
- assign grouped filenames;
- detect burst groups from enriched files.

Existing files to update:

- `types.go`: add grouped rename and burst-related fields.
- `planner.go`: use grouped rename mode before assigning destination names.
- `ingest.go`: pass enriched files through burst detection when enabled.
- `report.go`: include burst counts and group details.
- `cmd/photo.go`: add `--burst-window` and guided menu prompts.
- `README.md`: document grouped rename and burst detection.

## Error Handling

- Invalid `--burst-window` duration fails before planning.
- `--burst-window 0s` disables burst detection.
- `--rename grouped` works without `--burst-window`.
- Missing metadata uses fallback date and `unknown-camera`, consistent with the existing workflow.
- Hashing failures do not abort planning; they use original filename as deterministic tie-breaker.

## Testing Strategy

Unit tests:

- grouped rename keeps the first file as base and suffixes later files;
- equal timestamps use stable short-hash tie-breaking;
- grouped rename is stable when input order changes;
- burst detection groups files within the configured time window;
- burst detection does not group files across different cameras;
- burst detection ignores videos;
- invalid burst duration fails in command validation.

Integration-style tests:

- direct organize command with `--rename grouped`;
- direct organize command with `--rename grouped --burst-window 2s`;
- report includes burst summary.
