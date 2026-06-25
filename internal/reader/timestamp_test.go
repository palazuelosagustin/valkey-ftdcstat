package reader

import "testing"

func TestSampleTimestamp(t *testing.T) {
	line := []byte(`{"ts_ms":1781956751893,"valkey":{"info":{}}}`)
	ts, ok := sampleTimestamp(line)
	if !ok || ts.UnixMilli() != 1781956751893 {
		t.Fatalf("ts=%v ok=%v", ts, ok)
	}
}

func TestSampleTimestampSkipsOutsideRange(t *testing.T) {
	line := []byte(`{"ts_ms":1000}`)
	if _, ok := sampleTimestamp(line); !ok {
		t.Fatal("expected timestamp")
	}
	if _, ok := sampleTimestamp([]byte(`{"valkey":{}}`)); ok {
		t.Fatal("expected missing timestamp")
	}
}
