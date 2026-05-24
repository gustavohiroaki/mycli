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
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(os.PathSeparator)) {
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
	return runtime.GOOS != "windows" && strings.Contains(path, `:\`)
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
