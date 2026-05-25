package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"mycli/internal/photo"

	"github.com/spf13/cobra"
)

var (
	guidedPathStructure = "{type}/{year}/{year}-{month}/{year}-{month}-{day}/{camera}"

	photoOptions = photo.Options{
		Recursive:  true,
		Structure:  photo.DefaultStructure,
		Duplicates: photo.DuplicateSkip,
		Report:     photo.ReportText,
	}
	photoNoRecursive bool
)

var photoCmd = &cobra.Command{
	Use:   "photo [source destination]",
	Short: "Photography workflow utilities",
	Long:  "Guided and scriptable photography ingest workflows.",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 || len(args) == 2 {
			return nil
		}
		return fmt.Errorf("accepts either no arguments for the guided menu or <source> <destination>")
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 2 {
			return runPhotoMenu(cmd, args[0], args[1])
		}
		return runPhotoMenu(cmd, "", "")
	},
}

var photoOrganizeCmd = &cobra.Command{
	Use:   "organize <source> <destination>",
	Short: "Organize photos and videos into a photography library",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPhotoOrganize(cmd, args)
	},
}

func init() {
	rootCmd.AddCommand(photoCmd)
	photoCmd.AddCommand(photoOrganizeCmd)

	registerPhotoOrganizeFlags(photoCmd)
	registerPhotoOrganizeFlags(photoOrganizeCmd)
}

func registerPhotoOrganizeFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&photoOptions.Move, "move", false, "Move files instead of copying them")
	cmd.Flags().BoolVar(&photoOptions.Recursive, "recursive", true, "Scan source recursively")
	cmd.Flags().BoolVar(&photoNoRecursive, "no-recursive", false, "Disable recursive scan")
	cmd.Flags().StringArrayVar(&photoOptions.Excludes, "exclude", nil, "Relative path or basename pattern to exclude")
	cmd.Flags().StringVar(&photoOptions.Structure, "structure", photo.DefaultStructure, "Folder structure preset or template")
	cmd.Flags().StringVar(&photoOptions.Rename, "rename", "", "Optional rename template")
	cmd.Flags().Var((*duplicatePolicyValue)(&photoOptions.Duplicates), "duplicates", "Duplicate policy: skip, separate, suffix")
	cmd.Flags().BoolVar(&photoOptions.AllowFallback, "allow-fallback", false, "Continue without exiftool using filename/modtime fallback")
	cmd.Flags().Var((*reportFormatValue)(&photoOptions.Report), "report", "Report format: txt, json, none")
	cmd.Flags().DurationVar(&photoOptions.BurstWindow, "burst-window", 0, "Detect bursts using a duration window such as 2s")
	cmd.Flags().IntVar(&photoOptions.SimilarityThreshold, "similarity-threshold", 8, "Detect visually similar photos with this perceptual hash distance")
	cmd.Flags().BoolVar(&photoOptions.FullPerformance, "fullperformance", false, "Use parallel workers for faster planning and execution")
}

func runPhotoOrganize(cmd *cobra.Command, args []string) error {
	options := photoOptions
	options.Source = args[0]
	options.Destination = args[1]
	if photoNoRecursive {
		options.Recursive = false
	}
	if cmd.Flags().Changed("similarity-threshold") {
		options.SimilarityEnabled = true
	} else {
		options.SimilarityEnabled = false
	}
	if options.SimilarityThreshold < 0 {
		return fmt.Errorf("similarity-threshold cannot be negative")
	}

	provider := photo.ExiftoolProvider{}
	if !options.AllowFallback && !provider.Available() {
		return fmt.Errorf("exiftool not found; install exiftool or pass --allow-fallback")
	}

	plan, previewSummary, err := photo.PlanIngestWithProgress(options, provider, newPhotoPlanProgressPrinter())
	if err != nil {
		return err
	}
	printPlanPreview(plan, previewSummary)

	finalSummary := photo.ExecutePlanWithProgress(plan, printPhotoProgress)
	finalSummary.Scanned = previewSummary.Scanned
	reportPath, err := photo.WriteReport(plan, finalSummary)
	if err != nil {
		return err
	}
	printFinalSummary(finalSummary, reportPath)
	return nil
}

