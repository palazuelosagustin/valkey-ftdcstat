package render

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"valkey-ftdcstat/internal/derive"
)

func Report(w io.Writer, report derive.Report, jsonOut bool) error {
	if jsonOut {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(report)
	}
	fmt.Fprintf(w, "path: %s\n", report.Path)
	fmt.Fprintf(w, "files: %d\n", len(report.Files))
	fmt.Fprintf(w, "samples: %d\n", report.SampleCount)
	fmt.Fprintf(w, "range: %s .. %s\n\n", report.Start.Format("2006-01-02T15:04:05Z07:00"), report.End.Format("2006-01-02T15:04:05Z07:00"))

	if report.View == "commandstats" {
		return renderCommands(w, report.Commands)
	}
	return renderRows(w, report.Rows, report.Columns)
}

func renderRows(w io.Writer, rows []map[string]any, preferred []string) error {
	if len(rows) == 0 {
		_, err := fmt.Fprintln(w, "no derived rows")
		return err
	}
	cols := orderedColumns(rows, preferred)
	widths := make(map[string]int, len(cols))
	for _, col := range cols {
		widths[col] = len(col)
	}
	for _, row := range rows {
		for _, col := range cols {
			value := formatValue(row[col])
			if len(value) > widths[col] {
				widths[col] = len(value)
			}
		}
	}
	for i, col := range cols {
		if i > 0 {
			fmt.Fprint(w, " ")
		}
		fmt.Fprintf(w, "%*s", widths[col], col)
	}
	fmt.Fprintln(w)
	for _, row := range rows {
		for i, col := range cols {
			if i > 0 {
				fmt.Fprint(w, " ")
			}
			fmt.Fprintf(w, "%*s", widths[col], formatValue(row[col]))
		}
		fmt.Fprintln(w)
	}
	return nil
}

func renderCommands(w io.Writer, rows []derive.CommandRow) error {
	if len(rows) == 0 {
		_, err := fmt.Fprintln(w, "no commandstats deltas")
		return err
	}
	fmt.Fprintf(w, "%-16s %10s %10s %12s %8s\n", "command", "calls", "calls/s", "usec/call", "share%")
	for _, row := range rows {
		fmt.Fprintf(w, "%-16s %10.0f %10.2f %12.2f %8.2f\n", row.Command, row.Calls, row.CallsPerSec, row.UsecPerCall, row.SharePct)
	}
	return nil
}

func orderedColumns(rows []map[string]any, preferred []string) []string {
	seen := map[string]bool{}
	var cols []string
	for _, col := range preferred {
		for _, row := range rows {
			if _, ok := row[col]; ok && !seen[col] {
				cols = append(cols, col)
				seen[col] = true
				break
			}
		}
	}
	for _, row := range rows {
		var extra []string
		for key := range row {
			if seen[key] {
				continue
			}
			extra = append(extra, key)
			seen[key] = true
		}
		sort.Strings(extra)
		cols = append(cols, extra...)
	}
	return cols
}

func formatValue(v any) string {
	switch x := v.(type) {
	case nil:
		return "-"
	case string:
		if x == "" {
			return "-"
		}
		return x
	case float64:
		return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.2f", x), "0"), ".")
	default:
		return fmt.Sprint(x)
	}
}
