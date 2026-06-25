package reader_test

import (
	"os"
	"testing"

	"valkey-ftdcstat/internal/discovery"
	"valkey-ftdcstat/internal/flatten"
	"valkey-ftdcstat/internal/model"
	"valkey-ftdcstat/internal/reader"
)

func BenchmarkStreamSummaryCapture(b *testing.B) {
	path := os.Getenv("VALKEY_FTDC_BENCH_FILE")
	if path == "" {
		path = "/tmp/vm-metrics.vkftdc"
	}
	if _, err := os.Stat(path); err != nil {
		b.Skip("benchmark file not available")
	}
	files := []discovery.MetricFile{{Path: path, Kind: discovery.KindMetrics}}
	opts := reader.StreamOptions{Flatten: flatten.OptionsForView("summary", false)}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		count := 0
		_, _, err := reader.StreamSamples(files, opts, func(sample model.MetricSample) error {
			count++
			return nil
		})
		if err != nil {
			b.Fatal(err)
		}
		if count == 0 {
			b.Fatal("no samples")
		}
	}
}
