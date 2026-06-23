package model

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

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
