# netdiag 🌐

<a href="https://github.com/ARCoder181105/netdiag/releases"><img alt="Latest release" src="https://img.shields.io/github/v/release/ARCoder181105/netdiag"></a>
<a href="https://go.dev/"><img alt="Go 1.24+" src="https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat&logo=go"></a>
<a><img alt="License" src="https://img.shields.io/github/license/ARCoder181105/netdiag"></a>
<a href="https://github.com/ARCoder181105/netdiag/actions"><img alt="CI status" src="https://github.com/ARCoder181105/netdiag/actions/workflows/ci.yml/badge.svg"></a>
<a href="https://github.com/ARCoder181105/netdiag/releases"><img src="https://img.shields.io/github/downloads/ARCoder181105/netdiag/total"></a>

**netdiag** is a powerful, unified network diagnostic CLI tool built in Go. It combines the functionality of multiple network utilities (`ping`, `traceroute`, `nmap`, `dig`, `whois`, `speedtest`) into a single, fast, and easy-to-use command-line interface.

## 🚀 Features

- **🏓 Concurrent Ping** - Test connectivity to multiple hosts simultaneously
- **📡 Speed Test** - Measure your internet download/upload speeds
- **🗺️ Traceroute** - Discover the network path to any destination
- **🔍 Port Scanner** - Scan for open TCP ports with high-performance concurrency
- **🌐 HTTP Health Check** - Verify website status and SSL certificate validity
- **📋 DNS Lookup** - Query DNS records (A, AAAA, MX, TXT, NS, CNAME)
- **📖 WHOIS Lookup** - Retrieve domain registration information
- **🔎 Network Discovery** - Scan your local network for active devices

## 📋 Table of Contents

