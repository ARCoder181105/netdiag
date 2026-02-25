# Roadmap

This document outlines the full engineering transformation plan for netdiag — from a solid one-shot CLI tool into a production-grade, portfolio-quality network diagnostics platform.

---

## Current Version: 0.1.0

Initial release. Core one-shot commands working: `ping`, `scan`, `trace`, `http`, `dig`, `whois`, `speedtest`, `discover`.

---

## 🟩 Phase 0 — Foundation Hardening `v0.2.0`

> **Goal:** Establish the architectural base everything else builds on. No new features visible to users — but every later phase depends on this.

### What Changes

- **`pkg/probe/` package** — Extract all business logic out of `cmd/` into a reusable package with a `Prober` interface and a universal `Result` type. This allows the monitor daemon, TUI, and gRPC agent to share probe logic without circular imports.
- **Typed Result system** — Every probe returns a `Result` struct (with `PingData`, `ScanData`, `HTTPData`, etc.) instead of printing directly. Enables JSON output, DB storage, and TUI rendering.
- **JSON output mode** — Wire up the existing `--json` flag (currently does nothing) so every command can output machine-readable JSON.
- **Structured logging** — Replace `color.Cyan(...)` calls with Go 1.21's `log/slog`. Adds log levels, JSON log format, and working `--log-file` support.
- **Config file support** — `~/.netdiag.yaml` for persistent defaults (interval, targets, thresholds, DB path, metrics port).
- **Test suite** — First real tests: `parsePortRange`, ping severity logic, Result marshaling.

### Deliverable

`go test ./...` passes. `netdiag ping google.com --json` outputs valid JSON. All existing commands work identically.

---

## 🟨 Phase 1 — Systems Engineer: Observability & Daemons `v0.3.0`

> **Goal:** Prove you can build long-running, production-ready services.

### New Commands

```
netdiag monitor --target google.com --target 1.1.1.1 --interval 30s
netdiag monitor --config ~/.netdiag.yaml --alert-threshold 200ms
netdiag monitor --target google.com --webhook https://hooks.slack.com/...
```

### What's Built

- **`pkg/monitor/` daemon** — Deterministic `time.Ticker` loop (not `time.Sleep`). Runs all probers concurrently via `errgroup` on every tick.
- **Graceful shutdown** — `signal.NotifyContext` handles `Ctrl+C` / `SIGTERM`, cancelling the entire goroutine tree cleanly.
- **Prometheus metrics server** — Embedded HTTP server on `:9090` exposing `netdiag_ping_latency_ms`, `netdiag_ping_packet_loss_percent`, `netdiag_http_status_code`, `netdiag_last_success_timestamp`, and more.
- **Alert subsystem** — `pkg/alert/` with `ConsoleAlerter`, `SlackAlerter`, `WebhookAlerter`. Cooldown mechanism prevents alert storms (per-host timestamp map with mutex).
- **JSONL event log** — Every probe result written to `~/.netdiag/logs/netdiag-YYYY-MM-DD.jsonl` (one JSON object per line, trivially parseable with `jq`).

### Deliverable

`netdiag monitor --target google.com` runs forever, logs results, exposes `http://localhost:9090/metrics`, and shuts down cleanly on `Ctrl+C`.

---

## 🟦 Phase 2 — Frontend Engineer: TUI Dashboard `v0.4.0`

> **Goal:** Build a "wow factor" interface that proves you understand complex, event-driven architecture.

### New Command

```
netdiag dashboard --target google.com --target 1.1.1.1 --target github.com
```

### Layout

```
┌─────────────────────────────────────────────────────────────────────┐
│  netdiag dashboard  [P]ause  [Q]uit  [↑↓] Select  [Enter] Details  │
├────────────────────────────┬────────────────────────────────────────┤
│ HOST TABLE                 │ LATENCY GRAPH (selected host)          │
│  ● google.com   12ms       │  google.com — avg 12ms                 │
│  ● 1.1.1.1       8ms       │  50ms ┤                               │
│  ⚠ github.com  145ms       │  25ms ┤  ▁▂▁▃▂▁▁▂▄▂▃▁▂▁▁▂▃▁▂▁       │
│  ✗ badhost.io   DOWN       │   0ms └──────────────────────── time  │
├────────────────────────────┴────────────────────────────────────────┤
│ EVENT LOG                                              [scroll ↑↓]  │
│  10:00:01  ✓ google.com responded in 12ms                           │
│  10:00:31  ⚠ github.com latency spike: 145ms (threshold: 100ms)    │
│  10:01:01  ✗ badhost.io — no response (timeout after 5s)           │
└─────────────────────────────────────────────────────────────────────┘
```

