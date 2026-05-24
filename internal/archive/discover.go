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
