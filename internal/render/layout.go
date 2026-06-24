package render

type tableSection struct {
	Name       string
	Start, End int
}

type tableLayout struct {
	Columns  []string
	Sections []tableSection
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

func summarySectionForColumn() map[string]string {
	return map[string]string{
		"ops/s":    "server",
		"conn/s":   "server",
		"hit%":     "server",
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
		"repl":     "replication",
		"repls":    "replication",
		"us%":      "host",
		"sy%":      "host",
		"id%":      "host",
		"wa%":      "host",
		"load1":    "host",
		"availMB":  "host",
	}
}
