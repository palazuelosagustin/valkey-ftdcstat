# valkey-ftdcstat metric reference

## Input and discovery

`valkey-ftdcstat` reads a `valkey-ftdc` **directory**, not an individual `.vkftdc` file.

It discovers:

- `metadata.*.json` — module config (`interval-ms`, `collect-host-stats`, etc.)
- `metrics.*.vkftdc` — newline-delimited JSON samples

Each sample line contains:

- `ts_ms` — sample timestamp (UTC)
- `valkey.info.*` — flattened Valkey `INFO` sections
- `valkey.latency_latest.*` — `LATENCY LATEST` events
- `valkey.slowlog.*` — slowlog length and optional recent entries
- `host.*` — Linux host stats when collected

Flattened paths use dot notation, for example
`valkey.info.stats.total_commands_processed`.

## Views and flags

Views:

| View | Purpose |
|------|---------|
| `summary` | Compact rollup: server, commands, memory, stats, clients, replication, host |
| `server` | Throughput, hit rate, errors, connections, network gauges |
| `memory` | Allocation pressure and expiry/eviction rates |
| `clients` | Connection counts and throughput |
| `cpu` | Valkey CPU rates and host CPU/load |
| `persistence` | RDB/AOF flags and slowlog length |
| `replication` | Role, replica count, offset rates |
| `commandstats` | Per-interval rates for busiest commands (`get/s`, …) |
| `slowlog` | Deduplicated slow operations ranked slowest-first (aggregate, not time series) |
| `host` | vmstat-style host metrics |
| `network` | Valkey and host network throughput |
| `latency` | LATENCY LATEST events plus slowlog/blocked/fork/event-loop fallbacks |

Important flags:

- `--interval N` — display spacing in seconds (default `60`). Does **not** aggregate samples.
- `--avg DURATION` — average derived rows into fixed UTC wall-clock buckets (`1m`–`15m`). Cannot combine with `--interval`.
- `--from ISO_TIME` / `--to ISO_TIME` — filter samples; **from inclusive**, **to exclusive**.
- `--top N` — limit command columns in `summary` and `commandstats` (default `8`; `--top 0` = all commands with activity).
- `--device DEVICE` — host view only; filter disk rates to one block device from `/proc/diskstats`.
- `--json` — machine-readable report (cannot combine with `--web`).
- `--web` — local chart UI; still prints terminal output.
- `--listen ADDR` — web bind address (default `127.0.0.1:0`).
- `--verbose` — expands `memory`, `clients`, `replication`, `host`, and `network` only.

## Header fields

Terminal and web output include:

- **capture** — raw sample count and time span (`path`, `files`, `samples`, `range`).
- **metricsRange** — first/last **derived row** timestamps after filters and interval spacing.
- **serverInfo** — Valkey version, topology, `hz`, `maxclients`, build/git details.
- **moduleConfig** — collector settings when present in metadata.
- **hostInfo** — OS and memory summary when host stats are collected.

Process markers before rows:

- `--- valkey process: pid=… run_id=… time=… ---` on first rendered row.
- `--- valkey restart detected: … ---` when `run_id` or `process_id` changes.
- `gap …: rate baseline reset` when the gap between samples exceeds the gap threshold (default 10× interval or metadata override).

After a gap or restart, **rate columns are omitted** for that row; gauges still render from the current sample.

## Display semantics

- `0` — present and zero.
- `-` — missing, unavailable, not computable, zero denominator, or rate suppressed after gap/restart.
- JSON uses `null` for missing numeric values; string gauges like `repl`/`role` may be absent.
- Rates and percentages: typically one decimal place.
- Memory MiB columns: one decimal place.

## Summary sections

Column order:

```text
time | server | commands | memory | stats | clients | replication | host
```

Command columns (`get/s`, `info/s`, …) are ranked by total call delta over the capture window and limited by `--top`.

## Column formulas

### Server / throughput

