package model

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

type Warning struct {
	Source  string `json:"source,omitempty"`
	Message string `json:"message"`
}

func (w Warning) String() string {
	if w.Source == "" {
		return w.Message
	}
	return w.Source + ": " + w.Message
}

type TimeRange struct {
	From time.Time
	To   time.Time
}

func (r TimeRange) IsZero() bool {
	return r.From.IsZero() && r.To.IsZero()
}

func (r TimeRange) Contains(t time.Time) bool {
	if t.IsZero() {
		return false
	}
	if !r.From.IsZero() && t.Before(r.From) {
		return false
	}
	if !r.To.IsZero() && !t.Before(r.To) {
		return false
	}
	return true
}

func (r TimeRange) Overlaps(start, end time.Time) bool {
	if r.IsZero() || start.IsZero() || end.IsZero() || !end.After(start) {
		return true
	}
	if !r.To.IsZero() && !start.Before(r.To) {
		return false
	}
	if !r.From.IsZero() && !end.After(r.From) {
		return false
	}
	return true
}

// MetricSample is a flattened, path-addressable view of one capture sample.
type MetricSample struct {
	Time            time.Time
	Source          string
	SourceIndex     int
	Values          map[string]float64
	Text            map[string]string
	SlowlogEntries  []SlowlogItem
}

func (s MetricSample) Get(path string) (float64, bool) {
	if s.Values == nil {
		return 0, false
	}
	v, ok := s.Values[path]
	return v, ok
}

func (s MetricSample) GetAny(paths ...string) (float64, bool) {
	for _, path := range paths {
		if v, ok := s.Get(path); ok {
			return v, true
		}
	}
	return 0, false
}

func (s MetricSample) GetText(paths ...string) string {
	for _, path := range paths {
		if s.Text == nil {
			continue
		}
		if v, ok := s.Text[path]; ok && v != "" {
			return v
		}
	}
	return ""
}

type Metadata struct {
	Path          string         `json:"path"`
	Files         []string       `json:"files"`
	FormatVersion int            `json:"formatVersion,omitempty"`
	Module        string         `json:"module,omitempty"`
	Server        map[string]any `json:"server,omitempty"`
	Config        map[string]any `json:"config,omitempty"`
	MaxClients    float64        `json:"maxClients,omitempty"`
	Sidecars      []SidecarMeta  `json:"sidecars,omitempty"`
}

func (m Metadata) Summary() map[string]any {
	return map[string]any{
		"path":          m.Path,
		"formatVersion": m.FormatVersion,
		"module":        m.Module,
		"server":        m.Server,
		"config":        m.Config,
		"maxClients":    m.MaxClients,
	}
}

type SidecarMeta struct {
	Path        string `json:"path"`
	CurrentFile string `json:"currentFile,omitempty"`
	CreatedAtMS int64  `json:"createdAtMs,omitempty"`
}

func Number(m map[string]any, key string) float64 {
	if m == nil {
		return 0
	}
	return AsFloat(m[key])
}

func Text(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	switch v := m[key].(type) {
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	case nil:
		return ""
	default:
		return fmt.Sprint(v)
	}
}

func AsFloat(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case float32:
		return float64(n)
	case int:
		return float64(n)
	case int64:
		return float64(n)
	case int32:
		return float64(n)
	case uint64:
		return float64(n)
	case uint32:
		return float64(n)
	case json.Number:
		f, _ := n.Float64()
		return f
	case string:
		f, _ := strconv.ParseFloat(strings.TrimSpace(n), 64)
		return f
	default:
		return 0
	}
}

func ParseKBValue(s string) float64 {
	parts := strings.Fields(strings.TrimSpace(s))
	if len(parts) == 0 {
		return 0
	}
	f, _ := strconv.ParseFloat(parts[0], 64)
	return f / 1024.0
}

func ParseKBBytes(s string) float64 {
	parts := strings.Fields(strings.TrimSpace(s))
	if len(parts) == 0 {
		return 0
	}
	f, _ := strconv.ParseFloat(parts[0], 64)
	if len(parts) > 1 && strings.EqualFold(parts[1], "kB") {
		return f * 1024.0
	}
	return f
}

func RoundRate(v float64) float64 {
	if v == 0 || math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	return math.Round(v*100) / 100
}
