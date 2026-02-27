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

	sort.Ints(openPorts)

	duration := time.Since(startTime)
	ms := duration.Milliseconds()
	if ms == 0 {
		ms = 1
	}
	rate := int64(len(c.Ports)) / ms

	data := &ScanData{
		OpenPorts:  openPorts,
		TotalPorts: len(c.Ports),
		ScanMethod: "connect",
		ScanRateMs: rate,
	}

	message := fmt.Sprintf("Found %d open ports", len(openPorts))

	return Result{
		Target:    c.Host,
		TimeStamp: time.Now(),
		ProbeType: "scan",
		Success:   true,
		Severity:  SeverityOK,
		Message:   message,
		ScanData:  data,
	}, nil
}

// parsePortRange converts strings like "80,443,1000-1005" into a slice of integers
func ParsePortRange(portStr string) []int {
	var result []int
	parts := strings.Split(portStr, ",")

	for _, part := range parts {
		part = strings.TrimSpace(part)

		if strings.Contains(part, "-") {
			rangeParts := strings.Split(part, "-")
			if len(rangeParts) != 2 {
				continue
			}

			start, err1 := strconv.Atoi(strings.TrimSpace(rangeParts[0]))
			end, err2 := strconv.Atoi(strings.TrimSpace(rangeParts[1]))
			if err1 != nil || err2 != nil {
				continue
			}

			if start > end {
				start, end = end, start
			}

			if start < 1 || start > 65535 || end < 1 || end > 65535 {
				continue
			}

			for i := start; i <= end; i++ {
				result = append(result, i)
			}
		} else {
			num, err := strconv.Atoi(part)
			if err != nil {
				continue
			}
			if num < 1 || num > 65535 {
				continue
			}
			result = append(result, num)
		}
	}

	return result
}
