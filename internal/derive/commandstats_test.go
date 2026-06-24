package derive

import (
	"testing"
	"time"

	"valkey-ftdcstat/internal/model"
)

func TestBuildCommandstatsRows(t *testing.T) {
	samples := commandstatsSamples()
	report, err := Build(model.Capture{MetricSamples: samples}, Options{View: "commandstats", Interval: time.Minute, TopCommands: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Rows) != 1 {
		t.Fatalf("rows=%d", len(report.Rows))
	}
	if len(report.Columns) != 3 {
		t.Fatalf("columns=%v", report.Columns)
	}
	row := report.Rows[0]
	if row.Values["get/s"] != float64(1) {
		t.Fatalf("get/s=%v", row.Values["get/s"])
	}
	if row.Values["set/s"] != float64(0.5) {
		t.Fatalf("set/s=%v", row.Values["set/s"])
	}
}

func TestSummaryIncludesTopCommandRates(t *testing.T) {
	samples := commandstatsSamples()
	report, err := Build(model.Capture{MetricSamples: samples}, Options{View: "summary", Interval: time.Minute, TopCommands: 1})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, col := range report.Columns {
		if col == "get/s" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("columns=%v", report.Columns)
	}
	if report.Rows[0].Values["get/s"] != float64(1) {
		t.Fatalf("get/s=%v", report.Rows[0].Values["get/s"])
	}
}

func commandstatsSamples() []model.MetricSample {
	first := metricSample(0, 1000, 2000, 5000, 1000, 100<<20, 150<<20, 10, 1, 1.0, 1000, 500, 10000, 100, 5000, 100, "master", 2, nil)
	second := metricSample(60_000, 1120, 2600, 5600, 1120, 110<<20, 160<<20, 12, 0, 1.2, 1100, 540, 10300, 110, 5600, 110, "master", 2, nil)
	first.Values["valkey.info.commandstats.cmdstat_get.calls"] = 100
	first.Values["valkey.info.commandstats.cmdstat_set.calls"] = 40
	second.Values["valkey.info.commandstats.cmdstat_get.calls"] = 160
	second.Values["valkey.info.commandstats.cmdstat_set.calls"] = 70
	return []model.MetricSample{first, second}
}
