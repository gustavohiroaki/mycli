package cmd

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var moveOrganizedFiles bool

var imageExtensions = map[string]struct{}{
	".jpg":  {},
	".jpeg": {},
	".png":  {},
	".gif":  {},
	".webp": {},
	".bmp":  {},
	".tif":  {},
	".tiff": {},
	".heic": {},
	".heif": {},
	".dng":  {},
	".raw":  {},
	".cr2":  {},
	".cr3":  {},
	".crw":  {},
	".nef":  {},
	".nrw":  {},
	".arw":  {},
	".srf":  {},
	".sr2":  {},
	".raf":  {},
	".rw2":  {},
	".orf":  {},
	".pef":  {},
	".x3f":  {},
}

var videoExtensions = map[string]struct{}{
	".mp4":  {},
	".mov":  {},
	".avi":  {},
	".mkv":  {},
	".m4v":  {},
	".3gp":  {},
	".mts":  {},
	".m2ts": {},
	".mpg":  {},
	".mpeg": {},
	".wmv":  {},
	".flv":  {},
	".webm": {},
}

var datePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(20\d{2})[-_]?([01]\d)[-_]?([0-3]\d)[T _-]?([0-2]\d)?([0-5]\d)?([0-5]\d)?`),
	regexp.MustCompile(`(?i)(\d{2})[-_]?([01]\d)[-_]?(20\d{2})`),
}

func init() {
	rootCmd.AddCommand(organizeCmd)
	organizeCmd.Flags().BoolVar(&moveOrganizedFiles, "move", false, "Move files instead of copying them")
}

var organizeCmd = &cobra.Command{
	Use:   "organize <origem> <destino>",
	Short: "Organiza fotos e videos por data",
	Long: `Organiza arquivos de midia em pastas no formato:
ano/mes/dia/fotos
ano/mes/dia/videos

