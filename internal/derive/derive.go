package derive

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"valkey-ftdcstat/internal/model"
)

type Options struct {
	View     string
	Interval time.Duration
}

type Report struct {
	View        string           `json:"view"`
	Path        string           `json:"path"`
	Files       []string         `json:"files"`
	SampleCount int              `json:"sampleCount"`
	Start       time.Time        `json:"start"`
	End         time.Time        `json:"end"`
	Columns     []string         `json:"columns,omitempty"`
	Rows        []map[string]any `json:"rows,omitempty"`
	Commands    []CommandRow     `json:"commands,omitempty"`
	Latest      any              `json:"latest,omitempty"`
}

type CommandRow struct {
	Command     string  `json:"command"`
	Calls       float64 `json:"calls"`
	CallsPerSec float64 `json:"callsPerSec"`
	UsecPerCall float64 `json:"usecPerCall"`
	SharePct    float64 `json:"sharePct"`
}

func Build(capture model.Capture, opts Options) (Report, error) {
	if len(capture.Samples) == 0 {
		return Report{}, fmt.Errorf("no samples found")
	}
	report := Report{
		View:        opts.View,
		Path:        capture.Path,
		Files:       append([]string(nil), capture.Files...),
		SampleCount: len(capture.Samples),
		Start:       capture.Samples[0].Time(),
		End:         capture.Samples[len(capture.Samples)-1].Time(),
		Columns:     viewColumns(opts.View),
	}

	switch opts.View {
	case "summary", "memory", "clients", "cpu", "persistence", "replication", "host":
		rows := deriveRows(capture.Samples, opts.Interval, opts.View)
		report.Rows = rows
		report.Latest = latestForView(capture.Samples[len(capture.Samples)-1], opts.View)
		return report, nil
	case "commandstats":
		report.Commands = deriveCommands(capture.Samples)
		report.Latest = capture.Samples[len(capture.Samples)-1].Valkey.Info.Commandstats
		return report, nil
	default:
		return Report{}, fmt.Errorf("unknown view %q", opts.View)
	}
}

func deriveRows(samples []model.Sample, interval time.Duration, view string) []map[string]any {
	if len(samples) < 2 {
		return nil
	}
	var rows []map[string]any
	var lastShown time.Time
	for i := 1; i < len(samples); i++ {
		prev, curr := samples[i-1], samples[i]
		row := buildRow(prev, curr, view)
		if row == nil {
			continue
		}
		ts := curr.Time()
		if lastShown.IsZero() || ts.Sub(lastShown) >= interval {
			rows = append(rows, row)
			lastShown = ts
		}
	}
	return rows
}

