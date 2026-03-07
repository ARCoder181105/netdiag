# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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
  alongside the existing `output.Print*` colour functions. The `--log-file`
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
2. Update version in main.go
3. Commit changes: `git commit -am "Release vX.Y.Z"`
4. Create and push tag: `git tag vX.Y.Z && git push origin vX.Y.Z`
5. GitHub Actions will automatically build and publish the release
