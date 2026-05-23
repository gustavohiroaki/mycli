package photo

import "time"

type MediaType string

const (
	MediaTypePhoto MediaType = "photos"
	MediaTypeVideo MediaType = "videos"
	MediaTypeRaw   MediaType = "raw"
)

type DuplicatePolicy string

const (
	DuplicateSkip     DuplicatePolicy = "skip"
	DuplicateSeparate DuplicatePolicy = "separate"
	DuplicateSuffix   DuplicatePolicy = "suffix"
)

type ReportFormat string

const (
	ReportText ReportFormat = "txt"
	ReportJSON ReportFormat = "json"
	ReportNone ReportFormat = "none"
)

type Options struct {
	Source        string
	Destination   string
	Recursive     bool
	Excludes      []string
	Move          bool
	Structure     string
	Rename        string
	Duplicates    DuplicatePolicy
	AllowFallback bool
	Report        ReportFormat
}

type MediaFile struct {
	SourcePath string
	RelPath    string
	Type       MediaType
	Extension  string
	Size       int64
	ModTime    time.Time
}

type Metadata struct {
	Date         time.Time
	Camera       string
	Lens         string
	UsedFallback bool
}

type EnrichedFile struct {
	File     MediaFile
	Metadata Metadata
	Hash     string
}

type ActionKind string

const (
	ActionCopy ActionKind = "copy"
	ActionMove ActionKind = "move"
	ActionSkip ActionKind = "skip"
)

type PlannedAction struct {
	Kind         ActionKind
	SourcePath   string
	DestPath     string
	MediaType    MediaType
	Duplicate    bool
	UsedFallback bool
	Error        string
}

type Plan struct {
	Options Options
	Actions []PlannedAction
}

type Summary struct {
	Scanned       int
	Media         int
	Copied        int
	Moved         int
	Skipped       int
	Duplicates    int
	Failed        int
	Photos        int
	Videos        int
	Raw           int
	FallbackDates int
}
