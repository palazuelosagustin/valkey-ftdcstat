package derive

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"valkey-ftdcstat/internal/discovery"
	"valkey-ftdcstat/internal/model"
	"valkey-ftdcstat/internal/reader"
)

const (
	pathStatsCmds       = "valkey.info.stats.total_commands_processed"
	pathStatsConns      = "valkey.info.stats.total_connections_received"
	pathStatsHits       = "valkey.info.stats.keyspace_hits"
	pathStatsMisses     = "valkey.info.stats.keyspace_misses"
	pathStatsInKB       = "valkey.info.stats.instantaneous_input_kbps"
	pathStatsOutKB      = "valkey.info.stats.instantaneous_output_kbps"
	pathStatsExpired    = "valkey.info.stats.expired_keys"
	pathStatsEvicted    = "valkey.info.stats.evicted_keys"
	pathMemUsed         = "valkey.info.memory.used_memory"
	pathMemRSS          = "valkey.info.memory.used_memory_rss"
	pathMemMax          = "valkey.info.memory.maxmemory"
	pathClientsConn     = "valkey.info.clients.connected_clients"
	pathClientsBlocked  = "valkey.info.clients.blocked_clients"
	pathReplRole        = "valkey.info.replication.role"
	pathReplSlaves      = "valkey.info.replication.connected_slaves"
	pathReplOffset      = "valkey.info.replication.master_repl_offset"
	pathCPUUser         = "valkey.info.cpu.used_cpu_user"
	pathCPUSys          = "valkey.info.cpu.used_cpu_sys"
	pathRDB             = "valkey.info.persistence.rdb_bgsave_in_progress"
	pathAOF             = "valkey.info.persistence.aof_enabled"
	pathSlowlogLen      = "valkey.slowlog.len"
	pathHostSupported   = "host.supported"
	pathHostLoad1       = "host.loadavg.1m"
	pathHostMemAvail    = "host.memory.MemAvailable.mb"
	pathHostDiskstats   = "host.disk.diskstats"
	pathProcessID       = "valkey.info.server.process_id"
	pathRunID           = "valkey.info.server.run_id"
	pathUptime          = "valkey.info.server.uptime_in_seconds"
)

type Options struct {
	View           string
	Interval       time.Duration
	GapThreshold   time.Duration
	Metadata       model.Metadata
	TimeLocation   *time.Location
}

type Row struct {
	Time          time.Time      `json:"time"`
	Marker        string         `json:"marker,omitempty"`
	ProcessMarker string         `json:"processMarker,omitempty"`
	Values        map[string]any `json:"values"`
}

type Report struct {
	View        string           `json:"view"`
	Path        string           `json:"path"`
	Files       []string         `json:"files"`
	SampleCount int              `json:"sampleCount"`
	Start       time.Time        `json:"start"`
	End         time.Time        `json:"end"`
	Columns     []string         `json:"columns,omitempty"`
	Rows        []Row            `json:"rows,omitempty"`
	Commands    []CommandRow     `json:"commands,omitempty"`
	Metadata    model.Metadata   `json:"metadata,omitempty"`
	Latest      map[string]any   `json:"latest,omitempty"`
}

type CommandRow struct {
	Command     string  `json:"command"`
	Calls       float64 `json:"calls"`
	CallsPerSec float64 `json:"callsPerSec"`
	UsecPerCall float64 `json:"usecPerCall"`
	SharePct    float64 `json:"sharePct"`
}

func Build(capture model.Capture, opts Options) (Report, error) {
	if len(capture.MetricSamples) == 0 {
		return Report{}, fmt.Errorf("no samples found")
	}
	opts = normalizeOptions(opts, capture.Metadata)
	return buildFromSamples(capture.Path, capture.Files, capture.Metadata, capture.MetricSamples, opts)
}

func BuildStream(path string, files []discovery.MetricFile, metadata model.Metadata, opts Options, stream func(func(model.MetricSample) error) error) (Report, error) {
	opts = normalizeOptions(opts, metadata)
	streamer := NewStreamer(opts)
	var rows []Row
	var samples []model.MetricSample
	var first, last model.MetricSample
	var haveFirst bool

	err := stream(func(sample model.MetricSample) error {
		samples = append(samples, sample)
		if !haveFirst {
			first = sample
			haveFirst = true
		}
		last = sample
		if row, ok := streamer.Add(sample); ok {
			rows = append(rows, row)
		}
		return nil
	})
	if err != nil {
		return Report{}, err
	}
	if len(samples) == 0 {
		return Report{}, fmt.Errorf("no samples found")
	}
	return finalizeReport(path, filePaths(files), metadata, samples, first, last, rows, opts)
}

