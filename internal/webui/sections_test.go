package webui

import (
	"testing"

	"valkey-ftdcstat/internal/derive"
)

func TestBuildSectionsSummaryDashboard(t *testing.T) {
	columns := []string{
		"time", "ops/s", "conn/s", "hit%",
		"memMB", "rssMB", "frag%",
		"rej/s", "exp/s", "cli", "blk",
		"repl", "repls",
		"us%", "load1",
	}
	sections := buildSections("summary", columns, false)
	if len(sections) < 5 {
		t.Fatalf("sections=%+v", sections)
	}
	if sections[0].Name != "server" {
		t.Fatalf("first=%s", sections[0].Name)
	}
}

func TestHostSectionsSplitCPUAndDisks(t *testing.T) {
	columns := []string{"time", "us%", "sy%", "bi", "bo", "load1", "free"}
	sections := buildSections("host", columns, false)
	names := map[string]bool{}
	for _, section := range sections {
		names[section.Name] = true
	}
	for _, want := range []string{"host / CPU", "host / Memory", "host / Disks"} {
		if !names[want] {
			t.Fatalf("missing %s in %+v", want, sections)
		}
	}
}

func TestLatencySectionsSplitFallbackAndEvents(t *testing.T) {
	columns := []string{"time", "slowlog", "eloopUs", "fork", "forkMax"}
	sections := buildSections("latency", columns, false)
	if len(sections) != 2 {
		t.Fatalf("sections=%+v", sections)
	}
	if sections[1].Metrics[0].Column != "fork" {
		t.Fatalf("events=%+v", sections[1].Metrics)
	}
}

func TestBuildCommandstatsDataset(t *testing.T) {
	report := derive.Report{
		View: "commandstats",
		Commands: []derive.CommandRow{
			{Command: "get", Calls: 100, CallsPerSec: 1.5, UsecPerCall: 2.1, SharePct: 40},
			{Command: "set", Calls: 60, CallsPerSec: 0.9, UsecPerCall: 3.2, SharePct: 24},
		},
	}
	dataset := buildCommandstatsDataset(report, nil, Options{View: "commandstats"})
	if len(dataset.Data.Commands) != 2 {
		t.Fatalf("commands=%d", len(dataset.Data.Commands))
	}
	if dataset.Metadata.RowCount != 2 {
		t.Fatalf("rowCount=%d", dataset.Metadata.RowCount)
	}
}
