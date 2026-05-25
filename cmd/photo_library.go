package cmd

import (
	"fmt"
	"path/filepath"
	"time"

	"mycli/internal/photo"

	"github.com/spf13/cobra"
)

var (
	photoLibraryName        string
	photoLibraryMakeDefault = true
	photoImportLibrary      string
)

var photoLibraryCmd = &cobra.Command{
	Use:   "library",
	Short: "Manage photo libraries",
}

var photoLibraryInitCmd = &cobra.Command{
	Use:   "init <destination>",
	Short: "Initialize a photo library and save it globally",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		destination, err := filepath.Abs(args[0])
		if err != nil {
			return err
		}
		name := photoLibraryName
		if name == "" {
			name = filepath.Base(destination)
		}

		options := defaultPhotoLibraryOptions(destination)
		config := photo.ConfigFromOptions(options)
		if err := photo.SaveLibraryConfig(destination, config); err != nil {
			return err
		}
		if err := photo.InitLocalLibrary(destination); err != nil {
			return err
		}
		dbPath, err := photo.DefaultGlobalDBPath()
		if err != nil {
			return err
		}
		if err := photo.SaveGlobalLibrary(dbPath, photo.Library{Name: name, Path: destination, IsDefault: photoLibraryMakeDefault}); err != nil {
			return err
		}
		fmt.Printf("Photo library initialized: %s -> %s\n", name, destination)
		if photoLibraryMakeDefault {
			fmt.Println("Default photo library updated.")
		}
		return nil
	},
}

var photoLibraryListCmd = &cobra.Command{
	Use:   "list",
	Short: "List saved photo libraries",
	RunE: func(cmd *cobra.Command, args []string) error {
		dbPath, err := photo.DefaultGlobalDBPath()
		if err != nil {
			return err
		}
		libraries, err := photo.ListGlobalLibraries(dbPath)
		if err != nil {
			return err
		}
		if len(libraries) == 0 {
			fmt.Println("No photo libraries configured.")
			return nil
		}
		for _, library := range libraries {
			prefix := " "
			if library.IsDefault {
				prefix = "*"
			}
			fmt.Printf("%s %s  %s\n", prefix, library.Name, library.Path)
		}
		return nil
	},
}

var photoLibraryUseCmd = &cobra.Command{
	Use:   "use <name>",
	Short: "Set the default photo library",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dbPath, err := photo.DefaultGlobalDBPath()
		if err != nil {
			return err
		}
		if err := photo.SetDefaultLibrary(dbPath, args[0]); err != nil {
			return err
		}
		fmt.Printf("Default photo library: %s\n", args[0])
		return nil
	},
}

var photoImportCmd = &cobra.Command{
	Use:   "import <source>",
	Short: "Import media into the default photo library",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPhotoImport(args[0], photoImportLibrary)
	},
}

func init() {
	photoCmd.AddCommand(photoLibraryCmd)
	photoCmd.AddCommand(photoImportCmd)
	photoLibraryCmd.AddCommand(photoLibraryInitCmd, photoLibraryListCmd, photoLibraryUseCmd)

	photoLibraryInitCmd.Flags().StringVar(&photoLibraryName, "name", "", "Library name")
	photoLibraryInitCmd.Flags().BoolVar(&photoLibraryMakeDefault, "default", true, "Set as default photo library")
	photoImportCmd.Flags().StringVar(&photoImportLibrary, "library", "", "Library name or path; defaults to the global default library")
}

func runPhotoImport(source string, libraryRef string) error {
	dbPath, err := photo.DefaultGlobalDBPath()
	if err != nil {
		return err
	}
	library, err := photo.FindGlobalLibrary(dbPath, libraryRef)
	if err != nil {
		return err
	}
	config, err := photo.LoadLibraryConfig(library.Path)
	if err != nil {
		return err
	}
	options, err := config.ToOptions(source)
	if err != nil {
		return err
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

	provider := photo.ExiftoolProvider{}
	if !options.AllowFallback && !provider.Available() {
		return fmt.Errorf("exiftool not found; install exiftool or enable allowFallback in the library config")
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
	if err := photo.RecordImport(options.Destination, source, config, plan, finalSummary); err != nil {
		return err
	}
	printFinalSummary(finalSummary, reportPath)
	fmt.Printf("Library: %s (%s)\n", library.Name, library.Path)
	return nil
}

func defaultPhotoLibraryOptions(destination string) photo.Options {
	return photo.Options{
		Destination:         destination,
		Recursive:           true,
		Structure:           guidedPathStructure,
		Rename:              "grouped",
		Duplicates:          photo.DuplicateSeparate,
		Report:              photo.ReportText,
		BurstWindow:         2 * time.Second,
		SimilarityEnabled:   true,
		SimilarityThreshold: 8,
		FullPerformance:     true,
	}
}

func persistPhotoLibraryImport(options photo.Options, plan photo.Plan, summary photo.Summary) error {
	config := photo.ConfigFromOptions(options)
	if err := photo.SaveLibraryConfig(options.Destination, config); err != nil {
		return err
	}
	if err := photo.RecordImport(options.Destination, options.Source, config, plan, summary); err != nil {
		return err
	}
	dbPath, err := photo.DefaultGlobalDBPath()
	if err != nil {
		return err
	}
	name := filepath.Base(options.Destination)
	return photo.SaveGlobalLibrary(dbPath, photo.Library{Name: name, Path: options.Destination, IsDefault: true})
}