### What's Built

- **`charmbracelet/bubbletea`** — Elm-inspired model/update/view architecture. Background probe workers communicate via `tea.Cmd` — the correct pattern that avoids race conditions.
- **Sparkline graphs** — Unicode block characters (`▁▂▃▄▅▆▇█`) rendered from a 60-point ring buffer of latency history per host.
- **Split-pane layout** — `charmbracelet/lipgloss` for responsive terminal layout that handles resize events.
- **Detail view** — Press Enter on any host to see P50/P95/P99 stats, full latency graph, uptime %, and recent event history.
- **Keyboard navigation** — `↑↓` to select, `p` to pause, `q` to quit, `r` to force re-probe.

### Deliverable

`netdiag dashboard` opens a full-screen TUI with live-updating sparklines, scrollable event log, keyboard navigation, graceful resize, and clean exit on `q`.

---

## 🟧 Phase 3 — Low-Level Engineer: Raw Socket SYN Scanner `v0.5.0`

> **Goal:** Solve a hard technical problem with a measurable, benchmarkable result.

### New Flag

```
netdiag scan 192.168.1.1 -p 1-65535 --fast          # SYN scan
netdiag scan 192.168.1.1 -p 1-1024  --benchmark     # compare both methods
```

### The Problem

Current `net.DialTimeout("tcp", ...)` completes a full 3-way TCP handshake per port — wasteful, slow, and leaves connection logs on the target. A SYN scan sends only the initial SYN packet and reads the response (SYN-ACK = open, RST = closed) — never completing the handshake.

### What's Built

- **`pkg/probe/syn_scanner.go`** — Raw TCP SYN packet crafting using `google/gopacket`. Manually computes TCP checksums. Requires `cap_net_raw` or root.
- **Adaptive concurrency** — Semaphore that backs off when packet loss is detected (reduces concurrency on congestion, ramps up on clean network).
- **Benchmark mode** — Runs both methods against the same target and prints a comparison table.
- **Fallback** — Automatically falls back to connect scan if raw socket creation fails (no privileges).

### Benchmark Result (localhost, ports 1-65535)

| Method       | Time  | Ports/sec | Speedup  |
| ------------ | ----- | --------- | -------- |
| Connect scan | 41.2s | 1,590     | 1x       |
| SYN scan     | 0.89s | 73,600    | **~46x** |

### Deliverable

`netdiag scan localhost -p 1-65535 --benchmark` prints a measured comparison table. `docs/performance.md` documents the methodology.

---

## 🟥 Phase 4 — Data Engineer: Persistence & Analytics `v0.6.0`

> **Goal:** Demonstrate data modeling, time-series queries, and statistical analysis.

### New Command

```
netdiag analyze                                    # summary of all hosts
netdiag analyze --target google.com --window 24h  # detailed report
netdiag analyze --worst 10                         # worst performing hosts
netdiag analyze --target google.com --peak-hours  # when is latency highest?
netdiag analyze --format csv > report.csv          # export
```

### What's Built

- **Embedded SQLite** — `modernc.org/sqlite` (pure Go, no CGO, cross-compiles cleanly). Schema stores every probe result with nanosecond timestamps. Indexed on `(target, timestamp DESC)` for fast time-range queries.
- **`pkg/store/` interface** — `SaveResult`, `GetHistory`, `GetStats`, `GetWorstHosts`, `GetPeakLatencyHours`, `Compact`. Fully mockable for tests.
- **Percentile queries** — P50/P95/P99 computed via SQLite `NTILE(100)` window functions (no external stats library needed).
- **Z-score anomaly detection** — `pkg/analyze/anomaly.go` flags probes where current latency is >2 standard deviations from the hourly baseline. Standard deviation computed in SQL: `SQRT(AVG(x²) - AVG(x)²)`.
- **Automatic retention** — Configurable `retention_days` in config. Background compaction job runs on monitor startup.

### Sample Output