func runPhotoMenu(cmd *cobra.Command, source string, destination string) error {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Photo workflow")
	fmt.Println("1) Complete ingest")
	choice := promptLine(reader, "Choose an option [1]: ")
	if choice != "" && choice != "1" {
		return fmt.Errorf("invalid option %q", choice)
	}

	options := photoOptions
	if source != "" && destination != "" {
		applyGuidedPathDefaults(cmd, &options)
	}
	if cmd.Flags().Changed("similarity-threshold") {
		options.SimilarityEnabled = true
	}
	if photoNoRecursive {
		options.Recursive = false
	}
	options.Source = source
	options.Destination = destination
	if options.Source == "" {
		options.Source = promptRequired(reader, "Source directory: ")
	} else {
		fmt.Printf("Source directory: %s\n", options.Source)
	}
	if options.Destination == "" {
		options.Destination = promptRequired(reader, "Destination directory: ")
	} else {
		fmt.Printf("Destination directory: %s\n", options.Destination)
	}
	options.Recursive = promptYesNoOption(reader, "Scan subfolders?", options.Recursive)

	excludes := promptLine(reader, "Exclude paths, comma-separated: ")
	if excludes != "" {
		for _, item := range strings.Split(excludes, ",") {
			item = strings.TrimSpace(item)
			if item != "" {
				options.Excludes = append(options.Excludes, item)
			}
		}
	}

	options.Move = promptYesNoOption(reader, "Move files instead of copying?", options.Move)
	options.FullPerformance = promptYesNoOption(reader, "Use fullperformance mode?", options.FullPerformance)
	defaultStructure := options.Structure
	structure := promptLine(reader, fmt.Sprintf("Structure preset/template [%s]: ", defaultStructure))
	if structure != "" {
		options.Structure = structure
	} else {
		options.Structure = defaultStructure
		if defaultStructure == "" {
			options.Structure = photo.DefaultStructure
		}
	}

	if promptYesNoOption(reader, "Rename files?", options.Rename != "") {
		fmt.Println("1) Custom template")
		fmt.Println("2) Grouped names")
		renameDefault := "1"
		if options.Rename == "grouped" {
			renameDefault = "2"
		}
		renameChoice := promptLine(reader, fmt.Sprintf("Rename mode [%s]: ", renameDefault))
		if renameChoice == "" {
			renameChoice = renameDefault
		}
		if renameChoice == "2" {
			options.Rename = "grouped"
		} else {
			options.Rename = promptLine(reader, "Rename template [{date}_{time}_{camera}_{seq}{ext}]: ")
		}
		if options.Rename == "" && renameChoice != "2" {
			options.Rename = photo.DefaultRename
		}
	} else {
		options.Rename = ""
	}

	if options.Rename == "grouped" {
		if promptYesNoOption(reader, "Detect bursts by time window?", options.BurstWindow > 0) {
			defaultWindow := options.BurstWindow
			if defaultWindow <= 0 {
				defaultWindow = 2 * time.Second
			}
			window := promptLine(reader, fmt.Sprintf("Burst window [%s]: ", defaultWindow))
			if window == "" {
				window = defaultWindow.String()
			}
			parsed, err := time.ParseDuration(window)
			if err != nil {
				return err
			}
			options.BurstWindow = parsed
		} else {
			options.BurstWindow = 0
		}
		if promptYesNoOption(reader, "Detect visual similarity?", options.SimilarityEnabled) {
			threshold := promptLine(reader, fmt.Sprintf("Similarity threshold [%d]: ", options.SimilarityThreshold))
			if threshold == "" {
				threshold = strconv.Itoa(options.SimilarityThreshold)
			}
			parsed, err := strconv.Atoi(threshold)
			if err != nil {
				return err
			}
			if parsed < 0 {
				return fmt.Errorf("similarity threshold cannot be negative")
			}
			options.SimilarityThreshold = parsed
			options.SimilarityEnabled = true
		} else {
			options.SimilarityEnabled = false
		}
	}

	duplicates := promptLine(reader, fmt.Sprintf("Duplicates [skip|separate|suffix] (%s): ", options.Duplicates))
	if duplicates != "" {
		if err := validateDuplicatePolicy(photo.DuplicatePolicy(duplicates)); err != nil {
			return err
		}
		options.Duplicates = photo.DuplicatePolicy(duplicates)
	}

	provider := photo.ExiftoolProvider{}
	if !provider.Available() && !options.AllowFallback {
		if !promptYesNo(reader, "exiftool not found. Continue with fallback metadata? [y/N]: ", false) {
			return fmt.Errorf("exiftool not found")
		}
		options.AllowFallback = true
	}
	knownHashes, err := photo.ExistingHashes(options.Destination)
	if err != nil {
		return err
	}
	options.KnownHashes = knownHashes
	knownVisualHashes, err := photo.ExistingVisualHashes(options.Destination)
	if err != nil {
		return err
	}
	options.KnownVisualHashes = knownVisualHashes

	plan, summary, err := photo.PlanIngestWithProgress(options, provider, newPhotoPlanProgressPrinter())
	if err != nil {
		return err
	}
	printPlanPreview(plan, summary)
	if !promptYesNo(reader, "Execute this plan? [y/N]: ", false) {
		return nil
	}

	finalSummary := photo.ExecutePlanWithProgress(plan, printPhotoProgress)
	finalSummary.Scanned = summary.Scanned
	reportPath, err := photo.WriteReport(plan, finalSummary)
	if err != nil {
		return err
	}
	if err := persistPhotoLibraryImport(options, plan, finalSummary); err != nil {
		return err
	}
	printFinalSummary(finalSummary, reportPath)
	return nil
}

