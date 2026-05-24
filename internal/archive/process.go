package archive

import (
	"fmt"
	"hash/crc32"
	"io"
	"os"
)

type BatchProgressFunc func(done int, total int, result Result)

func Process(item Item, options Options) Result {
	files, skipped, err := Extract(item, options)
	if err != nil {
		return Result{Item: item, Status: StatusFail, SkippedLinks: skipped, OriginalKept: true, Error: err.Error()}
	}
	if err := Verify(files); err != nil {
		return Result{Item: item, Status: StatusFail, FilesExtracted: len(files), SkippedLinks: skipped, OriginalKept: true, Error: err.Error()}
	}
	if options.Keep {
		return Result{Item: item, Status: StatusKeep, FilesExtracted: len(files), SkippedLinks: skipped, OriginalKept: true}
	}
	if err := os.Remove(item.SourcePath); err != nil {
		return Result{Item: item, Status: StatusFail, FilesExtracted: len(files), SkippedLinks: skipped, OriginalKept: true, Error: fmt.Sprintf("delete verified archive: %v", err)}
	}
	return Result{Item: item, Status: StatusOK, FilesExtracted: len(files), SkippedLinks: skipped, OriginalDeleted: true}
}

func Verify(files []ExtractedFile) error {
	for _, file := range files {
		info, err := os.Stat(file.Path)
		if err != nil {
			return err
		}
		if info.Size() != file.Size {
			return fmt.Errorf("size mismatch: %s", file.Path)
		}
		if file.HasCRC32 {
			hash, err := fileCRC32(file.Path)
			if err != nil {
				return err
			}
			if hash != file.CRC32 {
				return fmt.Errorf("checksum mismatch: %s", file.Path)
			}
		}
	}
	return nil
}

func ProcessBatch(items []Item, options Options, skippedUnsupported int) ([]Result, Summary) {
	return ProcessBatchWithProgress(items, options, skippedUnsupported, nil)
}

func ProcessBatchWithProgress(items []Item, options Options, skippedUnsupported int, progress BatchProgressFunc) ([]Result, Summary) {
	summary := Summary{ArchivesFound: len(items), SkippedUnsupported: skippedUnsupported}
	results := make([]Result, 0, len(items))
	total := len(items)
	for index, item := range items {
		result := Process(item, options)
		results = append(results, result)
		switch result.Status {
		case StatusOK:
			summary.Extracted++
			summary.DeletedOriginals++
		case StatusKeep:
			summary.Extracted++
			summary.KeptOriginals++
		case StatusFail:
			summary.Failed++
			if result.OriginalKept {
				summary.KeptOriginals++
			}
		}
		if progress != nil {
			progress(index+1, total, result)
		}
	}
	return results, summary
}

func fileCRC32(path string) (uint32, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	hash := crc32.NewIEEE()
	if _, err := io.Copy(hash, file); err != nil {
		return 0, err
	}
	return hash.Sum32(), nil
}
