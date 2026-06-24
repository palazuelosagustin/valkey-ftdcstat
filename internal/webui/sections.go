package webui

import (
	"strings"

	"valkey-ftdcstat/internal/derive"
	"valkey-ftdcstat/internal/render"
)

var latencyFallbackColumns = map[string]struct{}{
	"slowlog": {}, "slowMaxMs": {}, "blocked": {}, "forkUsec": {}, "eloopUs": {},
}

func buildSections(view string, columns []string, verbose bool) []Section {
	switch view {
	case "summary":
		return sectionsFromLayout(view, columns)
	case "host":
		return hostSections(columns, verbose)
	case "latency":
		return latencySections(columns)
	case "memory":
		if verbose {
			return memorySections(columns)
		}
		return singleSection("memory", columns)
	case "clients":
		if verbose {
			return clientsSections(columns)
		}
		return singleSection("clients", columns)
	case "replication":
		if verbose {
			return replicationSections(columns)
		}
		return singleSection("replication", columns)
	case "network":
		if verbose {
			return networkSections(columns)
		}
		return singleSection("network", columns)
	case "commandstats":
		return nil
	default:
		return singleSection(view, columns)
	}
}

func sectionsFromLayout(view string, columns []string) []Section {
	layoutSections := render.ViewSections(view, columns)
	out := make([]Section, 0, len(layoutSections))
	for _, sec := range layoutSections {
		if section := sectionFromColumns(displaySectionName(view, sec.Name), sec.Columns, view); len(section.Metrics) > 0 {
			out = append(out, section)
		}
	}
	return out
}

func displaySectionName(view, name string) string {
	if view == "summary" {
		return name
	}
	return view + " / " + name
}

func hostSections(columns []string, verbose bool) []Section {
	groups := []struct {
		name string
		cols []string
	}{
		{name: "host / CPU", cols: []string{"r", "b", "us%", "sy%", "id%", "wa%", "st%", "load1", "cs/s", "forks/s"}},
		{name: "host / Memory", cols: []string{"swpd", "free", "buff", "cache", "availMB"}},
		{name: "host / Disks", cols: []string{"bi", "bo"}},
	}
	if verbose {
		groups[1].cols = append(groups[1].cols, "rssMB")
	}
	present := columnSet(columns)
	var out []Section
	for _, group := range groups {
		var cols []string
		for _, col := range group.cols {
			if present[col] {
				cols = append(cols, col)
			}
		}
		if section := sectionFromColumns(group.name, cols, "host"); len(section.Metrics) > 0 {
			out = append(out, section)
		}
	}
	if len(out) > 0 {
		return out
	}
	return singleSection("host", columns)
}

func memorySections(columns []string) []Section {
	groups := []struct {
		name string
		cols []string
	}{
		{name: "memory / Allocation", cols: []string{"usedMB", "rssMB", "maxMB", "rss%", "availMB"}},
		{name: "memory / Pressure", cols: []string{"exp/s", "evict/s", "frag%", "luaMB", "scripts", "defrag"}},
	}
	return groupedSections(groups, columns, "memory")
}

func clientsSections(columns []string) []Section {
	groups := []struct {
		name string
		cols []string
	}{
		{name: "clients / Connections", cols: []string{"conn", "blocked", "pubsub", "conn/s"}},
		{name: "clients / Throughput", cols: []string{"ops/s", "hit%", "rej/s"}},
	}
	return groupedSections(groups, columns, "clients")
}

func replicationSections(columns []string) []Section {
	groups := []struct {
		name string
		cols []string
	}{
		{name: "replication / Topology", cols: []string{"role", "replicas", "backlog", "backlogMB"}},
		{name: "replication / Offset", cols: []string{"offsetMB", "offMB/s"}},
	}
	return groupedSections(groups, columns, "replication")
}

func networkSections(columns []string) []Section {
	groups := []struct {
		name string
		cols []string
	}{
		{name: "network / Valkey", cols: []string{"inKB/s", "outKB/s", "conn/s"}},
		{name: "network / Host", cols: []string{"rxKB/s", "txKB/s", "rej/s", "pubsub"}},
	}
	return groupedSections(groups, columns, "network")
}

