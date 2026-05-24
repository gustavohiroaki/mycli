package photo

import (
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestDetectBurstGroupsByCameraAndWindow(t *testing.T) {
	files := []EnrichedFile{
		groupFile("/src/a.jpg", MediaTypePhoto, "Canon R6", at("2026-05-24 10:00:00"), "aaaa"),
		groupFile("/src/b.jpg", MediaTypePhoto, "Canon R6", at("2026-05-24 10:00:01"), "bbbb"),
		groupFile("/src/c.jpg", MediaTypePhoto, "Canon R6", at("2026-05-24 10:00:03"), "cccc"),
		groupFile("/src/d.jpg", MediaTypePhoto, "Canon R6", at("2026-05-24 10:01:00"), "dddd"),
	}

	groups := DetectBurstGroups(files, 2*time.Second)
	if len(groups) != 1 {
		t.Fatalf("len(groups) = %d, want 1", len(groups))
	}
	want := []string{"/src/a.jpg", "/src/b.jpg", "/src/c.jpg"}
	if !reflect.DeepEqual(groups[0].Files, want) {
		t.Fatalf("files = %#v, want %#v", groups[0].Files, want)
	}
}

func TestDetectBurstGroupsDoesNotCrossCameraOrVideo(t *testing.T) {
	files := []EnrichedFile{
		groupFile("/src/a.jpg", MediaTypePhoto, "Canon R6", at("2026-05-24 10:00:00"), "aaaa"),
		groupFile("/src/b.jpg", MediaTypePhoto, "Fuji X100V", at("2026-05-24 10:00:01"), "bbbb"),
		groupFile("/src/c.mov", MediaTypeVideo, "Canon R6", at("2026-05-24 10:00:02"), "cccc"),
	}

	groups := DetectBurstGroups(files, 2*time.Second)
	if len(groups) != 0 {
		t.Fatalf("len(groups) = %d, want 0", len(groups))
	}
}

func TestAssignGroupedNamesUsesBaseAndNumericSuffixes(t *testing.T) {
	files := []EnrichedFile{
		groupFile("/src/a.jpg", MediaTypePhoto, "Canon R6", at("2026-05-24 10:00:00"), "aaaa"),
		groupFile("/src/b.jpg", MediaTypePhoto, "Canon R6", at("2026-05-24 10:00:01"), "bbbb"),
		groupFile("/src/c.jpg", MediaTypePhoto, "Canon R6", at("2026-05-24 10:00:02"), "cccc"),
	}
	grouping := GroupingResult{PreferredGroupByFile: map[string]string{
		"/src/a.jpg": "burst-001",
		"/src/b.jpg": "burst-001",
		"/src/c.jpg": "burst-001",
	}, BurstGroups: []FileGroup{{ID: "burst-001", Type: GroupBurst, Files: []string{"/src/a.jpg", "/src/b.jpg", "/src/c.jpg"}}}}

	names := AssignGroupedNames(files, grouping)
	assertName(t, names, "/src/a.jpg", "2026-05-24_10-00-00_canon-r6_001.jpg")
	assertName(t, names, "/src/b.jpg", "2026-05-24_10-00-00_canon-r6_002.jpg")
	assertName(t, names, "/src/c.jpg", "2026-05-24_10-00-00_canon-r6_003.jpg")
}

func TestAssignGroupedNamesUsesNumericSuffixesForTimestampTies(t *testing.T) {
	files := []EnrichedFile{
		groupFile("/src/a.jpg", MediaTypePhoto, "Canon R6", at("2026-05-24 10:00:00"), "a91f0000"),
		groupFile("/src/b.jpg", MediaTypePhoto, "Canon R6", at("2026-05-24 10:00:00"), "c2040000"),
	}

	names := AssignGroupedNames(files, GroupingResult{})
	assertName(t, names, "/src/a.jpg", "2026-05-24_10-00-00_canon-r6_001.jpg")
	assertName(t, names, "/src/b.jpg", "2026-05-24_10-00-00_canon-r6_002.jpg")
}

func TestAssignGroupedNamesStableWhenInputOrderChanges(t *testing.T) {
	a := groupFile("/src/a.jpg", MediaTypePhoto, "Canon R6", at("2026-05-24 10:00:00"), "aaaa")
	b := groupFile("/src/b.jpg", MediaTypePhoto, "Canon R6", at("2026-05-24 10:00:01"), "bbbb")
	first := AssignGroupedNames([]EnrichedFile{a, b}, GroupingResult{})
	second := AssignGroupedNames([]EnrichedFile{b, a}, GroupingResult{})

	if !reflect.DeepEqual(first, second) {
		t.Fatalf("names differ:\nfirst=%#v\nsecond=%#v", first, second)
	}
}

func groupFile(path string, mediaType MediaType, camera string, date time.Time, hash string) EnrichedFile {
	return EnrichedFile{
		File: MediaFile{
			SourcePath: path,
			Type:       mediaType,
			Extension:  filepath.Ext(path),
		},
		Metadata: Metadata{Date: date, Camera: camera},
		Hash:     hash,
	}
}

func at(value string) time.Time {
	parsed, err := time.ParseInLocation("2006-01-02 15:04:05", value, time.Local)
	if err != nil {
		panic(err)
	}
	return parsed
}

func assertName(t *testing.T, names map[string]string, path string, want string) {
	t.Helper()
	if got := names[path]; got != want {
		t.Fatalf("name[%s] = %q, want %q", path, got, want)
	}
}
