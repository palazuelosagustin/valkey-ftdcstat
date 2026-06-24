package derive

import (
	"testing"

	"valkey-ftdcstat/internal/model"
)

func TestBuildHeaderIncludesModuleConfig(t *testing.T) {
	sample := metricSample(0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, "master", 0, nil)
	sample.Values["valkey.info.server.hz"] = 10
	metadata := model.Metadata{
		MaxClients: 10000,
		Config: map[string]any{
			"interval-ms":        float64(60000),
			"collect-host-stats": true,
		},
	}
	header := buildHeader(sample, metadata)
	if header.ModuleConfig["interval-ms"] != float64(60000) {
		t.Fatalf("moduleConfig=%v", header.ModuleConfig)
	}
	if header.BuildInfo["hz"] != float64(10) {
		t.Fatalf("hz=%v", header.BuildInfo["hz"])
	}
	if header.BuildInfo["maxClients"] != float64(10000) {
		t.Fatalf("maxClients=%v", header.BuildInfo["maxClients"])
	}
}
