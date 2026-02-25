# netdiag Implementation Plan

This document is the hands-on, task-by-task implementation guide. Each phase has a checklist of concrete files to create, code to write, and commands to run to verify the work is done. Follow phases in order — each one builds on the last.

For the high-level "why" behind each decision, see [`MASTERPLAN.md`](MASTERPLAN.md).  
For the version roadmap summary, see [`ROADMAP.md`](ROADMAP.md).

---

## How to Use This File

- Work through tasks top to bottom within each phase
- Check off tasks as you complete them
- Each phase ends with a **Verification** section — run those commands before moving on
- Code snippets are the intended final shape, not necessarily copy-paste-ready

---

## Phase 0 — Foundation Hardening

### 0.1 Create `pkg/probe/types.go`

This is the most important file in the entire refactor. Everything downstream depends on it.

- [ ] Create `pkg/probe/` directory
- [ ] Define `Severity` type with constants `SeverityOK`, `SeverityWarning`, `SeverityError`, `SeverityUnknown`
- [ ] Define `Result` struct with fields: `ProbeType`, `Target`, `Timestamp`, `Severity`, `Success`, `Message`, `Latency`, and optional payload pointers
- [ ] Define payload structs: `PingData`, `ScanData`, `TraceData`, `HTTPData`, `DNSData`
- [ ] Define `Prober` interface:
  ```go
  type Prober interface {
      Probe(ctx context.Context) (Result, error)
      Type() string
  }
  ```
- [ ] Add `json` struct tags to every exported field (needed for `--json` output and SQLite storage)
- [ ] Write `Result.IsAnomaly() bool` helper (stub for Phase 4)

### 0.2 Extract probe logic into `pkg/probe/`

For each file below: copy the core logic from `cmd/`, wrap it in a struct that implements `Prober`, keep `cmd/` as a thin wrapper that calls it.

- [ ] Create `pkg/probe/ping.go` — extract from `cmd/ping.go`
  - Struct `PingProber { Host string; Count int; Timeout time.Duration; Interval time.Duration }`
  - `Probe(ctx)` returns `Result` with `PingData` populated
