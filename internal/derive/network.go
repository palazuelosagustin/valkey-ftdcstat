package derive

import (
	"sort"
	"strings"

	"valkey-ftdcstat/internal/model"
)

func parseNetDev(blob string) (rxKB, txKB float64) {
	lines := strings.Split(blob, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.Contains(line, "|") || strings.HasPrefix(line, "Inter-") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 10 {
			continue
		}
		name := strings.TrimSuffix(fields[0], ":")
		if name == "lo" {
			continue
		}
		rxKB += parseFloat(fields[1]) / 1024.0
		txKB += parseFloat(fields[9]) / 1024.0
	}
	return rxKB, txKB
}

func latencyEventNames(sample model.MetricSample) []string {
	prefix := "valkey.latency_latest."
	suffix := ".latest_ms"
	seen := map[string]struct{}{}
	for path := range sample.Values {
		if !strings.HasPrefix(path, prefix) || !strings.HasSuffix(path, suffix) {
			continue
		}
		event := strings.TrimSuffix(strings.TrimPrefix(path, prefix), suffix)
		if event != "" {
			seen[event] = struct{}{}
		}
	}
	return sortedKeys(seen)
}

func mergeLatencyEvents(into map[string]struct{}, sample model.MetricSample) {
	if into == nil {
		return
	}
	for _, event := range latencyEventNames(sample) {
		into[event] = struct{}{}
	}
}

func sortedLatencyEvents(events map[string]struct{}) []string {
	return sortedKeys(events)
}

func sortedKeys(events map[string]struct{}) []string {
	out := make([]string, 0, len(events))
	for event := range events {
		out = append(out, event)
	}
	sortStrings(out)
	return out
}

func sortStrings(values []string) {
	sort.Strings(values)
}

func latencyColumns(events []string) []string {
	cols := []string{"time"}
	cols = append(cols, events...)
	return cols
}
