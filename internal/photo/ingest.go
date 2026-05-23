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

	duplicates := MarkDuplicates(enriched)
	plan, err := BuildPlan(enriched, duplicates, options)
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
	return summary
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