O comando percorre a origem recursivamente, tenta obter a data real da midia
usando exiftool quando disponivel, e usa o nome do arquivo ou a data de
modificacao como fallback.`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		sourceDir, err := filepath.Abs(args[0])
		if err != nil {
			return fmt.Errorf("falha ao resolver a origem: %w", err)
		}

		targetDir, err := filepath.Abs(args[1])
		if err != nil {
			return fmt.Errorf("falha ao resolver o destino: %w", err)
		}

		if sourceDir == targetDir {
			return errors.New("origem e destino nao podem ser iguais")
		}

		if isSubpath(targetDir, sourceDir) {
			return errors.New("o destino nao pode ficar dentro da origem")
		}

		sourceInfo, err := os.Stat(sourceDir)
		if err != nil {
			return fmt.Errorf("falha ao acessar a origem: %w", err)
		}
		if !sourceInfo.IsDir() {
			return errors.New("a origem precisa ser um diretorio")
		}

		if err := os.MkdirAll(targetDir, 0o755); err != nil {
			return fmt.Errorf("falha ao criar o destino: %w", err)
		}

		stats := organizeStats{}
		err = filepath.WalkDir(sourceDir, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}

			if d.IsDir() {
				return nil
			}

			mediaType, ok := detectMediaType(path)
			if !ok {
				return nil
			}

			mediaDate, err := resolveMediaDate(path)
			if err != nil {
				fmt.Printf("Ignorando data invalida para %s: %v\n", path, err)
				mediaDate = time.Now()
			}

			destinationDir := filepath.Join(
				targetDir,
				fmt.Sprintf("%04d", mediaDate.Year()),
				fmt.Sprintf("%02d", int(mediaDate.Month())),
				fmt.Sprintf("%02d", mediaDate.Day()),
				mediaType,
			)

			if err := os.MkdirAll(destinationDir, 0o755); err != nil {
				return fmt.Errorf("falha ao criar %s: %w", destinationDir, err)
			}

			destinationPath, err := uniqueDestinationPath(destinationDir, filepath.Base(path))
			if err != nil {
				return err
			}

			if moveOrganizedFiles {
				if err := moveFile(path, destinationPath); err != nil {
					return fmt.Errorf("falha ao mover %s: %w", path, err)
				}
			} else {
				if err := copyFile(path, destinationPath); err != nil {
					return fmt.Errorf("falha ao copiar %s: %w", path, err)
				}
			}

			stats.processed++
			if mediaType == "fotos" {
				stats.photos++
			} else {
				stats.videos++
			}
			return nil
		})
		if err != nil {
			return err
		}

		mode := "copiados"
		if moveOrganizedFiles {
			mode = "movidos"
		}

		fmt.Printf(
			"Arquivos %s: %d (fotos: %d, videos: %d)\n",
			mode,
			stats.processed,
			stats.photos,
			stats.videos,
		)
		return nil
	},
}

type organizeStats struct {
	processed int
	photos    int
	videos    int
}

func detectMediaType(path string) (string, bool) {
	extension := strings.ToLower(filepath.Ext(path))

	if _, ok := imageExtensions[extension]; ok {
		return "fotos", true
	}

	if _, ok := videoExtensions[extension]; ok {
		return "videos", true
	}

	return "", false
}

func resolveMediaDate(path string) (time.Time, error) {
	if mediaDate, err := dateFromExiftool(path); err == nil {
		return mediaDate, nil
	}

	if mediaDate, ok := dateFromFilename(filepath.Base(path)); ok {
		return mediaDate, nil
	}

	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}, err
	}

	return info.ModTime(), nil
}

func dateFromExiftool(path string) (time.Time, error) {
	if _, err := exec.LookPath("exiftool"); err != nil {
		return time.Time{}, errors.New("exiftool nao encontrado")
	}

	output, err := exec.Command(
		"exiftool",
		"-s3",
		"-d",
		"%Y-%m-%d %H:%M:%S",
		"-DateTimeOriginal",
		"-CreateDate",
		"-MediaCreateDate",
		"-TrackCreateDate",
		path,
	).Output()
	if err != nil {
		return time.Time{}, err
	}

	for _, line := range strings.Split(string(output), "\n") {
		candidate := strings.TrimSpace(line)
		if candidate == "" {
			continue
		}

		mediaDate, err := time.ParseInLocation("2006-01-02 15:04:05", candidate, time.Local)
		if err == nil {
			return mediaDate, nil
		}
	}

	return time.Time{}, errors.New("nenhuma data encontrada no metadata")
}

func dateFromFilename(name string) (time.Time, bool) {
	for index, pattern := range datePatterns {
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

func uniqueDestinationPath(dir string, fileName string) (string, error) {
	extension := filepath.Ext(fileName)
	baseName := strings.TrimSuffix(fileName, extension)
	candidate := filepath.Join(dir, fileName)

	if _, err := os.Stat(candidate); errors.Is(err, os.ErrNotExist) {
		return candidate, nil
	} else if err != nil {
		return "", err
	}

	for i := 1; ; i++ {
		candidate = filepath.Join(dir, fmt.Sprintf("%s_%d%s", baseName, i, extension))
		if _, err := os.Stat(candidate); errors.Is(err, os.ErrNotExist) {
			return candidate, nil
		} else if err != nil {
			return "", err
		}
	}
}

func moveFile(source string, target string) error {
	if err := os.Rename(source, target); err == nil {
		return nil
	}

	if err := copyFile(source, target); err != nil {
		return err
	}

	return os.Remove(source)
}

func copyFile(source string, target string) error {
	sourceFile, err := os.Open(source)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	info, err := sourceFile.Stat()
	if err != nil {
		return err
	}

	targetFile, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode().Perm())
	if err != nil {
		return err
	}

	_, copyErr := io.Copy(targetFile, sourceFile)
	closeErr := targetFile.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}

	return os.Chtimes(target, time.Now(), info.ModTime())
}

func isSubpath(candidate string, parent string) bool {
	relative, err := filepath.Rel(parent, candidate)
	if err != nil {
		return false
	}

	return relative != "." && relative != ".." && !strings.HasPrefix(relative, ".."+string(os.PathSeparator))
}
