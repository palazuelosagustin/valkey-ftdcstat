package flatten

import (
	"fmt"
	"strings"

	"valkey-ftdcstat/internal/model"
)

const (
	pathValkeyInfo = "valkey.info."
	pathValkey     = "valkey."
	pathHost       = "host."
)

// Sample converts a decoded capture sample into a flattened MetricSample.
func Sample(sample model.Sample, source string, sourceIndex int) model.MetricSample {
	out := model.MetricSample{
		Time:        sample.Time(),
		Source:      source,
		SourceIndex: sourceIndex,
		Values:      map[string]float64{},
		Text:        map[string]string{},
	}
	if sample.TsMS == 0 {
		out.Time = out.Time.UTC()
	}

	flattenMap(out, pathValkeyInfo+"server.", sample.Valkey.Info.Server)
	flattenMap(out, pathValkeyInfo+"clients.", sample.Valkey.Info.Clients)
	flattenMap(out, pathValkeyInfo+"memory.", sample.Valkey.Info.Memory)
	flattenMap(out, pathValkeyInfo+"persistence.", sample.Valkey.Info.Persistence)
	flattenMap(out, pathValkeyInfo+"stats.", sample.Valkey.Info.Stats)
	flattenMap(out, pathValkeyInfo+"replication.", sample.Valkey.Info.Replication)
	flattenMap(out, pathValkeyInfo+"cpu.", sample.Valkey.Info.CPU)
	flattenMap(out, pathValkeyInfo+"cluster.", sample.Valkey.Info.Cluster)

	for name, cmd := range sample.Valkey.Info.Commandstats {
		base := pathValkeyInfo + "commandstats." + name + "."
		putFloat(out, base+"calls", cmd.Calls)
		putFloat(out, base+"usec", cmd.Usec)
		putFloat(out, base+"usec_per_call", cmd.UsecPerCall)
	}

	for _, event := range sample.Valkey.LatencyLatest {
		if event.Event == "" {
			continue
		}
		base := pathValkey + "latency_latest." + event.Event + "."
		putFloat(out, base+"latest_ms", event.LatestMS)
		putFloat(out, base+"max_ms", event.MaxMS)
		putFloat(out, base+"all_time_ms", event.AllTimeMS)
	}

	putFloat(out, pathValkey+"slowlog.len", sample.Valkey.Slowlog.Len)

	host := sample.Host
	if host.Supported || host.Enabled || len(host.CPU) > 0 || host.Disk.Diskstats != "" {
		putFloat(out, pathHost+"supported", boolFloat(host.Supported))
	}
	flattenMap(out, pathHost+"loadavg.", host.LoadAvg)
	flattenMap(out, pathHost+"cpu.", host.CPU)

	for key, value := range host.Memory {
		path := pathHost + "memory." + key
		out.Text[path] = value
		if mb := model.ParseKBValue(value); mb > 0 || strings.Contains(value, "kB") {
			putFloat(out, path+".mb", mb)
		}
	}

	if host.Disk.Diskstats != "" {
		out.Text[pathHost+"disk.diskstats"] = host.Disk.Diskstats
	}
	if host.Network.NetDev != "" {
		out.Text[pathHost+"network.net_dev"] = host.Network.NetDev
	}

	for key, value := range host.Process.Status {
		path := pathHost + "process.status." + key
		out.Text[path] = value
		if strings.HasPrefix(key, "Vm") || strings.Contains(key, "_bytes") {
			if bytes := model.ParseKBBytes(value); bytes > 0 || strings.Contains(value, "kB") {
				putFloat(out, path+".bytes", bytes)
			}
		}
		if key == "voluntary_ctxt_switches" || key == "nonvoluntary_ctxt_switches" {
			putFloat(out, path, model.AsFloat(value))
		}
	}
	for key, value := range host.Process.IO {
		path := pathHost + "process.io." + key
		out.Text[path] = value
		putFloat(out, path, model.AsFloat(value))
	}

	if role := model.Text(sample.Valkey.Info.Replication, "role"); role != "" {
		out.Text[pathValkeyInfo+"replication.role"] = role
	}

	return out
}

func flattenMap(out model.MetricSample, prefix string, values map[string]any) {
	for key, value := range values {
		path := prefix + key
		switch typed := value.(type) {
		case map[string]any:
			flattenMap(out, path+".", typed)
		default:
			if f, ok := asFloatOK(typed); ok {
				putFloat(out, path, f)
				continue
			}
			if text := model.Text(map[string]any{key: typed}, key); text != "" {
				out.Text[path] = text
			}
		}
	}
}

func putFloat(out model.MetricSample, path string, value float64) {
	if out.Values == nil {
		out.Values = map[string]float64{}
	}
	out.Values[path] = value
}

func asFloatOK(v any) (float64, bool) {
	switch n := v.(type) {
	case nil:
		return 0, false
	case bool:
		if n {
			return 1, true
		}
		return 0, true
	case string:
		text := strings.TrimSpace(n)
		if text == "" {
			return 0, false
		}
		if text == "yes" || text == "true" {
			return 1, true
		}
		if text == "no" || text == "false" {
			return 0, true
		}
		f, err := parseFloat(text)
		if err != nil {
			return 0, false
		}
		return f, true
	default:
		f := model.AsFloat(v)
		if f == 0 && v == 0 {
			return 0, true
		}
		if f != 0 || isNumericZero(v) {
			return f, true
		}
		return 0, false
	}
}

func isNumericZero(v any) bool {
	switch n := v.(type) {
	case int, int32, int64, uint, uint32, uint64, float32, float64:
		return model.AsFloat(n) == 0
	default:
		return false
	}
}

func parseFloat(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}

func boolFloat(v bool) float64 {
	if v {
		return 1
	}
	return 0
}
