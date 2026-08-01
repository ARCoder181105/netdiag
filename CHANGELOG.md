# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.3.0] - 2026-08-01

Correctness, consistency and production-hardening release. No new commands.

### Fixed

- **`ping` and `discover` no longer require root.** Both hardcoded
  `SetPrivileged(true)`, so they failed for anyone without `CAP_NET_RAW` —
  which is the default state of a `go install`ed binary on Linux.
  `pkg/probe/icmp.go` now picks the socket type most likely to work on the
  platform, retries with the other on a permission error, and caches the
  result. When neither works, the error explains how to fix it instead of
  reporting a bare `socket: permission denied`.
- **`trace` no longer attributes other processes' ICMP packets to its own
  hops.** The echo ID was hardcoded to `1234` and any inbound ICMP packet was
  accepted as the current hop's reply. Probes are now identified by PID, and
  replies are matched on ID and sequence — including the quoted original header
  inside `TimeExceeded` messages.
- **`trace` reports a real severity.** It previously always returned
  `SeverityOK`, even when no hop responded at all.
- **`discover` handles non-/24 networks.** The local network was derived by
  slicing the IP string at the last dot and sweeping `1..254`, ignoring the
  actual netmask. It now enumerates hosts from the interface's `*net.IPNet`,
  capped at 1024 addresses.
- **`scan` reports a real throughput.** `ScanRateMs` used integer division of
  ports by elapsed milliseconds, so any scan slower than one port per
  millisecond reported `0`. Replaced by `ports_per_sec` (float).
- **`scan` no longer reports `SeverityOK` when it finds nothing**, matching how
  `dig` and `discover` already treated empty results.
- **`whois` and `speedtest` respect cancellation.** Both ignored the context
  entirely, so Ctrl+C did nothing. `speedtest` now uses the library's
  `*Context` methods; `whois` runs against the context with a real timeout.
- **`speedtest` shows the ISP name.** It rendered `user.String()`, dumping the
  whole struct into the ISP column.
- **HTTP response bodies are drained before closing**, and an expired
  certificate is now an error rather than a success with `tls_valid: false`.
- **Failed probes report their latency.** Most soft-failure paths left
  `Latency` at zero in JSON output.
- **`--json` is honored on every error path.** `trace` and `whois` printed
  plain text when a probe failed hard.
- **Progress banners print before the work, not after.** `discover` and `dig`
  announced what they were about to do once it had already finished.

### Added

- **Exit codes**: `0` healthy, `1` unhealthy or unreachable, `2` usage error,
  `3` probe could not run. Every command previously exited `0` unconditionally,
  which made netdiag unusable in scripts and CI.
- **Signal handling**: `Ctrl+C`/`SIGTERM` cancels an in-flight probe. All
  commands previously used `context.Background()`.
- **Input validation at the CLI boundary.** `ParsePortRange` now returns an
  error naming the offending token instead of silently skipping it — `--ports
  "abc,80"` used to scan one port and report success. URLs are validated and
  their scheme checked; DNS record types, host arguments, `--max-hops` and
  `--concurrency` are validated before a prober is built.
- `--log-level` flag. `pkg/logger` accepted a level parameter that `Init`
  always hardcoded to `info`.
- `dig AAAA` for IPv6 address lookups.
- `whois --timeout`.
- `SECURITY.md` with a vulnerability reporting process and scope.
- `.github/dependabot.yml` for Go modules and GitHub Actions.
- `make test-cover` and `make tidy`, both already referenced by the developer
  guide but absent from the Makefile.
- `bodyclose`, `errorlint` and `noctx` linters — each catches a class of bug
  found in this release.

### Changed

- **All 8 commands now share one execution path** (`cmd/run.go`). The same
  ~40-line block — build prober, run, synthesize an error result, log, handle
  `--json`, switch on severity — had been copy-pasted per command and had
  drifted. Commands are now flag parsing, validation, and table rendering only.
- `probe.ErrorResult` and `output.PrintBySeverity` replace six copies each of
  the same inline logic.
- Builds use `-trimpath` and `CGO_ENABLED=0`; `make fmt` pins its tool
  versions; the release workflow uses a single timestamp across all artifacts.
- CI lint no longer runs with `--fix`, which discarded its own fixes and let
  the job pass on issues a plain run would fail.
