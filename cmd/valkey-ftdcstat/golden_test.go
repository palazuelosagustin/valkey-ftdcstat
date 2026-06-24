package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"valkey-ftdcstat/internal/aggregate"
	"valkey-ftdcstat/internal/derive"
	"valkey-ftdcstat/internal/discovery"
	"valkey-ftdcstat/internal/reader"
	"valkey-ftdcstat/internal/render"
)

func TestGoldenOutputs(t *testing.T) {
	root := filepath.Join("..", "..", "testfixtures", "diagnostic.data")
	if _, err := os.Stat(root); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		view string
		opts cliOptions
	}{
		{view: "summary", opts: cliOptions{View: "summary", Interval: 60, TopCommands: -1}},
		{view: "server", opts: cliOptions{View: "server", Interval: 60, TopCommands: -1}},
		{view: "host", opts: cliOptions{View: "host", Interval: 60, TopCommands: -1}},
		{view: "latency", opts: cliOptions{View: "latency", Interval: 60, TopCommands: -1}},
		{view: "commandstats", opts: cliOptions{View: "commandstats", Interval: 60, TopCommands: 3}},
	}

	for _, tc := range cases {
		t.Run(tc.view, func(t *testing.T) {
			goldenPath := filepath.Join("..", "..", "testfixtures", "outputs", tc.view+".golden")
			wantBytes, err := os.ReadFile(goldenPath)
			if err != nil {
				t.Fatal(err)
			}
			got := renderCapture(t, root, tc.opts)
			want := string(wantBytes)
			if got != want {
				gotLines := strings.Split(got, "\n")
				wantLines := strings.Split(want, "\n")
				for i := 0; i < len(gotLines) && i < len(wantLines); i++ {
					if gotLines[i] != wantLines[i] {
						t.Fatalf("golden mismatch at line %d\n got: %q\nwant: %q", i+1, gotLines[i], wantLines[i])
					}
				}
				t.Fatalf("golden mismatch: got %d lines want %d lines", len(gotLines), len(wantLines))
			}
		})
	}
}

func renderCapture(t *testing.T, path string, opts cliOptions) string {
	t.Helper()
	files, warnings, err := discovery.Discover(path)
	if err != nil {
		t.Fatal(err)
	}
	files = discovery.FilterByTimeRange(files, opts.Range)
	metadata, _, err := reader.ReadMetadata(path, files)
	if err != nil {
		t.Fatal(err)
	}
	for _, warning := range warnings {
		t.Log("warning:", warning.String())
	}

	deriveOpts := derive.Options{
		View:         opts.View,
		Interval:     time.Duration(opts.Interval) * time.Second,
		Device:       opts.Device,
		Verbose:      opts.Verbose,
		TopCommands:  opts.TopCommands,
		Metadata:     metadata,
		TimeLocation: time.UTC,
	}
	report, err := derive.BuildFromReader(path, files, metadata, deriveOpts, reader.StreamOptions{TimeRange: opts.Range})
	if err != nil {
		t.Fatal(err)
	}
	if opts.Avg > 0 {
		report.Rows = aggregate.AverageRows(report.Rows, opts.Avg)
	}

	var buf bytes.Buffer
	if err := render.Report(&buf, report, render.DisplayOptions{}); err != nil {
		t.Fatal(err)
	}
	return normalizeGoldenOutput(buf.String())
}

func normalizeGoldenOutput(output string) string {
	lines := strings.Split(output, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "path:") {
			lines[i] = "path:    testfixtures/diagnostic.data"
		}
	}
	return strings.Join(lines, "\n")
}
