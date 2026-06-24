# valkey-ftdcstat

`valkey-ftdcstat` reads `valkey-ftdc` `diagnostic.data` directories and renders
derived vmstat-like tables from the raw JSON samples.

## Build

```bash
go build -o valkey-ftdcstat ./cmd/valkey-ftdcstat
```

## Usage

```bash
valkey-ftdcstat <path-to-diagnostic-data-directory> [--view VIEW] [--interval N] [--avg DURATION] [--device DEVICE] [--from ISO_TIME] [--to ISO_TIME] [--json] [--web] [--listen ADDR] [--verbose]
```

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

Views:

```text
summary       compact rollup across server, memory, clients, host, and replication
server        throughput, hit rate, errors, and connection activity
memory        allocation pressure and eviction/expiry rates
clients       connection counts and throughput
cpu           Valkey and host CPU
persistence   RDB/AOF state and slowlog length
replication   role, replicas, and replication offset
commandstats  command mix over the full capture
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
```

`--verbose` expands columns for `memory`, `clients`, `replication`, `host`, and
`network` views.