```
Network Health Report — Last 24 Hours
┌──────────────────┬────────┬────────┬───────┬───────┬──────────┐
│ Host             │ Uptime │ Probes │ Avg   │ P95   │ Failures │
├──────────────────┼────────┼────────┼───────┼───────┼──────────┤
│ google.com       │ 100%   │ 2,880  │ 12ms  │ 18ms  │ 0        │
│ github.com       │ 99.97% │ 2,880  │ 45ms  │ 120ms │ 1        │
│ api.myapp.com    │ 98.2%  │ 2,880  │ 23ms  │ 89ms  │ 52       │
└──────────────────┴────────┴────────┴───────┴───────┴──────────┘
⚠ api.myapp.com has elevated failure rate (1.8%). Investigate.
```

### Deliverable

`netdiag analyze --window 7d` generates a health report with percentile stats, peak-hour analysis, and anomaly flags. All monitor results are automatically persisted.

---

## 🔵 Phase 5 — Distributed Systems: Agent Mode `v0.7.0`

> **Goal:** Multi-region latency monitoring via gRPC — the feature that separates "side project" from "distributed systems experience."

### New Commands

```
# On a remote server (e.g. DigitalOcean droplet in Frankfurt)
netdiag agent --port 7777 --location "eu-west-1" --auth-token $SECRET

# Locally, aggregate from multiple regions
netdiag monitor \
  --agent agent-us.example.com:7777 \
  --agent agent-eu.example.com:7777 \
  --agent agent-ap.example.com:7777 \
  --target google.com
```

### What's Built

- **Protocol Buffers** — `proto/netdiag.proto` defines `RunProbe`, `StreamProbes`, `GetInfo` RPC methods.
- **gRPC agent server** — Listens for probe requests, executes them locally, streams results back with location metadata.
- **Aggregating monitor** — Fans out each probe to all connected agents, collects responses, correlates by target.
- **Multi-region TUI column** — Dashboard gets a third column showing per-region latency side by side:
  ```
  google.com │ 🇺🇸 us-east 12ms ● │ 🇩🇪 eu-west 98ms ● │ 🇯🇵 ap 180ms ●
  ```

### Deliverable

Three `netdiag agent` instances running in different regions, with `netdiag dashboard` showing geographic latency breakdown in real time.

---

## ⬛ Phase 6 — Portfolio Polish `v1.0.0`

> **Goal:** Make sure the engineering depth is visible before anyone reads the code.

### What's Built

- **`docs/performance.md`** — Full write-up of the SYN scanner: problem statement, methodology, benchmark environment, results table, and technical explanation.
- **`docs/architecture.md`** — Mermaid architecture diagram showing how CLI, monitor daemon, TUI, SQLite, Prometheus, and gRPC agent interact.
- **`deploy/prometheus.yml`** — Ready-to-use Prometheus scrape config.
- **`deploy/grafana-dashboard.json`** — Pre-built Grafana dashboard with latency time-series, packet loss heatmap, HTTP status history, and uptime gauges.
- **`Dockerfile`** — Multi-stage build. `setcap cap_net_raw+ep` so ICMP works without full root.
- **`deploy/docker-compose.yml`** — One `docker compose up` starts netdiag monitor + Prometheus + Grafana.
- **Demo GIF** — 30-second terminal recording (via `vhs`) at the top of the README showing the live dashboard.
- **README overhaul** — Leads with an "Engineering Highlights" table mapping each feature to the skill it demonstrates.

---

## Version Summary

| Version  | Phase   | Key Feature                                              |
| -------- | ------- | -------------------------------------------------------- |
| `v0.1.0` | —       | Initial release, all one-shot commands                   |
| `v0.2.0` | Phase 0 | `pkg/probe/` refactor, JSON output, config file, tests   |
| `v0.3.0` | Phase 1 | `monitor` daemon, Prometheus metrics, alerting           |
| `v0.4.0` | Phase 2 | `dashboard` TUI with sparklines                          |
| `v0.5.0` | Phase 3 | SYN scanner, `--fast` flag, benchmarks                   |
| `v0.6.0` | Phase 4 | SQLite persistence, `analyze` command, anomaly detection |
| `v0.7.0` | Phase 5 | gRPC agent mode, multi-region dashboard                  |
| `v1.0.0` | Phase 6 | Docker, Grafana, demo GIF, full documentation            |

---

**Last Updated:** 2026-02-26
