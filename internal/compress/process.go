package compress

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
)

type ProgressFunc func(done int, total int, result Result)

func ValidateOptions(options Options) error {
	if options.Level < 1 || options.Level > 100 {
		return fmt.Errorf("level must be between 1 and 100")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return fmt.Errorf("ffmpeg not found; install ffmpeg first")
	}
	if _, err := exec.LookPath("ffprobe"); err != nil {
		return fmt.Errorf("ffprobe not found; install ffmpeg first")
	}
	return nil
}

func ProcessBatch(items []Item, options Options, progress ProgressFunc) ([]Result, Summary) {
	results := make([]Result, 0, len(items))
	summary := Summary{Found: len(items)}
	for index, item := range items {
		result := Process(item, options)
		results = append(results, result)
		switch result.Status {
		case StatusOK, StatusReplace:
			summary.Compressed++
			if result.InputSize > result.OutputSize {
				summary.SavedBytes += result.InputSize - result.OutputSize
			}
		case StatusSkip:
			summary.Skipped++
		case StatusFail:
			summary.Failed++
		}
		if progress != nil {
			progress(index+1, len(items), result)
		}
	}
	return results, summary
}

func Process(item Item, options Options) Result {
	finalPath := item.DestPath
	if options.Replace {
		finalPath = replaceExt(item.SourcePath, ".mp4")
	}
	if !options.Overwrite {
		if _, err := os.Stat(finalPath); err == nil && finalPath != item.SourcePath {
			return Result{Item: item, Status: StatusSkip, InputSize: item.Size, Error: "destination exists"}
		}
	}
	if err := os.MkdirAll(filepath.Dir(finalPath), 0o755); err != nil {
		return Result{Item: item, Status: StatusFail, InputSize: item.Size, Error: err.Error()}
	}

	tempPath := finalPath + ".tmp.mp4"
	_ = os.Remove(tempPath)
	args := ffmpegArgs(item.SourcePath, tempPath, options.Level)
	if output, err := exec.Command("ffmpeg", args...).CombinedOutput(); err != nil {
		_ = os.Remove(tempPath)
		return Result{Item: item, Status: StatusFail, InputSize: item.Size, Error: string(output)}
	}
	if err := verifyVideo(tempPath); err != nil {
		_ = os.Remove(tempPath)
		return Result{Item: item, Status: StatusFail, InputSize: item.Size, Error: err.Error()}
	}
	info, err := os.Stat(tempPath)
	if err != nil {
		_ = os.Remove(tempPath)
		return Result{Item: item, Status: StatusFail, InputSize: item.Size, Error: err.Error()}
	}
	if info.Size() >= item.Size {
		_ = os.Remove(tempPath)
		return Result{Item: item, Status: StatusSkip, InputSize: item.Size, OutputSize: info.Size(), Error: "compressed file is not smaller"}
	}
	if options.Replace {
		if err := os.Remove(item.SourcePath); err != nil {
			_ = os.Remove(tempPath)
			return Result{Item: item, Status: StatusFail, InputSize: item.Size, OutputSize: info.Size(), Error: err.Error()}
		}
		if err := os.Rename(tempPath, finalPath); err != nil {
			_ = os.Remove(tempPath)
			return Result{Item: item, Status: StatusFail, InputSize: item.Size, OutputSize: info.Size(), Error: err.Error()}
		}
		item.DestPath = finalPath
		return Result{Item: item, Status: StatusReplace, InputSize: item.Size, OutputSize: info.Size()}
	}
	if err := os.Rename(tempPath, item.DestPath); err != nil {
		_ = os.Remove(tempPath)
		return Result{Item: item, Status: StatusFail, InputSize: item.Size, OutputSize: info.Size(), Error: err.Error()}
	}
	return Result{Item: item, Status: StatusOK, InputSize: item.Size, OutputSize: info.Size()}
}

func ffmpegArgs(input string, output string, level int) []string {
	crf := crfFromLevel(level)
	return []string{
		"-hide_banner",
		"-loglevel", "error",
		"-y",
		"-i", input,
		"-map", "0",
		"-c:v", "libx265",
		"-preset", "slow",
		"-crf", strconv.Itoa(crf),
		"-tag:v", "hvc1",
		"-c:a", "aac",
		"-b:a", "160k",
		"-c:s", "copy",
		"-movflags", "+faststart",
		output,
	}
}

func crfFromLevel(level int) int {
	return 18 + (level-1)*17/99
}

func verifyVideo(path string) error {
	return exec.Command("ffprobe", "-v", "error", "-select_streams", "v:0", "-show_entries", "stream=codec_type", "-of", "csv=p=0", path).Run()
}
