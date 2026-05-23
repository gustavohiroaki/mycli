package photo

import (
	"errors"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type MetadataProvider interface {
	Available() bool
	Read(path string) (Metadata, error)
}

type ExiftoolProvider struct{}

func (ExiftoolProvider) Available() bool {
	_, err := exec.LookPath("exiftool")
	return err == nil
}

func (provider ExiftoolProvider) Read(path string) (Metadata, error) {
	if !provider.Available() {
		return Metadata{}, errors.New("exiftool not found")
	}

	output, err := exec.Command(
		"exiftool",
		"-s3",
		"-d", "%Y-%m-%d %H:%M:%S",
		"-DateTimeOriginal",
		"-CreateDate",
		"-MediaCreateDate",
		"-TrackCreateDate",
		"-Model",
		"-LensModel",
		path,
	).Output()
	if err != nil {
		return Metadata{}, err
	}

	var metadata Metadata
	for _, line := range strings.Split(string(output), "\n") {
		value := strings.TrimSpace(line)
		if value == "" {
			continue
		}
		if metadata.Date.IsZero() {
			if parsed, err := time.ParseInLocation("2006-01-02 15:04:05", value, time.Local); err == nil {
				metadata.Date = parsed
				continue
			}
		}
		if metadata.Camera == "" {
			metadata.Camera = value
			continue
		}
		if metadata.Lens == "" {
			metadata.Lens = value
		}
	}
	if metadata.Date.IsZero() {
		return Metadata{}, errors.New("metadata date not found")
	}
	metadata.Camera = defaultMetadataValue(metadata.Camera, "unknown-camera")
	metadata.Lens = defaultMetadataValue(metadata.Lens, "unknown-lens")
	return metadata, nil
}

var filenameDatePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(20\d{2})[-_]?([01]\d)[-_]?([0-3]\d)[T _-]?([0-2]\d)?([0-5]\d)?([0-5]\d)?`),
	regexp.MustCompile(`(?i)(\d{2})[-_]?([01]\d)[-_]?(20\d{2})`),
}

func ResolveMetadata(file MediaFile, provider MetadataProvider) Metadata {
	if provider != nil && provider.Available() {
		if metadata, err := provider.Read(file.SourcePath); err == nil {
			metadata.UsedFallback = false
			metadata.Camera = defaultMetadataValue(metadata.Camera, "unknown-camera")
			metadata.Lens = defaultMetadataValue(metadata.Lens, "unknown-lens")
			return metadata
		}
	}

	metadata := Metadata{
		Date:         file.ModTime,
		Camera:       "unknown-camera",
		Lens:         "unknown-lens",
		UsedFallback: true,
	}
	if date, ok := DateFromFilename(filepath.Base(file.SourcePath)); ok {
		metadata.Date = date
	}
	return metadata
}

func DateFromFilename(name string) (time.Time, bool) {
	for index, pattern := range filenameDatePatterns {
		matches := pattern.FindStringSubmatch(name)
		if len(matches) == 0 {
			continue
		}

		var year, month, day int
		var hour, minute, second int
		var err error

		if index == 0 {
			year, err = strconv.Atoi(matches[1])
			if err != nil {
				continue
			}
			month, err = strconv.Atoi(matches[2])
			if err != nil {
				continue
			}
			day, err = strconv.Atoi(matches[3])
			if err != nil {
				continue
			}
			hour = atoiDefault(matches[4])
			minute = atoiDefault(matches[5])
			second = atoiDefault(matches[6])
		} else {
			day, err = strconv.Atoi(matches[1])
			if err != nil {
				continue
			}
			month, err = strconv.Atoi(matches[2])
			if err != nil {
				continue
			}
			year, err = strconv.Atoi(matches[3])
			if err != nil {
				continue
			}
		}

		mediaDate := time.Date(year, time.Month(month), day, hour, minute, second, 0, time.Local)
		if mediaDate.Year() == year && int(mediaDate.Month()) == month && mediaDate.Day() == day {
			return mediaDate, true
		}
	}
	return time.Time{}, false
}

func atoiDefault(value string) int {
	if value == "" {
		return 0
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return parsed
}

func defaultMetadataValue(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
