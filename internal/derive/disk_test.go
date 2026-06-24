package derive

import "testing"

func TestParseDiskstatsFiltersDevice(t *testing.T) {
	blob := "   8       0 sda 100 0 200 10 0 0 20 10 0 0 0 0 0 0 0 0 0\n   8      16 sdb 1000 0 2000 100 0 0 200 100 0 0 0 0 0 0 0 0 0\n"
	allRead, allWrite := parseDiskstats(blob, "")
	sdaRead, sdaWrite := parseDiskstats(blob, "sda")
	if allRead != 1100 || allWrite != 110 {
		t.Fatalf("all read=%v write=%v", allRead, allWrite)
	}
	if sdaRead != 100 || sdaWrite != 10 {
		t.Fatalf("sda read=%v write=%v", sdaRead, sdaWrite)
	}
}