func BuildFromReader(path string, files []discovery.MetricFile, metadata model.Metadata, opts Options) (Report, error) {
	opts = normalizeOptions(opts, metadata)
	var rows []Row
	var samples []model.MetricSample
	var first, last model.MetricSample
	var haveFirst bool
	streamer := NewStreamer(opts)

	_, streamWarnings, err := reader.StreamSamples(files, reader.StreamOptions{}, func(sample model.MetricSample) error {
		samples = append(samples, sample)
		if !haveFirst {
			first = sample
			haveFirst = true
		}
		last = sample
		if row, ok := streamer.Add(sample); ok {
			rows = append(rows, row)
		}
		return nil
	})
	if err != nil {
		return Report{}, err
	}
	_ = streamWarnings
	if len(samples) == 0 {
		return Report{}, fmt.Errorf("no samples found")
	}
	return finalizeReport(path, filePaths(files), metadata, samples, first, last, rows, opts)
}

func buildFromSamples(path string, files []string, metadata model.Metadata, samples []model.MetricSample, opts Options) (Report, error) {
	streamer := NewStreamer(opts)
	var rows []Row
	for _, sample := range samples {
		if row, ok := streamer.Add(sample); ok {
			rows = append(rows, row)
		}
	}
	return finalizeReport(path, files, metadata, samples, samples[0], samples[len(samples)-1], rows, opts)
}

func finalizeReport(path string, files []string, metadata model.Metadata, samples []model.MetricSample, first, last model.MetricSample, rows []Row, opts Options) (Report, error) {
	report := Report{
		View:        opts.View,
		Path:        path,
		Files:       append([]string(nil), files...),
		SampleCount: len(samples),
		Start:       samples[0].Time,
		End:         samples[len(samples)-1].Time,
		Columns:     viewColumns(opts.View),
		Metadata:    metadata,
	}
	switch opts.View {
	case "commandstats":
		report.Commands = deriveCommands(first, last)
		report.Latest = latestMap(last, opts.View)
		return report, nil
	default:
		report.Rows = rows
		report.Latest = latestMap(last, opts.View)
		return report, nil
	}
}

type Streamer struct {
	opts           Options
	lastRendered   time.Time
	printedProcess bool
	prev           model.MetricSample
	havePrev       bool
}

func NewStreamer(opts Options) *Streamer {
	opts = normalizeOptions(opts, opts.Metadata)
	return &Streamer{opts: opts}
}

func (s *Streamer) Add(cur model.MetricSample) (Row, bool) {
	if !s.havePrev {
		if cur.Time.IsZero() {
			return Row{}, false
		}
		s.prev = cur
		s.havePrev = true
		return Row{}, false
	}
	prev := s.prev
	s.prev = cur
	if cur.Time.IsZero() || prev.Time.IsZero() || !cur.Time.After(prev.Time) {
		return Row{}, false
	}
	if !s.lastRendered.IsZero() && cur.Time.Sub(s.lastRendered) < s.opts.Interval {
		return Row{}, false
	}

	calc := calculator{prev: prev, cur: cur, dt: cur.Time.Sub(prev.Time).Seconds()}
	row := Row{Time: cur.Time, Values: map[string]any{}}
	restarted := processRestart(prev, cur)
	if cur.Time.Sub(prev.Time) > s.opts.GapThreshold {
		row.Marker = fmt.Sprintf("gap %.0fs: rate baseline reset", cur.Time.Sub(prev.Time).Seconds())
	}
	if !s.printedProcess {
		row.ProcessMarker = processMarker("process", cur, s.opts.TimeLocation)
		s.printedProcess = row.ProcessMarker != ""
	}
	if restarted {
		row.ProcessMarker = processMarker("restart detected", cur, s.opts.TimeLocation)
	}
	reset := row.Marker != "" || restarted
	fillRow(&row, calc, s.opts.View, reset)
	s.lastRendered = cur.Time
	return row, true
}

type calculator struct {
	prev model.MetricSample
	cur  model.MetricSample
	dt   float64
}

func (c calculator) current(path string) (float64, bool) {
	return c.cur.Get(path)
}

func (c calculator) text(path string) string {
	return c.cur.GetText(path)
}

