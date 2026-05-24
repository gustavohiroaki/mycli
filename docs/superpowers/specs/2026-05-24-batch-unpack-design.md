# Batch Unpack Utility Design

## Context

`mycli` is a personal CLI with independent commands such as `prompt` and `photo`. The new utility should extract compressed archives and delete each original archive only after the extracted output is verified.

This command is destructive by design, so deletion must be gated by successful extraction and verification for each archive independently.

## Goals

- Add a root command:

```bash
mycli unpack <file-or-directory>
```

- Support these archive formats in the first implementation:

```text
.zip
.tar
.tar.gz
.tgz
```

- Accept a single archive file or a directory containing multiple archives.
- Process only the given directory level by default.
- Process subdirectories only when `--recursive` is passed.
- Delete each original archive automatically after that archive is extracted and verified.
- Preserve failed archives.
- Provide `--keep` to preserve originals even after successful verification.
- Provide `--dest` to choose an extraction root.
- Provide `--overwrite` to explicitly allow replacing existing files.
- Print a final batch summary.

## Non-Goals

- No `.rar` or `.7z` support in the first implementation.
- No password-protected archive support in the first implementation.
- No archive creation command.
- No parallel extraction in the first implementation.
- No move-to-trash behavior in the first implementation.

## CLI Behavior

Single file:

```bash
mycli unpack arquivo.zip
mycli unpack arquivo.tar.gz
```

Directory batch:

```bash
mycli unpack ./downloads
```

Recursive directory batch:

```bash
mycli unpack ./downloads --recursive
```

Custom destination:

```bash
mycli unpack ./downloads --dest ./extraidos
```

Keep original archives:

```bash
mycli unpack ./downloads --keep
```

Allow overwriting existing files:

```bash
mycli unpack ./downloads --overwrite
```

## Archive Discovery

If the input is a file:

- validate that the extension is supported;
- process only that file.

If the input is a directory:

- scan direct children only by default;
- include nested archives only with `--recursive`;
- ignore unsupported files;
- detect `.tar.gz` before `.gz`-style suffix handling, so archive base names are correct.

Supported extensions:

```text
.zip
.tar
.tar.gz
.tgz
```

## Destination Rules

Without `--dest`, extract beside each archive into a directory named after the archive base:

```text
downloads/fotos.zip     -> downloads/fotos/
downloads/aulas.tar.gz  -> downloads/aulas/
downloads/backup.tgz    -> downloads/backup/
```

With `--dest`, extract each archive into one child directory under the destination:

```text
mycli unpack ./downloads --dest ./extraidos

./extraidos/fotos/
./extraidos/aulas/
./extraidos/backup/
```

The command must create destination directories as needed.

Destination path collisions:

- if the destination directory already exists and is non-empty, fail that archive unless `--overwrite` is set;
- with `--overwrite`, files inside the extraction directory may be replaced by archive contents;
- `--overwrite` never deletes unrelated files that are not overwritten by archive entries.

## Extraction And Verification

Each archive is processed independently:

1. Determine archive type.
2. Determine extraction directory.
3. Validate destination safety.
4. Extract archive entries.
5. Verify extracted files against archive metadata.
6. Delete original archive only if verification succeeds and `--keep` is not set.
7. Record per-archive result.

Verification rules:

- `.zip`: validate each extracted regular file size and CRC32 against the zip entry metadata.
- `.tar`: validate each extracted regular file size against the tar entry metadata.
- `.tar.gz` and `.tgz`: decompress gzip stream, read tar entries, and validate each extracted regular file size against tar metadata.

Directory entries do not require file content verification.

If verification fails, the extracted directory may remain for inspection, but the original archive must not be deleted.

## Security Rules

Archive entries must not be allowed to write outside the destination directory.

Reject entries with:

```text
absolute paths
../ path traversal
Windows drive prefixes
```

Symlink and hardlink entries should be skipped in the first implementation. They should be reported as skipped entries rather than followed or created. This avoids links escaping the extraction directory.

## Output

For each archive, print a concise status:

```text
OK    fotos.zip -> downloads/fotos (12 files, original deleted)
KEEP  aulas.tar.gz -> downloads/aulas (8 files, --keep)
FAIL  backup.tgz -> checksum mismatch: images/001.jpg
```

Final summary:

```text
Archives found: 3
Extracted: 2
Deleted originals: 1
Kept originals: 1
Failed: 1
Skipped unsupported: 4
```

## Architecture

Create a focused package:

```text
internal/archive/
  types.go
  discover.go
  extract.go
  verify.go
```

Create a Cobra command:

```text
cmd/unpack.go
```

Responsibilities:

- `cmd/unpack.go`: argument parsing, flags, user-facing output.
- `internal/archive/types.go`: archive type, options, item result, batch summary.
- `internal/archive/discover.go`: detect supported archive types and discover files from file or directory input.
- `internal/archive/extract.go`: zip/tar/tar.gz extraction with path safety checks.
- `internal/archive/verify.go`: post-extraction verification and deletion gate.

## Error Handling

- Missing input path: command error.
- Unsupported single file: command error.
- Directory with no archives: successful no-op with summary.
- Unsupported files in directory batch: counted as skipped unsupported.
- Existing non-empty destination without `--overwrite`: fail that archive and keep original.
- Extraction error: fail that archive and keep original.
- Verification error: fail that archive and keep original.
- Delete error after verified extraction: report failure to delete, keep extraction, and count original as kept.

## Testing Strategy

Unit tests:

- archive type detection for `.zip`, `.tar`, `.tar.gz`, `.tgz`, and unsupported files;
- archive base directory naming;
- non-recursive discovery scans only direct children;
- recursive discovery includes nested archives;
- path traversal entries are rejected;
- existing non-empty destination fails without `--overwrite`;
- `--keep` preserves original after successful extraction.

Integration-style tests with temporary directories:

- extract and verify a zip, then delete original;
- extract and verify a tar, then delete original;
- extract and verify a tar.gz, then delete original;
- process a directory containing multiple supported archives and unsupported files;
- preserve failed archive when verification or extraction fails.

## Documentation Updates

Update `README.md` with:

```bash
mycli unpack arquivo.zip
mycli unpack ./downloads
mycli unpack ./downloads --recursive
mycli unpack ./downloads --dest ./extraidos --keep
```

Mention that originals are deleted automatically after successful verification, and that `--keep` disables deletion.
