package photo

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

const libraryDirName = ".mycli-photo"

type Library struct {
	Name       string
	Path       string
	IsDefault  bool
	CreatedAt  time.Time
	LastUsedAt time.Time
}

type LibraryConfig struct {
	Destination         string          `json:"destination"`
	Recursive           bool            `json:"recursive"`
	Excludes            []string        `json:"excludes,omitempty"`
	Move                bool            `json:"move"`
	Structure           string          `json:"structure"`
	Rename              string          `json:"rename"`
	Duplicates          DuplicatePolicy `json:"duplicates"`
	AllowFallback       bool            `json:"allowFallback"`
	Report              ReportFormat    `json:"report"`
	BurstWindow         string          `json:"burstWindow"`
	SimilarityEnabled   bool            `json:"similarityEnabled"`
	SimilarityThreshold int             `json:"similarityThreshold"`
	FullPerformance     bool            `json:"fullPerformance"`
}

func ConfigFromOptions(options Options) LibraryConfig {
	return LibraryConfig{
		Destination:         options.Destination,
		Recursive:           options.Recursive,
		Excludes:            options.Excludes,
		Move:                options.Move,
		Structure:           options.Structure,
		Rename:              options.Rename,
		Duplicates:          options.Duplicates,
		AllowFallback:       options.AllowFallback,
		Report:              options.Report,
		BurstWindow:         options.BurstWindow.String(),
		SimilarityEnabled:   options.SimilarityEnabled,
		SimilarityThreshold: options.SimilarityThreshold,
		FullPerformance:     options.FullPerformance,
	}
}

func (config LibraryConfig) ToOptions(source string) (Options, error) {
	burstWindow := time.Duration(0)
	if config.BurstWindow != "" {
		parsed, err := time.ParseDuration(config.BurstWindow)
		if err != nil {
			return Options{}, err
		}
		burstWindow = parsed
	}
	return Options{
		Source:              source,
		Destination:         config.Destination,
		Recursive:           config.Recursive,
		Excludes:            append([]string(nil), config.Excludes...),
		Move:                config.Move,
		Structure:           config.Structure,
		Rename:              config.Rename,
		Duplicates:          config.Duplicates,
		AllowFallback:       config.AllowFallback,
		Report:              config.Report,
		BurstWindow:         burstWindow,
		SimilarityEnabled:   config.SimilarityEnabled,
		SimilarityThreshold: config.SimilarityThreshold,
		FullPerformance:     config.FullPerformance,
	}, nil
}

func DefaultGlobalDBPath() (string, error) {
	if sudoUser := os.Getenv("SUDO_USER"); sudoUser != "" && sudoUser != "root" {
		account, err := user.Lookup(sudoUser)
		if err == nil && account.HomeDir != "" {
			return filepath.Join(account.HomeDir, ".config", "mycli", "mycli.db"), nil
		}
	}
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "mycli", "mycli.db"), nil
}

func SaveLibraryConfig(destination string, config LibraryConfig) error {
	root, err := filepath.Abs(destination)
	if err != nil {
		return err
	}
	config.Destination = root
	if err := os.MkdirAll(filepath.Join(root, libraryDirName), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(root, libraryDirName, "config.json"), data, 0o644)
}

func LoadLibraryConfig(destination string) (LibraryConfig, error) {
	root, err := filepath.Abs(destination)
	if err != nil {
		return LibraryConfig{}, err
	}
	data, err := os.ReadFile(filepath.Join(root, libraryDirName, "config.json"))
	if err != nil {
		return LibraryConfig{}, err
	}
	var config LibraryConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return LibraryConfig{}, err
	}
	if config.Destination == "" {
		config.Destination = root
	}
	return config, nil
}

func InitGlobalStore(path string) error {
	db, err := openSQLite(path)
	if err != nil {
		return err
	}
	defer db.Close()
	_, err = db.Exec(`
CREATE TABLE IF NOT EXISTS photo_libraries (
	name TEXT PRIMARY KEY,
	path TEXT NOT NULL,
	is_default INTEGER NOT NULL DEFAULT 0,
	created_at TEXT NOT NULL,
	last_used_at TEXT NOT NULL
);`)
	return err
}