func (c calculator) rate(path string) (float64, bool) {
	prev, ok := c.prev.Get(path)
	if !ok {
		return 0, false
	}
	cur, ok := c.cur.Get(path)
	if !ok || c.dt <= 0 || cur < prev {
		return 0, false
	}
	return (cur - prev) / c.dt, true
}

func (c calculator) delta(path string) (float64, bool) {
	prev, ok := c.prev.Get(path)
	if !ok {
		return 0, false
	}
	cur, ok := c.cur.Get(path)
	if !ok || cur < prev {
		return 0, false
	}
	return cur - prev, true
}

func fillRow(row *Row, c calculator, view string, reset bool) {
	switch view {
	case "summary":
		fillSummary(row, c, reset)
	case "memory":
		fillMemory(row, c, reset)
	case "clients":
		fillClients(row, c, reset)
	case "cpu":
		fillCPU(row, c, reset)
	case "persistence":
		fillPersistence(row, c, reset)
	case "replication":
		fillReplication(row, c, reset)
	case "host":
		fillHost(row, c, reset)
	}
}

func fillSummary(row *Row, c calculator, reset bool) {
	if !reset {
		setRate(row, "ops/s", c, pathStatsCmds)
		setRate(row, "conn/s", c, pathStatsConns)
		setHitPct(row, c)
	}
	setGauge(row, "memMB", c, pathMemUsed, bytesToMB)
	setGauge(row, "rssMB", c, pathMemRSS, bytesToMB)
	setGauge(row, "cli", c, pathClientsConn, identity)
	setGauge(row, "blk", c, pathClientsBlocked, identity)
	setGauge(row, "inKB/s", c, pathStatsInKB, identity)
	setGauge(row, "outKB/s", c, pathStatsOutKB, identity)
	appendHostCPU(row, c)
	setGauge(row, "load1", c, pathHostLoad1, identity)
	setGauge(row, "availMB", c, pathHostMemAvail, identity)
	row.Values["repl"] = c.text(pathReplRole)
	setGauge(row, "repls", c, pathReplSlaves, identity)
}

func fillMemory(row *Row, c calculator, reset bool) {
	setGauge(row, "usedMB", c, pathMemUsed, bytesToMB)
	setGauge(row, "rssMB", c, pathMemRSS, bytesToMB)
	setGauge(row, "maxMB", c, pathMemMax, bytesToMB)
	if used, ok := c.current(pathMemUsed); ok {
		if rss, ok := c.current(pathMemRSS); ok && used > 0 {
			put(row, "rss%", ratio(100*bytesToMB(rss), bytesToMB(used)))
		}
	}
	setGauge(row, "availMB", c, pathHostMemAvail, identity)
	if !reset {
		setRate(row, "exp/s", c, pathStatsExpired)
		setRate(row, "evict/s", c, pathStatsEvicted)
	}
}

func fillClients(row *Row, c calculator, reset bool) {
	setGauge(row, "conn", c, pathClientsConn, identity)
	setGauge(row, "blocked", c, pathClientsBlocked, identity)
	if !reset {
		setRate(row, "conn/s", c, pathStatsConns)
		setRate(row, "ops/s", c, pathStatsCmds)
		setHitPct(row, c)
	}
}

func fillCPU(row *Row, c calculator, reset bool) {
	if !reset {
		setRate(row, "ops/s", c, pathStatsCmds)
		if du, ok := c.delta(pathCPUUser); ok {
			put(row, "vkUsr%", du/c.dt*100)
		}
		if ds, ok := c.delta(pathCPUSys); ok {
			put(row, "vkSys%", ds/c.dt*100)
		}
	}
	appendHostCPUFromCalc(row, c)
	setGauge(row, "load1", c, pathHostLoad1, identity)
}

func fillPersistence(row *Row, c calculator, reset bool) {
	row.Values["rdb"] = boolishGauge(c, pathRDB)
	row.Values["aof"] = boolishGauge(c, pathAOF)
	if !reset {
		setRate(row, "exp/s", c, pathStatsExpired)
		setRate(row, "evict/s", c, pathStatsEvicted)
	}
	setGauge(row, "slowlog", c, pathSlowlogLen, identity)
}

func fillReplication(row *Row, c calculator, reset bool) {
	row.Values["role"] = c.text(pathReplRole)
	setGauge(row, "replicas", c, pathReplSlaves, identity)
	setGauge(row, "offsetMB", c, pathReplOffset, bytesToMB)
	if !reset {
		if delta, ok := c.delta(pathReplOffset); ok {
			put(row, "offMB/s", bytesToMB(delta)/c.dt)
		}
	}
}

