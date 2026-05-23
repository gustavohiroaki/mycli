package photo

import (
	"io"
	"os"
	"path/filepath"
	"time"
)

func ExecutePlan(plan Plan) Summary {
	summary := Summary{Media: len(plan.Actions)}
	for _, action := range plan.Actions {
		countAction(action, &summary)
		if action.Kind == ActionSkip {
			summary.Skipped++
			continue
		}
		if err := os.MkdirAll(filepath.Dir(action.DestPath), 0o755); err != nil {
			summary.Failed++
			continue
		}

		var err error
		if action.Kind == ActionMove {
			err = moveFile(action.SourcePath, action.DestPath)
		} else {
			err = copyFile(action.SourcePath, action.DestPath)
		}
		if err != nil {
			summary.Failed++
			continue
		}
		if action.Kind == ActionMove {
			summary.Moved++
		} else {
			summary.Copied++
		}
	}
	return summary
}

func countAction(action PlannedAction, summary *Summary) {
	if action.Duplicate {
		summary.Duplicates++
	}
	if action.UsedFallback {
		summary.FallbackDates++
	}
	switch action.MediaType {
	case MediaTypePhoto:
		summary.Photos++
	case MediaTypeVideo:
		summary.Videos++
	case MediaTypeRaw:
		summary.Raw++
	}
}

func moveFile(source string, target string) error {
	if err := os.Rename(source, target); err == nil {
		return nil
	}
	if err := copyFile(source, target); err != nil {
		return err
	}
	return os.Remove(source)
}

func copyFile(source string, target string) error {
	sourceFile, err := os.Open(source)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	info, err := sourceFile.Stat()
	if err != nil {
		return err
	}

	targetFile, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode().Perm())
	if err != nil {
		return err
	}

	_, copyErr := io.Copy(targetFile, sourceFile)
	closeErr := targetFile.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Chtimes(target, time.Now(), info.ModTime())
}
