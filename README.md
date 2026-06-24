# valkey-ftdcstat

`valkey-ftdcstat` reads `valkey-ftdc` `diagnostic.data` directories and renders
derived vmstat-like tables from the raw JSON samples.

## Build

```bash
go build -o valkey-ftdcstat ./cmd/valkey-ftdcstat
```

## Usage

```bash
valkey-ftdcstat <path-to-diagnostic-data-directory> [--view summary|memory|clients|cpu|persistence|replication|commandstats|host] [--interval N] [--json]
```

`--json` emits the raw capture data as valid JSON for debugging. It does not derive rows, build headers, or calculate deltas.

Default view:

```bash
valkey-ftdcstat diagnostic.data --view summary
```

Focused host vmstat-style view:

```bash
valkey-ftdcstat diagnostic.data --view host
```

Command mix over the full capture:

```bash
valkey-ftdcstat diagnostic.data --view commandstats
```
