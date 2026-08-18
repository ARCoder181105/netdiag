# Roadmap

netdiag is a finished tool, not a work in progress. This document records what
was built, what was deliberately **cut**, and why.

The original plan had six phases and would have turned a diagnostics CLI into a
monitoring platform: a daemon, Prometheus metrics, a TUI dashboard, SQLite
persistence, and a gRPC agent mode. Three of those phases shipped. Four were
cut, on purpose, because a smaller finished tool is worth more than a large
unfinished one — and because most of what was planned already exists, done
better, in Prometheus and Grafana.

| Phase | Status |
| ----- | ------ |
| Phase 0 — Foundation hardening | ✅ **Shipped** `v0.2.0` |
| Phase 1 — Monitor daemon, metrics, alerting | ❌ **Cut** |
| Phase 2 — TUI dashboard | ❌ **Cut** |
| Phase 3 — Raw socket SYN scanner | ✅ **Shipped** |
| Phase 4 — SQLite persistence and `analyze` | ❌ **Cut** |
| Phase 5 — gRPC agent mode | ❌ **Cut** |
| Phase 6 — Portfolio polish | ✅ **Shipped** |

---

## ✅ Phase 0 — Foundation hardening — SHIPPED `v0.2.0`

The architectural base: a `pkg/probe/` package with a `Prober` interface and a
universal `Result` type, so probes contain network logic and nothing else.

- **`pkg/probe/`** — all business logic moved out of `cmd/`, behind one
  interface.
- **Typed results** — every probe returns a `Result` carrying an outcome and one
  probe-specific payload, instead of printing.
- **JSON output** — `--json` wired up for every command, from the same value the
  table renderer receives.
- **Structured logging** — `log/slog`, to stderr by default so stdout stays a
  clean pipe.
- **Config file** — `~/.netdiag.yaml` via Viper, with `NETDIAG_` environment
  overrides.
- **Tests** — port range parsing, ping severity, result marshaling.

`v0.3.0` followed as a correctness and hardening release on the same base: exit
codes, signal handling, timeout bounds, and tests that bind to production
functions rather than reimplementations.

---

## ✅ Phase 3 — Raw socket SYN scanner — SHIPPED

> **Goal:** solve a hard technical problem with a measurable result.

### The problem

`net.DialTimeout` completes a full three-way handshake per port — wasteful, and
it leaves completed connections in the target's logs. A SYN scan sends only the
initial SYN and reads the reply: SYN-ACK means open, RST means closed, silence
means filtered. The handshake is never completed.

```bash
netdiag scan 192.168.1.1 -p 1-65535 --fast       # SYN scan
netdiag scan 127.0.0.1  -p 1-65535 --benchmark   # compare both methods
```

### What shipped

- **`pkg/probe/syn_scanner.go`** — raw TCP SYN crafting. `gopacket` lays out and
  decodes the header; the TCP checksum over the IPv4 pseudo-header is netdiag's
  own code, because handing that to a library helper is the part worth being
  able to explain. Needs `cap_net_raw`.
- **Correlation, not counting** — a raw socket receives every TCP segment on the
  machine, including this process's own SYNs. A reply counts only if it arrives
  on the scan's source port and acknowledges the sequence number sent to that
  port.
- **Adaptive concurrency** — additive-increase/multiplicative-decrease over a
  fixed window, backing off only on windows containing both replies and
  timeouts. Total silence means a filtered range, not congestion.
- **Fallback** — no capability, or no route-derived source address, and the scan
  degrades to the connect scanner with one notice on stderr.
- **Benchmark mode** — `--benchmark` runs both methods against the same target
  and reports if the SYN half fell back, so an unprivileged benchmark cannot be
  mistaken for a real comparison.

### Measured results

Full methodology, environment and caveats:
[`docs/performance.md`](docs/performance.md). Measured on bare metal — a 12th Gen
Intel Core i7-12650H running Linux 7.0.0-28-generic, with `setcap cap_net_raw+ep`
on the binary rather than root. Loopback rows are the median of 5 runs; the
filtered rows the median of 3.

| Target | Method | Time | Ports/sec | Speedup |
| ------ | ------ | ---- | --------- | ------- |
| 65,535 closed ports on loopback, `-c 100` | connect | 264 ms | 248,496 | 1.0x |
| 65,535 closed ports on loopback, `-c 100` | syn | 151 ms | 433,686 | **1.75x** |
| 65,535 closed ports on loopback, `-c 2000` | connect | 385 ms | 170,308 | 1.0x |
| 65,535 closed ports on loopback, `-c 2000` | syn | 128 ms | 510,140 | **3.0x** |
| 1,024 filtered ports, `-t 1s -c 100` | connect | 11.009 s | 93 | 1.0x |
| 1,024 filtered ports, `-t 1s -c 100` | syn | 11.008 s | 93 | **1.0x** |

1.75x, not the order of magnitude usually quoted for SYN scanners — because on
loopback there is no timeout to avoid and no round trip to overlap, which is
exactly what half-open scanning exists to exploit. Against a silent host both
methods are bound by ports ÷ concurrency × timeout, so they tie. Both methods
agreed on which ports were open in every loopback run.

