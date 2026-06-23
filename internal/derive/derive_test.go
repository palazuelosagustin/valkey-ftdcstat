package derive

import (
	"testing"
	"time"

	"valkey-ftdcstat/internal/model"
)

func TestBuildSummaryRows(t *testing.T) {
	capture := model.Capture{
		Path:  "diagnostic.data",
		Files: []string{"metrics.1.vkftdc"},
		Samples: []model.Sample{
			sample(0, 1000, 2000, 5000, 1000, 100<<20, 150<<20, 10, 1, 1.0, 1000, 500, 10000, 100, 5000, 100, "master", 2),
			sample(60_000, 1120, 2600, 5600, 1120, 110<<20, 160<<20, 12, 0, 1.2, 1100, 540, 10300, 110, 5600, 110, "master", 2),
		},
	}
	report, err := Build(capture, Options{View: "summary", Interval: time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Rows) != 1 {
		t.Fatalf("rows=%d", len(report.Rows))
	}
	row := report.Rows[0]
	if row["ops/s"] != float64(10) {
		t.Fatalf("ops/s=%v", row["ops/s"])
	}
	if row["conn/s"] != float64(2) {
		t.Fatalf("conn/s=%v", row["conn/s"])
	}
	if row["repl"] != "master" {
		t.Fatalf("repl=%v", row["repl"])
	}
}

func sample(ts, conns, cmds, hits, misses, used, rss int64, clients, blocked int, load float64, user, sys, idle, iowait, ctxt, procs int64, role string, replicas int) model.Sample {
	return model.Sample{
		TsMS: ts,
		Valkey: model.ValkeyMetrics{
			Info: model.InfoSections{
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
				Replication: map[string]any{"role": role, "connected_slaves": float64(replicas), "master_repl_offset": float64(1 << 20)},
				CPU:         map[string]any{"used_cpu_user": 10.0, "used_cpu_sys": 5.0},
				Persistence: map[string]any{"aof_enabled": 0.0, "rdb_bgsave_in_progress": 0.0},
				Commandstats: map[string]model.CommandMetrics{
					"cmdstat_get": {Calls: float64(cmds)},
				},
			},
			Slowlog: model.SlowlogSnapshot{Len: 0},
		},
		Host: model.HostMetrics{
			LoadAvg: map[string]any{"1m": load},
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
			Memory: map[string]string{"MemAvailable": "8192000 kB", "MemFree": "4096000 kB", "Buffers": "128000 kB", "Cached": "512000 kB"},
		},
	}
}
