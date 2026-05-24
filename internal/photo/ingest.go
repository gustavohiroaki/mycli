package photo

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

	enriched := make([]EnrichedFile, 0, len(files))
	for index, file := range files {
		metadata := ResolveMetadata(file, provider)
		hash, err := HashFile(file.SourcePath)
		if err != nil {
			hash = ""
		}
		enriched = append(enriched, EnrichedFile{File: file, Metadata: metadata, Hash: hash})
		reportPlanProgress(progress, "metadata", index+1, len(files), file.SourcePath)
	}

	visualSkipped := 0
	if options.SimilarityEnabled {
		enriched, visualSkipped = EnrichVisualHashesWithProgress(enriched, func(done int, total int, path string) {
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
