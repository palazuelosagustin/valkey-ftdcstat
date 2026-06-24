package flatten

import (
	"testing"

	"valkey-ftdcstat/internal/model"
)

func TestSampleFlattensInfoAndHostPaths(t *testing.T) {
	sample := model.Sample{
		TsMS: 60_000,
		Valkey: model.ValkeyMetrics{
			Info: model.InfoSections{
				Stats: map[string]any{
					"total_commands_processed": float64(100),
				},
				Replication: map[string]any{"role": "master"},
				Commandstats: map[string]model.CommandMetrics{
					"cmdstat_get": {Calls: 10, UsecPerCall: 2.5},
				},
			},
		},
		Host: model.HostMetrics{
			Supported: true,
			CPU:       map[string]any{"user": float64(1000), "idle": float64(9000)},
			Memory:    map[string]string{"MemAvailable": "8192 kB"},
			Disk:      model.HostDisk{Diskstats: "8 0 sda 1 2 3 4 5 6 7 8 9 10 11 12 13 14\n"},
		},
	}
	flat := Sample(sample, "metrics.1.vkftdc", 0)
	if got, ok := flat.Get("valkey.info.stats.total_commands_processed"); !ok || got != 100 {
		t.Fatalf("stats path=%v ok=%v", got, ok)
	}
	if flat.GetText("valkey.info.replication.role") != "master" {
		t.Fatalf("role=%q", flat.GetText("valkey.info.replication.role"))
	}
	if got, ok := flat.Get("valkey.info.commandstats.cmdstat_get.calls"); !ok || got != 10 {
		t.Fatalf("cmdstat=%v ok=%v", got, ok)
	}
	if got, ok := flat.Get("host.cpu.user"); !ok || got != 1000 {
		t.Fatalf("cpu=%v ok=%v", got, ok)
	}
	if got, ok := flat.Get("host.memory.MemAvailable.mb"); !ok || got != 8 {
		t.Fatalf("mem=%v ok=%v", got, ok)
	}
	if flat.Text["host.disk.diskstats"] == "" {
		t.Fatal("expected diskstats blob")
	}
}