func SaveGlobalLibrary(dbPath string, library Library) error {
	if err := InitGlobalStore(dbPath); err != nil {
		return err
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return err
	}
	defer db.Close()
	now := time.Now().Format(time.RFC3339)
	if library.IsDefault {
		if _, err := db.Exec(`UPDATE photo_libraries SET is_default = 0`); err != nil {
			return err
		}
	}
	_, err = db.Exec(`
INSERT INTO photo_libraries(name, path, is_default, created_at, last_used_at)
VALUES(?, ?, ?, ?, ?)
ON CONFLICT(name) DO UPDATE SET path = excluded.path, is_default = excluded.is_default, last_used_at = excluded.last_used_at;`,
		library.Name, library.Path, boolInt(library.IsDefault), now, now)
	return err
}

func SetDefaultLibrary(dbPath string, name string) error {
	if err := InitGlobalStore(dbPath); err != nil {
		return err
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return err
	}
	defer db.Close()
	var exists int
	if err := db.QueryRow(`SELECT COUNT(*) FROM photo_libraries WHERE name = ?`, name).Scan(&exists); err != nil {
		return err
	}
	if exists == 0 {
		return fmt.Errorf("library %q not found", name)
	}
	if _, err := db.Exec(`UPDATE photo_libraries SET is_default = CASE WHEN name = ? THEN 1 ELSE 0 END`, name); err != nil {
		return err
	}
	_, err = db.Exec(`UPDATE photo_libraries SET last_used_at = ? WHERE name = ?`, time.Now().Format(time.RFC3339), name)
	return err
}

