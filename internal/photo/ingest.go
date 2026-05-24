package photo

func PlanIngest(options Options, provider MetadataProvider) (Plan, Summary, error) {
	normalizeOptions(&options)

	if err := ValidateSourceDestination(options.Source, options.Destination); err != nil {
		return Plan{}, Summary{}, err
	}

	files, err := Scan(options.Source, ScanOptions{Recursive: options.Recursive, Excludes: options.Excludes})
	if err != nil {
		return Plan{}, Summary{}, err
	}

	enriched := make([]EnrichedFile, 0, len(files))
	for _, file := range files {
		metadata := ResolveMetadata(file, provider)
		hash, err := HashFile(file.SourcePath)
		if err != nil {
			hash = ""
		}
		enriched = append(enriched, EnrichedFile{File: file, Metadata: metadata, Hash: hash})
	}

	visualSkipped := 0
	if options.SimilarityEnabled {
		enriched, visualSkipped = EnrichVisualHashes(enriched)
	}
	burstGroups := DetectBurstGroups(enriched, options.BurstWindow)
	var similarGroups []FileGroup
	if options.SimilarityEnabled {
		similarGroups = DetectSimilarGroups(enriched, options.SimilarityThreshold)
	}
	grouping := MergeGrouping(enriched, burstGroups, similarGroups, visualSkipped, options)

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
