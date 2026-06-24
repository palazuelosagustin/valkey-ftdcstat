package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestJSONOutputsRawCapture(t *testing.T) {
	dir := t.TempDir()
	content := "VKFTDC1\n{\"format_version\":1,\"module\":\"valkey-ftdc\"}\n{\"ts_ms\":1000,\"valkey\":{\"info\":{\"server\":{\"valkey_version\":\"9.1.0\"},\"clients\":{},\"memory\":{},\"persistence\":{},\"stats\":{\"total_commands_processed\":6},\"replication\":{},\"cpu\":{},\"commandstats\":{},\"cluster\":{}},\"latency_latest\":[],\"slowlog\":{\"len\":0}},\"host\":{\"enabled\":false}}\n"
	if err := os.WriteFile(filepath.Join(dir, "metrics.1.vkftdc"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("go", "run", "./cmd/valkey-ftdcstat", "--json", dir)
	cmd.Dir = "/home/agustin.palazuelos/bin/valkey-ftdcstat"
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("go run failed: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(out, &parsed); err != nil {
		t.Fatalf("invalid json: %v\n%s", err, string(out))
	}
	if parsed["path"] != dir {
		t.Fatalf("path=%v", parsed["path"])
	}
	if _, ok := parsed["samples"]; !ok {
		t.Fatalf("missing samples: %v", parsed)
	}
	if _, ok := parsed["metadata"]; !ok {
		t.Fatalf("missing metadata: %v", parsed)
	}
	if _, ok := parsed["header"]; ok {
		t.Fatalf("unexpected derived header in raw json: %v", parsed["header"])
	}
	if _, ok := parsed["rows"]; ok {
		t.Fatalf("unexpected derived rows in raw json: %v", parsed["rows"])
	}
	if _, ok := parsed["latest"]; ok {
		t.Fatalf("unexpected derived latest in raw json: %v", parsed["latest"])
	}
	samples := parsed["samples"].([]any)
	first := samples[0].(map[string]any)
	if first["ts_ms"] != float64(1000) {
		t.Fatalf("ts_ms=%v", first["ts_ms"])
	}
	valkey := first["valkey"].(map[string]any)
	info := valkey["info"].(map[string]any)
	stats := info["stats"].(map[string]any)
	if stats["total_commands_processed"] != float64(6) {
		t.Fatalf("stats=%v", stats)
	}
}
