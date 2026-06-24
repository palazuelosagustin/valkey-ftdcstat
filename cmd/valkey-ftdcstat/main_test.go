package main

import (
	"strings"
	"testing"
	"time"
)

func TestParseArgsDefaultIntervalIsSixty(t *testing.T) {
	opts, err := parseArgs([]string{"diagnostic.data"})
	if err != nil {
		t.Fatal(err)
	}
	if opts.Interval != 60 {
		t.Fatalf("interval=%d", opts.Interval)
	}
	if opts.View != "summary" {
		t.Fatalf("view=%s", opts.View)
	}
}

func TestParseArgsAvg(t *testing.T) {
	opts, err := parseArgs([]string{"diagnostic.data", "--avg", "5m"})
	if err != nil {
		t.Fatal(err)
	}
	if opts.Avg != 5*time.Minute {
		t.Fatalf("avg=%s", opts.Avg)
	}
}

func TestParseArgsAvgRejectsInterval(t *testing.T) {
	_, err := parseArgs([]string{"diagnostic.data", "--avg", "5m", "--interval", "120"})
	if err == nil || !strings.Contains(err.Error(), "--avg cannot be combined with --interval") {
		t.Fatalf("err=%v", err)
	}
}

func TestParseArgsWebJSONConflict(t *testing.T) {
	_, err := parseArgs([]string{"diagnostic.data", "--web", "--json"})
	if err == nil || !strings.Contains(err.Error(), "--web cannot be combined with --json") {
		t.Fatalf("err=%v", err)
	}
}

func TestParseArgsTimeRange(t *testing.T) {
	opts, err := parseArgs([]string{"diagnostic.data", "--from", "2026-06-04T19:00:00", "--to", "2026-06-04T20:00:00"})
	if err != nil {
		t.Fatal(err)
	}
	if opts.Range.From.IsZero() || opts.Range.To.IsZero() {
		t.Fatalf("range=%+v", opts.Range)
	}
}

func TestParseArgsDeviceHostOnly(t *testing.T) {
	opts, err := parseArgs([]string{"diagnostic.data", "--view", "host", "--device", "sda"})
	if err != nil {
		t.Fatal(err)
	}
	if opts.Device != "sda" {
		t.Fatalf("device=%s", opts.Device)
	}
	_, err = parseArgs([]string{"diagnostic.data", "--view", "summary", "--device", "sda"})
	if err == nil || !strings.Contains(err.Error(), "--device is only supported for --view host") {
		t.Fatalf("err=%v", err)
	}
}