| Column | Type | Formula | Source path(s) |
|--------|------|---------|----------------|
| `ops/s` | rate | Δ`total_commands_processed` / Δt | `valkey.info.stats.total_commands_processed` |
| `conn/s` | rate | Δ`total_connections_received` / Δt | `valkey.info.stats.total_connections_received` |
| `hit%` | rate | Δhits / (Δhits + Δmisses) × 100 | `valkey.info.stats.keyspace_hits`, `keyspace_misses` |
| `rej/s` | rate | Δ`rejected_connections` / Δt | `valkey.info.stats.rejected_connections` |
| `err/s` | rate | Δ`total_error_replies` / Δt | `valkey.info.stats.total_error_replies` |
| `exp/s` | rate | Δ`expired_keys` / Δt | `valkey.info.stats.expired_keys` |
| `evict/s` | rate | Δ`evicted_keys` / Δt | `valkey.info.stats.evicted_keys` |
| `inKB/s` | gauge | instantaneous input kbps | `valkey.info.stats.instantaneous_input_kbps` |
| `outKB/s` | gauge | instantaneous output kbps | `valkey.info.stats.instantaneous_output_kbps` |
| `offKB/s` | rate | Δ`master_repl_offset` / Δt / 1024 | `valkey.info.replication.master_repl_offset` |

### Commands

| Column | Type | Formula | Source path(s) |
|--------|------|---------|----------------|
| `<cmd>/s` | rate | Δ`cmdstat_<cmd>.calls` / Δt | `valkey.info.commandstats.cmdstat_<cmd>.calls` |

Display names strip the `cmdstat_` prefix (`cmdstat_get` → `get/s`).

### Memory

| Column | Type | Formula | Source path(s) |
|--------|------|---------|----------------|
| `memMB` / `usedMB` | gauge | bytes / 1024² | `valkey.info.memory.used_memory` |
| `rssMB` | gauge | bytes / 1024² | `valkey.info.memory.used_memory_rss` |
| `maxMB` | gauge | bytes / 1024² | `valkey.info.memory.maxmemory` |
| `frag%` | gauge | fragmentation ratio | `valkey.info.memory.mem_fragmentation_ratio` |
| `rss%` | derived | rssMB / usedMB × 100 | memory used + rss |
| `availMB` | gauge | MemAvailable kB / 1024 | `host.memory.MemAvailable` |
| `luaMB` | gauge | bytes / 1024² | `valkey.info.memory.used_memory_lua` |
| `scripts` | gauge | cached scripts count | `valkey.info.memory.number_of_cached_scripts` |
| `defrag` | gauge | active defrag flag | `valkey.info.memory.active_defrag_running` |

### Clients

| Column | Type | Source path(s) |
|--------|------|----------------|
| `cli` / `conn` | gauge | `valkey.info.clients.connected_clients` |
| `blk` / `blocked` | gauge | `valkey.info.clients.blocked_clients` |
| `pubsub` | gauge | `valkey.info.clients.pubsub_clients` |

### Replication

| Column | Type | Source path(s) |
|--------|------|----------------|
| `repl` / `role` | text | `valkey.info.replication.role` |
| `repls` / `replicas` | gauge | `valkey.info.replication.connected_slaves` |
| `offsetMB` | gauge | `master_repl_offset` / 1024² |
| `offMB/s` | rate | Δ offset / Δt / 1024² |
| `backlog` | text | `repl_backlog_active` (yes/no) |
| `backlogMB` | gauge | `repl_backlog_size` / 1024² |

### Host CPU / load

Host CPU percentages use Δ jiffies from flattened `host.cpu.*` fields divided by Δ total CPU time:

| Column | Source |
|--------|--------|
| `us%` | `host.cpu.user` |
| `sy%` | `host.cpu.system` |
| `id%` | `host.cpu.idle` |
| `wa%` | `host.cpu.iowait` |
| `st%` | `host.cpu.steal` |
| `load1` | `host.loadavg.1m` |

Valkey CPU (`cpu` view):