The first working version was *slower* than the connect scan (0.75x). The fix
was not the packet library — building and checksumming all 65,535 packets costs
1.7 ms, half a percent of the scan — but sending from eight raw sockets instead
of one, since the kernel serializes writes per socket.

There is also a measured accuracy advantage under file descriptor pressure.
Scanning 200 open ports with `ulimit -n 32` and `-c 500`, five runs:

| Run | 1 | 2 | 3 | 4 | 5 |
| --- | - | - | - | - | - |
| connect — open ports found | 200 | 184 | 166 | 154 | 200 |
| syn — open ports found | 200 | 200 | 200 | 200 | 200 |

The connect scan needs a descriptor per port and reports an `EMFILE` failure as
a closed port, so it silently under-reports.

The WAN case usually cited for SYN scanning — a remote host that drops packets
to closed ports, making the connect scan pay a full timeout per port — **was not
measured**, because this environment has no authorized remote target. It is
untested, not proven.

---

## ✅ Phase 6 — Portfolio polish — SHIPPED

- **[`docs/performance.md`](docs/performance.md)** — the SYN scanner benchmark:
  problem, method, environment, real numbers, and the caveats that qualify them.
- **[`docs/architecture.md`](docs/architecture.md)** — Mermaid diagram of the
  layering, plus the reasoning behind the `Prober`/`Result` design and the
  exit-code contract.
- **`Dockerfile`** — multi-stage, ~28 MB, unprivileged user, `cap_net_raw` on
  the binary so ICMP and SYN scanning work without running as root.
- **README** — rewritten around an Engineering Highlights section.
- **Demo recording** — `docs/demo.tape` for [vhs](https://github.com/charmbracelet/vhs).

Deliberately **not** built, because they belong to the cut phases: Grafana
dashboards, `docker-compose.yml`, and a Prometheus scrape config.

---

# Cut phases

These were planned and are not being built. They are recorded here because a
decision not to build something is worth more to a reader than a list of
intentions.

## ❌ Phase 1 — Monitor daemon, Prometheus metrics, alerting — CUT

A `netdiag monitor` daemon with a ticker loop, an embedded Prometheus metrics
endpoint, Slack/webhook alerting with cooldowns, and a JSONL event log.

**Why cut.** This is a worse version of software that already exists. Anyone who
wants netdiag's measurements scraped can wrap the existing `--json` output in
four lines of shell and a `node_exporter` textfile collector; anyone who wants
real alerting wants Alertmanager, not a webhook poster with a cooldown map. The
one genuinely novel piece — the probes — already exists and is already
scriptable.

## ❌ Phase 2 — TUI dashboard — CUT

A full-screen `bubbletea` dashboard with sparklines, a split-pane layout, a
scrollable event log, and keyboard navigation.

**Why cut.** The most fun thing on the list and the least useful. It would have
been the largest single body of code in the project, in service of watching
numbers that Grafana already draws better, and every future probe would have
owed it a rendering path. The `Result` type makes it straightforward for anyone
who wants it.

## ❌ Phase 4 — SQLite persistence and `analyze` — CUT

Embedded SQLite storage for every probe result, a `pkg/store/` interface, an
`analyze` command with percentiles, peak-hour analysis, and z-score anomaly
detection.

**Why cut.** It depends entirely on Phase 1: without a daemon writing results
continuously, there is nothing to analyze. Cutting the daemon cut the data
source, and a time-series schema with no time series is just a schema.

## ❌ Phase 5 — gRPC agent mode — CUT

`netdiag agent` on remote hosts, protobuf definitions, and an aggregating
monitor fanning probes out to multiple regions.

**Why cut.** The distributed-systems credential is real, but so is the cost:
protobuf toolchain, auth, TLS between agents, version skew between agent and
aggregator, and a deployment story — all to run probes that already run fine
over SSH. It also depended on Phase 1 for the aggregating side.

---

## Version history

| Version | Phase | What landed |
| ------- | ----- | ----------- |
| `v0.1.0` | — | Initial release, all one-shot commands |
| `v0.2.0` | Phase 0 | `pkg/probe/` refactor, JSON output, config file, first tests |
| `v0.3.0` | — | Correctness and hardening: exit codes, signals, real tests |
| next | Phases 3 + 6 | SYN scanner, `--fast`, `--benchmark`, measured docs, Docker |

## What would actually be worth adding

Small, self-contained, and in keeping with what netdiag already is:

- A BPF filter on the SYN scanner's receive socket. It currently reads this
  process's own outbound SYNs back off the raw socket — 21,061 of them in one
  instrumented run — and filtering them in the kernel is the obvious next
  optimization. Untried, so no claim about what it would save.
- MTR-style continuous latency measurement, as a one-shot command rather than a
  daemon.
- Full IPv6 support across all commands (`dig AAAA` already works).
- IP geolocation, mDNS/Zeroconf discovery, PCAP export.
