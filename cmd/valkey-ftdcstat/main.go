package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"valkey-ftdcstat/internal/derive"
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

	capture, err := reader.ReadCapture(flag.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if len(capture.Samples) == 0 {
		fmt.Fprintln(os.Stderr, "no samples found")
		os.Exit(1)
	}

	opts := derive.Options{
		View:     *view,
		Interval: time.Duration(*interval) * time.Second,
	}
	report, err := derive.Build(capture, opts)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := render.Report(os.Stdout, report, *jsonOut); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