- [Installation](#-installation)
- [Uninstallation](#-uninstallation)
- [Quick Start](#-quick-start)
- [Global Flags](#-global-flags)
- [Exit Codes](#-exit-codes)
- [Responsible Use](#-responsible-use)
- [Commands Reference](#-commands-reference)
- [Configuration](#-configuration)
- [Architecture & Concepts](#-architecture--concepts)
- [Permissions](#-permissions)
- [Contributing](#-contributing)
- [License](#-license)

## 🛠️ Installation

### Quick Install (Recommended)

**Linux/macOS:**

```bash
curl -fsSL https://raw.githubusercontent.com/ARCoder181105/netdiag/main/install.sh | bash
```

**Windows (PowerShell as Administrator):**

```powershell
irm https://raw.githubusercontent.com/ARCoder181105/netdiag/main/install.ps1 | iex
```

---

### Other Installation Methods

<details>
<summary><b>📦 Package Managers</b></summary>

#### Go Install

```bash
go install github.com/ARCoder181105/netdiag@latest
```

</details>

<details>
<summary><b>⬇️ Download Pre-built Binaries</b></summary>

Download the latest release for your platform:

**[📥 Download Latest Release](https://github.com/ARCoder181105/netdiag/releases/latest)**

Available platforms:

- Linux (amd64, arm64)
- macOS (Intel, Apple Silicon)
- Windows (amd64)

After downloading:

**Linux/macOS:**

```bash
chmod +x netdiag-*
sudo mv netdiag-* /usr/local/bin/netdiag

# Linux only: Grant ICMP capabilities
sudo setcap cap_net_raw+ep /usr/local/bin/netdiag
```

**Windows:**

- Extract `netdiag.exe`
- Move to `C:\Windows\System32\` or add to PATH

</details>

<details>
<summary><b>🔨 Build from Source</b></summary>

**Prerequisites:**

- Go 1.24 or higher
- Git

```bash
# Clone repository
git clone https://github.com/ARCoder181105/netdiag.git
cd netdiag

# Build
go build -o netdiag

# Install (optional)
sudo mv netdiag /usr/local/bin/

# Linux: Grant ICMP capabilities
sudo setcap cap_net_raw+ep /usr/local/bin/netdiag
```

</details>

---

### Verify Installation

```bash
netdiag --version
netdiag --help
```

### Quick Test

```bash
# Test connectivity
netdiag ping google.com

# Run speed test
netdiag speedtest

# Scan ports
netdiag scan localhost -p 1-1000
```

## 🗑️ Uninstallation

If you need to remove netdiag, you can use the provided uninstallation scripts or remove it manually.

### Quick Uninstall

**Linux/macOS:**

```bash
curl -fsSL https://raw.githubusercontent.com/ARCoder181105/netdiag/main/uninstall.sh | bash
```

**Windows (PowerShell as Administrator):**

```powershell
irm https://raw.githubusercontent.com/ARCoder181105/netdiag/main/uninstall.ps1 | iex
```

---

### Manual Uninstall

<details>
<summary><b>Removing netdiag from your system</b></summary>

#### If installed via Makefile or script:

```bash
# Using Makefile
make uninstall

# Or manually remove the binary
sudo rm /usr/local/bin/netdiag  # Linux/macOS
```

#### If installed via Go:

```bash
rm $(go env GOPATH)/bin/netdiag
```

#### Windows:

```powershell
# If installed to System32
Remove-Item C:\Windows\System32\netdiag.exe

# Or remove from your custom PATH location
```

</details>

## 🚀 Quick Start

```bash
# Test connectivity to multiple hosts
netdiag ping google.com cloudflare.com

# Run an internet speed test
netdiag speedtest

# Trace the route to a destination
netdiag trace github.com

# Scan for open ports
netdiag scan 192.168.1.1 --ports 1-1024

# Check website health and SSL certificate
netdiag http https://example.com

# Lookup DNS records
netdiag dig google.com MX

# Get domain registration info
netdiag whois example.com

# Discover devices on your local network
netdiag discover
```

### 💻 Advanced JSON Parsing

`netdiag` natively supports JSON output for all commands using the `--json` flag. To parse and filter this output in the terminal, we highly recommend installing [jq](https://jqlang.github.io/jq/).

**Example:** Get the average ping latency:

```bash
netdiag ping 1.1.1.1 --json | jq '.[0].ping_data.avg_rtt'
```

## 🌍 Global Flags

These work on every command:

```text
  -j, --json                 Output machine-readable JSON instead of tables
  -l, --log-file string      Append structured logs to a file instead of stderr
      --log-format string    Log format: text or json (default: "text")
      --log-level string     Log level: debug, info, warn, error (default: "info")
```

`-v, --version` is available on `netdiag` itself (`netdiag --version`), not on
subcommands.

Logs always go to stderr (or `--log-file`), never stdout, so `--json` output
stays pipeable:

```bash
netdiag ping 1.1.1.1 --json --log-level debug | jq '.[0].ping_data.avg_rtt'
```

## 🔢 Exit Codes

| Code | Severity           | Meaning                                                     |
| ---- | ------------------ | ----------------------------------------------------------- |
| `0`  | `OK` or `Warning`  | The probe ran; the target is up                             |
| `1`  | `Error`            | The probe ran and the target failed                         |
| `2`  | —                  | Invalid arguments, flags, or configuration                  |
| `3`  | —                  | The probe could not run (no privileges, unresolvable host)  |

**Warnings exit `0` on purpose.** A certificate expiring in 10 days, 25% packet
loss on a host that is still up, or a traceroute that did not reach the final
hop are all degraded-but-alive states. They are reported in the output and in
`severity`, but they do not fail the command — otherwise every warning would
break a pipeline:

```bash
netdiag http https://api.example.com && ./deploy.sh
```

To treat warnings as failures, check `severity` yourself (`0` OK, `1` Warning,
`2` Error, `3` Unknown):

```bash
sev=$(netdiag http https://api.example.com --json | jq .severity)
[ "$sev" -eq 0 ] || exit 1
```

## ⚠ Responsible Use

`netdiag scan` and `netdiag discover` send unsolicited traffic to hosts. Scanning
or sweeping systems you do not own, or do not have explicit written permission to
test, is unlawful in many jurisdictions. Use these commands on your own
infrastructure or with documented authorization.

## 📖 Commands Reference

### `netdiag ping`

Send ICMP echo requests to one or more hosts concurrently.

```bash
netdiag ping <host> [more hosts...]

Flags:
  -c, --count int          Number of ICMP packets to send (default: 3)
  -t, --timeout duration   Timeout per host, e.g. 1s, 500ms (default: 1s)
  -i, --interval duration  Time to wait between packets, e.g. 1s (default: 1s)
      --concurrency int    Number of hosts to ping concurrently (default: 20)

Examples:
  netdiag ping google.com
  netdiag ping -c 10 8.8.8.8 1.1.1.1
  netdiag ping -t 2s -i 500ms github.com
```

**Output**: Displays a table with packet loss, average/min/max latency for each host.

---

### `netdiag speedtest`

Test your internet connection speed (download/upload).

```bash
netdiag speedtest

Flags:
  -u, --no-upload       Skip upload test
  -s, --server string   Specify server ID

Examples:
  netdiag speedtest
  netdiag speedtest --no-upload
  netdiag speedtest --server 12345
```

**Output**: Shows ISP info, server details, ping, download speed, and upload speed with quality assessment.

---

### `netdiag trace`

Perform a traceroute to discover the network path to a destination.

```bash
netdiag trace <host>

Flags:
  -m, --max-hops int       Maximum number of hops (default: 30)
  -t, --timeout duration   Timeout per hop, e.g. 2s, 500ms (default: 2s)

Examples:
  netdiag trace google.com
  netdiag trace 8.8.8.8 -m 20
```

**Output**: Displays each hop with IP address, hostname, and round-trip time.

---

### `netdiag scan`

Scan a target host for open TCP ports using a high-performance worker pool.

```bash
netdiag scan <host>

Flags:
  -p, --ports string        Ports to scan: list, range, or both (default: "1-1024")
  -t, --timeout duration    Connection timeout per port (default: 1s)
  -c, --concurrency int     Ports to probe concurrently (default: 100)

Examples:
  netdiag scan localhost
  netdiag scan 192.168.1.1 -p 80,443,8000-9000
  netdiag scan example.com -p 1-65535
```

**Output**: Lists all discovered open ports in a table format.

---

### `netdiag http`

Check HTTP status and SSL certificate information for a website.

```bash
netdiag http <url>

Flags:
  -t, --timeout duration   Timeout for the request, e.g. 5s, 500ms (default: 5s)
  -m, --method string      HTTP method (default: "GET")
      --skip-tls           Skip TLS certificate verification (insecure)

Examples:
  netdiag http example.com
  netdiag http https://github.com
  netdiag http https://expired.badssl.com --timeout 10s
```

**Output**:

- HTTP status code (color-coded by result)
- Request latency
- SSL certificate details (subject, issuer, validity period, expiration warning)

---

### `netdiag dig`

Perform DNS lookups for various record types.

```bash
netdiag dig <domain> [type]

Supported Types: A, AAAA, MX, TXT, NS, CNAME

Flags:
  -s, --server string      Custom DNS server (e.g. 8.8.8.8 or 8.8.8.8:5353)
  -t, --timeout duration   Query timeout (default: 5s)

Examples:
  netdiag dig google.com          # Default: A records (IPv4)
  netdiag dig google.com AAAA     # IPv6 addresses
  netdiag dig github.com MX       # Mail servers
  netdiag dig example.com TXT     # Text records
  netdiag dig google.com NS       # Name servers
  netdiag dig google.com --server 1.1.1.1
```

**Output**: Table of DNS records matching the specified type.

---

### `netdiag whois`

Retrieve domain registration and ownership information.

```bash
netdiag whois <domain>

Flags:
  -t, --timeout duration   Query timeout (default: 10s)

Examples:
  netdiag whois google.com
  netdiag whois github.com
```

**Output**: Full WHOIS record including registrar, creation date, expiration date, and nameservers.

---

### `netdiag discover`

Scan your local network for active devices using ping sweeps.

```bash
netdiag discover

Flags:
  -t, --timeout duration     Ping timeout per host (default: 500ms)

Examples:
  netdiag discover
  netdiag discover -t 1s
```

**Output**:

- Auto-detects your local IPv4 network, including its netmask
- Sweeps every usable host address in that network (capped at 1024 addresses)
- Displays table of discovered devices with IP, hostname, and latency

---

## ⚙ Configuration

netdiag reads `~/.netdiag.yaml` if present. CLI flags always override it.

```yaml
scan:
  # Connection timeout per port, used when --timeout is not passed.
  default_timeout: "1s"
```

Every key can also be set via the environment with a `NETDIAG_` prefix:

```bash
NETDIAG_SCAN_DEFAULT_TIMEOUT=2s netdiag scan localhost
```

Keys are added to the config schema only once a command actually reads them —
see [config.example.yaml](config.example.yaml) for the current set.

## 🏗️ Architecture & Concepts

### Layering

netdiag separates *what to measure* from *how to display it*. Commands are thin
Cobra wrappers; all network logic lives in `pkg/probe/`.

```text
main.go
  └── cmd/                 Cobra commands: flags, argument validation, rendering
        ├── root.go        global flags, logger + config wiring
        └── run.go         runProbe(): the shared execution path
              │
              ▼
      pkg/probe/           all network logic; no printing, no os.Exit
        ├── types.go       Result, Severity, the Prober interface
        ├── ping.go        PingProber
        ├── scan.go        ConnectScanner
        ├── tracer.go      TraceProber
        ├── http.go        HTTPProber
        ├── dig.go         DigProber
        ├── discover.go    DiscoverProber
        ├── whois.go       WhoisProber
        ├── speedtest.go   SpeedTestProber
        └── icmp.go        shared privileged/unprivileged ICMP handling
              │
              ▼
      pkg/output/          color, tables, JSON
      pkg/logger/          log/slog wrapper (stderr by default)
      pkg/config/          Viper loader for ~/.netdiag.yaml
```

### The `Prober` interface

Every probe implements the same two methods:

```go
type Prober interface {
    Probe(ctx context.Context) (Result, error)
    Type() string
}
```

This is what lets one shared runner drive most commands. `ping` targets multiple
hosts, so it runs a batch loop over the same helpers (`logResult`,
`exitForAll`) instead of calling `runProbe` directly.

### The `Result` type

Probes never print. They return a typed `Result` carrying an outcome plus one
probe-specific payload:

```go
type Result struct {
    TimeStamp time.Time
    ProbeType string
    Target    string

    PingData  *PingData   // only one payload is non-nil
    ScanData  *ScanData
    HTTPData  *HTTPData
    // ... DNSData, TraceData, DiscoverData, SpeedTestData, WhoisData

    Message  string
    Severity Severity     // OK | Warning | Error | Unknown
    Success  bool
    Latency  time.Duration
}
```

One type means `--json`, table rendering, exit codes, and structured logging are
all implemented once rather than per command.

### The shared runner

`cmd/run.go` owns everything that must behave identically across commands:

| Concern             | Behaviour                                              |
| ------------------- | ------------------------------------------------------ |
| Cancellation        | `signal.NotifyContext` — Ctrl+C stops a scan mid-flight |
| Error normalization | A hard error becomes a `Result` via `probe.ErrorResult` |
| Logging             | One structured line per probe, always to stderr        |
| `--json`            | Short-circuits rendering, prints the raw `Result`      |
| Color              | `output.PrintBySeverity` maps severity to color       |
| Exit code           | Derived from `Success` and `Severity`                  |

### Concurrency

- **Port scanner** — a semaphore-bounded worker pool (`--concurrency`, default
  100), with every dial carrying the cancellable context.
- **Ping** — `errgroup` with `SetLimit`, so pinging 100 hosts does not open 100
  sockets at once.
- **Discover** — bounded sweep of the detected network, capped at 1024
  addresses so a `/16` interface cannot launch a 65k-host scan.

### ICMP privileges

Raw ICMP sockets need `CAP_NET_RAW` or root. Many systems also offer
*unprivileged* ICMP datagram sockets, which need neither.

`pkg/probe/icmp.go` tries the mode most likely to work on the current platform,
transparently retries with the other on a permission error, and caches the
result for the rest of the process. If neither works, the error explains exactly
how to fix it rather than reporting a bare `socket: permission denied`.

## 🔐 Permissions

Only the ICMP-based commands (`ping`, `trace`, `discover`) need special
permissions. `scan`, `http`, `dig`, `whois`, and `speedtest` never do.

### Linux

`ping` and `discover` first try unprivileged ICMP datagram sockets, so on many
systems they work with no setup at all. If your kernel does not allow them,
pick one of:

```bash
# Preferred: grant only the capability this binary needs
sudo setcap cap_net_raw+ep /usr/local/bin/netdiag

# Or: allow unprivileged ICMP for all users
sudo sysctl -w net.ipv4.ping_group_range="0 2147483647"
```

`make install` applies the `setcap` step for you. netdiag prints whichever of
these applies if it hits a permission error.

`trace` always needs raw sockets, so it requires `cap_net_raw` or `sudo`.

### macOS

Unprivileged ICMP works out of the box for `ping` and `discover`. `trace`
requires `sudo`.

### Windows

Run Command Prompt or PowerShell as Administrator for the ICMP commands.

---

## 🤝 Contributing

Contributions are welcome! Here are some ideas for enhancements:

- [ ] MTR (My Traceroute) implementation for continuous latency monitoring
- [ ] IP geolocation lookup
- [ ] mDNS/Zeroconf service discovery
- [ ] Full IPv6 support across all commands (`dig AAAA` is done)
- [ ] Packet capture / PCAP export

Larger planned work is tracked in [ROADMAP.md](ROADMAP.md).

### Development Setup

```bash
# Clone the repo
git clone https://github.com/ARCoder181105/netdiag.git
cd netdiag

# Install dependencies
go mod download

# Run tests
go test ./...

# Build
go build -o netdiag
```

### Submitting Changes

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

---

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

## 🙏 Acknowledgments

Built with these excellent Go libraries:

- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [pro-bing](https://github.com/prometheus-community/pro-bing) - ICMP operations
- [tablewriter](https://github.com/olekukonko/tablewriter) - Table formatting
- [color](https://github.com/fatih/color) - Terminal colors
- [speedtest-go](https://github.com/showwin/speedtest-go) - Speed testing
- [whois](https://github.com/likexian/whois-go) - WHOIS queries

---

## 📞 Support

For issues, questions, or feature requests, please [open an issue](https://github.com/ARCoder181105/netdiag/issues).

---

**Made with ❤️ by ARCoder181105**
