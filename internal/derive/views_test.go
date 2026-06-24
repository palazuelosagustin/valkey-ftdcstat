package derive

import (
	"testing"
	"time"

	"valkey-ftdcstat/internal/model"
)

func TestFillServerView(t *testing.T) {
	samples := []model.MetricSample{
		metricSample(0, 100, 1000, 50, 10, 1<<20, 2<<20, 5, 0, 0, 0, 0, 0, 0, 0, 0, "master", 0, nil),
		metricSample(60_000, 110, 1060, 55, 12, 1<<20, 2<<20, 5, 0, 0, 0, 0, 0, 0, 0, 0, "master", 0, nil),
	}
	samples[0].Values[pathStatsErrors] = 100
	samples[1].Values[pathStatsErrors] = 103
	samples[0].Values[pathStatsRejected] = 1
	samples[1].Values[pathStatsRejected] = 2

	report, err := Build(model.Capture{MetricSamples: samples}, Options{View: "server", Interval: time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Rows) != 1 {
		t.Fatalf("rows=%d", len(report.Rows))
	}
	row := report.Rows[0]
	if row.Values["ops/s"] != float64(1) {
		t.Fatalf("ops/s=%v", row.Values["ops/s"])
	}
	if row.Values["err/s"] != float64(0.05) {
		t.Fatalf("err/s=%v", row.Values["err/s"])
	}
}

func TestLatencyViewCollectsEvents(t *testing.T) {
	first := metricSample(0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, "master", 0, nil)
	second := metricSample(60_000, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, "master", 0, nil)
	second.Values["valkey.latency_latest.fork.latest_ms"] = 12
	second.Values["valkey.latency_latest.active-defrag-cycle.latest_ms"] = 3

	report, err := Build(model.Capture{MetricSamples: []model.MetricSample{first, second}}, Options{View: "latency", Interval: time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Columns) != 3 {
		t.Fatalf("columns=%v", report.Columns)
	}
	if len(report.Rows) != 1 {
		t.Fatalf("rows=%d", len(report.Rows))
	}
	if report.Rows[0].Values["fork"] != float64(12) {
		t.Fatalf("fork=%v", report.Rows[0].Values["fork"])
	}
}

func TestNetworkViewParsesNetDev(t *testing.T) {
	first := metricSample(0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, "master", 0, nil)
	second := metricSample(60_000, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, "master", 0, nil)
	first.Text[pathHostNetDev] = "  eth0: 10240 0 0 0 0 0 0 0 5120 0 0 0 0 0 0 0\n"
	second.Text[pathHostNetDev] = "  eth0: 624640 0 0 0 0 0 0 0 614400 0 0 0 0 0 0 0\n"

	report, err := Build(model.Capture{MetricSamples: []model.MetricSample{first, second}}, Options{View: "network", Interval: time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	if report.Rows[0].Values["rxKB/s"] != float64(10) {
		t.Fatalf("rxKB/s=%v", report.Rows[0].Values["rxKB/s"])
	}
}
