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
	Source              string
	Destination         string
	Recursive           bool
	Excludes            []string
	Move                bool
	Structure           string
	Rename              string
	Duplicates          DuplicatePolicy
	AllowFallback       bool
	Report              ReportFormat
	BurstWindow         time.Duration
	SimilarityEnabled   bool
	SimilarityThreshold int
	FullPerformance     bool
	KnownHashes         map[string]string `json:"-"`
	KnownVisualHashes   []KnownVisualHash `json:"-"`
}

type KnownVisualHash struct {
	Path string
	Hash uint64
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
	File              MediaFile
	Metadata          Metadata
	Hash              string
	VisualHash        uint64
	HasVisualHash     bool
	VisualHashSkipped bool
}

type GroupType string

const (
	GroupBurst   GroupType = "burst"
	GroupSimilar GroupType = "similar"
)

type FileGroup struct {
	ID    string
	Type  GroupType
	Files []string
}

type GroupingResult struct {
	BurstGroups             []FileGroup
	SimilarGroups           []FileGroup
	PreferredGroupByFile    map[string]string
	VisualSimilaritySkipped int
	BurstWindow             time.Duration
	SimilarityEnabled       bool
	SimilarityThreshold     int
}

type ActionKind string

const (
	ActionCopy ActionKind = "copy"
	ActionMove ActionKind = "move"
	ActionSkip ActionKind = "skip"
)

type PlannedAction struct {
	Kind          ActionKind
	SourcePath    string
	DestPath      string
	MediaType     MediaType
	Duplicate     bool
	UsedFallback  bool
	Error         string
	Hash          string
	VisualHash    uint64
	HasVisualHash bool
	Metadata      Metadata
	SourceSize    int64
	Extension     string
}

type Plan struct {
	Options  Options
	Actions  []PlannedAction
	Grouping GroupingResult
}

type Summary struct {
	Scanned                 int
	Media                   int
	Copied                  int
	Moved                   int
	Skipped                 int
	Duplicates              int
	Failed                  int
	Photos                  int
	Videos                  int
	Raw                     int
	FallbackDates           int
	BurstGroups             int
	LargestBurst            int
	SimilarGroups           int
	LargestSimilar          int
	VisualSimilaritySkipped int
}
