package cmd

import (
	"fmt"

	"mycli/internal/archive"

	"github.com/spf13/cobra"
)

var unpackOptions archive.Options

var unpackCmd = &cobra.Command{
	Use:   "unpack <file-or-directory>",
	Short: "Extract archives and delete originals after verification",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		options := unpackOptions
		options.Input = args[0]

		items, skipped, err := archive.Discover(options.Input, options)
		if err != nil {
			return err
		}
		results, summary := archive.ProcessBatch(items, options, skipped)
		for _, result := range results {
			printUnpackResult(result)
		}
		printUnpackSummary(summary)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(unpackCmd)
	unpackCmd.Flags().StringVar(&unpackOptions.Dest, "dest", "", "Extraction root directory")
	unpackCmd.Flags().BoolVar(&unpackOptions.Recursive, "recursive", false, "Process archives in subdirectories")
	unpackCmd.Flags().BoolVar(&unpackOptions.Keep, "keep", false, "Keep original archives after successful verification")
	unpackCmd.Flags().BoolVar(&unpackOptions.Overwrite, "overwrite", false, "Allow overwriting files inside extraction directories")
}

func printUnpackResult(result archive.Result) {
	switch result.Status {
	case archive.StatusOK:
		fmt.Printf("OK    %s -> %s (%d files, original deleted)\n", result.Item.SourcePath, result.Item.DestDir, result.FilesExtracted)
	case archive.StatusKeep:
		fmt.Printf("KEEP  %s -> %s (%d files, original kept)\n", result.Item.SourcePath, result.Item.DestDir, result.FilesExtracted)
	case archive.StatusFail:
		fmt.Printf("FAIL  %s -> %s\n", result.Item.SourcePath, result.Error)
	}
}

func printUnpackSummary(summary archive.Summary) {
	fmt.Printf("Archives found: %d\n", summary.ArchivesFound)
	fmt.Printf("Extracted: %d\n", summary.Extracted)
	fmt.Printf("Deleted originals: %d\n", summary.DeletedOriginals)
	fmt.Printf("Kept originals: %d\n", summary.KeptOriginals)
	fmt.Printf("Failed: %d\n", summary.Failed)
	fmt.Printf("Skipped unsupported: %d\n", summary.SkippedUnsupported)
}
