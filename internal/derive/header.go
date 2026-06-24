package derive

import (
	"sort"
	"strings"

	"valkey-ftdcstat/internal/model"
)

func buildHeader(sample model.MetricSample, metadata model.Metadata) model.Header {
	buildInfo := buildBuildInfo(sample)
	if metadata.MaxClients > 0 {
		if buildInfo == nil {
			buildInfo = map[string]any{}
		}
		buildInfo["maxClients"] = metadata.MaxClients
	}
	if v, ok := sample.Get("valkey.info.server.hz"); ok && v > 0 {
		if buildInfo == nil {
			buildInfo = map[string]any{}
		}
		buildInfo["hz"] = v
	}
	return model.Header{
		HostInfo:        buildHostInfo(sample),
		BuildInfo:       buildInfo,
		ReplicationInfo: buildReplicationInfo(sample),
		ModuleConfig:    buildModuleConfig(metadata),
	}
}

func buildModuleConfig(metadata model.Metadata) map[string]any {
	if len(metadata.Config) == 0 {
		return nil
	}
	keys := []string{
		"interval-ms", "max-file-mb", "collect-host-stats", "collect-slowlog",
		"slowlog-redact", "path", "compression",
	}
	out := map[string]any{}
	for _, key := range keys {
		if v, ok := metadata.Config[key]; ok {
			out[key] = v
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func buildHostInfo(sample model.MetricSample) map[string]any {
	info := map[string]any{}
	if os := sample.GetText("valkey.info.server.os"); os != "" {
		info["os"] = os
	}
	memory := map[string]string{}
	for _, key := range []string{"MemAvailable", "Buffers", "Cached", "MemFree", "MemTotal"} {
		if v := sample.Text["host.memory."+key]; v != "" {
		switch key {
		case "MemAvailable":
			memory["available"] = v
		case "MemFree":
			memory["free"] = v
		case "MemTotal":
			memory["total"] = v
		case "Buffers":
			memory["buffers"] = v
		case "Cached":
			memory["cached"] = v
		default:
			memory[strings.ToLower(key)] = v
		}
		}
	}
	if len(memory) > 0 {
		info["memory"] = memory
	}
	if v, ok := sample.Get("host.loadavg.1m"); ok {
		loadavg := map[string]any{"1m": v}
		if v5, ok := sample.Get("host.loadavg.5m"); ok {
			loadavg["5m"] = v5
		}
		if v15, ok := sample.Get("host.loadavg.15m"); ok {
			loadavg["15m"] = v15
		}
		info["loadavg"] = loadavg
	}
	cpu := map[string]any{}
	for _, field := range []string{"user", "system", "idle", "iowait", "procs_running", "procs_blocked"} {
		if v, ok := sample.Get("host.cpu." + field); ok {
			cpu[field] = v
		}
	}
	if len(cpu) > 0 {
		info["cpu"] = cpu
	}
	if len(info) == 0 {
		return nil
	}
	return info
}

func buildBuildInfo(sample model.MetricSample) map[string]any {
	info := map[string]any{}
	putHeaderText(info, "valkeyVersion", sample.GetText("valkey.info.server.valkey_version"))
	putHeaderText(info, "redisVersion", sample.GetText("valkey.info.server.redis_version"))
	putHeaderText(info, "releaseStage", sample.GetText("valkey.info.server.valkey_release_stage"))
	putHeaderText(info, "buildID", sample.GetText("valkey.info.server.redis_build_id"))
	putHeaderText(info, "gccVersion", sample.GetText("valkey.info.server.gcc_version"))
	putHeaderText(info, "os", sample.GetText("valkey.info.server.os"))
	putHeaderText(info, "multiplexingAPI", sample.GetText("valkey.info.server.multiplexing_api"))
	putHeaderText(info, "serverMode", sample.GetText("valkey.info.server.server_mode"))
	putHeaderText(info, "gitSHA1", sample.GetText("valkey.info.server.redis_git_sha1"))
	if v, ok := sample.Get("valkey.info.server.arch_bits"); ok && v > 0 {
		info["archBits"] = v
	}
	if v, ok := sample.Get("valkey.info.server.redis_git_dirty"); ok {
		info["gitDirty"] = v != 0
	}
	if len(info) == 0 {
		return nil
	}
	return info
}

func buildReplicationInfo(sample model.MetricSample) map[string]any {
	info := map[string]any{}
	putHeaderText(info, "role", sample.GetText(pathReplRole))
	if v, ok := sample.Get(pathReplSlaves); ok {
		info["replicas"] = v
	}
	if v, ok := sample.Get("valkey.info.cluster.cluster_enabled"); ok {
		info["clusterEnabled"] = v != 0
	}
	if names := replicaNamesFromSample(sample); len(names) > 0 {
		info["replicaNames"] = names
	}
	if len(info) == 0 {
		return nil
	}
	return info
}

func replicaNamesFromSample(sample model.MetricSample) []string {
	var names []string
	for path, value := range sample.Text {
		if (strings.Contains(path, ".slave") || strings.Contains(path, ".replica")) && strings.HasSuffix(path, ".name") && value != "" {
			names = append(names, value)
		}
	}
	sort.Strings(names)
	return names
}

func putHeaderText(dst map[string]any, key, value string) {
	if value != "" {
		dst[key] = value
	}
}
