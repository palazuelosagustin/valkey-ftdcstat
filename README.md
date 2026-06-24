# valkey-ftdcstat

`valkey-ftdcstat` reads `valkey-ftdc` `diagnostic.data` directories and renders
derived vmstat-like tables from the raw JSON samples.

## Build

```bash
go build -o valkey-ftdcstat ./cmd/valkey-ftdcstat
```

## Usage

```bash
valkey-ftdcstat <path-to-diagnostic-data-directory> [--view VIEW] [--interval N] [--avg DURATION] [--top N] [--device DEVICE] [--from ISO_TIME] [--to ISO_TIME] [--json] [--web] [--listen ADDR] [--verbose]
```

See [docs/metric-reference.md](docs/metric-reference.md) for column definitions, formulas, and
diagnostic hints. Planned work is tracked in [docs/backlog.md](docs/backlog.md).

An agent skill bundle lives at [skills/valkey-ftdcstat-skill/](skills/valkey-ftdcstat-skill/).

### Time range

`--from` is inclusive and `--to` is exclusive:

```bash
valkey-ftdcstat diagnostic.data --from "2026-06-20T12:00:00" --to "2026-06-20T18:00:00"
```

### Averaging

`--avg` averages derived rows into fixed UTC wall-clock buckets between `1m`
and `15m`. It cannot be combined with `--interval`.

```bash
valkey-ftdcstat diagnostic.data --view summary --avg 5m
```

### Host disk filter

`--device` limits host-view disk rates to one block device from `/proc/diskstats`:

```bash
valkey-ftdcstat diagnostic.data --view host --device sda
```

### Web UI

`--web` starts a local HTTP server and still prints the normal terminal table.

```bash
valkey-ftdcstat diagnostic.data --web --view summary --avg 5m
valkey-ftdcstat diagnostic.data --web --listen 127.0.0.1:8080
```

`--web` cannot be combined with `--json`. For large captures, prefer `--avg` or
`--from`/`--to` to keep browser rendering responsive.

The web UI groups charts by section:

- **summary** — `server`, `memory`, `stats`, `clients`, `replication`, `host`
- **host** — `host / CPU`, `host / Memory`, `host / Disks`
- **latency** — fallback gauges plus dynamic `latency / events` columns
- **memory**, **clients**, **network**, **replication** — extra subpanels when `--verbose` is set
- **commandstats** — per-interval command rates (`get/s`, `set/s`, …)
- **slowlog** — ranked slow-operation table (aggregate view, not time series)

Views:

Terminal output includes:

- **capture** range from raw samples (`path`, `files`, `samples`, `range`)
- **metricsRange** from derived table rows (after filters and interval)
- **serverInfo** with Valkey version, topology, `hz`, and `maxclients`
- **moduleConfig** when present in capture metadata (`interval-ms`, `collect-host-stats`, etc.)
- **hostInfo** when host stats are collected

The summary view prints section labels (`server`, `commands`, `memory`, `stats`,
`clients`, `replication`, `host`) above column groups separated by `|`, and
repeats the header every 50 rows on long captures.

Top command rates (`get/s`, `set/s`, …) are included in **summary** by default
for the busiest commands in the capture. Use `--top N` to change how many command
columns are shown (`--top 0` shows all commands with activity).

Views:

```text
summary       compact rollup across server, memory, clients, host, and replication
server        throughput, hit rate, errors, and connection activity
memory        allocation pressure and eviction/expiry rates
clients       connection counts and throughput
cpu           Valkey and host CPU
persistence   RDB/AOF state and slowlog length
replication   role, replicas, and replication offset
commandstats  per-interval command rates for the busiest commands
slowlog       deduplicated slow operations ranked slowest-first (`--top` limits rows)
host          vmstat-style host metrics
network       Valkey and host network throughput
latency       LATENCY LATEST event gauges plus slowlog/blocked/fork/event-loop fallbacks
```

Examples:

```bash
valkey-ftdcstat diagnostic.data --view summary
valkey-ftdcstat diagnostic.data --view server --interval 300
valkey-ftdcstat diagnostic.data --view network --verbose
valkey-ftdcstat diagnostic.data --view latency --json
valkey-ftdcstat diagnostic.data --view commandstats
valkey-ftdcstat diagnostic.data --view slowlog --top 20
```

`--verbose` expands columns for `memory`, `clients`, `replication`, `host`, and
`network` views.

## Tests

```bash
go test ./...
```

Golden CLI output is pinned under `testfixtures/outputs/` against the small
fixture in `testfixtures/diagnostic.data/`. The full capture under `testdata/`
is optional local data (gitignored).

Regenerate golden files after intentional output changes — see
[skills/valkey-ftdcstat-skill/references/api_reference.md](skills/valkey-ftdcstat-skill/references/api_reference.md).
