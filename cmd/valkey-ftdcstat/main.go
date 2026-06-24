package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"valkey-ftdcstat/internal/aggregate"
	"valkey-ftdcstat/internal/derive"
	"valkey-ftdcstat/internal/discovery"
	"valkey-ftdcstat/internal/model"
	"valkey-ftdcstat/internal/reader"
	"valkey-ftdcstat/internal/render"
	"valkey-ftdcstat/internal/webui"
)

type cliOptions struct {
	Path        string
	View        string
	Interval    int
	IntervalSet bool
	Avg         time.Duration
	Device      string
	JSON        bool
	Web         bool
	Listen      string
	Verbose     bool
	Range       model.TimeRange
}

func main() {
	opts, err := parseArgs(os.Args[1:])
	if err != nil {
		printError(os.Stderr, err)
		usage(os.Stderr)
		os.Exit(2)
	}

	files, warnings, err := discovery.Discover(opts.Path)
	if err != nil {
		printError(os.Stderr, err)
		os.Exit(1)
	}
	files = discovery.FilterByTimeRange(files, opts.Range)
	for _, warning := range warnings {
		fmt.Fprintln(os.Stderr, "warning:", warning.String())
	}

	metadata, metaWarnings, err := reader.ReadMetadata(opts.Path, files)
	if err != nil {
		printError(os.Stderr, err)
		os.Exit(1)
	}
	for _, warning := range metaWarnings {
		fmt.Fprintln(os.Stderr, "warning:", warning.String())
	}

	streamOpts := reader.StreamOptions{TimeRange: opts.Range}
	deriveOpts := derive.Options{
		View:         opts.View,
		Interval:     time.Duration(opts.Interval) * time.Second,
		Device:       opts.Device,
		Verbose:      opts.Verbose,
		Metadata:     metadata,
		TimeLocation: time.UTC,
	}

	report, err := derive.BuildFromReader(opts.Path, files, metadata, deriveOpts, streamOpts)
	if err != nil {
		printError(os.Stderr, err)
		os.Exit(1)
	}
	if opts.Avg > 0 && opts.View != "commandstats" {
		report.Rows = aggregate.AverageRows(report.Rows, opts.Avg)
	}

	display := render.DisplayOptions{JSON: opts.JSON, AvgBucket: opts.Avg}
	if opts.Web {
		if err := runWeb(os.Stdout, report, warnings, display, opts); err != nil {
			printError(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if err := render.Report(os.Stdout, report, display); err != nil {
		printError(os.Stderr, err)
		os.Exit(1)
	}
}

func runWeb(w io.Writer, report derive.Report, warnings []model.Warning, display render.DisplayOptions, opts cliOptions) error {
	if opts.View == "commandstats" {
		return errors.New("--web is not supported for --view commandstats")
	}
	dataset := webui.BuildDataset(report, warnings, webui.Options{
		View:         opts.View,
		Avg:          opts.Avg,
		RowsAveraged: opts.Avg > 0,
		TimeRange:    opts.Range,
		TimeLocation: time.UTC,
	})
	if dataset.Metadata.RowCount > 5000 {
		fmt.Fprintln(os.Stderr, "warning: Large capture detected. Consider using --avg 5m or --from/--to for better browser performance.")
	}
	server, err := webui.NewServer(dataset)
	if err != nil {
		return err
	}
	address, err := server.Listen(opts.Listen)
	if err != nil {
		return err
	}
	display.WebURL = address
	if err := render.Report(w, report, display); err != nil {
		_ = server.Close()
		return err
	}
	return server.Serve()
}

func parseArgs(args []string) (cliOptions, error) {
	opts := cliOptions{View: "summary", Interval: 60}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-h" || arg == "--help":
			usage(os.Stdout)
			os.Exit(0)
		case arg == "--json":
			opts.JSON = true
		case arg == "--web":
			opts.Web = true
		case arg == "--verbose":
			opts.Verbose = true
		case arg == "--listen":
			i++
			if i >= len(args) {
				return opts, errors.New("--listen requires a value")
			}
			opts.Listen = args[i]
		case strings.HasPrefix(arg, "--listen="):
			opts.Listen = strings.TrimPrefix(arg, "--listen=")
		case arg == "--avg":
			i++
			if i >= len(args) {
				return opts, errors.New("--avg requires a duration, for example: --avg 5m")
			}
			d, err := time.ParseDuration(args[i])
			if err != nil {
				return opts, errors.New("--avg duration must be between 1m and 15m")
			}
			opts.Avg = d
		case strings.HasPrefix(arg, "--avg="):
			d, err := time.ParseDuration(strings.TrimPrefix(arg, "--avg="))
			if err != nil {
				return opts, errors.New("--avg duration must be between 1m and 15m")
			}
			opts.Avg = d
		case arg == "--view":
			i++
			if i >= len(args) {
				return opts, errors.New("--view requires a value")
			}
			opts.View = args[i]
		case strings.HasPrefix(arg, "--view="):
			opts.View = strings.TrimPrefix(arg, "--view=")
		case arg == "--interval":
			i++
			if i >= len(args) {
				return opts, errors.New("--interval requires a value")
			}
			n, err := strconv.Atoi(args[i])
			if err != nil || n <= 0 {
				return opts, errors.New("--interval must be a positive integer")
			}
			opts.Interval = n
			opts.IntervalSet = true
		case strings.HasPrefix(arg, "--interval="):
			n, err := strconv.Atoi(strings.TrimPrefix(arg, "--interval="))
			if err != nil || n <= 0 {
				return opts, errors.New("--interval must be a positive integer")
			}
			opts.Interval = n
			opts.IntervalSet = true
		case arg == "--device":
			i++
			if i >= len(args) {
				return opts, errors.New("--device requires a value")
			}
			opts.Device = args[i]
		case strings.HasPrefix(arg, "--device="):
			opts.Device = strings.TrimPrefix(arg, "--device=")
		case arg == "--from":
			i++
			if i >= len(args) {
				return opts, errors.New("--from requires a value")
			}
			t, err := parseTimeArg(args[i])
			if err != nil {
				return opts, fmt.Errorf("--from: %w", err)
			}
			opts.Range.From = t
		case strings.HasPrefix(arg, "--from="):
			t, err := parseTimeArg(strings.TrimPrefix(arg, "--from="))
			if err != nil {
				return opts, fmt.Errorf("--from: %w", err)
			}
			opts.Range.From = t
		case arg == "--to":
			i++
			if i >= len(args) {
				return opts, errors.New("--to requires a value")
			}
			t, err := parseTimeArg(args[i])
			if err != nil {
				return opts, fmt.Errorf("--to: %w", err)
			}
			opts.Range.To = t
		case strings.HasPrefix(arg, "--to="):
			t, err := parseTimeArg(strings.TrimPrefix(arg, "--to="))
			if err != nil {
				return opts, fmt.Errorf("--to: %w", err)
			}
			opts.Range.To = t
		case strings.HasPrefix(arg, "-"):
			return opts, fmt.Errorf("unknown option %s", arg)
		default:
			if opts.Path != "" {
				return opts, fmt.Errorf("unexpected argument %s", arg)
			}
			opts.Path = arg
		}
	}
	if opts.Path == "" {
		return opts, errors.New("path to diagnostic data directory is required")
	}
	if !opts.Range.From.IsZero() && !opts.Range.To.IsZero() && !opts.Range.From.Before(opts.Range.To) {
		return opts, errors.New("--from must be before --to")
	}
	if opts.Web && opts.JSON {
		return opts, errors.New("--web cannot be combined with --json")
	}
	if opts.Listen != "" && !opts.Web {
		return opts, errors.New("--listen is only supported with --web")
	}
	if opts.Avg > 0 && (opts.Avg < time.Minute || opts.Avg > 15*time.Minute) {
		return opts, errors.New("--avg duration must be between 1m and 15m")
	}
	if opts.Avg > 0 && opts.IntervalSet {
		return opts, errors.New("--avg cannot be combined with --interval")
	}
	if !derive.ValidView(opts.View) {
		return opts, errors.New("--view must be one of summary, server, memory, clients, cpu, persistence, replication, commandstats, host, network, latency")
	}
	if opts.Verbose && !derive.VerboseSupported(opts.View) {
		return opts, errors.New("--verbose is only supported for memory, clients, replication, host, and network views")
	}
	if opts.Device != "" && opts.View != "host" {
		return opts, errors.New("--device is only supported for --view host")
	}
	return opts, nil
}

func parseTimeArg(value string) (time.Time, error) {
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05Z07:00", "2006-01-02 15:04:05Z07:00"} {
		if t, err := time.Parse(layout, value); err == nil {
			return t, nil
		}
	}
	for _, layout := range []string{"2006-01-02T15:04:05", "2006-01-02 15:04:05", "2006-01-02"} {
		if t, err := time.ParseInLocation(layout, value, time.UTC); err == nil {
			return t, nil
		}
	}
	return time.Time{}, errors.New("expected ISO-8601 timestamp")
}

func usage(w io.Writer) {
	fmt.Fprintf(w, "usage: valkey-ftdcstat <path-to-diagnostic-data-directory> [--view VIEW] [--interval N] [--avg DURATION] [--device DEVICE] [--from ISO_TIME] [--to ISO_TIME] [--json] [--web] [--listen ADDR] [--verbose]\n")
}

func printError(w io.Writer, err error) {
	fmt.Fprintf(w, "valkey-ftdcstat: %v\n", err)
}
