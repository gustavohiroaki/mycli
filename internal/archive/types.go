package archive

type Type string

const (
	TypeZip   Type = "zip"
	TypeTar   Type = "tar"
	TypeTarGz Type = "tar.gz"
)

type Options struct {
	Input     string
	Dest      string
	Recursive bool
	Keep      bool
	Overwrite bool
}

type Item struct {
	SourcePath string
	Type       Type
	BaseName   string
	DestDir    string
}

type ExtractedFile struct {
	Path     string
	Size     int64
	CRC32    uint32
	HasCRC32 bool
}

type ResultStatus string

const (
	StatusOK   ResultStatus = "OK"
	StatusKeep ResultStatus = "KEEP"
	StatusFail ResultStatus = "FAIL"
)

type Result struct {
	Item            Item
	Status          ResultStatus
	FilesExtracted  int
	SkippedLinks    int
	OriginalDeleted bool
	OriginalKept    bool
	Error           string
}

type Summary struct {
	ArchivesFound      int
	Extracted          int
	DeletedOriginals   int
	KeptOriginals      int
	Failed             int
	SkippedUnsupported int
}
