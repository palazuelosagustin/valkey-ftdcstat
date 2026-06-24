package render

import (
	"fmt"
	"io"
	"strings"
	"time"

	"valkey-ftdcstat/internal/derive"
)

const headerRepeatRows = 50

type StreamingRenderer struct {
	w                io.Writer
	cols             []string
	sections         []tableSection
	widths           []int
	separators       map[int]bool
	header           []string
	loc              *time.Location
	headerRepeatRows int
	dataRows         int
}

type MetricsRange struct {
	Start time.Time
	End   time.Time
}

func MetricsRangeFromRows(rows []derive.Row) MetricsRange {
	var out MetricsRange
	for _, row := range rows {
		if row.Time.IsZero() {
			continue
		}
		if out.Start.IsZero() {
			out.Start = row.Time
		}
		out.End = row.Time
	}
	return out
}

func newStreamingRenderer(w io.Writer, layout tableLayout, loc *time.Location) *StreamingRenderer {
	if loc == nil {
		loc = time.UTC
	}
	cols := layout.Columns
	header := displayColumns(cols)
	return &StreamingRenderer{
		w:                w,
		cols:             cols,
		sections:         layout.Sections,
		widths:           baseColumnWidths(cols),
		separators:       separatorsFromSections(layout.Sections),
		header:           header,
		loc:              loc,
		headerRepeatRows: headerRepeatRows,
	}
}

func (r *StreamingRenderer) RenderRow(row derive.Row) error {
	line := tableLineForRow(r.cols, row, r.loc)
	growColumnWidths(r.widths, line)
	if r.dataRows == 0 {
		r.printHeader()
	}
	if r.dataRows > 0 && r.dataRows%r.headerRepeatRows == 0 {
		r.printHeader()
	}
	if row.ProcessMarker != "" {
		if _, err := fmt.Fprintln(r.w, row.ProcessMarker); err != nil {
			return err
		}
	}
	if row.Marker != "" {
		if _, err := fmt.Fprintf(r.w, "# %s\n", row.Marker); err != nil {
			return err
		}
	}
	printLine(r.w, line, r.cols, r.widths, r.separators, false)
	r.dataRows++
	return nil
}

func (r *StreamingRenderer) Close() error {
	return nil
}

func (r *StreamingRenderer) printHeader() {
	if len(r.sections) > 0 {
		printGroupLine(r.w, r.widths, r.sections, r.separators)
	}
	printLine(r.w, r.header, r.cols, r.widths, r.separators, true)
}

func tableLineForRow(cols []string, row derive.Row, loc *time.Location) []string {
	line := make([]string, len(cols))
	for i, col := range cols {
		if col == "time" {
			line[i] = formatRowTime(row.Time, loc)
			continue
		}
		line[i] = formatValue(row.Values[col])
	}
	return line
}

func printGroupLine(w io.Writer, widths []int, sections []tableSection, separators map[int]bool) {
	if len(sections) == 0 || len(widths) == 0 {
		return
	}
	positions, lineLen := columnPositions(widths, separators)
	line := []byte(strings.Repeat(" ", lineLen))
	for sep := range separators {
		pipe := positions[sep] - 2
		if pipe >= 0 && pipe < len(line) {
			line[pipe] = '|'
		}
	}
	for _, section := range sections {
		start := maxInt(section.Start, 0)
		end := minInt(section.End, len(widths))
		if end <= start {
			continue
		}
		startPos := positions[start]
		endPos := positions[end-1] + widths[end-1]
		span := endPos - startPos
		label := section.Name
		if len(label) > span {
			label = label[:span]
		}
		offset := startPos + (span-len(label))/2
		copy(line[offset:], label)
	}
	fmt.Fprintln(w, strings.TrimRight(string(line), " "))
}

func columnPositions(widths []int, separators map[int]bool) ([]int, int) {
	positions := make([]int, len(widths))
	lineLen := 0
	for i, width := range widths {
		if i > 0 {
			lineLen++
		}
		if separators[i] {
			lineLen += 2
		}
		positions[i] = lineLen
		lineLen += width
	}
	return positions, lineLen
}

func separatorsFromSections(sections []tableSection) map[int]bool {
	out := map[int]bool{}
	for _, section := range sections {
		if section.Start <= 0 {
			continue
		}
		out[section.Start] = true
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func printLine(w io.Writer, line, cols []string, widths []int, separators map[int]bool, header bool) {
	for j, cell := range line {
		if j > 0 {
			fmt.Fprint(w, " ")
		}
		if separators[j] {
			fmt.Fprint(w, "| ")
		}
		if header || isTextColumn(cols[j]) {
			fmt.Fprintf(w, "%-*s", widths[j], cell)
		} else {
			fmt.Fprintf(w, "%*s", widths[j], cell)
		}
	}
	fmt.Fprintln(w)
}

func displayColumns(cols []string) []string {
	out := append([]string(nil), cols...)
	for i, col := range out {
		if col == "time" {
			out[i] = "datetime"
		}
	}
	return out
}

func formatRowTime(t time.Time, loc *time.Location) string {
	if loc == nil {
		loc = time.UTC
	}
	return t.In(loc).Format("2006-01-02T15:04:05Z")
}

func baseColumnWidths(cols []string) []int {
	header := displayColumns(cols)
	out := make([]int, len(header))
	for i, cell := range header {
		out[i] = len(cell)
	}
	return out
}

func growColumnWidths(widths []int, line []string) {
	for i, cell := range line {
		if i >= len(widths) {
			break
		}
		if len(cell) > widths[i] {
			widths[i] = len(cell)
		}
	}
}

func isTextColumn(col string) bool {
	switch col {
	case "repl", "role", "rdb", "aof", "backlog":
		return true
	default:
		return false
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
