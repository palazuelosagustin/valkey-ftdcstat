package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestJSONOutputsReport(t *testing.T) {
	dir := t.TempDir()
	content := "VKFTDC1\n{\"format_version\":1,\"module\":\"valkey-ftdc\"}\n{\"ts_ms\":1000,\"valkey\":{\"info\":{\"server\":{\"valkey_version\":\"9.1.0\"},\"clients\":{},\"memory\":{},\"persistence\":{},\"stats\":{\"total_commands_processed\":6},\"replication\":{\"role\":\"master\"},\"cpu\":{},\"commandstats\":{},\"cluster\":{}},\"latency_latest\":[],\"slowlog\":{\"len\":0}},\"host\":{\"enabled\":false}}\n{\"ts_ms\":61000,\"valkey\":{\"info\":{\"server\":{\"valkey_version\":\"9.1.0\"},\"clients\":{},\"memory\":{},\"persistence\":{},\"stats\":{\"total_commands_processed\":16},\"replication\":{\"role\":\"master\"},\"cpu\":{},\"commandstats\":{},\"cluster\":{}},\"latency_latest\":[],\"slowlog\":{\"len\":0}},\"host\":{\"enabled\":false}}\n"
	if err := os.WriteFile(filepath.Join(dir, "metrics.2026-06-20T11-59-11Z.vkftdc"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	moduleRoot, err := findModuleRoot()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "run", "./cmd/valkey-ftdcstat", "--json", "--interval", "60", dir)
	cmd.Dir = moduleRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go run failed: %v\n%s", err, string(out))
	}

	var parsed map[string]any
	if err := json.Unmarshal(out, &parsed); err != nil {
		t.Fatalf("invalid json: %v\n%s", err, string(out))
	}
	if parsed["path"] != dir {
		t.Fatalf("path=%v", parsed["path"])
	}
	if parsed["view"] != "summary" {
		t.Fatalf("view=%v", parsed["view"])
	}
	if _, ok := parsed["rows"]; !ok {
		t.Fatalf("missing rows: %v", parsed)
	}
	header, ok := parsed["header"].(map[string]any)
	if !ok {
		t.Fatalf("missing header: %v", parsed["header"])
	}
	buildInfo, ok := header["buildInfo"].(map[string]any)
	if !ok || buildInfo["valkeyVersion"] != "9.1.0" {
		t.Fatalf("buildInfo=%v", header["buildInfo"])
	}
}

func findModuleRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", os.ErrNotExist
		}
		dir = parent
	}
}
