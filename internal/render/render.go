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
	renderHeader(w, report)
	if report.View == "commandstats" {
		return renderCommands(w, report.Commands)
	}
	return renderRows(w, report.Rows, report.Columns)
}

func renderHeader(w io.Writer, report derive.Report) {
	fmt.Fprintf(w, "path: %s\n", report.Path)
	fmt.Fprintf(w, "files: %d\n", len(report.Files))
	fmt.Fprintf(w, "samples: %d\n", report.SampleCount)
	fmt.Fprintf(w, "range: %s .. %s\n", report.Start.Format("2006-01-02T15:04:05Z07:00"), report.End.Format("2006-01-02T15:04:05Z07:00"))
	if report.Metadata.Module != "" || report.Metadata.FormatVersion > 0 {
		fmt.Fprintf(w, "module: %s format_version=%d\n", report.Metadata.Module, report.Metadata.FormatVersion)
	}
	if len(report.Metadata.Server) > 0 {
		fmt.Fprintln(w, "serverInfo")
		for _, key := range []string{"valkey_version", "redis_version", "server_mode", "process_id", "run_id", "hz"} {
			if value, ok := report.Metadata.Server[key]; ok && fmt.Sprint(value) != "" {
				fmt.Fprintf(w, "  %s: %v\n", key, value)
			}
		}
	}
	if report.Metadata.MaxClients > 0 {
		fmt.Fprintf(w, "clients.maxclients: %.0f\n", report.Metadata.MaxClients)
	}
	fmt.Fprintln(w)
}

func renderRows(w io.Writer, rows []derive.Row, preferred []string) error {
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
			value := formatRowValue(row, col)
			if len(value) > widths[col] {
				widths[col] = len(value)
			}
		}
	}
	writeHeader := func() {
		for i, col := range cols {
			if i > 0 {
				fmt.Fprint(w, " ")
			}
			fmt.Fprintf(w, "%*s", widths[col], col)
		}
		fmt.Fprintln(w)
	}
	writeHeader()
	for _, row := range rows {
		if row.ProcessMarker != "" {
			fmt.Fprintln(w, row.ProcessMarker)
		}
		if row.Marker != "" {
			fmt.Fprintf(w, "# %s\n", row.Marker)
		}
		for i, col := range cols {
			if i > 0 {
				fmt.Fprint(w, " ")
			}
			fmt.Fprintf(w, "%*s", widths[col], formatRowValue(row, col))
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

func orderedColumns(rows []derive.Row, preferred []string) []string {
	if len(preferred) > 0 {
		return append([]string(nil), preferred...)
	}
	seen := map[string]bool{}
	var cols []string
	if len(rows) > 0 && !rows[0].Time.IsZero() {
		cols = append(cols, "time")
		seen["time"] = true
	}
	for _, row := range rows {
		var extra []string
		for key := range row.Values {
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

func formatRowValue(row derive.Row, col string) string {
	if col == "time" {
		return row.Time.Format("2006-01-02T15:04:05Z")
	}
	return formatValue(row.Values[col])
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
