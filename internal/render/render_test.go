package render

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"valkey-ftdcstat/internal/derive"
	"valkey-ftdcstat/internal/model"
)

func TestReportRendersCompactHeader(t *testing.T) {
	report := derive.Report{
		View:        "summary",
		Path:        "/home/agustin.palazuelos/valkey-ftdc-run/node0/diagnostic.metrics",
		Files:       []string{"a", "b"},
		SampleCount: 4734,
		Start:       time.Date(2026, 6, 23, 23, 8, 36, 0, time.UTC),
		End:         time.Date(2026, 6, 24, 0, 27, 39, 0, time.UTC),
		Header: model.Header{
			HostInfo: map[string]any{
				"os": "Linux 6.8.0-110-generic x86_64",
				"memory": map[string]string{
					"available": "29420820 kB",
					"buffers":   "128000 kB",
					"cached":    "512000 kB",
					"free":      "4096000 kB",
					"total":     "32866112 kB",
				},
				"loadavg": map[string]any{"1m": 0.18, "5m": 0.07, "15m": 0.04},
				"cpu":     map[string]any{"idle": 3560049614.0, "iowait": 12659196.0, "system": 98876593.0, "user": 507973759.0, "procs_running": 2.0, "procs_blocked": 0.0},
			},
			BuildInfo: map[string]any{
				"valkeyVersion":   "9.1.0",
				"serverMode":      "standalone",
				"redisVersion":    "7.2.4",
				"buildID":         "7f033bf934c6ae79",
				"gccVersion":      "13.3.0",
				"os":              "Linux 6.8.0-110-generic x86_64",
				"archBits":        64.0,
				"multiplexingAPI": "epoll",
				"gitSHA1":         "00000000",
				"gitDirty":        true,
			},
			ReplicationInfo: map[string]any{
				"role":           "master",
				"replicas":       2.0,
				"clusterEnabled": false,
			},
		},
		Columns: []string{"time", "ops/s"},
		Rows:    []map[string]any{{"time": "2026-06-24T00:27:39Z", "ops/s": 10.0}},
	}
	var buf bytes.Buffer
	if err := Report(&buf, report); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	checks := []string{
		"valkey-ftdcstat report",
		"path:    /home/agustin.palazuelos/valkey-ftdc-run/node0/diagnostic.metrics",
		"files:   2",
		"samples: 4734",
		"range:   2026-06-23T23:08:36Z .. 2026-06-24T00:27:39Z  (1h 19m 03s)",
		"Valkey 9.1.0 | standalone | primary",
		"redis_version: 7.2.4 compatibility",
		"build: 7f033bf934c6ae79, gcc 13.3.0, Linux 6.8.0-110-generic x86_64, epoll",
		"git: 00000000 dirty",
		"role: primary",
		"replicas: 2",
		"cluster: disabled",
		"os: Linux 6.8.0-110-generic x86_64",
		"memory: 28.1 GiB available / 31.3 GiB total",
	}
	for _, check := range checks {
		if !strings.Contains(out, check) {
			t.Fatalf("missing %q in output:\n%s", check, out)
		}
	}
	for _, bad := range []string{"loadavg", "procs_running", "procs_blocked", "idle:", "iowait:", "system:", "user:", "buffers", "cached", "free"} {
		if strings.Contains(out, bad) {
			t.Fatalf("unexpected %q in output:\n%s", bad, out)
		}
	}
}

func TestReportRendersReplicaRoleAndNamesWhenPresent(t *testing.T) {
	report := derive.Report{
		View:        "summary",
		Path:        "diagnostic.metrics",
		Files:       []string{"a"},
		SampleCount: 2,
		Start:       time.Date(2026, 6, 24, 0, 0, 0, 0, time.UTC),
		End:         time.Date(2026, 6, 24, 0, 0, 5, 0, time.UTC),
		Header: model.Header{
			HostInfo:  map[string]any{"os": "Linux"},
			BuildInfo: map[string]any{"valkeyVersion": "9.1.0"},
			ReplicationInfo: map[string]any{
				"role":           "slave",
				"replicas":       2.0,
				"replicaNames":   []string{"node-a", "node-b"},
				"clusterEnabled": true,
			},
		},
		Columns: []string{"time"},
		Rows:    []map[string]any{{"time": "2026-06-24T00:00:05Z"}},
	}
	var buf bytes.Buffer
	if err := Report(&buf, report); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "role: replica") {
		t.Fatalf("expected replica role: %s", out)
	}
	if !strings.Contains(out, "replicas: 2 (node-a, node-b)") {
		t.Fatalf("expected replica names: %s", out)
	}
	if !strings.Contains(out, "cluster: enabled") {
		t.Fatalf("expected cluster enabled: %s", out)
	}
}
