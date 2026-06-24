package main

import (
	"os"
	"strings"
	"testing"
)

func TestMetricReferenceDocumentsViewsAndFormulas(t *testing.T) {
	data, err := os.ReadFile("../../docs/metric-reference.md")
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, want := range []string{
		"`summary`",
		"`commandstats`",
		"`latency`",
		"`--top N`",
		"`--avg DURATION`",
		"`--from ISO_TIME`",
		"`--device DEVICE`",
		"Δ`total_commands_processed`",
		"`<cmd>/s`",
		"gap …: rate baseline reset",
		"GET /api/metadata",
		"GET /api/data",
		"testfixtures/outputs/",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("metric-reference missing %q", want)
		}
	}
}

func TestBacklogDocumentsPhase8SlowlogView(t *testing.T) {
	data, err := os.ReadFile("../../docs/backlog.md")
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, want := range []string{
		"Phase 8",
		"`--view slowlog`",
		"Slowest first",
		"Deduplicate repeated queries",
		"Repeat counter",
		"count",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("backlog missing %q", want)
		}
	}
}

func TestREADMERefersToMetricReference(t *testing.T) {
	data, err := os.ReadFile("../../README.md")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "docs/metric-reference.md") {
		t.Fatal("README should link to docs/metric-reference.md")
	}
}
