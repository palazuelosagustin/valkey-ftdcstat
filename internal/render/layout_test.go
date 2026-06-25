package render

import "testing"

func TestSummaryLayoutGroupsColumns(t *testing.T) {
	columns := []string{
		"time", "ops/s", "conn/s", "hit%", "memMB", "rssMB", "frag%",
		"rej/s", "exp/s", "cli", "blk", "role", "node1", "node2", "us%", "load1",
	}
	layout := LayoutForView("summary", columns)
	if len(layout.Sections) != 7 {
		t.Fatalf("sections=%+v", layout.Sections)
	}
	if layout.Sections[0].Name != "server" || layout.Sections[0].Start != 1 {
		t.Fatalf("server section=%+v", layout.Sections[0])
	}
	if layout.Sections[len(layout.Sections)-1].Name != "host" {
		t.Fatalf("last section=%+v", layout.Sections[len(layout.Sections)-1])
	}
}

func TestSingleViewLayoutSkipsTimeColumn(t *testing.T) {
	layout := LayoutForView("server", []string{"time", "ops/s", "cli"})
	if len(layout.Sections) != 1 || layout.Sections[0].Start != 1 || layout.Sections[0].End != 3 {
		t.Fatalf("layout=%+v", layout)
	}
}