func groupedSections(groups []struct {
	name string
	cols []string
}, columns []string, view string) []Section {
	present := columnSet(columns)
	var out []Section
	for _, group := range groups {
		var cols []string
		for _, col := range group.cols {
			if present[col] {
				cols = append(cols, col)
			}
		}
		if section := sectionFromColumns(group.name, cols, view); len(section.Metrics) > 0 {
			out = append(out, section)
		}
	}
	if len(out) > 0 {
		return out
	}
	return singleSection(view, columns)
}

func latencySections(columns []string) []Section {
	var fallback []string
	var events []string
	for _, col := range columns {
		if col == "time" {
			continue
		}
		if _, ok := latencyFallbackColumns[col]; ok {
			fallback = append(fallback, col)
			continue
		}
		events = append(events, col)
	}
	var out []Section
	if section := sectionFromColumns("latency / fallbacks", fallback, "latency"); len(section.Metrics) > 0 {
		out = append(out, section)
	}
	if section := sectionFromColumns("latency / events", events, "latency"); len(section.Metrics) > 0 {
		out = append(out, section)
	}
	if len(out) > 0 {
		return out
	}
	return singleSection("latency", columns)
}

func singleSection(name string, columns []string) []Section {
	if section := sectionFromColumns(name, filterMetricColumns(columns), name); len(section.Metrics) > 0 {
		return []Section{section}
	}
	return nil
}

func sectionFromColumns(name string, columns []string, view string) Section {
	metrics := make([]Metric, 0, len(columns))
	for _, column := range columns {
		if column == "time" {
			continue
		}
		metrics = append(metrics, Metric{
			Column:   column,
			Label:    column,
			JSONName: column,
			Format:   metricFormat(column),
			Default:  defaultMetric(view, column),
		})
	}
	return Section{Name: name, Metrics: metrics}
}

func buildCommandRows(commands []derive.CommandRow) []CommandRow {
	out := make([]CommandRow, 0, len(commands))
	for _, row := range commands {
		out = append(out, CommandRow{
			Command:     row.Command,
			Calls:       row.Calls,
			CallsPerSec: row.CallsPerSec,
			UsecPerCall: row.UsecPerCall,
			SharePct:    row.SharePct,
		})
	}
	return out
}

func filterMetricColumns(columns []string) []string {
	out := make([]string, 0, len(columns))
	for _, col := range columns {
		if col != "time" {
			out = append(out, col)
		}
	}
	return out
}

func columnSet(columns []string) map[string]bool {
	out := make(map[string]bool, len(columns))
	for _, col := range columns {
		out[col] = true
	}
	return out
}

func metricFormat(column string) string {
	if strings.HasSuffix(column, "%") || column == "repl" || column == "role" || column == "rdb" || column == "aof" {
		return "text"
	}
	return "number"
}

func defaultMetric(view, column string) bool {
	switch view {
	case "summary":
		switch column {
		case "ops/s", "hit%", "memMB", "cli", "load1", "inKB/s", "outKB/s", "repl":
			return true
		}
	case "host":
		switch column {
		case "us%", "sy%", "load1", "bi", "bo", "availMB":
			return true
		}
	case "latency":
		switch column {
		case "eloopUs", "blocked", "slowlog":
			return true
		}
	case "memory":
		switch column {
		case "usedMB", "rssMB", "frag%", "exp/s", "evict/s":
			return true
		}
	case "clients":
		switch column {
		case "conn", "blocked", "ops/s", "hit%":
			return true
		}
	case "network":
		switch column {
		case "inKB/s", "outKB/s", "rxKB/s", "txKB/s":
			return true
		}
	case "cpu":
		switch column {
		case "ops/s", "vkUsr%", "vkSys%", "us%", "load1":
			return true
		}
	case "replication":
		switch column {
		case "role", "replicas", "offsetMB", "offMB/s":
			return true
		}
	}
	if view != "summary" && view != "host" && view != "latency" {
		return true
	}
	return false
}
