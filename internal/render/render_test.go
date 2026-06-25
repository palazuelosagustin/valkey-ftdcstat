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
				"nodes": map[string]string{
					"node0": "10.0.0.1",
					"node1": "10.0.0.2",
					"node3": "10.0.0.3",
				},
			},
		},
		Columns: []string{"time", "ops/s"},
		Rows: []derive.Row{
			{Time: time.Date(2026, 6, 24, 0, 27, 39, 0, time.UTC), Values: map[string]any{"ops/s": 10.0}},
		},
	}
	var buf bytes.Buffer
	if err := Report(&buf, report, DisplayOptions{}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	checks := []string{
		"valkey-ftdcstat report",
		"metricsRange:",
		"serverInfo:",
		"Valkey 9.1.0 | standalone | primary",
		"redis_version: 7.2.4 compatibility",
		"role: primary",
		"replicas: 2",
		"node0: 10.0.0.1",
		"node1: 10.0.0.2",
		"node3: 10.0.0.3",
		"hostInfo:",
	}
	for _, check := range checks {
		if !strings.Contains(out, check) {
			t.Fatalf("missing %q in output:\n%s", check, out)
		}
	}
}
