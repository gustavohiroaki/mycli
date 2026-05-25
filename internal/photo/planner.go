package photo

import (
	"fmt"
	"path/filepath"
)

func BuildPlan(files []EnrichedFile, duplicates map[string]bool, options Options) (Plan, error) {
	return BuildPlanWithGrouping(files, duplicates, options, GroupingResult{})
}

func BuildPlanWithGrouping(files []EnrichedFile, duplicates map[string]bool, options Options, grouping GroupingResult) (Plan, error) {
	structure, err := ResolveStructure(options.Structure)
	if err != nil {
		return Plan{}, err
	}
	if options.Duplicates == "" {
		options.Duplicates = DuplicateSkip
	}

	planned := Plan{Options: options, Grouping: grouping}
	usedDestinations := map[string]int{}
	groupedNames := map[string]string{}
	if options.Rename == "grouped" {
		groupedNames = AssignGroupedNames(files, grouping)
	}
	for index, file := range files {
		isDuplicate := duplicates[file.File.SourcePath]
		action := PlannedAction{
			Kind:          ActionCopy,
			SourcePath:    file.File.SourcePath,
			MediaType:     file.File.Type,
			Duplicate:     isDuplicate,
			UsedFallback:  file.Metadata.UsedFallback,
			Hash:          file.Hash,
			VisualHash:    file.VisualHash,
			HasVisualHash: file.HasVisualHash,
			Metadata:      file.Metadata,
			SourceSize:    file.File.Size,
			Extension:     file.File.Extension,
		}
		if options.Move {
			action.Kind = ActionMove
		}

		destinationRoot := options.Destination
		if isDuplicate {
			switch options.Duplicates {
			case DuplicateSkip:
				action.Kind = ActionSkip
				planned.Actions = append(planned.Actions, action)
				continue
			case DuplicateSeparate:
				destinationRoot = filepath.Join(options.Destination, "duplicates")
			case DuplicateSuffix:
			default:
				return Plan{}, fmt.Errorf("invalid duplicate policy %q", options.Duplicates)
			}
		}

		relativeDir, err := RenderTemplate(structure, file, index+1)
		if err != nil {
			return Plan{}, err
		}

		fileName := filepath.Base(file.File.SourcePath)
		if options.Rename == "grouped" {
			fileName = groupedNames[file.File.SourcePath]
		} else if options.Rename != "" {
			fileName, err = RenderTemplate(options.Rename, file, index+1)
			if err != nil {
				return Plan{}, err
			}
		}

		destination := filepath.Join(destinationRoot, filepath.FromSlash(relativeDir), fileName)
		if isDuplicate && options.Duplicates == DuplicateSuffix {
			destination = addSuffixBeforeExtension(destination, "duplicate")
		}
		action.DestPath = uniquePlannedDestination(destination, usedDestinations)
		planned.Actions = append(planned.Actions, action)
	}
	return planned, nil
}

func uniquePlannedDestination(path string, used map[string]int) string {
	if used[path] == 0 {
		used[path] = 1
		return path
	}
	used[path]++
	return addSuffixBeforeExtension(path, fmt.Sprintf("%d", used[path]))
}

func addSuffixBeforeExtension(path string, suffix string) string {
	ext := filepath.Ext(path)
	base := path[:len(path)-len(ext)]
	return base + "_" + suffix + ext
}
