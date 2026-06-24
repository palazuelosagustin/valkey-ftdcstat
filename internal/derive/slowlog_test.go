package derive

import (
	"testing"
	"time"

	"valkey-ftdcstat/internal/model"
)

func TestDeriveSlowlogDeduplicatesByIDAndFingerprint(t *testing.T) {
	base := time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC)
	samples := []model.MetricSample{
		{
			Time: base,
			SlowlogEntries: []model.SlowlogItem{
				{ID: 1, DurationUSec: 500_000, Args: []string{"GET", "user:1"}},
				{ID: 2, DurationUSec: 800_000, Args: []string{"HGETALL", "user:session:*"}},
			},
		},
		{
			Time: base.Add(time.Minute),
			SlowlogEntries: []model.SlowlogItem{
				{ID: 1, DurationUSec: 500_000, Args: []string{"GET", "user:1"}},
				{ID: 3, DurationUSec: 600_000, Args: []string{"GET", "user:1"}},
				{ID: 4, DurationUSec: 900_000, Args: []string{"HGETALL", "user:session:*"}},
			},
		},
	}
	rows, summary, _ := deriveSlowlog(samples, 10, map[string]any{"collect-slowlog": "yes"})
	if summary.TotalEntries != 4 {
		t.Fatalf("total=%d", summary.TotalEntries)
	}
	if summary.UniquePatterns != 2 {
		t.Fatalf("patterns=%d rows=%+v", summary.UniquePatterns, rows)
	}
	if rows[0].Command != "HGETALL" || rows[0].MaxMs != 900 || rows[0].Count != 2 {
		t.Fatalf("first=%+v", rows[0])
	}
	if rows[1].Command != "GET" || rows[1].Count != 2 || rows[1].MaxMs != 600 {
		t.Fatalf("second=%+v", rows[1])
	}
}

func TestDeriveSlowlogDisabledCollectorNote(t *testing.T) {
	_, summary, note := deriveSlowlog(nil, 10, map[string]any{"collect-slowlog": "no"})
	if summary.CollectEnabled {
		t.Fatal("expected disabled")
	}
	if note == "" {
		t.Fatal("expected note")
	}
}
