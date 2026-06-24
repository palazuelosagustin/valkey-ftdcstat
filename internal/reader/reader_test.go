package reader

import (
	"os"
	"path/filepath"
	"testing"

	"valkey-ftdcstat/internal/discovery"
	"valkey-ftdcstat/internal/model"
)

func TestReadCapture(t *testing.T) {
	dir := t.TempDir()
	content := "VKFTDC1\n{\"format_version\":1}\n{\"ts_ms\":1000,\"valkey\":{\"info\":{\"server\":{\"uptime_in_seconds\":1},\"clients\":{},\"memory\":{},\"persistence\":{},\"stats\":{},\"replication\":{},\"cpu\":{},\"commandstats\":{},\"cluster\":{}},\"latency_latest\":[],\"slowlog\":{\"len\":0}},\"host\":{\"enabled\":false}}\n"
	if err := os.WriteFile(filepath.Join(dir, "metrics.2026-06-20T11-59-11Z.vkftdc"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	capture, err := ReadCapture(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(capture.MetricSamples) != 1 {
		t.Fatalf("samples=%d", len(capture.MetricSamples))
	}
	if capture.MetricSamples[0].Time.UnixMilli() != 1000 {
		t.Fatalf("ts_ms=%d", capture.MetricSamples[0].Time.UnixMilli())
	}
}

func TestStreamSamplesAndMetadata(t *testing.T) {
	dir := t.TempDir()
	content := "VKFTDC1\n{\"format_version\":1,\"module\":\"valkey-ftdc\"}\n{\"ts_ms\":1000,\"valkey\":{\"info\":{\"server\":{\"process_id\":1,\"run_id\":\"a\"},\"clients\":{\"maxclients\":10000},\"memory\":{},\"persistence\":{},\"stats\":{\"total_commands_processed\":1},\"replication\":{\"role\":\"master\"},\"cpu\":{},\"commandstats\":{},\"cluster\":{}},\"latency_latest\":[],\"slowlog\":{\"len\":0}},\"host\":{\"supported\":true,\"loadavg\":{\"1m\":0.1},\"cpu\":{\"user\":1,\"idle\":9}}}\n{\"ts_ms\":61000,\"valkey\":{\"info\":{\"server\":{\"process_id\":1,\"run_id\":\"a\"},\"clients\":{},\"memory\":{},\"persistence\":{},\"stats\":{\"total_commands_processed\":11},\"replication\":{\"role\":\"master\"},\"cpu\":{},\"commandstats\":{},\"cluster\":{}},\"latency_latest\":[],\"slowlog\":{\"len\":0}},\"host\":{\"supported\":true,\"loadavg\":{\"1m\":0.2},\"cpu\":{\"user\":2,\"idle\":18}}}\n"
	if err := os.WriteFile(filepath.Join(dir, "metrics.2026-06-20T11-59-11Z.vkftdc"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	files, _, err := discovery.Discover(dir)
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	_, _, err = StreamSamples(files, StreamOptions{}, func(sample model.MetricSample) error {
		count++
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("count=%d", count)
	}
	metadata, _, err := ReadMetadata(dir, files)
	if err != nil {
		t.Fatal(err)
	}
	if metadata.Module != "valkey-ftdc" {
		t.Fatalf("module=%q", metadata.Module)
	}
	if metadata.MaxClients != 10000 {
		t.Fatalf("maxclients=%v", metadata.MaxClients)
	}
}
