package webui

import (
	"testing"
	"time"

	"valkey-ftdcstat/internal/derive"
	"valkey-ftdcstat/internal/model"
)

func TestBuildDatasetUsesReportColumns(t *testing.T) {
	report := derive.Report{
		View:    "server",
		Columns: []string{"time", "ops/s", "cli"},
		Rows: []derive.Row{
			{Time: time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC), Values: map[string]any{"ops/s": 10.0, "cli": 2.0}},
		},
		Metadata: model.Metadata{Module: "valkey-ftdc"},
	}
	dataset := BuildDataset(report, nil, Options{View: "server"})
	if len(dataset.Metadata.Sections) != 1 || len(dataset.Metadata.Sections[0].Metrics) != 2 {
		t.Fatalf("sections=%+v", dataset.Metadata.Sections)
	}
	if dataset.Data.Rows[0].Sections["server"]["ops/s"] != 10.0 {
		t.Fatalf("row=%+v", dataset.Data.Rows[0].Sections)
	}
}

func TestNewServerServesMetadata(t *testing.T) {
	report := derive.Report{
		View:    "summary",
		Columns: []string{"time", "ops/s"},
		Rows: []derive.Row{
			{Time: time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC), Values: map[string]any{"ops/s": 1.0}},
		},
	}
	server, err := NewServer(BuildDataset(report, nil, Options{View: "summary"}))
	if err != nil {
		t.Fatal(err)
	}
	body, _, status := server.route("/api/metadata")
	if status != 200 || len(body) == 0 {
		t.Fatalf("status=%d len=%d", status, len(body))
	}
}
