package compress

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

type ProgressFunc func(done int, total int, result Result)
type EncodeProgressFunc func(item Item, percent int)

func ValidateOptions(options Options) error {
	if options.Level < 1 || options.Level > 100 {
		return fmt.Errorf("level must be between 1 and 100")
	}
	if options.Workers < 0 {
		return fmt.Errorf("workers cannot be negative")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return fmt.Errorf("ffmpeg not found; install ffmpeg first")
	}
	if _, err := exec.LookPath("ffprobe"); err != nil {
		return fmt.Errorf("ffprobe not found; install ffmpeg first")
	}
	if options.GPU && !gpuEncoderAvailable() {
		return fmt.Errorf("hevc_amf encoder not available; install an ffmpeg build with AMF support (https://ffmpeg.org/download.html)")
	}
	return nil
}

func ProcessBatch(items []Item, options Options, progress ProgressFunc) ([]Result, Summary) {
	return ProcessBatchWithEncodeProgress(items, options, progress, nil)
}

func ProcessBatchWithEncodeProgress(items []Item, options Options, progress ProgressFunc, encodeProgress EncodeProgressFunc) ([]Result, Summary) {
	workers := workerCount(options, len(items))
	if workers > 1 {
		return processBatchParallel(items, options, progress, encodeProgress, workers)
	}
	return processBatchSequential(items, options, progress, encodeProgress)
}

func processBatchSequential(items []Item, options Options, progress ProgressFunc, encodeProgress EncodeProgressFunc) ([]Result, Summary) {
	results := make([]Result, 0, len(items))
	summary := Summary{Found: len(items)}
	for index, item := range items {
		result := ProcessWithProgress(item, options, encodeProgress)
		results = append(results, result)
		countResult(&summary, result)
		if progress != nil {
			progress(index+1, len(items), result)
		}
	}
	return results, summary
}

func processBatchParallel(items []Item, options Options, progress ProgressFunc, encodeProgress EncodeProgressFunc, workers int) ([]Result, Summary) {
	type jobResult struct {
		index  int
		result Result
	}

	jobs := make(chan int)
	resultCh := make(chan jobResult, len(items))
	var wg sync.WaitGroup
	var encodeMu sync.Mutex
	for worker := 0; worker < workers; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for index := range jobs {
				item := items[index]
				result := ProcessWithProgress(item, options, func(item Item, percent int) {
					if encodeProgress == nil {
						return
					}
					encodeMu.Lock()
					encodeProgress(item, percent)
					encodeMu.Unlock()
				})
				resultCh <- jobResult{index: index, result: result}
			}
		}()
	}
	for index := range items {
		jobs <- index
	}
	close(jobs)
	wg.Wait()
	close(resultCh)

	results := make([]Result, len(items))
	summary := Summary{Found: len(items)}
	completed := 0
	for item := range resultCh {
		completed++
		results[item.index] = item.result
		countResult(&summary, item.result)
		if progress != nil {
			progress(completed, len(items), item.result)
		}
	}
	return results, summary
}

func countResult(summary *Summary, result Result) {
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
}

func Process(item Item, options Options) Result {
	return ProcessWithProgress(item, options, nil)
}

func ProcessWithProgress(item Item, options Options, encodeProgress EncodeProgressFunc) Result {
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
	args := ffmpegArgs(item.SourcePath, tempPath, options)
	durationMS, _ := videoDurationMS(item.SourcePath)
	if output, err := runFFmpegWithProgress(item, args, durationMS, encodeProgress); err != nil {
		_ = os.Remove(tempPath)
		return Result{Item: item, Status: StatusFail, InputSize: item.Size, Error: output}
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

func ffmpegArgs(input string, output string, options Options) []string {
	if options.GPU {
		return ffmpegArgsAMF(input, output, options.Level)
	}
	return ffmpegArgsCPU(input, output, options.Level)
}

func ffmpegArgsCPU(input string, output string, level int) []string {
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
		"-progress", "pipe:1",
		"-nostats",
		output,
	}
}

func ffmpegArgsAMF(input string, output string, level int) []string {
	qp := crfFromLevel(level)
	return []string{
		"-hide_banner",
		"-loglevel", "error",
		"-y",
		"-i", input,
		"-map", "0",
		"-c:v", "hevc_amf",
		"-rc", "cqp",
		"-qp_i", strconv.Itoa(qp),
		"-qp_p", strconv.Itoa(qp),
		"-quality", "quality",
		"-tag:v", "hvc1",
		"-c:a", "aac",
		"-b:a", "160k",
		"-c:s", "copy",
		"-movflags", "+faststart",
		"-progress", "pipe:1",
		"-nostats",
		output,
	}
}

func gpuEncoderAvailable() bool {
	out, err := exec.Command("ffmpeg", "-hide_banner", "-encoders").Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), "hevc_amf")
}

func workerCount(options Options, itemCount int) int {
	if itemCount < 2 {
		return 1
	}
	if options.Workers > 0 {
		if options.Workers > itemCount {
			return itemCount
		}
		return options.Workers
	}
	if options.FullPerformance {
		workers := runtime.NumCPU() / 2
		if workers < 2 {
			workers = 2
		}
		if workers > itemCount {
			return itemCount
		}
		return workers
	}
	return 1
}

func crfFromLevel(level int) int {
	return 18 + (level-1)*17/99
}

func verifyVideo(path string) error {
	return exec.Command("ffprobe", "-v", "error", "-select_streams", "v:0", "-show_entries", "stream=codec_type", "-of", "csv=p=0", path).Run()
}

func videoDurationMS(path string) (int64, error) {
	output, err := exec.Command("ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", path).Output()
	if err != nil {
		return 0, err
	}
	value := strings.TrimSpace(string(output))
	if value == "" || value == "N/A" {
		return 0, fmt.Errorf("duration not available")
	}
	seconds, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, err
	}
	return int64(seconds * 1000), nil
}

func runFFmpegWithProgress(item Item, args []string, durationMS int64, progress EncodeProgressFunc) (string, error) {
	cmd := exec.Command("ffmpeg", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		return stderr.String(), err
	}

	lastPercent := -1
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "out_time_ms=") || durationMS <= 0 || progress == nil {
			continue
		}
		outTimeUS, err := strconv.ParseInt(strings.TrimPrefix(line, "out_time_ms="), 10, 64)
		if err != nil {
			continue
		}
		percent := int((outTimeUS / 1000) * 100 / durationMS)
		if percent < 0 {
			percent = 0
		}
		if percent > 100 {
			percent = 100
		}
		if percent != lastPercent {
			progress(item, percent)
			lastPercent = percent
		}
	}
	if scanErr := scanner.Err(); scanErr != nil {
		_ = cmd.Wait()
		return stderr.String(), scanErr
	}
	if err := cmd.Wait(); err != nil {
		return stderr.String(), err
	}
	if progress != nil && lastPercent < 100 {
		progress(item, 100)
	}
	return stderr.String(), nil
}
