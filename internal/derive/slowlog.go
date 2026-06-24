package derive

import (
	"sort"
	"strings"
	"time"

	"valkey-ftdcstat/internal/model"
)

const (
	defaultSlowlogTop = 50
	maxSlowlogArgLen  = 64
	maxArgsDisplayLen = 72
)

type SlowlogRow struct {
	Command     string    `json:"command"`
	Args        []string  `json:"args,omitempty"`
	ArgsDisplay string    `json:"argsDisplay"`
	MaxMs       float64   `json:"maxMs"`
	AvgMs       float64   `json:"avgMs"`
	TotalMs     float64   `json:"totalMs"`
	Count       int       `json:"count"`
	LastSeen    time.Time `json:"lastSeen"`
}

type SlowlogSummary struct {
	UniquePatterns int  `json:"uniquePatterns"`
	TotalEntries   int  `json:"totalEntries"`
	CollectEnabled bool `json:"collectEnabled"`
}

type slowlogOccurrence struct {
	id         int64
	observedAt time.Time
	durationMs float64
	command    string
	args       []string
}

func normalizeSlowlogTop(n int) int {
	if n < 0 {
		return defaultSlowlogTop
	}
	return n
}

func deriveSlowlog(samples []model.MetricSample, topN int, moduleConfig map[string]any) ([]SlowlogRow, SlowlogSummary, string) {
	collectEnabled := moduleConfigBool(moduleConfig, "collect-slowlog")
	occurrences := collectSlowlogOccurrences(samples)
	summary := SlowlogSummary{
		UniquePatterns: 0,
		TotalEntries:   len(occurrences),
		CollectEnabled: collectEnabled,
	}

	var note string
	switch {
	case len(occurrences) == 0 && !collectEnabled:
		note = "collect-slowlog is disabled in capture metadata; enable it in valkey-ftdc to record slowlog entries"
	case len(occurrences) == 0:
		note = "no slowlog entries found in capture window (each sample stores up to 8 recent SLOWLOG entries)"
	default:
		note = "slowlog view aggregates SLOWLOG GET 8 snapshots from each sample; not full slowlog history"
	}

	if len(occurrences) == 0 {
		return nil, summary, note
	}

	rows := aggregateSlowlogOccurrences(occurrences)
	summary.UniquePatterns = len(rows)
	if topN > 0 && len(rows) > topN {
		rows = append([]SlowlogRow(nil), rows[:topN]...)
	}
	return rows, summary, note
}

func collectSlowlogOccurrences(samples []model.MetricSample) []slowlogOccurrence {
	seenIDs := map[int64]struct{}{}
	var out []slowlogOccurrence
	for _, sample := range samples {
		for _, entry := range sample.SlowlogEntries {
			id := int64(entry.ID)
			if id == 0 && entry.DurationUSec == 0 && len(entry.Args) == 0 {
				continue
			}
			if _, ok := seenIDs[id]; ok {
				continue
			}
			seenIDs[id] = struct{}{}
			command, args := splitSlowlogArgs(entry.Args)
			if command == "" {
				continue
			}
			out = append(out, slowlogOccurrence{
				id:         id,
				observedAt: sample.Time,
				durationMs: entry.DurationUSec / 1000.0,
				command:    command,
				args:       args,
			})
		}
	}
	return out
}

func aggregateSlowlogOccurrences(occurrences []slowlogOccurrence) []SlowlogRow {
	type bucket struct {
		command  string
		args     []string
		count    int
		totalMs  float64
		maxMs    float64
		lastSeen time.Time
	}
	byFingerprint := map[string]*bucket{}
	for _, item := range occurrences {
		fp := slowlogFingerprint(item.command, item.args)
		group, ok := byFingerprint[fp]
		if !ok {
			group = &bucket{
				command: item.command,
				args:    append([]string(nil), item.args...),
			}
			byFingerprint[fp] = group
		}
		group.count++
		group.totalMs += item.durationMs
		if item.durationMs > group.maxMs {
			group.maxMs = item.durationMs
		}
		if item.observedAt.After(group.lastSeen) {
			group.lastSeen = item.observedAt
		}
	}

	rows := make([]SlowlogRow, 0, len(byFingerprint))
	for _, group := range byFingerprint {
		avgMs := group.totalMs / float64(group.count)
		rows = append(rows, SlowlogRow{
			Command:     group.command,
			Args:        append([]string(nil), group.args...),
			ArgsDisplay: formatSlowlogArgs(group.args),
			MaxMs:       model.RoundRate(group.maxMs),
			AvgMs:       model.RoundRate(avgMs),
			TotalMs:     model.RoundRate(group.totalMs),
			Count:       group.count,
			LastSeen:    group.lastSeen,
		})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].MaxMs == rows[j].MaxMs {
			if rows[i].Count == rows[j].Count {
				return rows[i].Command < rows[j].Command
			}
			return rows[i].Count > rows[j].Count
		}
		return rows[i].MaxMs > rows[j].MaxMs
	})
	return rows
}

func splitSlowlogArgs(args []string) (string, []string) {
	if len(args) == 0 {
		return "", nil
	}
	command := strings.ToUpper(strings.TrimSpace(args[0]))
	if command == "" {
		return "", nil
	}
	if len(args) == 1 {
		return command, nil
	}
	return command, append([]string(nil), args[1:]...)
}

func slowlogFingerprint(command string, args []string) string {
	parts := make([]string, 0, 1+len(args))
	parts = append(parts, strings.ToUpper(command))
	for _, arg := range args {
		parts = append(parts, normalizeSlowlogArg(arg))
	}
	return strings.Join(parts, "|")
}

func normalizeSlowlogArg(arg string) string {
	arg = strings.Join(strings.Fields(strings.TrimSpace(arg)), " ")
	if len(arg) > maxSlowlogArgLen {
		return arg[:maxSlowlogArgLen-3] + "..."
	}
	return arg
}

func formatSlowlogArgs(args []string) string {
	if len(args) == 0 {
		return "-"
	}
	parts := make([]string, 0, len(args))
	for _, arg := range args {
		parts = append(parts, normalizeSlowlogArg(arg))
	}
	display := strings.Join(parts, " ")
	if len(display) > maxArgsDisplayLen {
		return display[:maxArgsDisplayLen-3] + "..."
	}
	return display
}

func moduleConfigBool(config map[string]any, key string) bool {
	if config == nil {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(model.Text(config, key))) {
	case "yes", "true", "1", "on":
		return true
	default:
		return false
	}
}

func slowlogColumns() []string {
	return []string{"command", "args", "maxMs", "avgMs", "count", "lastSeen"}
}
