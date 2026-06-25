package derive

import (
	"fmt"
	"net"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"valkey-ftdcstat/internal/model"
)

const replInfoPrefix = "valkey.info.replication."

type Replica struct {
	Name       string
	IP         string
	Port       int
	OffsetPath string
}

func replicasFromSample(sample model.MetricSample) []Replica {
	byKey := map[string]*Replica{}
	indexFor := map[string]int{}

	for path, value := range sample.Text {
		key, field, ok := replicaField(path)
		if !ok || field != "name" && field != "ip" {
			continue
		}
		rep := replicaForKey(byKey, indexFor, key)
		switch field {
		case "name":
			rep.Name = value
		case "ip":
			rep.IP = value
		}
	}

	for path := range sample.Values {
		key, field, ok := replicaField(path)
		if !ok {
			continue
		}
		rep := replicaForKey(byKey, indexFor, key)
		switch field {
		case "offset":
			rep.OffsetPath = path
		case "port":
			rep.Port = int(sample.Values[path])
		}
	}

	keys := make([]string, 0, len(byKey))
	for key := range byKey {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		return indexFor[keys[i]] < indexFor[keys[j]]
	})

	out := make([]Replica, 0, len(keys))
	for _, key := range keys {
		rep := *byKey[key]
		if rep.Name == "" {
			rep.Name = key
		}
		out = append(out, rep)
	}
	return out
}

func replicaOffsetColumns(first, last model.MetricSample) []string {
	seen := map[string]bool{}
	var names []string
	for _, sample := range []model.MetricSample{first, last} {
		for _, rep := range replicasFromSample(sample) {
			if rep.Name == "" || seen[rep.Name] {
				continue
			}
			seen[rep.Name] = true
			names = append(names, rep.Name)
		}
	}
	sort.Strings(names)
	return names
}

func replicaNamesFromSample(sample model.MetricSample) []string {
	replicas := replicasFromSample(sample)
	names := make([]string, 0, len(replicas))
	for _, rep := range replicas {
		if rep.Name != "" {
			names = append(names, rep.Name)
		}
	}
	return names
}

func topologyNodes(sample model.MetricSample, capturePath string) map[string]string {
	nodes := map[string]string{}
	if name := localNodeName(capturePath); name != "" {
		if addr := localNodeAddr(sample); addr != "" {
			nodes[name] = addr
		}
	}
	for _, rep := range replicasFromSample(sample) {
		if rep.Name == "" || rep.IP == "" {
			continue
		}
		nodes[rep.Name] = formatNodeAddr(rep.IP, rep.Port)
	}
	if len(nodes) == 0 {
		return nil
	}
	return nodes
}

func fillReplicaOffsets(row *Row, c calculator) {
	for _, rep := range replicasFromSample(c.cur) {
		if rep.OffsetPath == "" {
			continue
		}
		setGauge(row, rep.Name, c, rep.OffsetPath, identity)
	}
}

func replicaField(path string) (key, field string, ok bool) {
	if !strings.HasPrefix(path, replInfoPrefix) {
		return "", "", false
	}
	rest := strings.TrimPrefix(path, replInfoPrefix)
	parts := strings.SplitN(rest, ".", 2)
	if len(parts) != 2 || !isReplicaKey(parts[0]) {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func isReplicaKey(key string) bool {
	return strings.HasPrefix(key, "slave") || strings.HasPrefix(key, "replica")
}

func replicaIndex(key string) int {
	digits := strings.TrimPrefix(strings.TrimPrefix(key, "slave"), "replica")
	n, _ := strconv.Atoi(digits)
	return n
}

func replicaForKey(byKey map[string]*Replica, indexFor map[string]int, key string) *Replica {
	if rep, ok := byKey[key]; ok {
		return rep
	}
	rep := &Replica{}
	byKey[key] = rep
	indexFor[key] = replicaIndex(key)
	return rep
}

var genericCaptureDirs = map[string]bool{
	"diagnostic.data": true,
	"testfixtures":    true,
}

func localNodeName(capturePath string) string {
	if capturePath == "" {
		return ""
	}
	dir := filepath.Base(filepath.Dir(filepath.Clean(capturePath)))
	if dir == "" || dir == "." || genericCaptureDirs[dir] {
		return ""
	}
	return dir
}

func localNodeAddr(sample model.MetricSample) string {
	ip, port := localNodeIPPort(sample)
	return formatNodeAddr(ip, port)
}

func localNodeIPPort(sample model.MetricSample) (string, int) {
	var ip string
	var port int
	for path, value := range sample.Text {
		if !strings.Contains(path, ".listener") || !strings.HasSuffix(path, ".bind") {
			continue
		}
		bindIP, bindPort := parseBindHostPort(value)
		if bindIP != "" {
			ip = bindIP
			port = bindPort
			break
		}
	}
	if port == 0 {
		if v, ok := sample.Get("valkey.info.server.tcp_port"); ok && v > 0 {
			port = int(v)
		}
	}
	return ip, port
}

func formatNodeAddr(ip string, port int) string {
	if ip == "" {
		return ""
	}
	if port > 0 {
		return fmt.Sprintf("%s:%d", ip, port)
	}
	return ip
}

func parseBindHostPort(bind string) (string, int) {
	bind = strings.TrimSpace(bind)
	if bind == "" || strings.HasPrefix(bind, "/") {
		return "", 0
	}
	host := bind
	port := 0
	if h, p, ok := strings.Cut(bind, ":"); ok {
		host = h
		if n, err := strconv.Atoi(p); err == nil {
			port = n
		}
	}
	if ip := net.ParseIP(strings.Trim(host, "[]")); ip != nil {
		return ip.String(), port
	}
	return "", 0
}
