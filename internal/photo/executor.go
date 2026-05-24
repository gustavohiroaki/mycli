package photo

import (
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type ProgressFunc func(done int, total int, action PlannedAction)

func ExecutePlan(plan Plan) Summary {
	return ExecutePlanWithProgress(plan, nil)
}

func ExecutePlanWithProgress(plan Plan, progress ProgressFunc) Summary {
	if plan.Options.FullPerformance {
		return executePlanParallel(plan, progress, photoWorkerCount(plan.Options))
	}
	return executePlanSequential(plan, progress)
}

func executePlanSequential(plan Plan, progress ProgressFunc) Summary {
	summary := Summary{Media: len(plan.Actions)}
	total := len(plan.Actions)
	for index, action := range plan.Actions {
		countAction(action, &summary)
		if action.Kind == ActionSkip {
			summary.Skipped++
			reportProgress(progress, index+1, total, action)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(action.DestPath), 0o755); err != nil {
			summary.Failed++
			reportProgress(progress, index+1, total, action)
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
			reportProgress(progress, index+1, total, action)
			continue
		}
		if action.Kind == ActionMove {
			summary.Moved++
		} else {
			summary.Copied++
		}
		reportProgress(progress, index+1, total, action)
	}
	return summary
}

func executePlanParallel(plan Plan, progress ProgressFunc, workers int) Summary {
	if workers <= 1 || len(plan.Actions) < 2 {
		return executePlanSequential(plan, progress)
	}

	type actionResult struct {
		action PlannedAction
		failed bool
	}

	jobs := make(chan int)
	results := make(chan actionResult, len(plan.Actions))
	var wg sync.WaitGroup
	for worker := 0; worker < workers; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for index := range jobs {
				action := plan.Actions[index]
				results <- actionResult{
					action: action,
					failed: executeOneAction(action),
				}
			}
		}()
	}
	for index := range plan.Actions {
		jobs <- index
	}
	close(jobs)
	wg.Wait()
	close(results)

	summary := Summary{Media: len(plan.Actions)}
	completed := 0
	for result := range results {
		completed++
		countAction(result.action, &summary)
		switch {
		case result.action.Kind == ActionSkip:
			summary.Skipped++
		case result.failed:
			summary.Failed++
		case result.action.Kind == ActionMove:
			summary.Moved++
		default:
			summary.Copied++
		}
		reportProgress(progress, completed, len(plan.Actions), result.action)
	}
	return summary
}

func executeOneAction(action PlannedAction) bool {
	if action.Kind == ActionSkip {
		return false
	}
	if err := os.MkdirAll(filepath.Dir(action.DestPath), 0o755); err != nil {
		return true
	}

	var err error
	if action.Kind == ActionMove {
		err = moveFile(action.SourcePath, action.DestPath)
	} else {
		err = copyFile(action.SourcePath, action.DestPath)
	}
	return err != nil
}

func reportProgress(progress ProgressFunc, done int, total int, action PlannedAction) {
	if progress != nil {
		progress(done, total, action)
	}
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
