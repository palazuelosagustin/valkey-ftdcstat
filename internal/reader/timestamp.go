package reader

import (
	"bytes"
	"strconv"
	"time"
)

const tsMSKey = `"ts_ms":`

func sampleTimestamp(line []byte) (time.Time, bool) {
	idx := bytes.Index(line, []byte(tsMSKey))
	if idx < 0 {
		return time.Time{}, false
	}
	rest := bytes.TrimSpace(line[idx+len(tsMSKey):])
	end := 0
	for end < len(rest) && rest[end] >= '0' && rest[end] <= '9' {
		end++
	}
	if end == 0 {
		return time.Time{}, false
	}
	ms, err := strconv.ParseInt(string(rest[:end]), 10, 64)
	if err != nil {
		return time.Time{}, false
	}
	return time.UnixMilli(ms).UTC(), true
}
