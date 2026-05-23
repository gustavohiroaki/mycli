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

type ScanOptions struct {
	Recursive bool
	Excludes  []string
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

func isSubpath(candidate string, parent string) bool {
	relative, err := filepath.Rel(parent, candidate)
	if err != nil {
		return false
	}
	return relative != "." && relative != ".." && !strings.HasPrefix(relative, ".."+string(os.PathSeparator))
}