func fillHost(row *Row, c calculator, reset bool) {
	setGauge(row, "r", c, "host.cpu.procs_running", identity)
	setGauge(row, "b", c, "host.cpu.procs_blocked", identity)
	swapTotal, _ := c.current("host.memory.SwapTotal.mb")
	swapFree, _ := c.current("host.memory.SwapFree.mb")
	put(row, "swpd", max(0, swapTotal-swapFree))
	setGauge(row, "free", c, "host.memory.MemFree.mb", identity)
	setGauge(row, "buff", c, "host.memory.Buffers.mb", identity)
	setGauge(row, "cache", c, "host.memory.Cached.mb", identity)
	if !reset {
		rdPrev, wrPrev := parseDiskstats(c.prev.GetText(pathHostDiskstats))
		rdCurr, wrCurr := parseDiskstats(c.cur.GetText(pathHostDiskstats))
		put(row, "bi", delta(rdCurr, rdPrev)/c.dt)
		put(row, "bo", delta(wrCurr, wrPrev)/c.dt)
		setRate(row, "forks/s", c, "host.cpu.processes")
		setRate(row, "cs/s", c, "host.cpu.ctxt")
	}
	appendHostCPUFromCalc(row, c)
	if total := hostTotalDelta(c.prev, c.cur); total > 0 {
		if ds, ok := c.delta("host.cpu.steal"); ok {
			put(row, "st%", ds/total*100)
		}
	}
	setGauge(row, "load1", c, pathHostLoad1, identity)
}

