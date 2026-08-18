package probe

import (
	"context"
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type ConnectScanner struct {
	Host        string
	Ports       []int
	Timeout     time.Duration
	Concurrency int
}

func (c *ConnectScanner) Type() string {
	return "scan"
}

func (c *ConnectScanner) Probe(ctx context.Context) (Result, error) {
	startTime := time.Now()

	var openPorts []int
	results := make(chan int)
	var wg sync.WaitGroup
	sem := make(chan struct{}, c.Concurrency)

	for _, port := range c.Ports {
		wg.Add(1)

		go func(port int) {
			defer wg.Done()

			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				return
			}
			defer func() { <-sem }()

			dialer := net.Dialer{Timeout: c.Timeout}
			address := net.JoinHostPort(c.Host, strconv.Itoa(port))

			conn, err := dialer.DialContext(ctx, "tcp", address)
			if err == nil {
				_ = conn.Close()
				results <- port
			}
		}(port)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	for port := range results {
		openPorts = append(openPorts, port)
	}

	return scanResult(c.Host, len(c.Ports), openPorts, "connect", time.Since(startTime)), nil
}

// scanResult builds the Result for any scan method. Both scanners go through
// it so a --fast scan and a connect scan are byte-for-byte the same JSON shape,
// differing only in scan_method.
func scanResult(host string, totalPorts int, openPorts []int, method string, duration time.Duration) Result {
	sort.Ints(openPorts)

	// A scan that found nothing is reported the same way dig and discover
	// report an empty result: it succeeded, but there is nothing to show.
	severity := SeverityOK
	if len(openPorts) == 0 {
		severity = SeverityWarning
	}

	return Result{
		Target:    host,
		TimeStamp: time.Now(),
		ProbeType: "scan",
		Success:   true,
		Severity:  severity,
		Message:   fmt.Sprintf("Found %d open ports", len(openPorts)),
		ScanData: &ScanData{
			OpenPorts:   openPorts,
			TotalPorts:  totalPorts,
			ScanMethod:  method,
			PortsPerSec: portsPerSec(totalPorts, duration),
		},
		Latency: duration,
	}
}

// portsPerSec is the scan throughput. The previous ScanRateMs did integer
// division of ports by elapsed milliseconds, so any scan slower than one port
// per millisecond reported a flat 0.
func portsPerSec(ports int, elapsed time.Duration) float64 {
	if elapsed <= 0 {
		return 0
	}
	return float64(ports) / elapsed.Seconds()
}

// ParsePortRange converts strings like "80,443,1000-1005" into a slice of
// integers. Malformed input is rejected rather than skipped: silently dropping
// "abc" from "abc,80" would scan one port and report success.
func ParsePortRange(portStr string) ([]int, error) {
	if strings.TrimSpace(portStr) == "" {
		return nil, fmt.Errorf("no ports specified")
	}

	var result []int
	seen := make(map[int]bool)

	for _, part := range strings.Split(portStr, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			return nil, fmt.Errorf("empty port entry in %q", portStr)
		}

		start, end, found := strings.Cut(part, "-")
		if !found {
			port, err := parsePort(part)
			if err != nil {
				return nil, err
			}
			if !seen[port] {
				seen[port] = true
				result = append(result, port)
			}
			continue
		}

		low, err := parsePort(start)
		if err != nil {
			return nil, err
		}
		high, err := parsePort(end)
		if err != nil {
			return nil, err
		}
		if low > high {
			low, high = high, low
		}

		for i := low; i <= high; i++ {
			if !seen[i] {
				seen[i] = true
				result = append(result, i)
			}
		}
	}

	return result, nil
}

func parsePort(s string) (int, error) {
	s = strings.TrimSpace(s)
	port, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("invalid port %q: not a number", s)
	}
	if port < 1 || port > 65535 {
		return 0, fmt.Errorf("invalid port %d: must be between 1 and 65535", port)
	}
	return port, nil
}