func ListGlobalLibraries(dbPath string) ([]Library, error) {
	if err := InitGlobalStore(dbPath); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	rows, err := db.Query(`SELECT name, path, is_default, created_at, last_used_at FROM photo_libraries ORDER BY is_default DESC, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var libraries []Library
	for rows.Next() {
		var library Library
		var isDefault int
		var createdAt, lastUsedAt string
		if err := rows.Scan(&library.Name, &library.Path, &isDefault, &createdAt, &lastUsedAt); err != nil {
			return nil, err
		}
		library.IsDefault = isDefault == 1
		library.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		library.LastUsedAt, _ = time.Parse(time.RFC3339, lastUsedAt)
		libraries = append(libraries, library)
	}
	return libraries, rows.Err()
}

func DefaultLibrary(dbPath string) (Library, error) {
	libraries, err := ListGlobalLibraries(dbPath)
	if err != nil {
		return Library{}, err
	}
	for _, library := range libraries {
		if library.IsDefault {
			return library, nil
		}
	}
	return Library{}, errors.New("no default photo library configured")
}

func FindGlobalLibrary(dbPath string, nameOrPath string) (Library, error) {
	if nameOrPath == "" {
		return DefaultLibrary(dbPath)
	}
	libraries, err := ListGlobalLibraries(dbPath)
	if err != nil {
		return Library{}, err
	}
	for _, library := range libraries {
		if library.Name == nameOrPath || library.Path == nameOrPath {
			return library, nil
		}
	}
	abs, err := filepath.Abs(nameOrPath)
	if err == nil {
		for _, library := range libraries {
			if library.Path == abs {
				return library, nil
			}
		}
	}
	return Library{}, fmt.Errorf("photo library %q not found", nameOrPath)
}

func InitLocalLibrary(destination string) error {
	db, err := openSQLite(localLibraryDBPath(destination))
	if err != nil {
		return err
	}
	defer db.Close()
	_, err = db.Exec(`
CREATE TABLE IF NOT EXISTS imports (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	source_path TEXT NOT NULL,
	destination_path TEXT NOT NULL,
	started_at TEXT NOT NULL,
	finished_at TEXT NOT NULL,
	config_snapshot TEXT NOT NULL,
	summary TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS media_files (
	destination_path TEXT PRIMARY KEY,
	source_path_original TEXT NOT NULL,
	sha256 TEXT,
	visual_hash INTEGER,
	has_visual_hash INTEGER NOT NULL,
	date_taken TEXT,
	camera TEXT,
	lens TEXT,
	media_type TEXT,
	extension TEXT,
	size INTEGER,
	imported_at TEXT NOT NULL,
	import_id INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_media_sha256 ON media_files(sha256);
CREATE INDEX IF NOT EXISTS idx_media_visual_hash ON media_files(visual_hash);`)
	return err
}

func ExistingHashes(destination string) (map[string]string, error) {
	if err := InitLocalLibrary(destination); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", localLibraryDBPath(destination))
	if err != nil {
		return nil, err
	}
	defer db.Close()
	rows, err := db.Query(`SELECT sha256, destination_path FROM media_files WHERE sha256 IS NOT NULL AND sha256 != ''`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	hashes := map[string]string{}
	for rows.Next() {
		var hash, path string
		if err := rows.Scan(&hash, &path); err != nil {
			return nil, err
		}
		hashes[hash] = path
	}
	return hashes, rows.Err()
}

func ExistingVisualHashes(destination string) ([]KnownVisualHash, error) {
	if err := InitLocalLibrary(destination); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", localLibraryDBPath(destination))
	if err != nil {
		return nil, err
	}
	defer db.Close()
	rows, err := db.Query(`SELECT destination_path, visual_hash FROM media_files WHERE has_visual_hash = 1`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var hashes []KnownVisualHash
	for rows.Next() {
		var item KnownVisualHash
		var hash int64
		if err := rows.Scan(&item.Path, &hash); err != nil {
			return nil, err
		}
		item.Hash = uint64(hash)
		hashes = append(hashes, item)
	}
	return hashes, rows.Err()
}

func RecordImport(destination string, source string, config LibraryConfig, plan Plan, summary Summary) error {
	if err := InitLocalLibrary(destination); err != nil {
		return err
	}
	db, err := sql.Open("sqlite", localLibraryDBPath(destination))
	if err != nil {
		return err
	}
	defer db.Close()
	started := time.Now()
	configData, err := json.Marshal(config)
	if err != nil {
		return err
	}
	summaryData, err := json.Marshal(summary)
	if err != nil {
		return err
	}
	result, err := db.Exec(`INSERT INTO imports(source_path, destination_path, started_at, finished_at, config_snapshot, summary) VALUES(?, ?, ?, ?, ?, ?)`,
		source, destination, started.Format(time.RFC3339), time.Now().Format(time.RFC3339), string(configData), string(summaryData))
	if err != nil {
		return err
	}
	importID, err := result.LastInsertId()
	if err != nil {
		return err
	}
	importedAt := time.Now().Format(time.RFC3339)
	for _, action := range plan.Actions {
		if action.Kind == ActionSkip || action.DestPath == "" || action.Hash == "" {
			continue
		}
		info, err := os.Stat(action.DestPath)
		if err != nil || info.IsDir() {
			continue
		}
		_, err = db.Exec(`
INSERT OR REPLACE INTO media_files(destination_path, source_path_original, sha256, visual_hash, has_visual_hash, date_taken, camera, lens, media_type, extension, size, imported_at, import_id)
VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);`,
			action.DestPath,
			action.SourcePath,
			action.Hash,
			int64(action.VisualHash),
			boolInt(action.HasVisualHash),
			action.Metadata.Date.Format(time.RFC3339),
			action.Metadata.Camera,
			action.Metadata.Lens,
			string(action.MediaType),
			action.Extension,
			action.SourceSize,
			importedAt,
			importID,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func localLibraryDBPath(destination string) string {
	return filepath.Join(destination, libraryDirName, "library.db")
}

func openSQLite(path string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	return sql.Open("sqlite", path)
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
