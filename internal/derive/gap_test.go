package derive

import (
	"strings"
	"testing"
	"time"
)

func TestGapResetsRatesButKeepsGauges(t *testing.T) {
	first := metricSample(0, 100, 1000, 50, 10, 1<<20, 2<<20, 5, 0, 0, 0, 0, 0, 0, 0, 0, "master", 0, nil)
	second := metricSample(60_000, 110, 1060, 55, 12, 1<<20, 2<<20, 5, 0, 0, 0, 0, 0, 0, 0, 0, "master", 0, nil)
	third := metricSample(960_000, 120, 1120, 60, 14, 1<<20, 2<<20, 6, 0, 0, 0, 0, 0, 0, 0, 0, "master", 0, nil)
	second.Values["valkey.info.commandstats.cmdstat_get.calls"] = 100
	third.Values["valkey.info.commandstats.cmdstat_get.calls"] = 160

	streamer := NewStreamer(Options{View: "summary", Interval: time.Minute, GapThreshold: 10 * time.Minute, TopCommands: 1})
	if _, ok := streamer.Add(first); ok {
		t.Fatal("first sample should not emit row")
	}
	if _, ok := streamer.Add(second); !ok {
		t.Fatal("expected row after first interval")
	}
	row, ok := streamer.Add(third)
	if !ok {
		t.Fatal("expected row after gap")
	}
	if row.Marker == "" || !strings.Contains(row.Marker, "gap") {
		t.Fatalf("marker=%q", row.Marker)
	}
	if _, ok := row.Values["ops/s"]; ok {
		t.Fatalf("ops/s should be omitted after gap reset: %v", row.Values["ops/s"])
	}
	if _, ok := row.Values["get/s"]; ok {
		t.Fatalf("get/s should be omitted after gap reset: %v", row.Values["get/s"])
	}
	if row.Values["memMB"] == nil {
		t.Fatal("expected memMB gauge after gap")
	}
}
