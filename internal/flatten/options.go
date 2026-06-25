package flatten

// Options controls how much work Sample does for a given view.
type Options struct {
	IncludeSlowlogEntries bool
	IncludeLatency        bool
	SkipPersistence       bool
	SkipCluster           bool
	SkipValkeyCPU         bool
	SkipHostDisk          bool
	SkipHostNetwork       bool
	SkipProcessIO         bool
	SkipProcessStatus     bool
}

// OptionsForView returns flatten settings tuned for the selected report view.
func OptionsForView(view string, verbose bool) Options {
	opts := Options{
		IncludeSlowlogEntries: view == "slowlog",
		IncludeLatency:        view == "latency",
	}
	switch view {
	case "summary":
		opts.SkipPersistence = true
		opts.SkipCluster = true
		opts.SkipValkeyCPU = true
		opts.SkipProcessIO = true
		opts.SkipProcessStatus = true
		if !verbose {
			opts.SkipHostDisk = true
			opts.SkipHostNetwork = true
		}
	case "server", "clients", "memory", "cpu", "persistence", "replication", "commandstats":
		opts.SkipHostDisk = true
		opts.SkipHostNetwork = true
		opts.SkipProcessIO = true
		opts.SkipProcessStatus = true
	case "network":
		opts.SkipHostDisk = true
		opts.SkipProcessIO = true
		opts.SkipProcessStatus = true
	case "host":
		if !verbose {
			opts.SkipProcessIO = true
		}
	}
	return opts
}