func buildRow(prev, curr model.Sample, view string) map[string]any {
	dt := curr.Time().Sub(prev.Time()).Seconds()
	if dt <= 0 {
		return nil
	}

	row := map[string]any{"time": curr.Time().Format(time.RFC3339)}
	statsPrev, statsCurr := prev.Valkey.Info.Stats, curr.Valkey.Info.Stats
	memCurr := curr.Valkey.Info.Memory
	clientsCurr := curr.Valkey.Info.Clients
	replCurr := curr.Valkey.Info.Replication
	cpuPrev, cpuCurr := prev.Valkey.Info.CPU, curr.Valkey.Info.CPU

	opsPerSec := delta(model.Number(statsCurr, "total_commands_processed"), model.Number(statsPrev, "total_commands_processed")) / dt
	connPerSec := delta(model.Number(statsCurr, "total_connections_received"), model.Number(statsPrev, "total_connections_received")) / dt
	expiredPerSec := delta(model.Number(statsCurr, "expired_keys"), model.Number(statsPrev, "expired_keys")) / dt
	evictedPerSec := delta(model.Number(statsCurr, "evicted_keys"), model.Number(statsPrev, "evicted_keys")) / dt
	hitPct := ratioPct(
		delta(model.Number(statsCurr, "keyspace_hits"), model.Number(statsPrev, "keyspace_hits")),
		delta(model.Number(statsCurr, "keyspace_misses"), model.Number(statsPrev, "keyspace_misses")),
	)

	switch view {
	case "summary":
		put(row, "ops/s", opsPerSec)
		put(row, "conn/s", connPerSec)
		put(row, "hit%", hitPct)
		put(row, "memMB", bytesToMB(model.Number(memCurr, "used_memory")))
		put(row, "rssMB", bytesToMB(model.Number(memCurr, "used_memory_rss")))
		put(row, "cli", model.Number(clientsCurr, "connected_clients"))
		put(row, "blk", model.Number(clientsCurr, "blocked_clients"))
		put(row, "inKB/s", model.Number(statsCurr, "instantaneous_input_kbps"))
		put(row, "outKB/s", model.Number(statsCurr, "instantaneous_output_kbps"))
		appendHostCPU(row, prev.Host, curr.Host)
		put(row, "load1", model.Number(curr.Host.LoadAvg, "1m"))
		put(row, "availMB", model.ParseKBValue(curr.Host.Memory["MemAvailable"]))
		row["repl"] = model.Text(replCurr, "role")
		put(row, "repls", model.Number(replCurr, "connected_slaves"))
	case "memory":
		used := bytesToMB(model.Number(memCurr, "used_memory"))
		rss := bytesToMB(model.Number(memCurr, "used_memory_rss"))
		max := bytesToMB(model.Number(memCurr, "maxmemory"))
		put(row, "usedMB", used)
		put(row, "rssMB", rss)
		put(row, "maxMB", max)
		put(row, "rss%", ratio(100*rss, used))
		put(row, "availMB", model.ParseKBValue(curr.Host.Memory["MemAvailable"]))
		put(row, "exp/s", expiredPerSec)
		put(row, "evict/s", evictedPerSec)
	case "clients":
		put(row, "conn", model.Number(clientsCurr, "connected_clients"))
		put(row, "blocked", model.Number(clientsCurr, "blocked_clients"))
		put(row, "conn/s", connPerSec)
		put(row, "ops/s", opsPerSec)
		put(row, "hit%", hitPct)
	case "cpu":
		put(row, "ops/s", opsPerSec)
		put(row, "vkUsr%", delta(model.Number(cpuCurr, "used_cpu_user"), model.Number(cpuPrev, "used_cpu_user"))/dt*100)
		put(row, "vkSys%", delta(model.Number(cpuCurr, "used_cpu_sys"), model.Number(cpuPrev, "used_cpu_sys"))/dt*100)
		appendHostCPU(row, prev.Host, curr.Host)
		put(row, "load1", model.Number(curr.Host.LoadAvg, "1m"))
	case "persistence":
		row["rdb"] = boolish(curr.Valkey.Info.Persistence, "rdb_bgsave_in_progress")
		row["aof"] = boolish(curr.Valkey.Info.Persistence, "aof_enabled")
		put(row, "exp/s", expiredPerSec)
		put(row, "evict/s", evictedPerSec)
		put(row, "slowlog", curr.Valkey.Slowlog.Len)
	case "replication":
		row["role"] = model.Text(replCurr, "role")
		put(row, "replicas", model.Number(replCurr, "connected_slaves"))
		put(row, "offsetMB", bytesToMB(model.Number(replCurr, "master_repl_offset")))
		put(row, "offMB/s", bytesToMB(delta(model.Number(replCurr, "master_repl_offset"), model.Number(prev.Valkey.Info.Replication, "master_repl_offset")))/dt)
	case "host":
		appendHostVMStat(row, prev.Host, curr.Host, dt)
	}
	return row
}

func latestForView(sample model.Sample, view string) any {
	switch view {
	case "summary":
		return map[string]any{
			"server":      sample.Valkey.Info.Server,
			"memory":      sample.Valkey.Info.Memory,
			"stats":       sample.Valkey.Info.Stats,
			"replication": sample.Valkey.Info.Replication,
			"host":        sample.Host,
		}
	case "memory":
		return sample.Valkey.Info.Memory
	case "clients":
		return sample.Valkey.Info.Clients
	case "cpu":
		return map[string]any{"valkey": sample.Valkey.Info.CPU, "host": sample.Host.CPU}
	case "persistence":
		return sample.Valkey.Info.Persistence
	case "replication":
		return sample.Valkey.Info.Replication
	case "host":
		return sample.Host
	default:
		return nil
	}
}

func deriveCommands(samples []model.Sample) []CommandRow {
	if len(samples) < 2 {
		return nil
	}
	first, last := samples[0], samples[len(samples)-1]
	dt := last.Time().Sub(first.Time()).Seconds()
	if dt <= 0 {
		return nil
	}

	totalCalls := 0.0
	for name, curr := range last.Valkey.Info.Commandstats {
		prev := first.Valkey.Info.Commandstats[name]
		totalCalls += delta(curr.Calls, prev.Calls)
	}

	rows := make([]CommandRow, 0, len(last.Valkey.Info.Commandstats))
	for name, curr := range last.Valkey.Info.Commandstats {
		prev := first.Valkey.Info.Commandstats[name]
		calls := delta(curr.Calls, prev.Calls)
		if calls <= 0 {
			continue
		}
		rows = append(rows, CommandRow{
			Command:     strings.TrimPrefix(name, "cmdstat_"),
			Calls:       calls,
			CallsPerSec: calls / dt,
			UsecPerCall: curr.UsecPerCall,
			SharePct:    ratioPct(calls, totalCalls-calls),
		})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].CallsPerSec == rows[j].CallsPerSec {
			return rows[i].Command < rows[j].Command
		}
		return rows[i].CallsPerSec > rows[j].CallsPerSec
	})
	return rows
}