- Config now contains only keys a command actually reads. The `monitor`,
  `database` and `metrics` sections were inert and are deferred to the phases
  that will consume them.
- Logs are confirmed to go to stderr only, so `--json | jq` works with logging
  enabled.

### Removed

- `Result.IsAnomaly()` — an unused Phase 4 stub with no callers.

### Documentation

- README: corrected the Go version (1.24, not 1.25), documented six flags that
  existed but were undocumented (`scan -c`, `http --skip-tls`, `trace -t`,
  `dig -s/-t`, and the global logging flags), fixed the ping `--timeout` and
  `--interval` types (durations, not ints), and added Global Flags, Exit Codes,
  Responsible Use, and Configuration sections.
- README: removed the Homebrew install instructions. The formula is pinned to
  v0.1.0 with placeholder checksums and cannot work.
- README: rewrote "Architecture & Concepts", which still described the
  pre-`pkg/probe` layout with inline code samples.
- README and CONTRIBUTING: dropped `--json`, config file support and custom DNS
  servers from the "wanted features" lists — all three shipped in v0.2.0.
- ROADMAP/MASTERPLAN/Plan: the SYN scanner benchmark tables were presented as
  measured results for code that does not exist, in three mutually
  inconsistent versions. They are now clearly marked as targets.
- CONTRIBUTING: fixed find/replace damage that broke a copy-pasteable git
  command, refreshed the project structure, and added a commit authorship
  policy.

## [0.2.1] - 2026-03-07

### Fixed

- **`pkg/probe/ping.go`** — Ping now correctly emits `SeverityWarning` for
  degraded-but-alive hosts (partial packet loss > 0% or avg latency > 150ms).
  Previously only `SeverityOK` or `SeverityError` were ever set, meaning
  `ping_test.go`'s `SeverityWarning` assertions were testing mock logic rather
  than the real probe. The success message now also includes avg RTT and loss %.

- **`cmd/speedtest.go`** — Hard errors returned from `prober.Probe()` now
  respect the `--json` flag instead of always printing to stderr as plain text.
  Structured logging added for both success and failure paths.

- **`cmd/scan.go`** — Structured logging added via `logger.Log` for scan
  completion and failure. All other `cmd/` files were already inconsistent.

- **`cmd/http.go`** — Structured logging added (status code, latency, TLS
  validity, TLS days remaining on success; error message on failure).

- **`cmd/tracer.go`** — Structured logging added (hop count, latency on
  success; error on failure).

- **`cmd/dig.go`** — Structured logging added (record type, record count on
  success; error on failure).

- **`cmd/whois.go`** — Structured logging added (latency on success; error
  on failure).

- **`cmd/discover.go`** — Structured logging added (subnet prefix, device
  count, latency on success; error on failure).

### Changed

- All `cmd/` files now consistently use `logger.Log` for structured output
  alongside the existing `output.Print*` color functions. The `--log-file`
  and `--log-format` flags now capture output from every command, not just
  `ping`.

## [0.2.0] - 2026-02-26

### Added

- `pkg/probe/` package — all business logic extracted from `cmd/` into a
  reusable package with a `Prober` interface and a universal `Result` type
- Typed result system (`PingData`, `ScanData`, `HTTPData`, `TraceData`,
  `DNSData`, `DiscoverData`, `SpeedTestData`, `WhoisData`) with JSON tags
- `--json` flag wired across all commands for machine-readable output
- Structured logging via `log/slog` (`pkg/logger/`) with `--log-file` and
  `--log-format` flags on the root command
- Config file support (`~/.netdiag.yaml`) via `pkg/config/` (Viper)
- Test suite: `ParsePortRange`, ping severity, TLS days remaining, `Severity.String()`

## [0.1.0] - 2026-01-14

### Added
- Initial project setup
- Core CLI structure using Cobra
- Basic command framework

---

## Release Process

1. Update this CHANGELOG.md with all changes since last release
2. Commit changes: `git commit -am "chore: release vX.Y.Z"`
3. Create and push tag: `git tag vX.Y.Z && git push origin vX.Y.Z`
4. GitHub Actions will automatically build and publish the release

The version is injected from the git tag at build time via `-ldflags`; there is
no version string to edit in source.
