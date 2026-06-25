package render

import "strings"

type tableSection struct {
	Name       string
	Start, End int
}

type tableLayout struct {
	Columns  []string
	Sections []tableSection
}

// ViewSection names a chart group and the report columns it contains.
type ViewSection struct {
	Name    string
	Columns []string
}

// ViewSections maps a view and its columns into chart/table section groups.
func ViewSections(view string, columns []string) []ViewSection {
	layout := LayoutForView(view, columns)
	out := make([]ViewSection, 0, len(layout.Sections))
	for _, sec := range layout.Sections {
		if sec.End <= sec.Start {
			continue
		}
		out = append(out, ViewSection{
			Name:    sec.Name,
			Columns: append([]string(nil), layout.Columns[sec.Start:sec.End]...),
		})
	}
	return out
}

func LayoutForView(view string, columns []string) tableLayout {
	if len(columns) == 0 {
		return tableLayout{}
	}
	if view != "summary" {
		return singleSectionLayout(view, columns)
	}
	return summaryLayout(columns)
}

func singleSectionLayout(name string, columns []string) tableLayout {
	start := 0
	if len(columns) > 0 && columns[0] == "time" {
		start = 1
	}
	sections := []tableSection{}
	if start < len(columns) {
		sections = append(sections, tableSection{Name: name, Start: start, End: len(columns)})
	}
	return tableLayout{Columns: append([]string(nil), columns...), Sections: sections}
}

func summaryLayout(columns []string) tableLayout {
	sectionFor := summarySectionForColumn()
	hostStart := indexOfColumn(columns, "us%")
	var sections []tableSection
	var current string
	var currentStart int
	flush := func(end int) {
		if current == "" || end <= currentStart {
			return
		}
		sections = append(sections, tableSection{Name: current, Start: currentStart, End: end})
		current = ""
	}
	for i, col := range columns {
		name := sectionFor[col]
		if name == "" && isSummaryCommandColumn(col) {
			name = "commands"
		}
		if name == "" && isSummaryOffsetColumn(columns, i, hostStart) {
			name = "offset"
		}
		if col == "time" {
			flush(i)
			continue
		}
		if name == "" {
			flush(i)
			currentStart = i
			continue
		}
		if name != current {
			flush(i)
			current = name
			currentStart = i
		}
	}
	flush(len(columns))
	return tableLayout{Columns: append([]string(nil), columns...), Sections: sections}
}

func indexOfColumn(columns []string, name string) int {
	for i, col := range columns {
		if col == name {
			return i
		}
	}
	return -1
}

func isSummaryOffsetColumn(columns []string, idx, hostStart int) bool {
	roleIdx := indexOfColumn(columns, "role")
	if roleIdx < 0 || idx <= roleIdx {
		return false
	}
	if hostStart >= 0 && idx >= hostStart {
		return false
	}
	return columns[idx] != "role"
}

func isSummaryCommandColumn(column string) bool {
	if !strings.HasSuffix(column, "/s") {
		return false
	}
	switch column {
	case "ops/s", "conn/s", "rej/s", "exp/s", "evict/s", "offKB/s", "inKB/s", "outKB/s":
		return false
	default:
		return true
	}
}

func summarySectionForColumn() map[string]string {
	return map[string]string{
		"ops/s":  "server",
		"conn/s": "server",
		"hit%":   "server",
		"memMB":    "memory",
		"rssMB":    "memory",
		"frag%":    "memory",
		"rej/s":    "stats",
		"exp/s":    "stats",
		"evict/s":  "stats",
		"offKB/s":  "stats",
		"inKB/s":   "stats",
		"outKB/s":  "stats",
		"cli":      "clients",
		"blk":      "clients",
		"role":     "replication",
		"us%":      "host",
		"sy%":      "host",
		"id%":      "host",
		"wa%":      "host",
		"load1":    "host",
		"availMB":  "host",
	}
}
