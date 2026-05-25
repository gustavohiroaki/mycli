package compress

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var videoExtensions = map[string]struct{}{
	".mp4": {}, ".mov": {}, ".avi": {}, ".mkv": {}, ".m4v": {},
	".3gp": {}, ".mts": {}, ".m2ts": {}, ".mpg": {}, ".mpeg": {},
	".wmv": {}, ".flv": {}, ".webm": {},
}

func Discover(input string, options Options) ([]Item, int, error) {
	absInput, err := filepath.Abs(input)
	if err != nil {
		return nil, 0, err
	}
	info, err := os.Stat(absInput)
	if err != nil {
		return nil, 0, err
	}
	if !info.IsDir() {
		if !isVideo(absInput) {
			return nil, 1, fmt.Errorf("unsupported video: %s", input)
		}
		return []Item{buildItem(absInput, filepath.Base(absInput), info.Size(), options)}, 0, nil
	}

	var items []Item
	skipped := 0
	err = filepath.WalkDir(absInput, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == absInput {
			return nil
		}
		if entry.IsDir() {
			if !options.Recursive {
				return filepath.SkipDir
			}
			return nil
		}
		if !isVideo(path) {
			skipped++
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(absInput, path)
		if err != nil {
			return err
		}
		items = append(items, buildItem(path, rel, info.Size(), options))
		return nil
	})
	return items, skipped, err
}

func isVideo(path string) bool {
	_, ok := videoExtensions[strings.ToLower(filepath.Ext(path))]
	return ok
}

func buildItem(path string, rel string, size int64, options Options) Item {
	dest := options.Dest
	if dest == "" {
		dir := filepath.Dir(path)
		base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		dest = filepath.Join(dir, base+"_compressed.mp4")
	} else if stat, err := os.Stat(dest); err == nil && stat.IsDir() {
		dest = filepath.Join(dest, replaceExt(rel, ".mp4"))
	} else if len(rel) > 0 && filepath.Ext(dest) == "" {
		dest = filepath.Join(dest, replaceExt(rel, ".mp4"))
	}
	return Item{SourcePath: path, RelPath: rel, DestPath: dest, Size: size}
}

func replaceExt(path string, ext string) string {
	return strings.TrimSuffix(path, filepath.Ext(path)) + ext
}