func appendHostCPU(row map[string]any, prev, curr model.HostMetrics) {
	currCPU, prevCPU := curr.CPU, prev.CPU
	totalDelta := 0.0
	fields := []string{"user", "nice", "system", "idle", "iowait", "irq", "softirq", "steal", "guest", "guest_nice"}
	for _, field := range fields {
		totalDelta += delta(model.Number(currCPU, field), model.Number(prevCPU, field))
	}
	if totalDelta <= 0 {
		return
	}
	put(row, "us%", delta(model.Number(currCPU, "user"), model.Number(prevCPU, "user"))/totalDelta*100)
	put(row, "sy%", delta(model.Number(currCPU, "system"), model.Number(prevCPU, "system"))/totalDelta*100)
	put(row, "id%", delta(model.Number(currCPU, "idle"), model.Number(prevCPU, "idle"))/totalDelta*100)
	put(row, "wa%", delta(model.Number(currCPU, "iowait"), model.Number(prevCPU, "iowait"))/totalDelta*100)
}

func appendHostVMStat(row map[string]any, prev, curr model.HostMetrics, dt float64) {
	put(row, "r", model.Number(curr.CPU, "procs_running"))
	put(row, "b", model.Number(curr.CPU, "procs_blocked"))
	swapTotal := model.ParseKBValue(curr.Memory["SwapTotal"])
	swapFree := model.ParseKBValue(curr.Memory["SwapFree"])
	put(row, "swpd", max(0, swapTotal-swapFree))
	put(row, "free", model.ParseKBValue(curr.Memory["MemFree"]))
	put(row, "buff", model.ParseKBValue(curr.Memory["Buffers"]))
	put(row, "cache", model.ParseKBValue(curr.Memory["Cached"]))

	rdPrev, wrPrev := parseDiskstats(prev.Disk.Diskstats)
	rdCurr, wrCurr := parseDiskstats(curr.Disk.Diskstats)
	put(row, "bi", delta(rdCurr, rdPrev)/dt)
	put(row, "bo", delta(wrCurr, wrPrev)/dt)
	put(row, "forks/s", delta(model.Number(curr.CPU, "processes"), model.Number(prev.CPU, "processes"))/dt)
	put(row, "cs/s", delta(model.Number(curr.CPU, "ctxt"), model.Number(prev.CPU, "ctxt"))/dt)
	appendHostCPU(row, prev, curr)
	put(row, "st%", delta(model.Number(curr.CPU, "steal"), model.Number(prev.CPU, "steal"))/hostTotalDelta(prev.CPU, curr.CPU)*100)
	put(row, "load1", model.Number(curr.LoadAvg, "1m"))
}

func parseDiskstats(blob string) (readKB, writeKB float64) {
	lines := strings.Split(blob, "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 14 {
			continue
		}
		readSectors := parseFloat(fields[5])
		writeSectors := parseFloat(fields[9])
		readKB += readSectors / 2.0
		writeKB += writeSectors / 2.0
	}
	return readKB, writeKB
}

func hostTotalDelta(prev, curr map[string]any) float64 {
	fields := []string{"user", "nice", "system", "idle", "iowait", "irq", "softirq", "steal", "guest", "guest_nice"}
	total := 0.0
	for _, field := range fields {
		total += delta(model.Number(curr, field), model.Number(prev, field))
	}
	return total
}

func delta(curr, prev float64) float64 {
	if curr < prev {
		return 0
	}
	return curr - prev
}

func ratioPct(a, b float64) float64 {
	denom := a + b
	if denom <= 0 {
		return 0
	}
	return a / denom * 100
}

func ratio(a, b float64) float64 {
	if b <= 0 {
		return 0
	}
	return a / b
}

func bytesToMB(v float64) float64 {
	return v / (1024.0 * 1024.0)
}

func parseFloat(s string) float64 {
	var out float64
	fmt.Sscanf(s, "%f", &out)
	return out
}

func put(row map[string]any, key string, value float64) {
	row[key] = round(value)
}

func round(v float64) float64 {
	if v == 0 {
		return 0
	}
	return float64(int(v*100+0.5)) / 100
}

func boolish(m map[string]any, key string) string {
	if model.Number(m, key) != 0 {
		return "yes"
	}
	return "no"
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func viewColumns(view string) []string {
	switch view {
	case "summary":
		return []string{"time", "ops/s", "conn/s", "hit%", "memMB", "rssMB", "cli", "blk", "inKB/s", "outKB/s", "us%", "sy%", "id%", "wa%", "load1", "availMB", "repl", "repls"}
	case "memory":
		return []string{"time", "usedMB", "rssMB", "maxMB", "rss%", "availMB", "exp/s", "evict/s"}
	case "clients":
		return []string{"time", "conn", "blocked", "conn/s", "ops/s", "hit%"}
	case "cpu":
		return []string{"time", "ops/s", "vkUsr%", "vkSys%", "us%", "sy%", "id%", "wa%", "load1"}
	case "persistence":
		return []string{"time", "rdb", "aof", "exp/s", "evict/s", "slowlog"}
	case "replication":
		return []string{"time", "role", "replicas", "offsetMB", "offMB/s"}
	case "host":
		return []string{"time", "r", "b", "swpd", "free", "buff", "cache", "bi", "bo", "forks/s", "cs/s", "us%", "sy%", "id%", "wa%", "st%", "load1"}
	default:
		return []string{"time"}
	}
}
