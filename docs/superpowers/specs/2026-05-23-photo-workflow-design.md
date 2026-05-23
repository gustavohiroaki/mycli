# Photo Workflow Utility Design

## Context

`mycli` currently has a root-level `organize` command that organizes photos and videos from a source directory into a destination directory by date:

```text
year/month/day/fotos
year/month/day/videos
```

The command already scans recursively, detects media by extension, reads dates with `exiftool` when available, falls back to filename and modification time, supports copy by default, and supports move with `--move`.

The new goal is to turn this into a broader photography workflow utility under `mycli photo`, with a guided terminal menu and a direct command path for scripting.

## Goals

- Add `mycli photo` as the main photography workflow entrypoint.
- Open a guided terminal menu when `mycli photo` is run without a subcommand.
- Move the current organize behavior into the photography utility.
- Keep a direct command for non-interactive use: `mycli photo organize <source> <destination>`.
- Support recursive source scanning by default.
- Ignore directories only when the user passes explicit exclude rules through the menu or flags.
- Make folder structure configurable through presets and custom templates.
- Support organizing by metadata such as camera model.
- Use `exiftool` as the recommended metadata source.
- Add duplicate handling choices.
- Add optional renaming.
- Generate an ingest report.

## Non-Goals For First Implementation

- No automatic ignored directory heuristics.
- No persistent saved presets or config files.
- No external database or catalog management.
- No destructive deletion of duplicates.
- No image editing, conversion, rating, or culling.
- No GUI.

## User Interface

### Guided Menu

Running:

```bash
mycli photo
```

opens a guided workflow. The first menu action is "Complete ingest", covering:

1. Choose source directory.
2. Choose destination directory.
3. Choose whether to scan recursively.
4. Choose exclude patterns, if any.
5. Choose copy or move.
6. Choose folder structure preset or custom template.
7. Choose whether to rename files.
8. Choose duplicate handling behavior.
9. Preview the planned work.
10. Confirm execution.
11. Show final summary and write report.

### Direct Command

The direct command remains available for automation:

```bash
mycli photo organize <source> <destination>
```

It is recursive by default and supports flags:

```bash
--move
--no-recursive
--exclude <pattern>
--structure <template-or-preset>
--rename <template-or-preset>
--duplicates <skip|separate|suffix>
--allow-fallback
--report <txt|json|none>
```

The old root-level `mycli organize` command will be removed from the public command tree in the first implementation. The behavior migrates to `mycli photo organize`, and the README documents the new command.

## Folder Structure

The workflow supports presets and custom templates.

Initial presets:

```text
{year}/{month}/{day}/{type}
{year}/{year}-{month}-{day}/{type}
{camera}/{year}/{month}/{day}/{type}
{year}/{camera}/{year}-{month}-{day}/{type}
```

Custom templates use named tokens:

```text
{year}/{month}/{day}/{camera}/{type}
```

Initial folder tokens:

```text
{year}
{month}
{day}
{date}
{time}
{camera}
{lens}
{type}
{extension}
```

Token values must be sanitized before becoming path segments. Missing values use stable fallback labels such as `unknown-camera` or `unknown-lens`.

The `{type}` token maps files into categories such as:

```text
photos
videos
raw
```

The existing Portuguese folder names `fotos` and `videos` can be preserved as a compatibility preset, but the new template engine should use neutral token values internally.

## File Renaming

Renaming is optional. The default is to preserve the original filename.

Initial rename preset:

```text
{date}_{time}_{camera}_{seq}{ext}
```

The planner resolves `{seq}` after grouping files with the same rendered name. Name collisions are handled deterministically by adding a numeric suffix.

## Metadata

`exiftool` is the primary metadata provider. The workflow checks for it at the start.

In the guided menu:

- If `exiftool` is present, continue normally.
- If `exiftool` is missing, warn that metadata support will be limited and ask whether to continue with fallback behavior.

In direct command mode:

- Missing `exiftool` aborts by default.
- `--allow-fallback` allows fallback behavior.

Metadata resolution order:

1. `exiftool` date fields such as `DateTimeOriginal`, `CreateDate`, `MediaCreateDate`, and `TrackCreateDate`.
2. Filename date patterns.
3. File modification time.

Camera model, lens, and other non-date metadata come from `exiftool`. If unavailable, fallback labels are used.

## Recursive Scanning

The source is scanned recursively by default. This applies to both guided and direct modes.

The destination must not be the same as the source and must not be inside the source. This prevents the workflow from reprocessing files it just wrote.

No directories are ignored automatically. The only ignored paths are those matching user-provided exclude patterns.

## Duplicate Handling

The guided workflow asks for duplicate behavior on each run:

```text
skip
separate
suffix
```

Default: `skip`.

Behavior:

- `skip`: calculate content hashes and do not copy/move duplicate files.
- `separate`: put duplicates under a `duplicates` area inside the destination.
- `suffix`: copy duplicates to the planned folder with a unique suffix.

No duplicate behavior deletes files from source or destination.

## Planner And Execution Flow

The implementation should separate planning from execution.

1. Scanner finds candidate media files.
2. Metadata reader enriches each file.
3. Duplicate detector computes hashes and marks duplicates.
4. Template renderer computes destination directories and optional new names.
5. Planner builds a list of actions.
6. Preview summarizes actions before anything is written.
7. Executor copies or moves files.
8. Reporter writes final summary.

Planning must not mutate the filesystem except for optional report path validation. Execution is the only phase that writes media files.

## Architecture

Recommended package layout:

```text
cmd/
  photo.go

internal/photo/
  ingest.go
  scanner.go
  metadata.go
  templates.go
  duplicates.go
  planner.go
  executor.go
  report.go
```

Responsibilities:

- `cmd/photo.go`: Cobra commands, menu prompts, flag parsing, user-facing output.
- `internal/photo/scanner.go`: recursive walk, extension filtering, exclude matching.
- `internal/photo/metadata.go`: `exiftool` integration and fallback date parsing.
- `internal/photo/templates.go`: presets, token rendering, path sanitization.
- `internal/photo/duplicates.go`: hashing and duplicate classification.
- `internal/photo/planner.go`: produce a filesystem action plan.
- `internal/photo/executor.go`: copy/move actions and collision handling.
- `internal/photo/report.go`: text and JSON reports.
- `internal/photo/ingest.go`: orchestration and public package API.

The current `cmd/organize.go` logic should be split into these units rather than expanded in place.

## Error Handling

- Invalid source: fail before scanning.
- Invalid destination: fail before scanning.
- Destination inside source: fail before scanning.
- Missing `exiftool`: guided mode asks, direct mode fails unless `--allow-fallback` is set.
- Invalid template token: fail during option validation.
- File read/copy/move error: record failure, continue when possible, and include it in the report.
- Hashing error: record failure and treat the file as non-deduplicated unless the error prevents copying.
- Name collision: resolve with deterministic suffix.

## Reporting

Reports should include:

- Source and destination.
- Execution timestamp.
- Recursive mode.
- Exclude patterns.
- Copy or move mode.
- Folder template.
- Rename template, if any.
- Duplicate policy.
- Total scanned files.
- Total media files.
- Total copied, moved, skipped, duplicate, and failed files.
- Counts by media type.
- Count of files using fallback metadata.
- Failure details.

Default report format: text. JSON should be available for later automation.

## Testing Strategy

Unit tests:

- Filename date parsing.
- Template rendering and sanitization.
- Recursive scanner behavior.
- Exclude pattern behavior.
- Duplicate hash classification.
- Destination collision naming.
- Destination safety checks.

Integration-style tests with temporary directories:

- Copy workflow with nested source directories.
- Move workflow with nested source directories.
- Duplicate policies.
- Missing metadata fallback.

`exiftool` should be abstracted behind an interface so tests do not require the binary.

## Documentation Updates

Update `README.md` to show:

```bash
mycli photo
mycli photo organize <source> <destination>
```

Document that `exiftool` is recommended for full metadata support and that scanning is recursive by default.
