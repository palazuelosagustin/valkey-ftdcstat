package discovery

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"valkey-ftdcstat/internal/model"
)

type FileKind string

const (
	KindMetrics  FileKind = "metrics"
	KindSidecar  FileKind = "sidecar"
	KindInterim  FileKind = "interim"
)

type MetricFile struct {
	Path      string
	Kind      FileKind
	Timestamp time.Time
	Sequence  int
}

var metricsTimestampRE = regexp.MustCompile(`^metrics\.(\d{4}-\d{2}-\d{2}T\d{2}-\d{2}-\d{2}Z)\.vkftdc$`)
var sidecarTimestampRE = regexp.MustCompile(`^metadata\.(\d{4}-\d{2}-\d{2}T\d{2}-\d{2}-\d{2}Z)\.json$`)

func Discover(root string) ([]MetricFile, []model.Warning, error) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, nil, err
	}
	if !info.IsDir() {
		return nil, nil, errors.New("path must be a directory")
	}

	var files []MetricFile
	var warnings []model.Warning
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			warnings = append(warnings, model.Warning{Source: path, Message: walkErr.Error()})
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		kind, ok := classify(entry.Name())
		if !ok {
			return nil
		}
		ts, tsOK := parseTimestamp(entry.Name(), kind)
		if !tsOK && kind == KindMetrics {
			warnings = append(warnings, model.Warning{Source: path, Message: "metrics file name does not contain a parseable rotation timestamp"})
		}
		files = append(files, MetricFile{Path: path, Kind: kind, Timestamp: ts})
		return nil
	})
	if err != nil {
		return nil, warnings, err
	}

	sort.SliceStable(files, func(i, j int) bool {
		a, b := files[i], files[j]
		ap, bp := sortPriority(a), sortPriority(b)
		if ap != bp {
			return ap < bp
		}
		if !a.Timestamp.IsZero() && !b.Timestamp.IsZero() && !a.Timestamp.Equal(b.Timestamp) {
			return a.Timestamp.Before(b.Timestamp)
		}
		return filepath.Base(a.Path) < filepath.Base(b.Path)
	})
	for i := range files {
		files[i].Sequence = i
	}
	if metricCount(files) == 0 {
		return nil, warnings, errors.New("no FTDC metrics files found")
	}
	return files, warnings, nil
}

func FilterByTimeRange(files []MetricFile, tr model.TimeRange) []MetricFile {
	if tr.IsZero() || len(files) == 0 {
		return files
	}
	out := make([]MetricFile, 0, len(files))
	for i, file := range files {
		if file.Kind != KindMetrics || file.Timestamp.IsZero() {
			out = append(out, file)
			continue
		}
		next := nextMetricsTimestamp(files, i)
		if next.IsZero() {
			out = append(out, file)
			continue
		}
		if tr.Overlaps(file.Timestamp, next) {
			out = append(out, file)
		}
	}
	for i := range out {
		out[i].Sequence = i
	}
	return out
}

func MetricFiles(files []MetricFile) []MetricFile {
	out := make([]MetricFile, 0, len(files))
	for _, file := range files {
		if file.Kind == KindMetrics {
			out = append(out, file)
		}
	}
	return out
}

func SidecarFiles(files []MetricFile) []MetricFile {
	out := make([]MetricFile, 0, len(files))
	for _, file := range files {
		if file.Kind == KindSidecar {
			out = append(out, file)
		}
	}
	return out
}

func nextMetricsTimestamp(files []MetricFile, start int) time.Time {
	for i := start + 1; i < len(files); i++ {
		if files[i].Kind == KindMetrics && !files[i].Timestamp.IsZero() {
			return files[i].Timestamp
		}
	}
	return time.Time{}
}

func metricCount(files []MetricFile) int {
	n := 0
	for _, file := range files {
		if file.Kind == KindMetrics {
			n++
		}
	}
	return n
}

func classify(name string) (FileKind, bool) {
	switch {
	case metricsTimestampRE.MatchString(name):
		return KindMetrics, true
	case sidecarTimestampRE.MatchString(name):
		return KindSidecar, true
	case strings.HasSuffix(strings.ToLower(name), ".vkftdc"):
		return KindMetrics, true
	case strings.Contains(name, ".interim"):
		return KindInterim, true
	default:
		return "", false
	}
}

func parseTimestamp(name string, kind FileKind) (time.Time, bool) {
	var matches []string
	switch kind {
	case KindMetrics:
		matches = metricsTimestampRE.FindStringSubmatch(name)
	case KindSidecar:
		matches = sidecarTimestampRE.FindStringSubmatch(name)
	default:
		return time.Time{}, false
	}
	if len(matches) != 2 {
		return time.Time{}, false
	}
	t, err := time.Parse("2006-01-02T15-04-05Z", matches[1])
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

func sortPriority(file MetricFile) int {
	switch file.Kind {
	case KindMetrics:
		if file.Timestamp.IsZero() {
			return 1
		}
		return 0
	case KindSidecar:
		return 2
	case KindInterim:
		return 3
	default:
		return 4
	}
}
