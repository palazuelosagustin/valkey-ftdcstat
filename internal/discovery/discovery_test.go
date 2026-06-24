package discovery

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"valkey-ftdcstat/internal/model"
)

func TestDiscoverMetricsAndSidecars(t *testing.T) {
	dir := t.TempDir()
	metrics := filepath.Join(dir, "metrics.2026-06-20T11-59-11Z.vkftdc")
	sidecar := filepath.Join(dir, "metadata.2026-06-20T11-59-11Z.json")
	if err := os.WriteFile(metrics, []byte("VKFTDC1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sidecar, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	files, warnings, err := Discover(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Fatalf("warnings=%v", warnings)
	}
	if len(files) != 2 {
		t.Fatalf("files=%d", len(files))
	}
	if files[0].Kind != KindMetrics {
		t.Fatalf("first kind=%q", files[0].Kind)
	}
	if files[0].Timestamp != time.Date(2026, 6, 20, 11, 59, 11, 0, time.UTC) {
		t.Fatalf("timestamp=%v", files[0].Timestamp)
	}
}

func TestFilterByTimeRange(t *testing.T) {
	files := []MetricFile{
		{Path: "a", Kind: KindMetrics, Timestamp: time.Date(2026, 6, 20, 10, 0, 0, 0, time.UTC)},
		{Path: "b", Kind: KindMetrics, Timestamp: time.Date(2026, 6, 20, 11, 0, 0, 0, time.UTC)},
		{Path: "c", Kind: KindSidecar, Timestamp: time.Date(2026, 6, 20, 11, 0, 0, 0, time.UTC)},
	}
	filtered := FilterByTimeRange(files, model.TimeRange{
		From: time.Date(2026, 6, 20, 11, 30, 0, 0, time.UTC),
		To:   time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC),
	})
	if len(filtered) != 2 {
		t.Fatalf("filtered=%d", len(filtered))
	}
	if filtered[0].Path != "b" {
		t.Fatalf("first=%q", filtered[0].Path)
	}
}
