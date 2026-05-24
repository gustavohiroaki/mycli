package photo

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

func DetectBurstGroups(files []EnrichedFile, window time.Duration) []FileGroup {
	if window <= 0 {
		return nil
	}

	byCamera := map[string][]EnrichedFile{}
	for _, file := range files {
		if file.File.Type != MediaTypePhoto && file.File.Type != MediaTypeRaw {
			continue
		}
		camera := sanitizeTokenValue(defaultString(file.Metadata.Camera, "unknown-camera"))
		byCamera[camera] = append(byCamera[camera], file)
	}

	var groups []FileGroup
	for _, cameraFiles := range byCamera {
		sortEnriched(cameraFiles)
		var current []EnrichedFile
		for _, file := range cameraFiles {
			if len(current) == 0 {
				current = append(current, file)
				continue
			}
			last := current[len(current)-1]
			if file.Metadata.Date.Sub(last.Metadata.Date) <= window {
				current = append(current, file)
				continue
			}
			if len(current) > 1 {
				groups = append(groups, makeGroup(GroupBurst, len(groups)+1, current))
			}
			current = []EnrichedFile{file}
		}
		if len(current) > 1 {
			groups = append(groups, makeGroup(GroupBurst, len(groups)+1, current))
		}
	}

	sort.Slice(groups, func(i, j int) bool {
		return groups[i].Files[0] < groups[j].Files[0]
	})
	for i := range groups {
		groups[i].ID = fmt.Sprintf("burst-%03d", i+1)
	}
	return groups
}

func AssignGroupedNames(files []EnrichedFile, grouping GroupingResult) map[string]string {
	filesByPath := map[string]EnrichedFile{}
	for _, file := range files {
		filesByPath[file.File.SourcePath] = file
	}

	buckets := map[string][]EnrichedFile{}
	groupBases := map[string]EnrichedFile{}
	for _, group := range append(grouping.BurstGroups, grouping.SimilarGroups...) {
		groupFiles := filesForGroup(group, filesByPath)
		if len(groupFiles) == 0 {
			continue
		}
		sortEnriched(groupFiles)
		groupBases[group.ID] = groupFiles[0]
	}

	for _, file := range files {
		key := grouping.PreferredGroupByFile[file.File.SourcePath]
		if key == "" {
			key = defaultGroupKey(file)
		}
		buckets[key] = append(buckets[key], file)
	}

	names := map[string]string{}
	keys := make([]string, 0, len(buckets))
	for key := range buckets {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		bucket := buckets[key]
		sortEnriched(bucket)
		baseFile := bucket[0]
		if groupedBase, ok := groupBases[key]; ok {
			baseFile = groupedBase
		}
		base := groupedBaseName(baseFile)
		for index, file := range bucket {
			name := base + strings.ToLower(file.File.Extension)
			if len(bucket) > 1 {
				name = fmt.Sprintf("%s_%03d%s", base, index+1, strings.ToLower(file.File.Extension))
			}
			names[file.File.SourcePath] = name
		}
	}

	return names
}

func MergeGrouping(files []EnrichedFile, burstGroups []FileGroup, similarGroups []FileGroup, visualSkipped int, options Options) GroupingResult {
	preferred := map[string]string{}
	for _, group := range similarGroups {
		for _, path := range group.Files {
			preferred[path] = group.ID
		}
	}
	for _, group := range burstGroups {
		for _, path := range group.Files {
			preferred[path] = group.ID
		}
	}
	return GroupingResult{
		BurstGroups:             burstGroups,
		SimilarGroups:           similarGroups,
		PreferredGroupByFile:    preferred,
		VisualSimilaritySkipped: visualSkipped,
		BurstWindow:             options.BurstWindow,
		SimilarityEnabled:       options.SimilarityEnabled,
		SimilarityThreshold:     options.SimilarityThreshold,
	}
}

func filesForGroup(group FileGroup, filesByPath map[string]EnrichedFile) []EnrichedFile {
	files := make([]EnrichedFile, 0, len(group.Files))
	for _, path := range group.Files {
		if file, ok := filesByPath[path]; ok {
			files = append(files, file)
		}
	}
	return files
}

func makeGroup(groupType GroupType, index int, files []EnrichedFile) FileGroup {
	paths := make([]string, 0, len(files))
	for _, file := range files {
		paths = append(paths, file.File.SourcePath)
	}
	return FileGroup{ID: fmt.Sprintf("%s-%03d", groupType, index), Type: groupType, Files: paths}
}

func sortEnriched(files []EnrichedFile) {
	sort.Slice(files, func(i, j int) bool {
		left := files[i]
		right := files[j]
		if !left.Metadata.Date.Equal(right.Metadata.Date) {
			return left.Metadata.Date.Before(right.Metadata.Date)
		}
		return left.File.SourcePath < right.File.SourcePath
	})
}

func defaultGroupKey(file EnrichedFile) string {
	return strings.Join([]string{
		file.Metadata.Date.Format("2006-01-02 15:04:05"),
		sanitizeTokenValue(defaultString(file.Metadata.Camera, "unknown-camera")),
		string(file.File.Type),
		strings.ToLower(file.File.Extension),
	}, "|")
}

func groupedBaseName(file EnrichedFile) string {
	return strings.Join([]string{
		file.Metadata.Date.Format("2006-01-02"),
		file.Metadata.Date.Format("15-04-05"),
	}, "_")
}
