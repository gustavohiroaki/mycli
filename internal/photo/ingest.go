package photo

import (
	"runtime"
	"sync"
)

type PlanProgressFunc func(stage string, done int, total int, path string)

func PlanIngest(options Options, provider MetadataProvider) (Plan, Summary, error) {
	return PlanIngestWithProgress(options, provider, nil)
}

func PlanIngestWithProgress(options Options, provider MetadataProvider, progress PlanProgressFunc) (Plan, Summary, error) {
	normalizeOptions(&options)

	reportPlanProgress(progress, "validate", 0, 0, options.Source)
	if err := ValidateSourceDestination(options.Source, options.Destination); err != nil {
		return Plan{}, Summary{}, err
	}

	reportPlanProgress(progress, "scan", 0, 0, options.Source)
	files, err := Scan(options.Source, ScanOptions{Recursive: options.Recursive, Excludes: options.Excludes})
	if err != nil {
		return Plan{}, Summary{}, err
	}
	reportPlanProgress(progress, "scan", len(files), len(files), "")

	workers := photoWorkerCount(options)
	enriched := enrichFiles(files, provider, workers, progress)

	visualSkipped := 0
	if options.SimilarityEnabled {
		enriched, visualSkipped = EnrichVisualHashesWithOptions(enriched, workers, func(done int, total int, path string) {
			reportPlanProgress(progress, "visual-hash", done, total, path)
		})
	}
	reportPlanProgress(progress, "grouping", 0, 0, "")
	burstGroups := DetectBurstGroups(enriched, options.BurstWindow)
	var similarGroups []FileGroup
	if options.SimilarityEnabled {
		similarGroups = DetectSimilarGroups(enriched, options.SimilarityThreshold)
	}
	grouping := MergeGrouping(enriched, burstGroups, similarGroups, visualSkipped, options)

	reportPlanProgress(progress, "plan", 0, 0, "")
	duplicates := MarkDuplicates(enriched)
	plan, err := BuildPlanWithGrouping(enriched, duplicates, options, grouping)
	if err != nil {
		return Plan{}, Summary{}, err
	}

	summary := SummarizePlan(plan)
	summary.Scanned = len(files)
	summary.Media = len(files)
	return plan, summary, nil
}

func reportPlanProgress(progress PlanProgressFunc, stage string, done int, total int, path string) {
	if progress != nil {
		progress(stage, done, total, path)
	}
}

func enrichFiles(files []MediaFile, provider MetadataProvider, workers int, progress PlanProgressFunc) []EnrichedFile {
	if workers <= 1 || len(files) < 2 {
		enriched := make([]EnrichedFile, 0, len(files))
		for index, file := range files {
			enriched = append(enriched, enrichFile(file, provider))
			reportPlanProgress(progress, "metadata", index+1, len(files), file.SourcePath)
		}
		return enriched
	}

	enriched := make([]EnrichedFile, len(files))
	jobs := make(chan int)
	var wg sync.WaitGroup
	var progressMu sync.Mutex
	completed := 0
	for worker := 0; worker < workers; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for index := range jobs {
				file := files[index]
				enriched[index] = enrichFile(file, provider)
				progressMu.Lock()
				completed++
				reportPlanProgress(progress, "metadata", completed, len(files), file.SourcePath)
				progressMu.Unlock()
			}
		}()
	}
	for index := range files {
		jobs <- index
	}
	close(jobs)
	wg.Wait()
	return enriched
}

func enrichFile(file MediaFile, provider MetadataProvider) EnrichedFile {
	metadata := ResolveMetadata(file, provider)
	hash, err := HashFile(file.SourcePath)
	if err != nil {
		hash = ""
	}
	return EnrichedFile{File: file, Metadata: metadata, Hash: hash}
}

func photoWorkerCount(options Options) int {
	if !options.FullPerformance {
		return 1
	}
	workers := runtime.NumCPU()
	if workers < 2 {
		return 2
	}
	return workers
}

func ExecuteIngest(options Options, provider MetadataProvider) (Plan, Summary, string, error) {
	plan, _, err := PlanIngest(options, provider)
	if err != nil {
		return Plan{}, Summary{}, "", err
	}
	summary := ExecutePlan(plan)
	summary.Scanned = len(plan.Actions)
	summary.Media = len(plan.Actions)
	reportPath, err := WriteReport(plan, summary)
	return plan, summary, reportPath, err
}

func SummarizePlan(plan Plan) Summary {
	summary := Summary{Media: len(plan.Actions)}
	for _, action := range plan.Actions {
		countAction(action, &summary)
		if action.Kind == ActionSkip {
			summary.Skipped++
		}
	}
	summary.BurstGroups = len(plan.Grouping.BurstGroups)
	summary.LargestBurst = largestGroupSize(plan.Grouping.BurstGroups)
	summary.SimilarGroups = len(plan.Grouping.SimilarGroups)
	summary.LargestSimilar = largestGroupSize(plan.Grouping.SimilarGroups)
	summary.VisualSimilaritySkipped = plan.Grouping.VisualSimilaritySkipped
	return summary
}

func largestGroupSize(groups []FileGroup) int {
	largest := 0
	for _, group := range groups {
		if len(group.Files) > largest {
			largest = len(group.Files)
		}
	}
	return largest
}

func normalizeOptions(options *Options) {
	if options.Structure == "" {
		options.Structure = DefaultStructure
	}
	if options.Duplicates == "" {
		options.Duplicates = DuplicateSkip
	}
	if options.Report == "" {
		options.Report = ReportText
	}
}
