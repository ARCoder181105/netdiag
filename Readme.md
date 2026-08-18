# netdiag

[![Latest release](https://img.shields.io/github/v/release/ARCoder181105/netdiag)](https://github.com/ARCoder181105/netdiag/releases)
[![Go 1.24+](https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![CI](https://github.com/ARCoder181105/netdiag/actions/workflows/ci.yml/badge.svg)](https://github.com/ARCoder181105/netdiag/actions)
[![License](https://img.shields.io/github/license/ARCoder181105/netdiag)](LICENSE)

One network diagnostics CLI instead of eight. `ping`, `traceroute`, port
scanning, DNS, HTTP and TLS checks, WHOIS, speed test, and LAN discovery — with
consistent JSON output, consistent exit codes, and consistent behavior on
Ctrl+C across every one of them.

```bash
netdiag ping google.com cloudflare.com
netdiag scan 192.168.1.1 -p 1-65535 --fast
netdiag http https://example.com --json | jq .http_data.tls_days_left
```

<!--
  Demo recording: run `vhs docs/demo.tape` to produce docs/demo.gif,
  then uncomment the line below.

![netdiag demo](docs/demo.gif)
-->

## Contents

- [Engineering highlights](#engineering-highlights)
- [Install](#install)
- [Commands](#commands)
- [Scripting: JSON and exit codes](#scripting-json-and-exit-codes)
- [Configuration](#configuration)
- [Permissions](#permissions)
- [Docker](#docker)
- [Architecture](#architecture)
- [Responsible use](#responsible-use)
- [Contributing](#contributing)

## Engineering highlights

The parts worth reading the source for.

### Half-open SYN scanning, with the checksum done by hand

`scan --fast` sends a bare TCP SYN and reads the reply without ever completing
the handshake — SYN-ACK means open, RST means closed, silence means filtered.
The TCP checksum is computed over the IPv4 pseudo-header by netdiag's own code
rather than handed to a library helper, and it is tested against the RFC 1071
worked example plus a vector generated independently in Python.

Replies are correlated on source port *and* sequence number, not arrival order,
because a raw socket receives every TCP segment on the machine — including this
process's own outbound SYNs and the kernel's RSTs.

Measured on bare metal, 65,535 loopback ports, median of 5 runs:

| Concurrency | connect | syn | |
|---|---|---|---|
| `-c 100` | 264 ms | 151 ms | **1.75x** |
| `-c 2000` | 385 ms | 128 ms | **3.0x** |

It is also more *accurate* under file descriptor pressure: the connect scan
needs one descriptor per port and reports an `EMFILE` failure as a closed port,
so at `ulimit -n 32` it found 200, 184, 166, 154 and 200 of 200 open ports
across five runs, where the SYN scan found all 200 every time.

Full methodology, the caveats, and the two bugs that benchmarking found —
including the version that was *slower* than the connect scan, and why the fix
was not the packet library — are in [docs/performance.md](docs/performance.md).

### ICMP privilege negotiation

Raw ICMP needs `CAP_NET_RAW`. Most systems also offer unprivileged ICMP
datagram sockets — but Windows has none, and Linux gates them behind a sysctl
that some distributions leave empty.

netdiag picks the mode most likely to work on the current platform, retries with
the other one on a permission error, and caches whichever worked for the rest of
the process, so a 1,024-host `discover` sweep pays that cost at most once. If
neither works, the error tells you the exact command to fix it instead of
printing `socket: permission denied`.

The same degrade-rather-than-fail pattern covers `scan --fast`: no capability,
or no route to derive a source address from, and it falls back to the connect
scanner with one notice on stderr.

### A severity contract that does not break pipelines

Every probe returns a typed `Result` with a `Severity` that describes *the
target*, separately from whether the probe itself could run. That split is what
makes the exit codes usable in scripts, and it is why a warning exits `0` —
see [Scripting](#scripting-json-and-exit-codes).

### Probes never print

All network logic lives behind one interface in `pkg/probe/`, returns a value,
and touches neither stdout nor `os.Exit`. `--json`, table rendering, structured
logging, and exit codes are each implemented once in the shared runner rather
than per command. Details in [docs/architecture.md](docs/architecture.md).

## Install

**Linux / macOS**

```bash
curl -fsSL https://raw.githubusercontent.com/ARCoder181105/netdiag/main/install.sh | bash
```

**Windows** (PowerShell as Administrator)

```powershell
irm https://raw.githubusercontent.com/ARCoder181105/netdiag/main/install.ps1 | iex
```

**Go**

```bash
go install github.com/ARCoder181105/netdiag@latest
```

Both install scripts are fetched from `main` and piped straight into a shell.
If you would rather not do that, read
[install.sh](install.sh) first, or use `go install` or a
[release binary](#pre-built-binaries) instead — releases are versioned and
publish checksums.

<details>
<summary>Pre-built binaries, building from source, uninstalling</summary>

### Pre-built binaries

[Download the latest release](https://github.com/ARCoder181105/netdiag/releases/latest)
for Linux (amd64/arm64), macOS (Intel/Apple Silicon), or Windows (amd64).

Every release publishes a `checksums.txt`. Verify before installing — download
it alongside the binary, then:

```bash
sha256sum --check --ignore-missing checksums.txt
```

```bash
chmod +x netdiag-*
sudo mv netdiag-* /usr/local/bin/netdiag
sudo setcap cap_net_raw+ep /usr/local/bin/netdiag   # Linux, for ICMP and --fast
```

On Windows, extract `netdiag.exe` and put it somewhere on your `PATH`.

### From source

Requires Go 1.24+.

```bash
git clone https://github.com/ARCoder181105/netdiag.git
cd netdiag
make build            # or: go build -o netdiag
sudo make install     # installs to /usr/local/bin and applies setcap
```

### Uninstall

```bash
curl -fsSL https://raw.githubusercontent.com/ARCoder181105/netdiag/main/uninstall.sh | bash
```

Or manually: `sudo make uninstall`, `sudo rm /usr/local/bin/netdiag`, or
`rm $(go env GOPATH)/bin/netdiag` if you installed with `go install`.

</details>

Verify with `netdiag --version`.

## Commands

| Command | What it does | Needs privileges |
|---|---|---|
| [`ping`](#ping) | ICMP echo to one or more hosts, concurrently | usually not — see [Permissions](#permissions) |
| [`scan`](#scan) | TCP port scan, connect or half-open SYN | only for `--fast` |
| [`trace`](#trace) | Traceroute to a destination | yes |
| [`http`](#http) | HTTP status, latency, and TLS certificate check | no |
| [`dig`](#dig) | DNS lookups (A, AAAA, MX, TXT, NS, CNAME) | no |
| [`whois`](#whois) | Domain registration lookup | no |
| [`discover`](#discover) | Sweep the local network for active devices | usually not — see [Permissions](#permissions) |
| [`speedtest`](#speedtest) | Download and upload throughput | no |

`ping` and `discover` try unprivileged ICMP datagram sockets first and work
without setup on macOS and most Linux systems. `trace` and `scan --fast` always
need `CAP_NET_RAW` or the equivalent.

Global flags, valid on every command:

```text
  -j, --json                Output machine-readable JSON instead of tables
  -l, --log-file string     Append structured logs to a file instead of stderr
      --log-format string   Log format: text or json (default "text")
      --log-level string    Log level: debug, info, warn, error (default "info")
```

`--version` is on `netdiag` itself, not on subcommands.

### ping

Sends ICMP echo requests to any number of hosts concurrently.

```text
  -c, --count int           Number of ICMP packets to send (default 3)
  -t, --timeout duration    Total timeout for the whole run, not per packet (default 5s)
  -i, --interval duration   Time to wait between packets (default 1s)
      --concurrency int     Number of hosts to ping concurrently (default 20)
```

```bash
netdiag ping google.com
netdiag ping -c 10 8.8.8.8 1.1.1.1
netdiag ping -t 2s -i 500ms github.com
```

Reports packet loss and min/avg/max/stddev latency per host. Exits non-zero if
any host failed.

### scan

Scans TCP ports, either with ordinary connections or with half-open SYN probes.

```text
  -p, --ports string        Ports to scan: a list, a range, or both (default "1-1024")
  -t, --timeout duration    Connection timeout per port (default 1s)
  -c, --concurrency int     Number of ports to probe concurrently (default 100)
      --fast                Use a half-open SYN scan (needs CAP_NET_RAW; falls back)
      --benchmark           Run both scan methods against the target and compare them
```

```bash
netdiag scan localhost
netdiag scan 192.168.1.1 -p 80,443,8000-9000
netdiag scan 192.168.1.1 -p 1-65535 --fast
netdiag scan 127.0.0.1 -p 1-65535 --benchmark
```

`--fast` needs `CAP_NET_RAW`; without it the scan falls back to the connect
method, says so once on stderr, and still returns results. `--benchmark` runs
both methods against the same target and prints a comparison — and tells you if
the SYN half fell back, so an unprivileged benchmark cannot be mistaken for a
real comparison.

A scan that finds nothing is a warning, not an error: it ran fine, there was
just nothing to report. A scan interrupted with Ctrl+C says its results are
incomplete rather than claiming the unscanned ports were closed.

### trace

```text
  -m, --max-hops int        Maximum number of hops (default 30)
  -t, --timeout duration    Timeout per hop (default 2s)
```

```bash
netdiag trace google.com
netdiag trace 8.8.8.8 -m 20
```

Prints each hop with its IP, resolved hostname, and round-trip time. Replies are
matched by parsing the quoted original header out of the ICMP body and checking
the echo ID and sequence, so concurrent pings elsewhere on the machine cannot
pollute the hop list.

### http

```text
  -t, --timeout duration    Timeout for the request (default 5s)
  -m, --method string       HTTP method for the request (default "GET")
      --skip-tls            Skip TLS certificate verification (insecure)
```

```bash
netdiag http example.com
netdiag http https://github.com
netdiag http https://expired.badssl.com --timeout 10s
```

Reports status code, latency, redirect count, and certificate issuer plus days
remaining. A certificate close to expiry is a warning, so it is visible without
failing a deploy pipeline.

### dig

```text
  -s, --server string       Custom DNS server (e.g. 8.8.8.8 or 8.8.8.8:5353)
  -t, --timeout duration    Query timeout (default 5s)
```

```bash
netdiag dig google.com          # A records by default
netdiag dig google.com AAAA
netdiag dig github.com MX
netdiag dig google.com --server 1.1.1.1
```

Supported types: `A`, `AAAA`, `MX`, `TXT`, `NS`, `CNAME`.

### whois

```text
  -t, --timeout duration    Query timeout (default 10s)
```

```bash
netdiag whois example.com
```

The timeout bounds the whole IANA → registry → registrar chain, not each hop
individually.

### discover

```text
  -t, --timeout duration    Ping timeout per host (default 500ms)
```

```bash
netdiag discover
netdiag discover -t 1s
```

Detects your primary IPv4 network by asking the kernel which source address it
would use for outbound traffic — so a Docker bridge does not get swept instead
of your actual LAN — then sweeps it, capped at 1,024 addresses.

### speedtest

```text
  -u, --no-upload           Skip upload test
  -s, --server string       Specify speedtest server ID
```

```bash
netdiag speedtest
netdiag speedtest --no-upload
```

## Scripting: JSON and exit codes

Every command supports `--json`. Logs and diagnostics go to stderr by default —
or to a file with `--log-file` — and never to stdout, so the JSON stays a clean
pipe either way. The examples below use [jq](https://jqlang.github.io/jq/),
which is not required to run netdiag but makes the JSON output far easier to
work with:

```bash
netdiag ping 1.1.1.1 --json | jq '.[0].ping_data.avg_rtt'
netdiag scan localhost -p 1-1024 --json | jq '.scan_data.open_ports'
netdiag http https://example.com --json | jq '.http_data.tls_days_left'
```

| Code | Severity | Meaning |
|---|---|---|
| `0` | `OK` or `Warning` | The probe ran; the target is up |
| `1` | `Error` | The probe ran and the target failed |
| `2` | — | Invalid arguments, flags, or configuration |
| `3` | — | The probe could not run at all (for example, ICMP is not permitted) |

A host that cannot be resolved is a failed *target*, not a probe that could not
run, so it exits `1` rather than `3`.

**Warnings exit `0` deliberately.** A certificate with 10 days left, 25% packet
loss on a host that still answers, or a traceroute that never reached the final
hop are degraded-but-alive states. They appear in the output and in `severity`,
but they do not fail the command — otherwise every warning would break this:

```bash
netdiag http https://api.example.com && ./deploy.sh
```

To treat warnings as failures, read `severity` yourself (`0` OK, `1` Warning,
`2` Error, `3` Unknown):

```bash
sev=$(netdiag http https://api.example.com --json | jq .severity)
[ "$sev" -eq 0 ] || exit 1
```

## Configuration

netdiag reads `~/.netdiag.yaml` if it exists. CLI flags always win.

```yaml
scan:
  # Connection timeout per port, used when --timeout is not passed.
  default_timeout: "1s"
```

Any key can be set through the environment with a `NETDIAG_` prefix:

```bash
NETDIAG_SCAN_DEFAULT_TIMEOUT=2s netdiag scan localhost
```

Keys enter the schema only once a command actually reads them — see
[config.example.yaml](config.example.yaml) for the current set.

## Permissions

Only `ping`, `trace`, `discover`, and `scan --fast` need anything special.
`scan` (default), `http`, `dig`, `whois`, and `speedtest` never do.

**Linux.** `ping` and `discover` try unprivileged ICMP datagram sockets first,
so they often work with no setup. If your kernel does not allow them, or you
want `trace` and `scan --fast`:

```bash
# Preferred: grant only the capability this binary needs
sudo setcap cap_net_raw+ep /usr/local/bin/netdiag

# Or: allow unprivileged ICMP for all users (does not help --fast or trace)
sudo sysctl -w net.ipv4.ping_group_range="0 2147483647"
```

`make install` applies the `setcap` step. netdiag prints whichever of these
applies when it hits a permission error.

**macOS.** Unprivileged ICMP works out of the box for `ping` and `discover`.
`trace` and `scan --fast` need `sudo`.

**Windows.** Run the terminal as Administrator for the ICMP commands.

## Docker

```bash
docker build -t netdiag .
docker run --rm netdiag ping 1.1.1.1
docker run --rm netdiag scan 192.168.1.1 -p 1-65535 --fast
```

The image is ~28 MB and runs as an unprivileged user. `cap_net_raw` is applied
to the binary itself, so ICMP and SYN scanning work without running the
container as root.

For a hardened deployment, drop everything and add back the one capability:

```bash
docker run --rm --cap-drop=ALL --cap-add=NET_RAW netdiag scan host --fast
```

`--cap-drop=ALL` on its own will not start the image: Linux refuses to exec a
binary carrying permitted capabilities the process could never be granted, so
you get an exec error rather than a fallback to unprivileged scanning.

## Architecture

```text
main.go
  └── cmd/           Cobra commands: flags, validation, rendering
        └── run.go   runProbe(): cancellation, logging, --json, exit codes
              │
              ▼
      pkg/probe/     all network logic behind the Prober interface
              │
              ▼
      pkg/output/    tables, color, JSON
      pkg/logger/    log/slog, stderr by default
      pkg/config/    Viper loader for ~/.netdiag.yaml
```

`cmd/` imports `pkg/probe/`, never the reverse. Every probe implements the same
two-method interface and returns a typed `Result`; one shared runner turns that
into output, logs, and an exit code.

[docs/architecture.md](docs/architecture.md) has the full picture, including a
diagram and the reasoning behind the `Prober`/`Result` design.

## Responsible use

`netdiag scan` and `netdiag discover` send unsolicited traffic to hosts.
Scanning or sweeping systems you do not own, or do not have explicit written
permission to test, is unlawful in many jurisdictions. Use them on your own
infrastructure or with documented authorization.

## Contributing

Contributions welcome. Ideas that would fit:

- MTR-style continuous latency monitoring
- IP geolocation lookup
- mDNS/Zeroconf service discovery
- Full IPv6 support across all commands (`dig AAAA` is done)
- Packet capture / PCAP export

Scope decisions, including the phases that were deliberately cut, are in
[ROADMAP.md](ROADMAP.md).

```bash
git clone https://github.com/ARCoder181105/netdiag.git
cd netdiag
go mod download
make test      # go test ./...
make lint      # golangci-lint
make fmt       # gofumpt + gci
make build
```

Then fork, branch, and open a pull request.
[CONTRIBUTING.md](CONTRIBUTING.md) has the details.

## License

MIT — see [LICENSE](LICENSE).

Built with [Cobra](https://github.com/spf13/cobra),
[pro-bing](https://github.com/prometheus-community/pro-bing),
[gopacket](https://github.com/gopacket/gopacket),
[tablewriter](https://github.com/olekukonko/tablewriter),
[color](https://github.com/fatih/color),
[speedtest-go](https://github.com/showwin/speedtest-go), and
[whois](https://github.com/likexian/whois).