| Column | Formula |
|--------|---------|
| `vkUsr%` | Δ`used_cpu_user` / Δt × 100 |
| `vkSys%` | Δ`used_cpu_sys` / Δt × 100 |

### Host memory / disk (`host` view)

| Column | Meaning |
|--------|---------|
| `r`, `b` | runnable/blocked processes from `/proc/stat` |
| `swpd`, `free`, `buff`, `cache` | memory snapshot (MiB) |
| `bi`, `bo` | disk read/write KiB/s from `/proc/diskstats` (all devices or `--device`) |
| `forks/s` | Δ processes / Δt |
| `cs/s` | Δ context switches / Δt |
| `rssMB` | Valkey process RSS (verbose) |

Disk parsing: sector deltas × 512 / 1024 / Δt.

### Network

| Column | Formula |
|--------|---------|
| `rxKB/s`, `txKB/s` | Δ bytes from `/proc/net/dev` (excluding `lo`) / 1024 / Δt |

### Latency

Fallback columns (always present):

| Column | Source |
|--------|--------|
| `slowlog` | `valkey.slowlog.len` |
| `slowMaxMs` | max recent slowlog entry ms |
| `blocked` | blocked clients |
| `forkUsec` | `latest_fork_usec` |
| `eloopUs` | `instantaneous_eventloop_duration_usec` |

Dynamic event columns from `LATENCY LATEST`:

- `<event>` — `latest_ms` gauge
- `<event>Max` — `max_ms` gauge

When no LATENCY events exist in the capture, a note explains that fallbacks are shown.

### Slowlog (`--view slowlog`)

Aggregate view over all samples in the capture window (respects `--from`/`--to`).

| Column | Meaning |
|--------|---------|
| `command` | Valkey command name (uppercase) |
| `args` | Argument summary (truncated; collector redaction preserved) |
| `maxMs` | Slowest occurrence |
| `avgMs` | Mean duration across deduplicated occurrences |
| `count` | Times this command+args fingerprint appeared |
| `lastSeen` | Sample time when the most recent matching entry was observed |

Dedup rules:

1. Each slowlog `id` is counted once across overlapping sample snapshots.
2. Rows with the same command+args fingerprint merge into one line with `count > 1`.

Data source: `valkey.slowlog.entries[]` from each sample (`SLOWLOG GET 8` when `collect-slowlog yes`).
This is not full slowlog history — only recent entries snapshotted per interval.

`--top N` limits displayed rows (default **50**; `--top 0` = all patterns).
`--avg` and `--interval` do not apply.

### Persistence

| Column | Source |
|--------|--------|
| `rdb` | `rdb_bgsave_in_progress` |
| `aof` | `aof_enabled` |

## Web UI sections

Charts mirror terminal section groups:

- **summary** — `server`, `commands`, `memory`, `stats`, `clients`, `replication`, `host`
- **host** — `host / CPU`, `host / Memory`, `host / Disks`
- **latency** — fallbacks + `latency / events`
- **commandstats** — all `<cmd>/s` columns in one panel
- **memory/clients/network/replication** — extra subpanels with `--verbose`

API:

- `GET /api/metadata` — view, sections, header text, row count, warnings
- `GET /api/data` — derived rows grouped by section name

## Diagnostic hints

| Symptom | Columns to inspect |
|---------|-------------------|
| Command hot spots | `summary` commands section, `--view commandstats` |
| Memory pressure | `memMB`, `frag%`, `exp/s`, `evict/s` |
| Client pile-up | `cli`, `blk`, `conn/s`, `rej/s` |
| Replication lag | `offKB/s`, `offsetMB`, `repls` |
| Disk bottleneck | `host` view `bi`/`bo`, `wa%` |
| Event-loop stalls | `latency` view `eloopUs`, LATENCY events |
| Slow commands building up | `--view slowlog`, or `latency.slowlog` / `slowMaxMs` gauges |

## Updating this reference

When changing column names, formulas, flags, JSON shape, or output semantics in code, update this file and golden tests under `testfixtures/outputs/` together.
