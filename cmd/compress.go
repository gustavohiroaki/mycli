package cmd

import (
	"fmt"

	"mycli/internal/compress"

	"github.com/spf13/cobra"
)

var compressOptions = compress.Options{Level: 35}

var compressCmd = &cobra.Command{
	Use:   "compress <video-or-directory>",
	Short: "Compress videos with quality-focused settings",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		options := compressOptions
		options.Input = args[0]
		if err := compress.ValidateOptions(options); err != nil {
			return err
		}
		items, skipped, err := compress.Discover(options.Input, options)
		if err != nil {
			return err
		}
		results, summary := compress.ProcessBatch(items, options, printCompressProgress)
		for _, result := range results {
			printCompressResult(result)
		}
		printCompressSummary(summary, skipped)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(compressCmd)
	compressCmd.Flags().StringVar(&compressOptions.Dest, "dest", "", "Output file or directory")
	compressCmd.Flags().IntVar(&compressOptions.Level, "level", 35, "Compression level from 1 to 100")
	compressCmd.Flags().BoolVar(&compressOptions.Recursive, "recursive", false, "Process videos in subdirectories")
	compressCmd.Flags().BoolVar(&compressOptions.Replace, "replace", false, "Replace original only when compressed output is valid and smaller")
	compressCmd.Flags().BoolVar(&compressOptions.Overwrite, "overwrite", false, "Overwrite existing output files")
}

func printCompressProgress(done int, total int, result compress.Result) {
	fmt.Printf("Progress: %d/%d (%d%%) %s %s\n", done, total, progressPercent(done, total), result.Status, result.Item.SourcePath)
}

func printCompressResult(result compress.Result) {
	switch result.Status {
	case compress.StatusOK:
		fmt.Printf("OK      %s -> %s (%s -> %s)\n", result.Item.SourcePath, result.Item.DestPath, formatBytes(result.InputSize), formatBytes(result.OutputSize))
	case compress.StatusReplace:
		fmt.Printf("REPLACE %s (%s -> %s)\n", result.Item.SourcePath, formatBytes(result.InputSize), formatBytes(result.OutputSize))
	case compress.StatusSkip:
		fmt.Printf("SKIP    %s (%s)\n", result.Item.SourcePath, result.Error)
	case compress.StatusFail:
		fmt.Printf("FAIL    %s (%s)\n", result.Item.SourcePath, firstLine(result.Error))
	}
}

func printCompressSummary(summary compress.Summary, skippedUnsupported int) {
	fmt.Printf("Videos found: %d\n", summary.Found)
	fmt.Printf("Compressed: %d\n", summary.Compressed)
	fmt.Printf("Skipped: %d\n", summary.Skipped)
	fmt.Printf("Failed: %d\n", summary.Failed)
	fmt.Printf("Skipped unsupported: %d\n", skippedUnsupported)
	fmt.Printf("Saved: %s\n", formatBytes(summary.SavedBytes))
}

func formatBytes(value int64) string {
	const unit = 1024
	if value < unit {
		return fmt.Sprintf("%d B", value)
	}
	div := int64(unit)
	exp := 0
	for n := value / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(value)/float64(div), "KMGTPE"[exp])
}

func firstLine(value string) string {
	for index, char := range value {
		if char == '\n' || char == '\r' {
			return value[:index]
		}
	}
	return value
}
