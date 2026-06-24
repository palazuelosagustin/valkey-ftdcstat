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
	seen := map[string]struct{}{}
	for path := range sample.Values {
		if !strings.HasPrefix(path, prefix) {
			continue
		}
		rest := strings.TrimPrefix(path, prefix)
		for _, suffix := range []string{".latest_ms", ".max_ms", ".all_time_ms"} {
			if strings.HasSuffix(rest, suffix) {
				event := strings.TrimSuffix(rest, suffix)
				if event != "" {
					seen[event] = struct{}{}
				}
				break
			}
		}
	}
	return sortedKeys(seen)
}

var latencyBaseColumns = []string{"slowlog", "slowMaxMs", "blocked", "forkUsec", "eloopUs"}

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
	cols = append(cols, latencyBaseColumns...)
	for _, event := range events {
		cols = append(cols, event, event+"Max")
	}
	return cols
}
