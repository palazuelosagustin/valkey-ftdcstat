# Backlog

## Phase 8 — Dedicated `slowlog` view

**Status:** planned (not started)

### Goal

Add `--view slowlog` as a **unique aggregate view** (not a time series) that surfaces slow operations from the capture's slowlog samples.

### Requirements

1. **Dedicated view** — separate from `--view latency` (which tracks LATENCY LATEST gauges and fallbacks over time).
2. **Slowest first** — sort entries by duration descending.
3. **Deduplicate repeated queries** — identical command patterns (same command + normalized args fingerprint) appear **once**.
4. **Repeat counter** — show how many times each unique slow operation occurred in the capture window (`count` column).
5. **Useful columns** (initial proposal):

| Column | Meaning |
|--------|---------|
| `command` | Valkey command name |
| `args` | Redacted or truncated argument summary |
| `maxMs` | Slowest occurrence (duration) |
| `avgMs` | Average duration across occurrences |
| `count` | Number of times this fingerprint appeared |
| `lastSeen` | Timestamp of most recent occurrence |

### Data source

Slowlog entries are already collected by `valkey-ftdc` when `collect-slowlog yes`:

- `valkey.slowlog.len` — queue length gauge (used today in `latency` view)
- `valkey.slowlog.entries[]` — up to 8 recent entries per sample with `id`, `ts`, `duration_usec`, `args`

Phase 8 needs to **accumulate entries across all samples** in the selected time range, merge duplicates, and render one ranked table.

### Open design questions

- **Fingerprinting:** hash `(command, arg0, arg1, …)` with optional normalization (strip volatile IDs, collapse large keys).
- **Arg redaction:** respect collector `slowlog-redact` setting; never expand secrets in output.
- **Web UI:** static ranked table (similar to early commandstats design) or expandable rows — default to table.
- **Limits:** `--top N` to cap rows; default 50?
- **JSON shape:** `{ "entries": [ { "command", "args", "maxMs", "count", … } ] }`

### Non-goals for Phase 8

- Per-interval slowlog time series (keep `latency.slowlog` gauge for that).
- Storing full slowlog history beyond what the collector already captures per sample.

### Suggested implementation order

1. Derive `slowlog.Report` with fingerprint + aggregation across samples.
2. Terminal renderer (table) + `--json` output.
3. `--view slowlog` CLI wiring and validation.
4. Web UI ranked table panel.
5. Golden tests + metric-reference update + skill update.
