# valkey-ftdcstat

`valkey-ftdcstat` reads `valkey-ftdc` `diagnostic.data` directories and renders
derived vmstat-like tables from the raw JSON samples.

## Build

```bash
go build -o valkey-ftdcstat ./cmd/valkey-ftdcstat
```

## Usage

```bash
valkey-ftdcstat <path-to-diagnostic-data-directory> [--view VIEW] [--interval N] [--verbose] [--json]
```

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
latency       LATENCY LATEST event gauges
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
