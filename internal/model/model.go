package model

import "time"

type Capture struct {
	Path            string              `json:"path"`
	Files           []string            `json:"files"`
	Samples         []Sample            `json:"samples,omitempty"`
	MetricSamples   []MetricSample      `json:"metricSamples,omitempty"`
	Metadata        Metadata            `json:"metadata,omitempty"`
	FileMetadata    []any               `json:"fileMetadata,omitempty"`
	Warnings        []Warning           `json:"warnings,omitempty"`
}

type Sample struct {
	TsMS   int64         `json:"ts_ms"`
	Valkey ValkeyMetrics `json:"valkey"`
	Host   HostMetrics   `json:"host"`
}

func (s Sample) Time() time.Time {
	return time.UnixMilli(s.TsMS).UTC()
}

type ValkeyMetrics struct {
	Info          InfoSections    `json:"info"`
	LatencyLatest []LatencyEvent  `json:"latency_latest"`
	Slowlog       SlowlogSnapshot `json:"slowlog"`
}

type InfoSections struct {
	Server       map[string]any            `json:"server"`
	Clients      map[string]any            `json:"clients"`
	Memory       map[string]any            `json:"memory"`
	Persistence  map[string]any            `json:"persistence"`
	Stats        map[string]any            `json:"stats"`
	Replication  map[string]any            `json:"replication"`
	CPU          map[string]any            `json:"cpu"`
	Commandstats map[string]CommandMetrics `json:"commandstats"`
	Cluster      map[string]any            `json:"cluster"`
}

type CommandMetrics struct {
	Calls       float64 `json:"calls"`
	Usec        float64 `json:"usec"`
	UsecPerCall float64 `json:"usec_per_call"`
}

type LatencyEvent struct {
	Event     string  `json:"event"`
	LatestMS  float64 `json:"latest_ms"`
	MaxMS     float64 `json:"max_ms"`
	AllTimeMS float64 `json:"all_time_ms"`
}

type SlowlogSnapshot struct {
	Len     float64       `json:"len"`
	Entries []SlowlogItem `json:"entries,omitempty"`
}

type SlowlogItem struct {
	ID           float64  `json:"id"`
	TS           float64  `json:"ts"`
	DurationUSec float64  `json:"duration_usec"`
	Args         []string `json:"args"`
}

type HostMetrics struct {
	Enabled   bool              `json:"enabled,omitempty"`
	Supported bool              `json:"supported,omitempty"`
	LoadAvg   map[string]any    `json:"loadavg"`
	CPU       map[string]any    `json:"cpu"`
	Memory    map[string]string `json:"memory"`
	Disk      HostDisk          `json:"disk"`
	Network   HostNetwork       `json:"network"`
	Process   HostProcess       `json:"process"`
}

type HostDisk struct {
	Diskstats string `json:"diskstats"`
}

type HostNetwork struct {
	NetDev string `json:"net_dev"`
}

type HostProcess struct {
	Status map[string]string `json:"status"`
	IO     map[string]string `json:"io"`
}
