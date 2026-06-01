package compress

type Options struct {
	Input           string
	Dest            string
	Level           int
	Recursive       bool
	Replace         bool
	Overwrite       bool
	Workers         int
	FullPerformance bool
	GPU             bool
}

type Item struct {
	SourcePath string
	DestPath   string
	RelPath    string
	Size       int64
}

type Result struct {
	Item       Item
	Status     string
	InputSize  int64
	OutputSize int64
	Error      string
}

type Summary struct {
	Found      int
	Compressed int
	Skipped    int
	Failed     int
	SavedBytes int64
}

const (
	StatusOK      = "OK"
	StatusSkip    = "SKIP"
	StatusFail    = "FAIL"
	StatusReplace = "REPLACE"
)
