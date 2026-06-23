package reader

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadCapture(t *testing.T) {
	dir := t.TempDir()
	content := "VKFTDC1\n{\"format_version\":1}\n{\"ts_ms\":1000,\"valkey\":{\"info\":{\"server\":{\"uptime_in_seconds\":1},\"clients\":{},\"memory\":{},\"persistence\":{},\"stats\":{},\"replication\":{},\"cpu\":{},\"commandstats\":{},\"cluster\":{}},\"latency_latest\":[],\"slowlog\":{\"len\":0}},\"host\":{\"enabled\":false}}\n"
	if err := os.WriteFile(filepath.Join(dir, "metrics.1.vkftdc"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	capture, err := ReadCapture(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(capture.Samples) != 1 {
		t.Fatalf("samples=%d", len(capture.Samples))
	}
	if capture.Samples[0].TsMS != 1000 {
		t.Fatalf("ts_ms=%d", capture.Samples[0].TsMS)
	}
}
