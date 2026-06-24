package webui

import "testing"

func TestBuildSectionsSummaryIncludesCommands(t *testing.T) {
	columns := []string{
		"time", "ops/s", "conn/s", "hit%", "get/s", "set/s",
		"memMB", "rssMB", "frag%",
	}
	sections := buildSections("summary", columns, false)
	names := map[string]bool{}
	for _, section := range sections {
		names[section.Name] = true
	}
	if !names["commands"] {
		t.Fatalf("sections=%+v", sections)
	}
}

func TestBuildSectionsCommandstats(t *testing.T) {
	columns := []string{"time", "get/s", "set/s"}
	sections := buildSections("commandstats", columns, false)
	if len(sections) != 1 || sections[0].Name != "commandstats" {
		t.Fatalf("sections=%+v", sections)
	}
	if len(sections[0].Metrics) != 2 {
		t.Fatalf("metrics=%+v", sections[0].Metrics)
	}
}