- [ ] Create `pkg/probe/scan.go` — extract from `cmd/scan.go`
  - Move `parsePortRange()` here (it's a pure function, easiest to test)
  - Struct `ConnectScanner { Host string; Ports []int; Timeout time.Duration; Concurrency int }`
- [ ] Create `pkg/probe/trace.go` — extract from `cmd/tracer.go`
  - Struct `TraceProber { Host string; MaxHops int }`
- [ ] Create `pkg/probe/http.go` — extract from `cmd/http.go`
  - Struct `HTTPProber { URL string; Timeout time.Duration; Method string }`
- [ ] Create `pkg/probe/dns.go` — extract from `cmd/dig.go`
  - Struct `DNSProber { Domain string; RecordType string }`
- [ ] Create `pkg/probe/discover.go` — extract from `cmd/discover.go`
  - Struct `DiscoverProber { Timeout time.Duration }`
- [ ] Update all `cmd/*.go` files to call `pkg/probe/` instead of doing work directly
- [ ] Confirm `go build .` still passes after refactor

### 0.3 Create `pkg/logger/logger.go`

- [ ] Create `pkg/logger/` directory
- [ ] Wrap `log/slog` with a `New(level, format, writer)` constructor
- [ ] Add a package-level default logger var
- [ ] Wire `--log-file` flag in `cmd/root.go` to write to file
- [ ] Wire `--log-format json` flag to switch to JSON handler
- [ ] Replace all `output.PrintInfo/PrintError` calls in `cmd/` with logger calls (or keep both — logger for file, color output for terminal)

### 0.4 Create `pkg/config/config.go`

- [ ] Add `github.com/spf13/viper` dependency: `go get github.com/spf13/viper`
- [ ] Create `pkg/config/` directory
- [ ] Define `Config` struct matching this shape:
  ```go
  type Config struct {
      Monitor  MonitorConfig  `mapstructure:"monitor"`
      Database DatabaseConfig `mapstructure:"database"`
      Metrics  MetricsConfig  `mapstructure:"metrics"`
      Scan     ScanConfig     `mapstructure:"scan"`
  }
  ```
- [ ] `Load()` function that reads `~/.netdiag.yaml`, falls back to defaults if missing
- [ ] Create a sample `~/.netdiag.yaml` and add it as `config.example.yaml` in the repo root
- [ ] Wire Viper into `cmd/root.go` `PersistentPreRun` hook so config loads before any command runs

### 0.5 Wire `--json` output mode

- [ ] Add `outputJSON(result probe.Result)` function to `pkg/output/printer.go`
- [ ] In each `cmd/*.go` Run function, check `jsonOutput` flag and branch:
  ```go
  if jsonOutput {
      output.PrintJSON(result)
      return
  }
  output.PrintTable(headers, rows)
  ```
- [ ] Test: `netdiag ping 1.1.1.1 --json | jq '.ping.avg_rtt_ns'`

### 0.6 Write tests

- [ ] Create `pkg/probe/scan_test.go` — table-driven tests for `parsePortRange`:
  - Single port, comma list, range, reversed range, invalid input, out-of-range ports
- [ ] Create `pkg/probe/ping_test.go` — test `Result.Severity` logic based on packet loss and latency
- [ ] Create `pkg/probe/http_test.go` — test TLS days remaining calculation
- [ ] Run `go test ./...` — all pass

### Phase 0 Verification

```bash
go build .                                        # builds clean
go test ./...                                     # all tests pass
./netdiag ping 1.1.1.1 --json | jq .             # valid JSON output
./netdiag scan localhost -p 80,443 --json | jq . # valid JSON output
./netdiag --log-file /tmp/netdiag.log ping google.com
cat /tmp/netdiag.log                              # log file written
```

---

## Phase 1 — Monitor Daemon & Prometheus

### 1.1 Create `pkg/monitor/monitor.go`

- [ ] Create `pkg/monitor/` directory
- [ ] Define `Monitor` struct:
  ```go
  type Monitor struct {
      probers  []probe.Prober
      interval time.Duration
      events   chan probe.Result
      logger   *slog.Logger
      // Phase 4: store store.Store
      // Phase 1.2: metrics *metrics.Registry
  }
  ```
- [ ] `New(probers []probe.Prober, interval time.Duration, logger *slog.Logger) *Monitor`
- [ ] `Run(ctx context.Context) error` — ticker loop, calls `runAllProbes` on each tick and immediately on start
- [ ] `runAllProbes(ctx context.Context)` — errgroup with `SetLimit(20)`, sends results to `events` channel
- [ ] `Events() <-chan probe.Result` — exposes read-only channel for consumers
- [ ] Write `pkg/monitor/monitor_test.go` — mock probers, verify events are emitted

### 1.2 Create `pkg/alert/`

- [ ] Create `pkg/alert/alerter.go` — define `Alerter` interface and `AlertRule` struct
- [ ] Create `pkg/alert/console.go` — prints colored alert to stderr
- [ ] Create `pkg/alert/slack.go` — HTTP POST to Slack webhook URL
- [ ] Create `pkg/alert/webhook.go` — generic HTTP POST with JSON body
- [ ] Create `pkg/alert/cooldown.go` — `CooldownManager` with mutex-protected `map[string]time.Time`
- [ ] Add `--webhook` and `--alert-threshold` flags to monitor command

### 1.3 Create `pkg/metrics/`

- [ ] Add Prometheus dependency: `go get github.com/prometheus/client_golang`
- [ ] Create `pkg/metrics/registry.go` — define all `GaugeVec` and `CounterVec` metrics:
  - `netdiag_ping_latency_ms{target, ip}`
  - `netdiag_ping_packet_loss_percent{target}`
  - `netdiag_http_status_code{target}`
  - `netdiag_http_latency_ms{target}`
  - `netdiag_http_tls_days_remaining{target}`
  - `netdiag_probe_total{target, probe_type, result}` (Counter)
  - `netdiag_last_success_timestamp{target, probe_type}` (Gauge — unix timestamp)
- [ ] `Update(result probe.Result)` method that sets the right metrics based on result type
- [ ] Create `pkg/metrics/server.go` — HTTP server with `/metrics` and `/health` endpoints, shuts down via context
- [ ] Wire into Monitor: start metrics server in `Run()` if enabled, call `metrics.Update(result)` for each result

### 1.4 Create `pkg/eventlog/writer.go`

- [ ] Create `pkg/eventlog/` directory
- [ ] `Writer` struct that holds an `*os.File` rotated by date
- [ ] `Write(result probe.Result) error` — marshals result to JSON, appends newline
- [ ] Auto-creates `~/.netdiag/logs/` directory if it doesn't exist
- [ ] Rotates to new file at midnight (check date on each write)
- [ ] Wire into Monitor: start log writer, write each result from `events` channel

### 1.5 Create `cmd/monitor.go`

- [ ] Define `monitorCmd` with Cobra
- [ ] Flags: `--target` (string slice), `--interval` (duration, default 30s), `--webhook` (string), `--alert-threshold` (duration), `--metrics-port` (int, default 9090), `--no-metrics` (bool)
- [ ] Load config, override with flags
- [ ] Build `[]probe.Prober` from target list (auto-detect probe type: URL → HTTP, hostname/IP → ping)
- [ ] Create monitor, start metrics server in goroutine, call `monitor.Run(ctx)`
- [ ] Use `signal.NotifyContext` for shutdown

### Phase 1 Verification

```bash
./netdiag monitor --target google.com --target 1.1.1.1 --interval 10s &
sleep 15
curl -s http://localhost:9090/metrics | grep netdiag_ping
curl -s http://localhost:9090/health
cat ~/.netdiag/logs/*.jsonl | jq .
kill %1   # Ctrl+C — should print "monitor shutting down gracefully"
```

---

## Phase 2 — TUI Dashboard

### 2.1 Add Bubbletea dependencies

```bash
go get github.com/charmbracelet/bubbletea
go get github.com/charmbracelet/lipgloss
go get github.com/charmbracelet/bubbles
```

### 2.2 Create `pkg/tui/messages.go`

- [ ] Define all `tea.Msg` types:
  - `ProbeResultMsg { Result probe.Result }`
  - `TickMsg time.Time`
  - `ErrorMsg { Err error }`

### 2.3 Create `pkg/tui/model.go`

- [ ] Define `TargetState` struct with: `Host`, `ProbeType`, `Status`, `LastLatency`, `LastChecked`, `LatencyHistory [60]float64`, `historyIdx int`
- [ ] Define `Model` struct with: `targets`, `eventLog []string`, `logViewport`, `probers`, `interval`, `selectedTarget`, `paused`, `width`, `height`
- [ ] `InitialModel(probers []probe.Prober, interval time.Duration) Model`
- [ ] `Init() tea.Cmd` — kick off first probe batch and start tick timer

### 2.4 Create `pkg/tui/update.go`

- [ ] `Update(msg tea.Msg) (tea.Model, tea.Cmd)` — handle all message types:
  - `ProbeResultMsg` → update matching `TargetState`, append to event log
  - `TickMsg` → re-schedule all probes via `tea.Batch`
  - `tea.KeyMsg` → handle `q`, `p`, `r`, `↑`, `↓`, `enter`
  - `tea.WindowSizeMsg` → update `width`, `height`, resize panes

### 2.5 Create `pkg/tui/sparkline/sparkline.go`

- [ ] `Render(data []float64, width int, maxVal float64) string` using `▁▂▃▄▅▆▇█`
- [ ] `RenderWithScale(data []float64, width int) (string, float64)` — auto-scales to data max
- [ ] Write unit test: known input → expected Unicode string output

### 2.6 Create `pkg/tui/view.go`

- [ ] `View() string` — compose the full terminal layout using lipgloss:
  - Left pane: host table with colored status indicators
  - Right pane: sparkline graph + stats for selected host
  - Bottom pane: scrollable event log via `bubbles/viewport`
  - Top bar: title + keybinding hints
- [ ] Create `pkg/tui/styles.go` — all lipgloss style definitions (colors, borders, padding)
- [ ] Handle both "list view" and "detail view" (toggled by Enter)

### 2.7 Create `cmd/dashboard.go`

- [ ] Define `dashboardCmd` with same flags as `monitor`
- [ ] Build probers, create model, run `tea.NewProgram(model, tea.WithAltScreen()).Run()`

### Phase 2 Verification

```bash
./netdiag dashboard --target google.com --target 1.1.1.1 --target github.com
# Should open full-screen TUI
# Press ↓ to select second host — sparkline changes
# Press Enter — detail view opens
# Press Esc — back to list
# Press p — pauses probing
# Press q — exits cleanly, terminal restored
# Resize terminal — layout adjusts
```

---

## Phase 3 — SYN Scanner

### 3.1 Add gopacket dependency

```bash
go get github.com/google/gopacket
# Note: requires libpcap-dev on Linux, WinPcap on Windows
# Document this in README
```

### 3.2 Create `pkg/probe/syn_scanner.go`

- [ ] Define `SYNScanner` struct: `{ Host string; Ports []int; Timeout time.Duration; Concurrency int; SourceIP net.IP }`
- [ ] `getSourceIP() (net.IP, error)` — detect local IP via `net.Dial("udp", "8.8.8.8:80")`
- [ ] `craftSYNPacket(dstIP net.IP, srcPort, dstPort int) ([]byte, error)`:
  - Build `layers.IPv4` with TTL=64, Protocol=TCP
  - Build `layers.TCP` with `SYN: true`, random `Seq`, random `SrcPort`
  - Call `tcpLayer.SetNetworkLayerForChecksum(ipLayer)` — this is the step most people miss
  - Serialize with `ComputeChecksums: true`
- [ ] `waitForResponse(conn net.PacketConn, srcPort int, deadline time.Time) (bool, error)`:
  - Read packets until deadline
  - Parse with `icmp.ParseMessage` — look for SYN-ACK (`tcp.SYN && tcp.ACK`) or RST
  - Match on `srcPort` to avoid false positives from other traffic
- [ ] `Scan() ([]int, error)` — semaphore worker pool, returns open ports
- [ ] `Probe(ctx) (probe.Result, error)` — implements `Prober` interface
- [ ] **Fallback logic**: if `net.ListenPacket("ip4:tcp", ...)` fails with permission error, log warning and fall back to connect scan automatically

### 3.3 Add `--fast` and `--benchmark` flags to `cmd/scan.go`

- [ ] `--fast` flag — uses `SYNScanner` instead of `ConnectScanner`
- [ ] `--benchmark` flag — runs both, prints comparison table:
  ```
  ┌──────────────┬────────┬────────────┬─────────┐
  │ Method       │ Time   │ Ports/sec  │ Speedup │
  ├──────────────┼────────┼────────────┼─────────┤
  │ Connect scan │ 41.2s  │ 1,590      │ 1x      │
  │ SYN scan     │ 0.89s  │ 73,600     │ 46.3x   │
  └──────────────┴────────┴────────────┴─────────┘
  Open ports: [22, 80, 443] (results identical ✓)
  ```

### 3.4 Write `docs/performance.md`

- [ ] Section: "The Problem with net.Dial"
- [ ] Section: "How SYN Scanning Works" (diagram of packet flow)
- [ ] Section: "Implementation Challenges" (checksum calculation, matching responses, false positives)
- [ ] Section: "Benchmark Methodology" (hardware, Go version, command used)
- [ ] Section: "Results" (table)
- [ ] Section: "Limitations" (requires privileges, stateful firewalls may block)

### Phase 3 Verification

```bash
sudo ./netdiag scan localhost -p 1-1024 --fast
sudo ./netdiag scan localhost -p 1-1024 --benchmark
# Verify: open ports match between both methods
# Verify: SYN scan is measurably faster
# Without sudo:
./netdiag scan localhost -p 1-1024 --fast
# Should print warning and fall back to connect scan automatically
```

---

## Phase 4 — SQLite Persistence & Analytics

### 4.1 Add SQLite dependency

```bash
go get modernc.org/sqlite   # pure Go, no CGO required
```

### 4.2 Create `pkg/store/`

- [ ] Create `pkg/store/store.go` — define `Store` interface:
  ```go
  type Store interface {
      SaveResult(ctx context.Context, r probe.Result) error
      GetHistory(ctx context.Context, target string, since time.Time) ([]probe.Result, error)
      GetStats(ctx context.Context, target string, window time.Duration) (*Stats, error)
      GetUptimePercent(ctx context.Context, target string, window time.Duration) (float64, error)
      GetWorstHosts(ctx context.Context, limit int, window time.Duration) ([]HostSummary, error)
      GetPeakLatencyHours(ctx context.Context, target string) ([]HourlyAvg, error)
      Compact(ctx context.Context, olderThan time.Duration) (int64, error)
      Close() error
  }
  ```
- [ ] Create `pkg/store/schema.go` — all `CREATE TABLE` and `CREATE INDEX` statements
- [ ] Create `pkg/store/sqlite.go` — `SQLiteStore` implementing `Store`
  - `Open(path string) (*SQLiteStore, error)` — creates dirs, runs schema migration
  - Use prepared statements for all queries (store them in struct fields)
  - Percentile query via `NTILE(100)` window function
  - Stddev query via `SQRT(AVG(x*x) - AVG(x)*AVG(x))`
- [ ] Create `pkg/store/sqlite_test.go` — integration tests using in-memory DB (`:memory:`)
  - Insert 100 results, verify percentiles, verify uptime calculation
- [ ] Wire store into monitor: `monitor.SetStore(store)` — saves every result after probing

### 4.3 Create `pkg/analyze/`

- [ ] Create `pkg/analyze/stats.go` — `Stats` struct and helper functions
- [ ] Create `pkg/analyze/anomaly.go` — `AnomalyDetector` with Z-score logic:
  ```go
  func (a *AnomalyDetector) IsAnomaly(ctx context.Context, target string, current time.Duration) (bool, string, error)
  ```
- [ ] Wire anomaly detector into monitor's event consumer — tag results with `Severity: SeverityWarning` if anomaly detected
- [ ] Write tests for anomaly detection with synthetic data

### 4.4 Create `cmd/analyze.go`

- [ ] Define `analyzeCmd` with Cobra
- [ ] Flags: `--target`, `--window` (duration, default 24h), `--worst` (int), `--peak-hours` (bool), `--format` (table/csv/json)
- [ ] Open store from config path
- [ ] `--target` specified: show detailed single-host report
- [ ] No `--target`: show summary table of all hosts
- [ ] `--worst N`: rank by failure rate descending
- [ ] `--peak-hours`: show hourly average latency breakdown
- [ ] `--format csv`: output raw CSV rows
- [ ] Add `netdiag db compact` subcommand to manually run retention cleanup

### Phase 4 Verification

```bash
# Run monitor for a few minutes first to populate DB
./netdiag monitor --target google.com --target 1.1.1.1 --interval 5s &
sleep 60
kill %1

./netdiag analyze
./netdiag analyze --target google.com --window 1h
./netdiag analyze --worst 5
./netdiag analyze --target google.com --peak-hours
./netdiag analyze --target google.com --format csv
ls ~/.netdiag/data.db   # file exists
```

---

## Phase 5 — gRPC Agent Mode

### 5.1 Set up protobuf toolchain

```bash
# Install protoc and Go plugins
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
go get google.golang.org/grpc
go get google.golang.org/protobuf
```

- [ ] Add to `Makefile`:
  ```makefile
  proto:
    protoc --go_out=. --go-grpc_out=. proto/netdiag.proto
  ```

### 5.2 Write `proto/netdiag.proto`

- [ ] Define `NetdiagAgent` service with `RunProbe`, `StreamProbes`, `GetInfo` RPCs
- [ ] Define `ProbeRequest`, `ProbeResponse`, `StreamRequest`, `AgentInfo` messages
- [ ] `ProbeResponse` must include `agent_id`, `agent_location`, `timestamp_ns`
- [ ] Run `make proto` — generates `internal/pb/netdiag.pb.go` and `netdiag_grpc.pb.go`

### 5.3 Create agent server

- [ ] Create `pkg/agent/server.go` — implements `NetdiagAgentServer` gRPC interface
- [ ] `RunProbe` handler: parse request, build prober, call `Probe(ctx)`, return response
- [ ] `StreamProbes` handler: accept stream request, run probes on interval, stream results back
- [ ] `GetInfo` handler: return agent metadata (location from flag, capabilities, version)
- [ ] Token-based auth via gRPC interceptor: check `Authorization: Bearer <token>` metadata

### 5.4 Create agent client

- [ ] Create `pkg/agent/client.go` — wraps gRPC stub, adds retry with exponential backoff
- [ ] `Connect(addr, token string) (*Client, error)`
- [ ] `RunProbe(ctx, request) (probe.Result, error)`
- [ ] `StreamProbes(ctx, request) (<-chan probe.Result, error)`

### 5.5 Create `cmd/agent.go`

- [ ] Flags: `--port`, `--location`, `--auth-token`
- [ ] Start gRPC server, handle shutdown via context

### 5.6 Wire agents into monitor

- [ ] Add `--agent` (string slice) flag to `monitorCmd`
- [ ] In `Monitor.runAllProbes`: for each target, fan out to all connected agents + local probe
- [ ] Aggregate results, tag with `agent_location`
- [ ] Update TUI to show per-agent column when agents are connected

### Phase 5 Verification

```bash
# Terminal 1
./netdiag agent --port 7777 --location "local-1" --auth-token secret

# Terminal 2
./netdiag agent --port 7778 --location "local-2" --auth-token secret

# Terminal 3
./netdiag dashboard \
  --agent localhost:7777 \
  --agent localhost:7778 \
  --target google.com
# Dashboard should show two location columns
```

---

## Phase 6 — Portfolio Polish

### 6.1 Documentation

- [ ] Write `docs/performance.md` (if not already done in Phase 3)
- [ ] Write `docs/architecture.md` with Mermaid diagram:

  ````markdown
  ```mermaid
  graph TB
      CLI --> Core
      Monitor --> Core
      Dashboard --> Core
      Agent -- gRPC --> Monitor
      Core --> SQLite
      Monitor --> Prometheus
  ```
  ````

  ```

  ```

- [ ] Rewrite top section of `README.md`:
  - Add demo GIF immediately after title
  - Add "Engineering Highlights" table
  - Update installation section to include Docker
  - Update commands reference with new commands

### 6.2 DevOps files

- [ ] Create `Dockerfile` — multi-stage build, `setcap cap_net_raw+ep`
- [ ] Create `deploy/prometheus.yml`
- [ ] Create `deploy/grafana-dashboard.json` — panels for latency, packet loss, uptime, HTTP status
- [ ] Create `deploy/docker-compose.yml` — netdiag + Prometheus + Grafana
- [ ] Test: `docker compose up` → `http://localhost:3000` shows Grafana dashboard with data

### 6.3 CI enhancements

- [ ] Add benchmark job to `.github/workflows/ci.yml`:
  ```yaml
  benchmark:
    runs-on: ubuntu-latest
    steps:
      - run: go test -bench=. -benchmem ./pkg/probe/... | tee benchmark.txt
      - uses: actions/upload-artifact@v4
        with:
          name: benchmarks
          path: benchmark.txt
  ```
- [ ] Add `go test -race ./...` to CI (already present but verify it's running)
- [ ] Add coverage badge to README

### 6.4 Demo recording

- [ ] Install `vhs`: `go install github.com/charmbracelet/vhs@latest`
- [ ] Write `demo.tape` VHS script showing dashboard in action
- [ ] Run `vhs demo.tape` — generates `docs/demo.gif`
- [ ] Add `![Dashboard Demo](docs/demo.gif)` to top of README

### Phase 6 Verification

```bash
docker compose -f deploy/docker-compose.yml up -d
sleep 10
curl http://localhost:9090/metrics | grep netdiag  # Prometheus scraping
open http://localhost:3000                          # Grafana dashboard loaded
docker compose -f deploy/docker-compose.yml down
```

---

## Dependency Installation Reference

Run these in order as you reach each phase:

```bash
# Phase 0
go get github.com/spf13/viper

# Phase 1
go get github.com/prometheus/client_golang/prometheus
go get github.com/prometheus/client_golang/prometheus/promhttp

# Phase 2
go get github.com/charmbracelet/bubbletea
go get github.com/charmbracelet/lipgloss
go get github.com/charmbracelet/bubbles

# Phase 3
go get github.com/google/gopacket
# Also install system lib:
# Linux:  sudo apt-get install libpcap-dev
# macOS:  brew install libpcap

# Phase 4
go get modernc.org/sqlite

# Phase 5
go get google.golang.org/grpc
go get google.golang.org/protobuf
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# After each phase
go mod tidy
```

---

## Quick Reference: New Files Per Phase

| Phase | New Files                                                                                                                                                          |
| ----- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| 0     | `pkg/probe/types.go`, `pkg/probe/*.go`, `pkg/logger/logger.go`, `pkg/config/config.go`, `config.example.yaml`, `pkg/probe/*_test.go`                               |
| 1     | `pkg/monitor/monitor.go`, `pkg/alert/*.go`, `pkg/metrics/registry.go`, `pkg/metrics/server.go`, `pkg/eventlog/writer.go`, `cmd/monitor.go`                         |
| 2     | `pkg/tui/model.go`, `pkg/tui/update.go`, `pkg/tui/view.go`, `pkg/tui/messages.go`, `pkg/tui/styles.go`, `pkg/tui/sparkline/sparkline.go`, `cmd/dashboard.go`       |
| 3     | `pkg/probe/syn_scanner.go`, `docs/performance.md`                                                                                                                  |
| 4     | `pkg/store/store.go`, `pkg/store/sqlite.go`, `pkg/store/schema.go`, `pkg/store/sqlite_test.go`, `pkg/analyze/stats.go`, `pkg/analyze/anomaly.go`, `cmd/analyze.go` |
| 5     | `proto/netdiag.proto`, `internal/pb/*.go` (generated), `pkg/agent/server.go`, `pkg/agent/client.go`, `cmd/agent.go`                                                |
| 6     | `Dockerfile`, `deploy/prometheus.yml`, `deploy/grafana-dashboard.json`, `deploy/docker-compose.yml`, `demo.tape`, `docs/architecture.md`                           |
