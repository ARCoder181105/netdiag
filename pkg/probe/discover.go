package probe

import (
	"context"
	"fmt"
	"net"
	"sort"
	"strings"
	"sync"
	"time"

	probing "github.com/prometheus-community/pro-bing"
)

type DiscoverProber struct {
	Timeout time.Duration
}

func (d *DiscoverProber) Type() string {
	return "discover"
}

func (d *DiscoverProber) Probe(ctx context.Context) (Result, error) {

	start := time.Now()

	localIP, prefix, err := getLocalIPPrefix()
	if err != nil {
		return Result{
			TimeStamp: time.Now(),
			ProbeType: "discover",
			Target:    "local-network",
			Success:   false,
			Severity:  SeverityError,
			Message:   fmt.Sprintf("Failed to detect local network: %v", err),
		}, nil
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	sem := make(chan struct{}, 50)

	var devices []DiscoverDevice

	for i := 1; i < 255; i++ {

		targetIP := fmt.Sprintf("%s.%d", prefix, i)

		if targetIP == localIP {
			continue
		}

		wg.Add(1)
		go func(ip string) {
			defer wg.Done()

			if ctx.Err() != nil {
				return
			}

			sem <- struct{}{}
			defer func() { <-sem }()

			pinger, err := probing.NewPinger(ip)
			if err != nil {
				return
			}

			pinger.Count = 1
			pinger.Timeout = d.Timeout
			pinger.SetPrivileged(true)

			if err := pinger.RunWithContext(ctx); err != nil {
				return
			}

			stats := pinger.Statistics()
			if stats.PacketsRecv > 0 {

				host := resolveHostname(ip)

				mu.Lock()
				devices = append(devices, DiscoverDevice{
					IP:       ip,
					HostName: host,
					Latency:  stats.AvgRtt,
				})
				mu.Unlock()
			}

		}(targetIP)
	}

	wg.Wait()

	sort.Slice(devices, func(i, j int) bool {
		return devices[i].IP < devices[j].IP
	})

	data := &DiscoverData{
		LocalIP: localIP,
		Prefix:  prefix,
		Devices: devices,
	}

	severity := SeverityOK
	message := fmt.Sprintf("Scan complete. Found %d devices.", len(devices))

	if len(devices) == 0 {
		severity = SeverityWarning
		message = "No devices found"
	}

	return Result{
		TimeStamp:    time.Now(),
		ProbeType:    "discover",
		Target:       prefix,
		DiscoverData: data,
		Success:      true,
		Severity:     severity,
		Message:      message,
		Latency:      time.Since(start),
	}, nil
}

func getLocalIPPrefix() (string, string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", "", err
	}

	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				ip := ipnet.IP.String()

				if strings.HasPrefix(ip, "169.254") {
					continue
				}

				lastDot := strings.LastIndex(ip, ".")
				if lastDot != -1 {
					return ip, ip[:lastDot], nil
				}
			}
		}
	}

	return "", "", fmt.Errorf("no active local IP found")
}

func resolveHostname(ip string) string {
	names, err := net.LookupAddr(ip)
	if err == nil && len(names) > 0 {
		return strings.TrimSuffix(names[0], ".")
	}
	return "(Unknown)"
}