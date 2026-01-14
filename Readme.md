# netdiag 🌐

[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

**netdiag** is a powerful, unified network diagnostic CLI tool built in Go. It combines the functionality of multiple network utilities (`ping`, `traceroute`, `nmap`, `dig`, `whois`, `speedtest`) into a single, fast, and easy-to-use command-line interface.

## 🚀 Features

- **🏓 Concurrent Ping** - Test connectivity to multiple hosts simultaneously
- **📡 Speed Test** - Measure your internet download/upload speeds
- **🗺️ Traceroute** - Discover the network path to any destination
- **🔍 Port Scanner** - Scan for open TCP ports with high-performance concurrency
- **🌐 HTTP Health Check** - Verify website status and SSL certificate validity
- **📋 DNS Lookup** - Query DNS records (A, MX, TXT, NS, CNAME)
- **📖 WHOIS Lookup** - Retrieve domain registration information
- **🔎 Network Discovery** - Scan your local network for active devices

## 📋 Table of Contents

- [Installation](#installation)
- [Quick Start](#quick-start)
- [Commands Reference](#commands-reference)
- [Architecture & Concepts](#architecture--concepts)
- [Permissions](#permissions)
- [Contributing](#contributing)
- [License](#license)

## 🛠️ Installation

### Prerequisites

- Go 1.25 or higher
- Administrator/root privileges (for ICMP operations)

### Quick Install

Choose your platform and follow the instructions:

<details>
<summary><b>🐧 Linux (Bash/Zsh)</b></summary>

```bash
# Clone the repository
git clone https://github.com/ARCoder181105/netdiag.git
cd netdiag

# Build the binary
go build -o netdiag

# Install globally
sudo mv netdiag /usr/local/bin/

# Grant ICMP permissions (IMPORTANT - see Security Note below)
sudo setcap cap_net_raw+ep /usr/local/bin/netdiag

# Verify installation
netdiag --help
```

**Security Note**: Instead of running as root, we grant only the `CAP_NET_RAW` capability for raw socket access. This follows the principle of least privilege.

</details>

<details>
<summary><b>🍎 macOS (Bash/Zsh)</b></summary>

```bash
# Clone the repository
git clone https://github.com/ARCoder181105/netdiag.git
cd netdiag

# Build the binary
go build -o netdiag

# Install globally
sudo mv netdiag /usr/local/bin/

# Make executable
sudo chmod +x /usr/local/bin/netdiag

# Verify installation
netdiag --help
```

**Note**: macOS doesn't support Linux capabilities. You'll need to run ICMP-based commands (ping, trace, discover) with `sudo`:

```bash
sudo netdiag ping google.com
sudo netdiag trace github.com
sudo netdiag discover
```

For non-ICMP commands (scan, http, dig, whois, speedtest), sudo is not required.

</details>

<details>
<summary><b>🪟 Windows (PowerShell)</b></summary>

```powershell
# Clone the repository
git clone https://github.com/ARCoder181105/netdiag.git
cd netdiag

# Build the binary
go build -o netdiag.exe

# Create installation directory (if it doesn't exist)
New-Item -ItemType Directory -Force -Path "$env:USERPROFILE\bin"

# Move binary to user bin directory
Move-Item -Force netdiag.exe "$env:USERPROFILE\bin\"

# Add to PATH (if not already added)
$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($userPath -notlike "*$env:USERPROFILE\bin*") {
    [Environment]::SetEnvironmentVariable(
        "Path",
        "$userPath;$env:USERPROFILE\bin",
        "User"
    )
}

# Refresh PATH in current session
$env:Path = [System.Environment]::GetEnvironmentVariable("Path", "Machine") + ";" + [System.Environment]::GetEnvironmentVariable("Path", "User")

# Verify installation
netdiag --help
```

**Important**: 
- Run PowerShell as Administrator for ICMP operations (ping, trace, discover)
- Regular user mode works for: scan, http, dig, whois, speedtest
- Restart your terminal after installation to refresh PATH

**Alternative Installation (System-wide)**:

```powershell
# Run PowerShell as Administrator

# Build and install
go build -o netdiag.exe
Move-Item -Force netdiag.exe "C:\Windows\System32\"

# Verify
netdiag --help
```

</details>

<details>
<summary><b>🐠 Fish Shell (Linux/macOS)</b></summary>

```fish
# Clone the repository
git clone https://github.com/ARCoder181105/netdiag.git
cd netdiag

# Build the binary
go build -o netdiag

# Install globally
sudo mv netdiag /usr/local/bin/

# Grant ICMP permissions (Linux only)
sudo setcap cap_net_raw+ep /usr/local/bin/netdiag

# Verify installation
netdiag --help
```

</details>

### Build Options

#### Static Binary (for distribution)

```bash
# Linux
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o netdiag

# macOS (Intel)
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o netdiag

# macOS (Apple Silicon)
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o netdiag

# Windows
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o netdiag.exe
```

#### Cross-Platform Build Script

**Linux/macOS** (`build.sh`):
```bash
#!/bin/bash

echo "Building netdiag for multiple platforms..."

# Linux
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o dist/netdiag-linux-amd64
GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o dist/netdiag-linux-arm64

# macOS
GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o dist/netdiag-darwin-amd64
GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o dist/netdiag-darwin-arm64

# Windows
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o dist/netdiag-windows-amd64.exe

echo "Build complete! Binaries are in ./dist/"
```

**Windows** (`build.ps1`):
```powershell
Write-Host "Building netdiag for multiple platforms..." -ForegroundColor Cyan

# Create dist directory
New-Item -ItemType Directory -Force -Path "dist" | Out-Null

# Linux
$env:GOOS = "linux"; $env:GOARCH = "amd64"
go build -ldflags="-s -w" -o dist/netdiag-linux-amd64
$env:GOOS = "linux"; $env:GOARCH = "arm64"
go build -ldflags="-s -w" -o dist/netdiag-linux-arm64

# macOS
$env:GOOS = "darwin"; $env:GOARCH = "amd64"
go build -ldflags="-s -w" -o dist/netdiag-darwin-amd64
$env:GOOS = "darwin"; $env:GOARCH = "arm64"
go build -ldflags="-s -w" -o dist/netdiag-darwin-arm64

# Windows
$env:GOOS = "windows"; $env:GOARCH = "amd64"
go build -ldflags="-s -w" -o dist/netdiag-windows-amd64.exe

Write-Host "Build complete! Binaries are in ./dist/" -ForegroundColor Green
```

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

## 📖 Commands Reference

### `netdiag ping`

Send ICMP echo requests to one or more hosts concurrently.

```bash
netdiag ping <host> [more hosts...]

Flags:
  -c, --count int       Number of ICMP packets to send (default: 3)
  -t, --timeout int     Timeout per packet in seconds (default: 1)
  -i, --interval int    Time to wait between packets in seconds (default: 1)

Examples:
  netdiag ping google.com
  netdiag ping -c 10 8.8.8.8 1.1.1.1
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
  -m, --max-hops int    Maximum number of hops (default: 30)

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
  -p, --ports string    Port range to scan (default: "1-1024")
  -t, --timeout int     Timeout in seconds (default: 1)

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
  -t, --timeout int     Timeout for the request in seconds (default: 5)
  -m, --method string   HTTP method (default: "GET")

Examples:
  netdiag http example.com
  netdiag http https://github.com
  netdiag http https://expired.badssl.com --timeout 10
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

Supported Types: A, MX, TXT, NS, CNAME

Examples:
  netdiag dig google.com          # Default: A records (IPv4)
  netdiag dig github.com MX       # Mail servers
  netdiag dig example.com TXT     # Text records
  netdiag dig google.com NS       # Name servers
```

**Output**: Table of DNS records matching the specified type.

---

### `netdiag whois`

Retrieve domain registration and ownership information.

```bash
netdiag whois <domain>

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
  -t, --timeout int     Ping timeout in milliseconds (default: 500)

Examples:
  netdiag discover
  netdiag discover -t 1000
```

**Output**: 
- Auto-detects your local IP range (e.g., 192.168.1.0/24)
- Scans all 254 addresses
- Displays table of discovered devices with IP, hostname, and latency

---

## 🏗️ Architecture & Concepts

### Design Philosophy

**netdiag** is architected as an **extensible platform**, not just a collection of scripts. The modular design allows for easy addition of new diagnostic features without refactoring core functionality.

### Key Technologies & Libraries

#### 1. **CLI Framework: Cobra**
- **Library**: [`github.com/spf13/cobra`](https://github.com/spf13/cobra)
- **Why**: Industry-standard for complex CLI apps (used by Kubernetes, Hugo, GitHub CLI)
- **Benefits**: 
  - Powerful nested subcommand support
  - Robust flag parsing
  - Auto-generated help text
  - Easy integration with configuration files via Viper

#### 2. **Concurrency Management: errgroup**
- **Library**: [`golang.org/x/sync/errgroup`](https://pkg.go.dev/golang.org/x/sync/errgroup)
- **Why**: Professional error handling for concurrent operations
- **Benefits**:
  - Automatic error propagation from goroutines
  - Context cancellation on first error
  - Built-in concurrency limiting with `SetLimit()`
  - Prevents resource exhaustion

**Example Use Case**: When pinging 100 hosts, errgroup limits concurrent operations to 20, preventing system overload while efficiently managing errors.

#### 3. **ICMP Operations: pro-bing**
- **Library**: [`github.com/prometheus-community/pro-bing`](https://github.com/prometheus-community/pro-bing)
- **Why**: Production-grade ICMP library from Prometheus community
- **Features**:
  - Detailed statistics (min/max/avg RTT, packet loss, jitter)
  - Privileged and unprivileged mode support
  - Callback-based result handling

#### 4. **Output Formatting**

**Table Rendering**: [`github.com/olekukonko/tablewriter`](https://github.com/olekukonko/tablewriter)
- Transforms raw data into clean, aligned ASCII tables
- Automatic column width calculation
- Border customization

**Semantic Colors**: [`github.com/fatih/color`](https://github.com/fatih/color)
- Cross-platform color support
- Semantic color scheme:
  - 🟢 **Green**: Success (host up, port open, SSL valid)
  - 🔴 **Red**: Failure (host down, connection refused, SSL expired)
  - 🟡 **Yellow**: Warning (high latency, SSL expiring soon)
  - 🔵 **Cyan**: Informational (headers, progress updates)

#### 5. **Network Operations**

- **Port Scanning**: Go's `net.DialTimeout()` with semaphore-based concurrency control
- **Traceroute**: Raw ICMP sockets via `golang.org/x/net/icmp` with TTL manipulation
- **DNS Queries**: Go's standard `net` package for DNS lookups
- **WHOIS**: [`github.com/likexian/whois`](https://github.com/likexian/whois-go)
- **Speed Test**: [`github.com/showwin/speedtest-go`](https://github.com/showwin/speedtest-go)

### Concurrency Patterns

#### Worker Pool with Semaphore (Port Scanner)

```go
semaphore := make(chan struct{}, 100) // Limit to 100 concurrent scans

for _, port := range ports {
    wg.Add(1)
    go func(p int) {
        defer wg.Done()
        
        semaphore <- struct{}{}        // Acquire slot (blocks if full)
        defer func() { <-semaphore }() // Release slot
        
        // Perform scan
        conn, err := net.DialTimeout("tcp", address, timeout)
        if err == nil {
            conn.Close()
            results <- p // Port is open
        }
    }(port)
}
```

**Benefits**: Prevents "too many open files" errors while maximizing throughput.

#### errgroup Pattern (Concurrent Ping)

```go
group, ctx := errgroup.WithContext(context.Background())
group.SetLimit(20) // Max 20 concurrent pings

for _, host := range hosts {
    h := host
    group.Go(func() error {
        if ctx.Err() != nil {
            return ctx.Err() // Stop if another goroutine failed
        }
        
        pinger, err := probing.NewPinger(h)
        if err != nil {
            return err // Error propagates, cancels context
        }
        
        return pinger.Run()
    })
}

if err := group.Wait(); err != nil {
    // Handle first error from any goroutine
}
```

**Benefits**: Automatic error handling, context cancellation, and controlled concurrency.

### Raw Socket Operations & Privileges

Many network diagnostic operations (ping, traceroute) require **raw socket access** to craft custom ICMP packets. This is a privileged operation for security reasons.

#### The Problem
- Raw sockets allow packet crafting, which could be used maliciously
- Operating systems restrict this capability to root/Administrator

#### The Solution: Linux Capabilities
Instead of running the entire program as root (`sudo netdiag`), grant only the specific capability needed:

```bash
sudo setcap cap_net_raw+ep /usr/local/bin/netdiag
```

This grants `CAP_NET_RAW` (raw socket creation) to the binary while keeping everything else unprivileged—following the **principle of least privilege**.

---

## 🔐 Permissions

### Linux/macOS

After building, grant ICMP capabilities:

```bash
sudo setcap cap_net_raw+ep ./netdiag
```

Alternatively, run with sudo (not recommended):

```bash
sudo ./netdiag ping google.com
```

### Windows

Run Command Prompt or PowerShell as Administrator.

---

## 🤝 Contributing

Contributions are welcome! Here are some ideas for enhancements:

- [ ] MTR (My Traceroute) implementation for continuous latency monitoring
- [ ] IP geolocation lookup
- [ ] mDNS/Zeroconf service discovery
- [ ] JSON output mode (`--json` flag)
- [ ] Configuration file support
- [ ] IPv6 support for all commands

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