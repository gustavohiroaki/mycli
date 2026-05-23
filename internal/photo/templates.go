package photo

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	DefaultStructure = "{year}/{month}/{day}/{type}"
	DefaultRename    = "{date}_{time}_{camera}_{seq}{ext}"
)

var structurePresets = map[string]string{
	"date":        "{year}/{month}/{day}/{type}",
	"date-folder": "{year}/{year}-{month}-{day}/{type}",
	"camera-date": "{camera}/{year}/{month}/{day}/{type}",
	"year-camera": "{year}/{camera}/{year}-{month}-{day}/{type}",
	"legacy-pt":   "{year}/{month}/{day}/{type}",
}

var tokenPattern = regexp.MustCompile(`\{([a-z]+)\}`)
var unsafeSegmentChars = regexp.MustCompile(`[^a-z0-9._-]+`)

func ResolveStructure(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return DefaultStructure, nil
	}
	if preset, ok := structurePresets[value]; ok {
		return preset, nil
	}
	if err := ValidateTemplate(value); err != nil {
		return "", err
	}
	return value, nil
}

func ValidateTemplate(template string) error {
	tokens := tokenPattern.FindAllStringSubmatch(template, -1)
	for _, token := range tokens {
		if _, ok := allowedTokenSet()[token[1]]; !ok {
			return fmt.Errorf("unknown template token %q", token[1])
		}
	}
	return nil
}

func RenderTemplate(template string, file EnrichedFile, seq int) (string, error) {
	if err := ValidateTemplate(template); err != nil {
		return "", err
	}

	values := map[string]string{
		"year":      file.Metadata.Date.Format("2006"),
		"month":     file.Metadata.Date.Format("01"),
		"day":       file.Metadata.Date.Format("02"),
		"date":      file.Metadata.Date.Format("2006-01-02"),
		"time":      file.Metadata.Date.Format("15-04-05"),
		"camera":    sanitizeTokenValue(defaultString(file.Metadata.Camera, "unknown-camera")),
		"lens":      sanitizeTokenValue(defaultString(file.Metadata.Lens, "unknown-lens")),
		"type":      string(file.File.Type),
		"extension": strings.TrimPrefix(strings.ToLower(file.File.Extension), "."),
		"ext":       strings.ToLower(file.File.Extension),
		"seq":       fmt.Sprintf("%03d", seq),
	}

	return tokenPattern.ReplaceAllStringFunc(template, func(token string) string {
		key := strings.TrimSuffix(strings.TrimPrefix(token, "{"), "}")
		return values[key]
	}), nil
}

func allowedTokenSet() map[string]struct{} {
	return map[string]struct{}{
		"year":      {},
		"month":     {},
		"day":       {},
		"date":      {},
		"time":      {},
		"camera":    {},
		"lens":      {},
		"type":      {},
		"extension": {},
		"ext":       {},
		"seq":       {},
	}
}

func sanitizeTokenValue(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = unsafeSegmentChars.ReplaceAllString(value, "-")
	value = strings.Trim(value, "-")
	if value == "" {
		return "unknown"
	}
	return value
}

func defaultString(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
