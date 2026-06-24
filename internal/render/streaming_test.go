package render

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"valkey-ftdcstat/internal/derive"
)

func TestSummaryTableUsesSectionSeparators(t *testing.T) {
	report := derive.Report{
		View: "summary",
		Columns: []string{
			"time", "ops/s", "conn/s", "hit%", "memMB", "rssMB", "cli", "blk", "us%", "load1",
		},
		Rows: []derive.Row{
			{
				Time: time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC),
				Values: map[string]any{
					"ops/s": 10.0, "conn/s": 1.0, "hit%": 99.0,
					"memMB": 1.0, "rssMB": 14.0,
					"cli": 2.0, "blk": 0.0,
					"us%": 3.0, "load1": 0.5,
				},
			},
		},
	}
	var buf bytes.Buffer
	if err := Report(&buf, report, DisplayOptions{}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "datetime") || !strings.Contains(out, "|") {
		t.Fatalf("expected grouped header with pipes:\n%s", out)
	}
	for _, label := range []string{"server", "memory", "clients", "host"} {
		if !strings.Contains(out, label) {
			t.Fatalf("missing section label %q:\n%s", label, out)
		}
	}
}

func TestStreamingRendererRepeatsHeader(t *testing.T) {
	layout := singleSectionLayout("server", []string{"time", "ops/s"})
	renderer := newStreamingRenderer(&bytes.Buffer{}, layout, time.UTC)
	row := derive.Row{
		Time:   time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC),
		Values: map[string]any{"ops/s": 1.0},
	}
	for i := 0; i < 51; i++ {
		row.Time = row.Time.Add(time.Minute)
		if err := renderer.RenderRow(row); err != nil {
			t.Fatal(err)
		}
	}
	if renderer.dataRows != 51 {
		t.Fatalf("rows=%d", renderer.dataRows)
	}
}
