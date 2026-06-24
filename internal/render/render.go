package render

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"sort"
	"strings"
	"time"

	"valkey-ftdcstat/internal/derive"
	"valkey-ftdcstat/internal/model"
)

func Report(w io.Writer, report derive.Report, jsonOut bool) error {
	if jsonOut {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(report)
	}

	fmt.Fprintln(w, "valkey-ftdcstat report")
	fmt.Fprintf(w, "path:    %s\n", report.Path)
	fmt.Fprintf(w, "files:   %d\n", len(report.Files))
	fmt.Fprintf(w, "samples: %d\n", report.SampleCount)
	fmt.Fprintf(w, "range:   %s .. %s", report.Start.Format("2006-01-02T15:04:05Z07:00"), report.End.Format("2006-01-02T15:04:05Z07:00"))
	if !report.Start.IsZero() && !report.End.IsZero() && report.End.After(report.Start) {
		fmt.Fprintf(w, "  (%s)", formatDuration(report.End.Sub(report.Start)))
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w)
	renderCompactHeader(w, report.Header)

	if report.View == "commandstats" {
		return renderCommands(w, report.Commands)
	}
	if report.LatencyNote != "" {
		fmt.Fprintf(w, "note: %s\n\n", report.LatencyNote)
	}
	return renderRows(w, report.Rows, report.Columns)
}

func renderCompactHeader(w io.Writer, header model.Header) {
	renderServerSection(w, header.BuildInfo, header.ReplicationInfo)
	fmt.Fprintln(w)
	renderTopologySection(w, header.ReplicationInfo)
	fmt.Fprintln(w)
	renderHostSection(w, header.HostInfo)
	fmt.Fprintln(w)
}

func renderServerSection(w io.Writer, buildInfo, replicationInfo map[string]any) {
	role := normalizeRole(stringValue(replicationInfo, "role"))
	mode := stringValue(buildInfo, "serverMode")
	valkeyVersion := stringValue(buildInfo, "valkeyVersion")
	redisVersion := stringValue(buildInfo, "redisVersion")
	buildID := stringValue(buildInfo, "buildID")
	gccVersion := stringValue(buildInfo, "gccVersion")
	osValue := stringValue(buildInfo, "os")
	archBits := int(numberValue(buildInfo, "archBits"))
	multiplexingAPI := stringValue(buildInfo, "multiplexingAPI")
	gitSHA := stringValue(buildInfo, "gitSHA1")
	gitDirty, hasGitDirty := boolValue(buildInfo, "gitDirty")

	fmt.Fprintln(w, "server:")
	first := []string{}
	if valkeyVersion != "" {
		first = append(first, "Valkey "+valkeyVersion)
	}
	if mode != "" {
		first = append(first, mode)
	}
	if role != "" {
		first = append(first, role)
	}
	if len(first) > 0 {
		fmt.Fprintf(w, "  %s\n", strings.Join(first, " | "))
	}
	if redisVersion != "" {
		fmt.Fprintf(w, "  redis_version: %s compatibility\n", redisVersion)
	}
	buildParts := []string{}
	if buildID != "" {
		buildParts = append(buildParts, buildID)
	}
	if gccVersion != "" {
		buildParts = append(buildParts, "gcc "+gccVersion)
	}
	if osValue != "" {
		buildParts = append(buildParts, compactOS(osValue, archBits))
	}
	if multiplexingAPI != "" {
		buildParts = append(buildParts, multiplexingAPI)
	}
	if len(buildParts) > 0 {
		fmt.Fprintf(w, "  build: %s\n", strings.Join(buildParts, ", "))
	}
	if gitSHA != "" || hasGitDirty {
		gitLine := gitSHA
		if gitLine == "" {
			gitLine = "unknown"
		}
		if hasGitDirty {
			if gitDirty {
				gitLine += " dirty"
			} else {
				gitLine += " clean"
			}
		}
		fmt.Fprintf(w, "  git: %s\n", gitLine)
	}
}

func renderTopologySection(w io.Writer, replicationInfo map[string]any) {
	fmt.Fprintln(w, "topology:")
	role := normalizeRole(stringValue(replicationInfo, "role"))
	if role == "" {
		role = "unknown"
	}
	fmt.Fprintf(w, "  role: %s\n", role)
	replicas := int(numberValue(replicationInfo, "replicas"))
	if names := stringSlice(replicationInfo, "replicaNames"); len(names) > 0 {
		fmt.Fprintf(w, "  replicas: %d (%s)\n", replicas, strings.Join(names, ", "))
	} else {
		fmt.Fprintf(w, "  replicas: %d\n", replicas)
	}
	clusterEnabled, _ := boolValue(replicationInfo, "clusterEnabled")
	cluster := "disabled"
	if clusterEnabled {
		cluster = "enabled"
	}
	fmt.Fprintf(w, "  cluster: %s\n", cluster)
}

func renderHostSection(w io.Writer, hostInfo map[string]any) {
	fmt.Fprintln(w, "host:")
	if osValue := stringValue(hostInfo, "os"); osValue != "" {
		fmt.Fprintf(w, "  os: %s\n", osValue)
	}
	memory, _ := hostInfo["memory"].(map[string]string)
	available := model.ParseKBValue(memory["available"])
	total := model.ParseKBValue(memory["total"])
	if available > 0 || total > 0 {
		fmt.Fprintf(w, "  memory: %s available / %s total\n", formatBinaryMB(available), formatBinaryMB(total))
	}
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

func formatDuration(d time.Duration) string {
	seconds := int(d.Seconds())
	if seconds < 0 {
		seconds = 0
	}
	h := seconds / 3600
	m := (seconds % 3600) / 60
	s := seconds % 60
	if h > 0 {
		return fmt.Sprintf("%dh %02dm %02ds", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%dm %02ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}

func normalizeRole(role string) string {
	switch role {
	case "master":
		return "primary"
	case "slave":
		return "replica"
	case "":
		return "unknown"
	default:
		return role
	}
}

func compactOS(osValue string, archBits int) string {
	if osValue == "" {
		return ""
	}
	if archBits <= 0 {
		return osValue
	}
	suffix := fmt.Sprintf(" %d", archBits)
	if strings.HasSuffix(osValue, suffix) {
		return strings.TrimSuffix(osValue, suffix)
	}
	return osValue
}

func formatBinaryMB(mb float64) string {
	if mb <= 0 {
		return "0 B"
	}
	gib := mb / 1024.0
	if gib >= 1 {
		return fmt.Sprintf("%.1f GiB", round1(gib))
	}
	return fmt.Sprintf("%.1f MiB", round1(mb))
}

func round1(v float64) float64 {
	return math.Round(v*10) / 10
}

func stringValue(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	return model.Text(m, key)
}

func numberValue(m map[string]any, key string) float64 {
	if m == nil {
		return 0
	}
	return model.AsFloat(m[key])
}

func boolValue(m map[string]any, key string) (bool, bool) {
	if m == nil {
		return false, false
	}
	value, ok := m[key]
	if !ok {
		return false, false
	}
	b, ok := value.(bool)
	if ok {
		return b, true
	}
	return model.AsFloat(value) != 0, true
}

func stringSlice(m map[string]any, key string) []string {
	if m == nil {
		return nil
	}
	values, ok := m[key].([]string)
	if ok {
		return values
	}
	items, ok := m[key].([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		text := fmt.Sprint(item)
		if text != "" {
			out = append(out, text)
		}
	}
	return out
}
