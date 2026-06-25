package derive

import (
	"testing"
	"time"

	"valkey-ftdcstat/internal/flatten"
	"valkey-ftdcstat/internal/model"
)

func TestBuildSummaryRows(t *testing.T) {
	samples := []model.MetricSample{
		metricSample(0, 1000, 2000, 5000, 1000, 100<<20, 150<<20, 10, 1, 1.0, 1000, 500, 10000, 100, 5000, 100, "master", 2, nil),
		metricSample(60_000, 1120, 2600, 5600, 1120, 110<<20, 160<<20, 12, 0, 1.2, 1100, 540, 10300, 110, 5600, 110, "master", 2, nil),
	}
	report, err := Build(model.Capture{MetricSamples: samples}, Options{View: "summary", Interval: time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Rows) != 1 {
		t.Fatalf("rows=%d", len(report.Rows))
	}
	row := report.Rows[0]
	if row.Values["ops/s"] != float64(10) {
		t.Fatalf("ops/s=%v", row.Values["ops/s"])
	}
	if row.Values["conn/s"] != float64(2) {
		t.Fatalf("conn/s=%v", row.Values["conn/s"])
	}
	if row.Values["role"] != "master" {
		t.Fatalf("role=%v", row.Values["role"])
	}
}

func TestBuildHeaderIncludesRawDetails(t *testing.T) {
	samples := []model.MetricSample{
		metricSample(0, 1000, 2000, 5000, 1000, 100<<20, 150<<20, 10, 1, 1.0, 1000, 500, 10000, 100, 5000, 100, "master", 2, []string{"node-a", "node-b"}),
		metricSample(60_000, 1120, 2600, 5600, 1120, 110<<20, 160<<20, 12, 0, 1.2, 1100, 540, 10300, 110, 5600, 110, "master", 2, []string{"node-a", "node-b"}),
	}
	report, err := Build(model.Capture{MetricSamples: samples}, Options{View: "summary", Interval: time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	if got := report.Header.BuildInfo["valkeyVersion"]; got != "9.1.0" {
		t.Fatalf("valkeyVersion=%v", got)
	}
	if got := report.Header.HostInfo["os"]; got != "Linux 6.8.0-110-generic x86_64" {
		t.Fatalf("os=%v", got)
	}
	memory, ok := report.Header.HostInfo["memory"].(map[string]string)
	if !ok || memory["buffers"] != "128000 kB" || memory["cached"] != "512000 kB" || memory["free"] != "4096000 kB" {
		t.Fatalf("memory=%v", report.Header.HostInfo["memory"])
	}
	cpu, ok := report.Header.HostInfo["cpu"].(map[string]any)
	if !ok || cpu["procs_running"] != float64(2) || cpu["procs_blocked"] != float64(1) {
		t.Fatalf("cpu=%v", report.Header.HostInfo["cpu"])
	}
	names, ok := report.Header.ReplicationInfo["replicaNames"].([]string)
	if !ok || len(names) != 2 || names[0] != "node-a" || names[1] != "node-b" {
		t.Fatalf("replicaNames=%v", report.Header.ReplicationInfo["replicaNames"])
	}
}

func TestStreamerDetectsRestart(t *testing.T) {
	first := metricSample(0, 0, 100, 0, 0, 1<<20, 2<<20, 1, 0, 0, 0, 0, 1000, 0, 0, 0, "master", 0, nil)
	first.Text[pathRunID] = "abc"
	first.Values[pathProcessID] = 100
	first.Values[pathUptime] = 3600
	second := metricSample(60_000, 0, 200, 0, 0, 1<<20, 2<<20, 1, 0, 0, 0, 0, 1000, 0, 0, 0, "master", 0, nil)
	second.Text[pathRunID] = "def"
	second.Values[pathProcessID] = 200
	second.Values[pathUptime] = 10

	streamer := NewStreamer(Options{View: "summary", Interval: time.Minute})
	if _, ok := streamer.Add(first); ok {
		t.Fatal("first sample should not emit row")
	}
	row, ok := streamer.Add(second)
	if !ok {
		t.Fatal("expected row")
	}
	if row.ProcessMarker == "" {
		t.Fatal("expected restart marker")
	}
}

func metricSample(ts int64, conns, cmds, hits, misses, used, rss int64, clients, blocked int, load float64, user, sys, idle, iowait, ctxt, procs int64, role string, replicas int, replicaNames []string) model.MetricSample {
	replication := map[string]any{
		"role":               role,
		"connected_slaves":   float64(replicas),
		"master_repl_offset": float64(1 << 20),
	}
	for i := 0; i < replicas; i++ {
		entry := map[string]any{
			"ip":    "127.0.0.1",
			"port":  float64(6001 + i),
			"state": "online",
		}
		if i < len(replicaNames) {
			entry["name"] = replicaNames[i]
		}
		replication["slave"+string(rune('0'+i))] = entry
	}
	return flatten.Sample(model.Sample{
		TsMS: ts,
		Valkey: model.ValkeyMetrics{
			Info: model.InfoSections{
				Server: map[string]any{
					"valkey_version":       "9.1.0",
					"redis_version":        "7.2.4",
					"redis_build_id":       "7f033bf934c6ae79",
					"gcc_version":          "13.3.0",
					"os":                   "Linux 6.8.0-110-generic x86_64",
					"arch_bits":            64,
					"multiplexing_api":     "epoll",
					"server_mode":          "standalone",
					"redis_git_sha1":       "00000000",
					"redis_git_dirty":      1,
					"valkey_release_stage": "ga",
					"process_id":           float64(1),
					"run_id":               role + "-run",
				},
				Clients: map[string]any{"connected_clients": float64(clients), "blocked_clients": float64(blocked)},
				Memory:  map[string]any{"used_memory": float64(used), "used_memory_rss": float64(rss), "maxmemory": float64(200 << 20)},
				Stats: map[string]any{
					"total_connections_received": float64(conns),
					"total_commands_processed":   float64(cmds),
					"keyspace_hits":              float64(hits),
					"keyspace_misses":            float64(misses),
					"instantaneous_input_kbps":   10.5,
					"instantaneous_output_kbps":  20.5,
				},
				Replication: replication,
				Cluster:     map[string]any{"cluster_enabled": 0.0},
				CPU:         map[string]any{"used_cpu_user": 10.0, "used_cpu_sys": 5.0},
				Persistence: map[string]any{"aof_enabled": 0.0, "rdb_bgsave_in_progress": 0.0},
				Commandstats: map[string]model.CommandMetrics{
					"cmdstat_get": {Calls: float64(cmds)},
				},
			},
			Slowlog: model.SlowlogSnapshot{Len: 0},
		},
		Host: model.HostMetrics{
			Supported: true,
			LoadAvg:   map[string]any{"1m": load, "5m": 0.8, "15m": 0.4},
			CPU: map[string]any{
				"user":          float64(user),
				"system":        float64(sys),
				"idle":          float64(idle),
				"iowait":        float64(iowait),
				"nice":          0.0,
				"irq":           0.0,
				"softirq":       0.0,
				"steal":         0.0,
				"guest":         0.0,
				"guest_nice":    0.0,
				"ctxt":          float64(ctxt),
				"processes":     float64(procs),
				"procs_running": 2.0,
				"procs_blocked": 1.0,
			},
			Memory: map[string]string{"MemTotal": "32866112 kB", "MemAvailable": "29420820 kB", "MemFree": "4096000 kB", "Buffers": "128000 kB", "Cached": "512000 kB"},
		},
	}, "test", 0)
}
