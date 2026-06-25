package derive

import (
	"strings"

	"valkey-ftdcstat/internal/model"
)

const (
	commandstatsPathPrefix = "valkey.info.commandstats."
	defaultTopCommands     = 8
)

func normalizeTopCommands(n int) int {
	if n < 0 {
		return defaultTopCommands
	}
	return n
}

func displayCommandName(fullName string) string {
	return strings.TrimPrefix(fullName, "cmdstat_")
}

func commandRateColumn(command string) string {
	return displayCommandName(command) + "/s"
}

func fillCommandRates(row *Row, c calculator, reset bool) {
	if reset {
		return
	}
	for path := range c.cur.Values {
		if !strings.HasPrefix(path, commandstatsPathPrefix) || !strings.HasSuffix(path, ".calls") {
			continue
		}
		fullName := strings.TrimPrefix(strings.TrimSuffix(path, ".calls"), commandstatsPathPrefix)
		if fullName == "" {
			continue
		}
		if rate, ok := c.rate(path); ok {
			put(row, commandRateColumn(fullName), rate)
		}
	}
}

func topCommandNames(first, last model.MetricSample, topN int) []string {
	rows := deriveCommands(first, last)
	if len(rows) == 0 {
		return nil
	}
	if topN == 0 || topN >= len(rows) {
		out := make([]string, 0, len(rows))
		for _, row := range rows {
			out = append(out, row.Command)
		}
		return out
	}
	out := make([]string, 0, topN)
	for i := 0; i < len(rows) && i < topN; i++ {
		out = append(out, rows[i].Command)
	}
	return out
}

func commandstatsColumns(commands []string) []string {
	cols := []string{"time"}
	for _, command := range commands {
		cols = append(cols, commandRateColumn("cmdstat_"+command))
	}
	return cols
}

func summaryColumns(topCommands []string, replicaOffsets []string) []string {
	cols := []string{
		"time",
		"ops/s", "conn/s", "hit%",
	}
	for _, command := range topCommands {
		cols = append(cols, commandRateColumn("cmdstat_"+command))
	}
	cols = append(cols,
		"memMB", "rssMB", "frag%",
		"rej/s", "exp/s", "evict/s", "offKB/s", "inKB/s", "outKB/s",
		"cli", "blk",
		"role",
	)
	cols = append(cols, replicaOffsets...)
	cols = append(cols,
		"us%", "sy%", "id%", "wa%", "load1", "availMB",
	)
	return cols
}
