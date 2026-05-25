package photo

import (
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"math/bits"
	"os"
	"sort"
	"strings"
	"sync"
)

func EnrichVisualHashes(files []EnrichedFile) ([]EnrichedFile, int) {
	return EnrichVisualHashesWithProgress(files, nil)
}

func EnrichVisualHashesWithProgress(files []EnrichedFile, progress func(done int, total int, path string)) ([]EnrichedFile, int) {
	return EnrichVisualHashesWithOptions(files, 1, progress)
}

func EnrichVisualHashesWithOptions(files []EnrichedFile, workers int, progress func(done int, total int, path string)) ([]EnrichedFile, int) {
	if workers <= 1 || len(files) < 2 {
		return enrichVisualHashesSequential(files, progress)
	}

	enriched := make([]EnrichedFile, len(files))
	copy(enriched, files)
	skipped := 0
	jobs := make(chan int)
	var wg sync.WaitGroup
	var mu sync.Mutex
	completed := 0

	for worker := 0; worker < workers; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for index := range jobs {
				file := enriched[index]
				skippedFile := applyVisualHash(&enriched[index])
				mu.Lock()
				if skippedFile {
					skipped++
				}
				completed++
				reportVisualHashProgress(progress, completed, len(enriched), file.File.SourcePath)
				mu.Unlock()
			}
		}()
	}
	for index := range enriched {
		jobs <- index
	}
	close(jobs)
	wg.Wait()

	return enriched, skipped
}

func enrichVisualHashesSequential(files []EnrichedFile, progress func(done int, total int, path string)) ([]EnrichedFile, int) {
	enriched := make([]EnrichedFile, len(files))
	copy(enriched, files)
	skipped := 0

	for i := range enriched {
		if applyVisualHash(&enriched[i]) {
			skipped++
		}
		reportVisualHashProgress(progress, i+1, len(enriched), enriched[i].File.SourcePath)
	}

	return enriched, skipped
}

func applyVisualHash(file *EnrichedFile) bool {
	if file.File.Type != MediaTypePhoto || !isVisualHashExtension(file.File.Extension) {
		return false
	}
	hash, ok, err := VisualHashFile(file.File.SourcePath)
	if err != nil || !ok {
		file.VisualHashSkipped = true
		return true
	}
	file.VisualHash = hash
	file.HasVisualHash = true
	return false
}

func reportVisualHashProgress(progress func(done int, total int, path string), done int, total int, path string) {
	if progress != nil {
		progress(done, total, path)
	}
}

func VisualHashFile(path string) (uint64, bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, false, err
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return 0, false, err
	}
	return averageHash(img), true, nil
}

func DetectSimilarGroups(files []EnrichedFile, threshold int) []FileGroup {
	if threshold < 0 {
		return nil
	}

	candidates := make([]EnrichedFile, 0, len(files))
	for _, file := range files {
		if file.File.Type == MediaTypePhoto && file.HasVisualHash {
			candidates = append(candidates, file)
		}
	}
	sortEnriched(candidates)
	if len(candidates) < 2 {
		return nil
	}

	parent := make([]int, len(candidates))
	for i := range parent {
		parent[i] = i
	}
	for i := 0; i < len(candidates); i++ {
		for j := i + 1; j < len(candidates); j++ {
			if HammingDistance(candidates[i].VisualHash, candidates[j].VisualHash) <= threshold {
				union(parent, i, j)
			}
		}
	}

	sets := map[int][]EnrichedFile{}
	for i, file := range candidates {
		root := find(parent, i)
		sets[root] = append(sets[root], file)
	}

	var groups []FileGroup
	for _, groupFiles := range sets {
		if len(groupFiles) < 2 {
			continue
		}
		sortEnriched(groupFiles)
		groups = append(groups, makeGroup(GroupSimilar, len(groups)+1, groupFiles))
	}
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].Files[0] < groups[j].Files[0]
	})
	for i := range groups {
		groups[i].ID = "similar-" + shortSequence(i+1)
	}
	return groups
}

func DetectSimilarGroupsAgainstKnown(files []EnrichedFile, known []KnownVisualHash, threshold int) []FileGroup {
	if threshold < 0 || len(known) == 0 {
		return nil
	}

	sort.Slice(known, func(i, j int) bool {
		return known[i].Path < known[j].Path
	})
	groupsByKnownPath := map[string][]EnrichedFile{}
	for _, file := range files {
		if file.File.Type != MediaTypePhoto || !file.HasVisualHash {
			continue
		}
		for _, existing := range known {
			if HammingDistance(file.VisualHash, existing.Hash) <= threshold {
				groupsByKnownPath[existing.Path] = append(groupsByKnownPath[existing.Path], file)
				break
			}
		}
	}

	keys := make([]string, 0, len(groupsByKnownPath))
	for key := range groupsByKnownPath {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	groups := make([]FileGroup, 0, len(keys))
	for _, key := range keys {
		groupFiles := groupsByKnownPath[key]
		sortEnriched(groupFiles)
		groups = append(groups, makeGroup(GroupSimilar, len(groups)+1, groupFiles))
	}
	return groups
}

func HammingDistance(left uint64, right uint64) int {
	return bits.OnesCount64(left ^ right)
}

func averageHash(img image.Image) uint64 {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	var values [64]uint32
	var total uint32

	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			sampleX := bounds.Min.X + (x*width+width/2)/8
			sampleY := bounds.Min.Y + (y*height+height/2)/8
			r, g, b, _ := img.At(sampleX, sampleY).RGBA()
			brightness := uint32((299*(r>>8) + 587*(g>>8) + 114*(b>>8)) / 1000)
			index := y*8 + x
			values[index] = brightness
			total += brightness
		}
	}

	average := total / 64
	var hash uint64
	for i, value := range values {
		if value >= average {
			hash |= 1 << uint(63-i)
		}
	}
	return hash
}

func isVisualHashExtension(extension string) bool {
	switch strings.ToLower(extension) {
	case ".jpg", ".jpeg", ".png":
		return true
	default:
		return false
	}
}

func find(parent []int, value int) int {
	for parent[value] != value {
		parent[value] = parent[parent[value]]
		value = parent[value]
	}
	return value
}

func union(parent []int, left int, right int) {
	leftRoot := find(parent, left)
	rightRoot := find(parent, right)
	if leftRoot != rightRoot {
		parent[rightRoot] = leftRoot
	}
}

func shortSequence(value int) string {
	return fmt.Sprintf("%03d", value)
}
