package derive

import (
	"reflect"
	"testing"
	"time"

	"valkey-ftdcstat/internal/model"
)

func TestReplicasFromSampleUsesNamesAndOffsets(t *testing.T) {
	sample := metricSample(0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, "master", 2, []string{"node1", "node2"})
	sample.Values["valkey.info.replication.slave0.offset"] = 1000
	sample.Values["valkey.info.replication.slave1.offset"] = 2000
	sample.Text["valkey.info.replication.slave0.ip"] = "10.0.0.2"
	sample.Text["valkey.info.replication.slave1.ip"] = "10.0.0.3"

	replicas := replicasFromSample(sample)
	if len(replicas) != 2 {
		t.Fatalf("replicas=%v", replicas)
	}
	if replicas[0].Name != "node1" || replicas[0].IP != "10.0.0.2" || replicas[0].OffsetPath == "" {
		t.Fatalf("replica0=%+v", replicas[0])
	}
	if replicas[1].Name != "node2" || replicas[1].IP != "10.0.0.3" {
		t.Fatalf("replica1=%+v", replicas[1])
	}
}

func TestTopologyNodesIncludesLocalAndReplicas(t *testing.T) {
	sample := metricSample(0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, "master", 2, []string{"node1", "node3"})
	sample.Text["valkey.info.replication.slave0.ip"] = "10.0.0.2"
	sample.Text["valkey.info.replication.slave1.ip"] = "10.0.0.3"
	sample.Text["valkey.info.server.listener1.bind"] = "10.0.0.1"
	sample.Values["valkey.info.server.tcp_port"] = 6379

	nodes := topologyNodes(sample, "/data/node0/diagnostic.data")
	want := map[string]string{
		"node0": "10.0.0.1:6379",
		"node1": "10.0.0.2:6001",
		"node3": "10.0.0.3:6002",
	}
	if !reflect.DeepEqual(nodes, want) {
		t.Fatalf("nodes=%v want=%v", nodes, want)
	}
}

func TestSummaryColumnsIncludeReplicaOffsets(t *testing.T) {
	first := metricSample(0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, "master", 2, []string{"node1", "node2"})
	last := first
	cols := summaryColumns(nil, replicaOffsetColumns(first, last))
	roleIdx := indexOfString(cols, "role")
	if roleIdx < 0 || cols[roleIdx+1] != "node1" || cols[roleIdx+2] != "node2" {
		t.Fatalf("columns=%v", cols)
	}
	if cols[roleIdx+3] != "us%" {
		t.Fatalf("host columns should follow replica offsets, columns=%v", cols)
	}
}

func indexOfString(items []string, want string) int {
	for i, item := range items {
		if item == want {
			return i
		}
	}
	return -1
}

func TestFillSummaryUsesRoleAndReplicaOffsets(t *testing.T) {
	samples := []model.MetricSample{
		metricSample(0, 1000, 2000, 5000, 1000, 100<<20, 150<<20, 10, 1, 1.0, 1000, 500, 10000, 100, 5000, 100, "master", 2, []string{"node1", "node2"}),
		metricSample(60_000, 1120, 2600, 5600, 1120, 110<<20, 160<<20, 12, 0, 1.2, 1100, 540, 10300, 110, 5600, 110, "master", 2, []string{"node1", "node2"}),
	}
	samples[0].Values["valkey.info.replication.slave0.offset"] = 1000
	samples[0].Values["valkey.info.replication.slave1.offset"] = 900
	samples[1].Values["valkey.info.replication.slave0.offset"] = 1500
	samples[1].Values["valkey.info.replication.slave1.offset"] = 1200

	report, err := Build(model.Capture{MetricSamples: samples}, Options{View: "summary", Interval: time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	row := report.Rows[0]
	if row.Values["role"] != "master" {
		t.Fatalf("role=%v", row.Values["role"])
	}
	if _, ok := row.Values["repls"]; ok {
		t.Fatal("repls column should be removed")
	}
	if row.Values["node1"] != float64(1500) || row.Values["node2"] != float64(1200) {
		t.Fatalf("offsets=%v", row.Values)
	}
}