func deriveCommands(first, last model.MetricSample) []CommandRow {
	dt := last.Time.Sub(first.Time).Seconds()
	if dt <= 0 {
		return nil
	}
	prefix := "valkey.info.commandstats."
	totalCalls := 0.0
	type cmdDelta struct {
		name string
		calls float64
		usecPerCall float64
	}
	var deltas []cmdDelta
	for path, lastCalls := range last.Values {
		if !strings.HasPrefix(path, prefix) || !strings.HasSuffix(path, ".calls") {
			continue
		}
		name := strings.TrimPrefix(path, prefix)
		name = strings.TrimSuffix(name, ".calls")
		prevCalls, _ := first.Get(path)
		calls := delta(lastCalls, prevCalls)
		if calls <= 0 {
			continue
		}
		usecPerCall, _ := last.Get(prefix + name + ".usec_per_call")
		totalCalls += calls
		deltas = append(deltas, cmdDelta{name: name, calls: calls, usecPerCall: usecPerCall})
	}
	rows := make([]CommandRow, 0, len(deltas))
	for _, item := range deltas {
		rows = append(rows, CommandRow{
			Command:     strings.TrimPrefix(item.name, "cmdstat_"),
			Calls:       item.calls,
			CallsPerSec: item.calls / dt,
			UsecPerCall: item.usecPerCall,
			SharePct:    ratioPct(item.calls, totalCalls-item.calls),
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

func appendHostCPU(row *Row, c calculator) {
	if supported, ok := c.current(pathHostSupported); ok && supported == 0 {
		return
	}
	appendHostCPUFromCalc(row, c)
}

func appendHostCPUFromCalc(row *Row, c calculator) {
	total := hostTotalDelta(c.prev, c.cur)
	if total <= 0 {
		return
	}
	if du, ok := c.delta("host.cpu.user"); ok {
		put(row, "us%", du/total*100)
	}
	if ds, ok := c.delta("host.cpu.system"); ok {
		put(row, "sy%", ds/total*100)
	}
	if di, ok := c.delta("host.cpu.idle"); ok {
		put(row, "id%", di/total*100)
	}
	if dw, ok := c.delta("host.cpu.iowait"); ok {
		put(row, "wa%", dw/total*100)
	}
}

func hostTotalDelta(prev, cur model.MetricSample) float64 {
	fields := []string{"user", "nice", "system", "idle", "iowait", "irq", "softirq", "steal", "guest", "guest_nice"}
	total := 0.0
	for _, field := range fields {
		path := "host.cpu." + field
		if d, ok := deltaOK(cur, prev, path); ok {
			total += d
		}
	}
	return total
}

func deltaOK(cur, prev model.MetricSample, path string) (float64, bool) {
	p, ok := prev.Get(path)
	if !ok {
		return 0, false
	}
	c, ok := cur.Get(path)
	if !ok || c < p {
		return 0, false
	}
	return c - p, true
}

func processRestart(prev, cur model.MetricSample) bool {
	if p, ok := prev.Get(pathUptime); ok {
		if c, ok := cur.Get(pathUptime); ok && c < p {
			return true
		}
	}
	if p, ok := prev.Get(pathProcessID); ok {
		if c, ok := cur.Get(pathProcessID); ok && c != p {
			return true
		}
	}
	if pr := prev.GetText(pathRunID); pr != "" {
		if cr := cur.GetText(pathRunID); cr != "" && cr != pr {
			return true
		}
	}
	return false
}

func processMarker(event string, sample model.MetricSample, loc *time.Location) string {
	pid := "-"
	if v, ok := sample.Get(pathProcessID); ok {
		pid = fmt.Sprintf("%.0f", v)
	}
	runID := sample.GetText(pathRunID)
	if pid == "-" && runID == "" {
		return ""
	}
	if loc == nil {
		loc = time.UTC
	}
	return fmt.Sprintf("--- valkey %s: pid=%s run_id=%s time=%s ---", event, pid, valueOrDash(runID), sample.Time.In(loc).Format(time.RFC3339))
}

func valueOrDash(v string) string {
	if v == "" {
		return "-"
	}
	return v
}

func latestMap(sample model.MetricSample, view string) map[string]any {
	out := map[string]any{"time": sample.Time.Format(time.RFC3339)}
	switch view {
	case "summary":
		out["repl"] = sample.GetText(pathReplRole)
		out["memMB"] = bytesToMB(get(sample, pathMemUsed))
		out["cli"] = get(sample, pathClientsConn)
	case "memory":
		out["usedMB"] = bytesToMB(get(sample, pathMemUsed))
	case "clients":
		out["conn"] = get(sample, pathClientsConn)
	case "cpu":
		out["load1"] = get(sample, pathHostLoad1)
	case "persistence":
		out["slowlog"] = get(sample, pathSlowlogLen)
	case "replication":
		out["role"] = sample.GetText(pathReplRole)
	case "host":
		out["load1"] = get(sample, pathHostLoad1)
	case "commandstats":
		var cmds []string
		for path := range sample.Values {
			if strings.HasPrefix(path, "valkey.info.commandstats.") && strings.HasSuffix(path, ".calls") {
				cmds = append(cmds, strings.TrimPrefix(strings.TrimSuffix(path, ".calls"), "valkey.info.commandstats."))
			}
		}
		sort.Strings(cmds)
		out["commands"] = cmds
	}
	return out
}

func get(sample model.MetricSample, path string) float64 {
	v, _ := sample.Get(path)
	return v
}

type valueTransform func(float64) float64

func identity(v float64) float64 { return v }

func setRate(row *Row, key string, c calculator, path string) {
	if rate, ok := c.rate(path); ok {
		put(row, key, rate)
	}
}

func setHitPct(row *Row, c calculator) {
	hits, okH := c.delta(pathStatsHits)
	misses, okM := c.delta(pathStatsMisses)
	if okH || okM {
		put(row, "hit%", ratioPct(hits, misses))
	}
}

func setGauge(row *Row, key string, c calculator, path string, transform valueTransform) {
	if v, ok := c.current(path); ok {
		put(row, key, transform(v))
	}
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

func normalizeOptions(opts Options, metadata model.Metadata) Options {
	if opts.Interval <= 0 {
		opts.Interval = time.Minute
	}
	if opts.GapThreshold <= 0 {
		seconds := int(opts.Interval / time.Second)
		if seconds < 60 {
			seconds = 60
		}
		gapSeconds := seconds * 10
		if gapSeconds < 60 {
			gapSeconds = 60
		}
		opts.GapThreshold = time.Duration(gapSeconds) * time.Second
	}
	if opts.TimeLocation == nil {
		opts.TimeLocation = time.UTC
	}
	if opts.Metadata.Path == "" {
		opts.Metadata = metadata
	}
	return opts
}

func filePaths(files []discovery.MetricFile) []string {
	out := make([]string, 0, len(files))
	for _, file := range discovery.MetricFiles(files) {
		out = append(out, file.Path)
	}
	return out
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

func put(row *Row, key string, value float64) {
	row.Values[key] = model.RoundRate(value)
}

func boolishGauge(c calculator, path string) string {
	if v, ok := c.current(path); ok && v != 0 {
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