func applyGuidedPathDefaults(cmd *cobra.Command, options *photo.Options) {
	if !cmd.Flags().Changed("structure") {
		options.Structure = guidedPathStructure
	}
	if !cmd.Flags().Changed("rename") {
		options.Rename = "grouped"
	}
	if !cmd.Flags().Changed("burst-window") {
		options.BurstWindow = 2 * time.Second
	}
	if !cmd.Flags().Changed("similarity-threshold") {
		options.SimilarityThreshold = 8
		options.SimilarityEnabled = true
	}
	if !cmd.Flags().Changed("duplicates") {
		options.Duplicates = photo.DuplicateSeparate
	}
}

func promptLine(reader *bufio.Reader, label string) string {
	fmt.Print(label)
	value, _ := reader.ReadString('\n')
	return strings.TrimSpace(value)
}

func promptRequired(reader *bufio.Reader, label string) string {
	for {
		value := promptLine(reader, label)
		if value != "" {
			return value
		}
		fmt.Println("Value required.")
	}
}

func promptYesNo(reader *bufio.Reader, label string, defaultValue bool) bool {
	value := strings.ToLower(promptLine(reader, label))
	if value == "" {
		return defaultValue
	}
	return value == "y" || value == "yes" || value == "s" || value == "sim"
}

func promptYesNoOption(reader *bufio.Reader, question string, defaultValue bool) bool {
	suffix := "[y/N]"
	if defaultValue {
		suffix = "[Y/n]"
	}
	return promptYesNo(reader, fmt.Sprintf("%s %s: ", question, suffix), defaultValue)
}

func printPlanPreview(plan photo.Plan, summary photo.Summary) {
	fmt.Printf("Found %d media files\n", summary.Media)
	fmt.Printf("Photos: %d, Raw: %d, Videos: %d\n", summary.Photos, summary.Raw, summary.Videos)
	fmt.Printf("Duplicates: %d, fallback metadata: %d\n", summary.Duplicates, summary.FallbackDates)
	fmt.Printf("Burst groups: %d, largest burst: %d\n", summary.BurstGroups, summary.LargestBurst)
	fmt.Printf("Similar groups: %d, visual similarity skipped: %d\n", summary.SimilarGroups, summary.VisualSimilaritySkipped)
	for i, action := range plan.Actions {
		if i >= 5 {
			break
		}
		if action.Kind == photo.ActionSkip {
			fmt.Printf("%s -> skipped\n", action.SourcePath)
			continue
		}
		fmt.Printf("%s -> %s\n", action.SourcePath, action.DestPath)
	}
}

func printFinalSummary(summary photo.Summary, reportPath string) {
	fmt.Printf("Copied: %d, moved: %d, skipped: %d, failed: %d\n", summary.Copied, summary.Moved, summary.Skipped, summary.Failed)
	if reportPath != "" {
		fmt.Printf("Report: %s\n", reportPath)
	}
}

func printPhotoProgress(done int, total int, action photo.PlannedAction) {
	fmt.Printf("Progress: %d/%d (%d%%) %s %s\n", done, total, progressPercent(done, total), action.Kind, action.SourcePath)
}

func newPhotoPlanProgressPrinter() photo.PlanProgressFunc {
	var mu sync.Mutex
	lastStage := ""
	lastPercent := -1
	return func(stage string, done int, total int, path string) {
		mu.Lock()
		defer mu.Unlock()

		if total <= 0 {
			if stage != lastStage {
				fmt.Printf("Planning: %s\n", stage)
				lastStage = stage
				lastPercent = -1
			}
			return
		}

		percent := progressPercent(done, total)
		if stage == lastStage && percent == lastPercent && done != total {
			return
		}
		if path == "" {
			fmt.Printf("Planning: %s %d/%d (%d%%)\n", stage, done, total, percent)
		} else {
			fmt.Printf("Planning: %s %d/%d (%d%%) %s\n", stage, done, total, percent, path)
		}
		lastStage = stage
		lastPercent = percent
	}
}

type duplicatePolicyValue photo.DuplicatePolicy

func (v *duplicatePolicyValue) String() string {
	return string(*v)
}

func (v *duplicatePolicyValue) Set(value string) error {
	policy := photo.DuplicatePolicy(value)
	if err := validateDuplicatePolicy(policy); err != nil {
		return err
	}
	*v = duplicatePolicyValue(policy)
	return nil
}

func (v *duplicatePolicyValue) Type() string {
	return "duplicate-policy"
}

func validateDuplicatePolicy(value photo.DuplicatePolicy) error {
	switch value {
	case photo.DuplicateSkip, photo.DuplicateSeparate, photo.DuplicateSuffix:
		return nil
	default:
		return fmt.Errorf("invalid duplicate policy %q", value)
	}
}

type reportFormatValue photo.ReportFormat

func (v *reportFormatValue) String() string {
	return string(*v)
}

func (v *reportFormatValue) Set(value string) error {
	format := photo.ReportFormat(value)
	switch format {
	case photo.ReportText, photo.ReportJSON, photo.ReportNone:
		*v = reportFormatValue(format)
		return nil
	default:
		return fmt.Errorf("invalid report format %q", value)
	}
}

func (v *reportFormatValue) Type() string {
	return "report-format"
}
