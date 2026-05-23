package photo

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
)

func HashFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func MarkDuplicates(files []EnrichedFile) map[string]bool {
	seen := map[string]string{}
	duplicates := map[string]bool{}
	for _, file := range files {
		if file.Hash == "" {
			continue
		}
		if _, ok := seen[file.Hash]; ok {
			duplicates[file.File.SourcePath] = true
			continue
		}
		seen[file.Hash] = file.File.SourcePath
		duplicates[file.File.SourcePath] = false
	}
	return duplicates
}
