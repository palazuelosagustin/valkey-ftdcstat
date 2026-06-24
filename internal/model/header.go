package model

type Header struct {
	HostInfo        map[string]any `json:"hostInfo,omitempty"`
	BuildInfo       map[string]any `json:"buildInfo,omitempty"`
	ReplicationInfo map[string]any `json:"replicationInfo,omitempty"`
	ModuleConfig    map[string]any `json:"moduleConfig,omitempty"`
}
