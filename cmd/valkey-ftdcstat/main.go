package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"valkey-ftdcstat/internal/derive"
	"valkey-ftdcstat/internal/discovery"
	"valkey-ftdcstat/internal/reader"
	"valkey-ftdcstat/internal/render"
)

func main() {
	var (
		view     = flag.String("view", "summary", "summary|memory|clients|cpu|persistence|replication|commandstats|host")
		jsonOut  = flag.Bool("json", false, "emit JSON")
		interval = flag.Int("interval", 60, "display interval in seconds")
	)
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "usage: valkey-ftdcstat [--json] [--interval seconds] [--view name] <diagnostic.data>\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(2)
	}
	if *interval < 1 {
		fmt.Fprintln(os.Stderr, "interval must be >= 1")
		os.Exit(2)
	}

	path := flag.Arg(0)
	files, warnings, err := discovery.Discover(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	for _, warning := range warnings {
		fmt.Fprintln(os.Stderr, "warning:", warning.String())
	}

	metadata, metaWarnings, err := reader.ReadMetadata(path, files)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	for _, warning := range metaWarnings {
		fmt.Fprintln(os.Stderr, "warning:", warning.String())
	}

	opts := derive.Options{
		View:         *view,
		Interval:     time.Duration(*interval) * time.Second,
		Metadata:     metadata,
		TimeLocation: time.UTC,
	}
	report, err := derive.BuildFromReader(path, files, metadata, opts)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := render.Report(os.Stdout, report, *jsonOut); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
