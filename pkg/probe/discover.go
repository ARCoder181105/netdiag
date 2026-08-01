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

// maxSweepHosts caps how many addresses a discover run will probe. A /16 is
// 65k hosts, which is not a "scan my LAN" operation.
// ponytail: fixed cap; make it a --max-hosts flag if anyone needs a bigger sweep.
const maxSweepHosts = 1024

// discoverConcurrency bounds in-flight pings during a sweep.
const discoverConcurrency = 50

type DiscoverProber struct {
	Timeout time.Duration
}

func (d *DiscoverProber) Type() string {
	return "discover"
}

func (d *DiscoverProber) Probe(ctx context.Context) (Result, error) {
	start := time.Now()

	localIP, ipnet, err := localIPv4Network()
	if err != nil {
		return Result{
			TimeStamp: time.Now(),
			ProbeType: "discover",
			Target:    "local-network",
			Success:   false,
			Severity:  SeverityError,
			Message:   fmt.Sprintf("Failed to detect local network: %v", err),
			Latency:   time.Since(start),
		}, nil
	}

	hosts, truncated := hostAddresses(ipnet, localIP, maxSweepHosts)
	if len(hosts) == 0 {
		return Result{
			TimeStamp: time.Now(),
			ProbeType: "discover",
			Target:    ipnet.String(),
			Success:   false,
			Severity:  SeverityWarning,
			Message:   fmt.Sprintf("No scannable hosts in %s", ipnet),
			Latency:   time.Since(start),
		}, nil
	}

	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		devices []DiscoverDevice
	)
	sem := make(chan struct{}, discoverConcurrency)

	for _, targetIP := range hosts {
		wg.Add(1)

		go func(ip string) {
			defer wg.Done()

			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				return
			}
			defer func() { <-sem }()

			stats, _, err := RunPinger(ctx, ip, func(pinger *probing.Pinger) {
				pinger.Count = 1
				pinger.Timeout = d.Timeout
			})
			if err != nil || stats.PacketsRecv == 0 {
				return
			}

			device := DiscoverDevice{
				IP:       ip,
				HostName: resolveHostname(ip),
				Latency:  stats.AvgRtt,
			}

			mu.Lock()
			devices = append(devices, device)
			mu.Unlock()
		}(targetIP)
	}

	wg.Wait()

	sort.Slice(devices, func(i, j int) bool {
		return compareIPv4(devices[i].IP, devices[j].IP) < 0
	})

	data := &DiscoverData{
		LocalIP: localIP.String(),
		Prefix:  ipnet.String(),
		Devices: devices,
	}

	severity := SeverityOK
	message := fmt.Sprintf("Scan complete. Found %d devices in %s.", len(devices), ipnet)

	switch {
	case len(devices) == 0:
		severity = SeverityWarning
		message = fmt.Sprintf("No devices found in %s", ipnet)
	case truncated:
		severity = SeverityWarning
		message = fmt.Sprintf(
			"Found %d devices. %s is larger than %d addresses; only the first %d were scanned.",
			len(devices), ipnet, maxSweepHosts, maxSweepHosts,
		)
	}

	return Result{
		TimeStamp:    time.Now(),
		ProbeType:    "discover",
		Target:       ipnet.String(),
		DiscoverData: data,
		Success:      true,
		Severity:     severity,
		Message:      message,
		Latency:      time.Since(start),
	}, nil
}

// localIPv4Network returns this machine's primary IPv4 address and the network
// it belongs to.
//
// It asks the kernel which source address it would use to reach the internet,
// rather than taking the first address it finds. On a host running Docker,
// InterfaceAddrs also reports bridges like docker0 (172.17.0.1/16), and picking
// one of those would sweep an empty bridge network instead of the user's LAN.
func localIPv4Network() (net.IP, *net.IPNet, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil, nil, err
	}

	preferred := preferredIPv4()

	var fallback *net.IPNet
	var fallbackIP net.IP

	for _, address := range addrs {
		ipnet, ok := address.(*net.IPNet)
		if !ok {
			continue
		}

		ip := ipnet.IP.To4()
		if ip == nil || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
			continue
		}

		network := &net.IPNet{IP: ip.Mask(ipnet.Mask), Mask: ipnet.Mask}

		if preferred != nil && ip.Equal(preferred) {
			return ip, network, nil
		}

		if fallback == nil {
			fallbackIP, fallback = ip, network
		}
	}

	if fallback != nil {
		return fallbackIP, fallback, nil
	}

	return nil, nil, fmt.Errorf("no active local IPv4 address found")
}

// preferredIPv4 reports the source address the kernel would use for outbound
// traffic. The UDP "connection" is only a routing table lookup — no packets are
// sent — so this works offline and costs nothing. Returns nil if it cannot tell.
func preferredIPv4() net.IP {
	conn, err := net.Dial("udp4", "8.8.8.8:80")
	if err != nil {
		return nil
	}
	defer func() { _ = conn.Close() }()

	addr, ok := conn.LocalAddr().(*net.UDPAddr)
	if !ok {
		return nil
	}
	return addr.IP.To4()
}

// hostAddresses enumerates the usable host addresses of an IPv4 network,
// excluding the network address, the broadcast address and skip. It returns
// whether the list was truncated at limit.
func hostAddresses(ipnet *net.IPNet, skip net.IP, limit int) (hosts []string, truncated bool) {
	ones, bits := ipnet.Mask.Size()
	if bits != 32 {
		return nil, false
	}

	// /31 and /32 have no conventional host range; probe the address itself.
	if ones >= 31 {
		if ip := ipnet.IP.To4(); ip != nil && !ip.Equal(skip) {
			hosts = append(hosts, ip.String())
		}
		return hosts, false
	}

	network := binaryIPv4(ipnet.IP)
	broadcast := network | ^maskBits(ones)

	for addr := network + 1; addr < broadcast; addr++ {
		if len(hosts) >= limit {
			return hosts, true
		}
		ip := ipv4FromBinary(addr)
		if ip.Equal(skip) {
			continue
		}
		hosts = append(hosts, ip.String())
	}

	return hosts, false
}

func maskBits(ones int) uint32 {
	if ones == 0 {
		return 0
	}
	return ^uint32(0) << (32 - ones)
}

func binaryIPv4(ip net.IP) uint32 {
	v4 := ip.To4()
	if v4 == nil {
		return 0
	}
	return uint32(v4[0])<<24 | uint32(v4[1])<<16 | uint32(v4[2])<<8 | uint32(v4[3])
}

func ipv4FromBinary(addr uint32) net.IP {
	return net.IPv4(byte(addr>>24), byte(addr>>16), byte(addr>>8), byte(addr))
}

// compareIPv4 orders dotted-quad strings numerically rather than lexically, so
// .9 sorts before .10.
func compareIPv4(a, b string) int {
	ipA, ipB := binaryIPv4(net.ParseIP(a)), binaryIPv4(net.ParseIP(b))
	switch {
	case ipA < ipB:
		return -1
	case ipA > ipB:
		return 1
	default:
		return 0
	}
}

func resolveHostname(ip string) string {
	names, err := net.LookupAddr(ip)
	if err == nil && len(names) > 0 {
		return strings.TrimSuffix(names[0], ".")
	}
	return "(Unknown)"
}
