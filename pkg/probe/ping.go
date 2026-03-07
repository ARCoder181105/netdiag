package probe

import (
	"context"
	"fmt"
	"time"

	probing "github.com/prometheus-community/pro-bing"
)

type PingProber struct {
	Host     string
	Count    int
	Timeout  time.Duration
	Interval time.Duration
}

func (p *PingProber) Type() string {
	return "ping"
}

func (p *PingProber) Probe(ctx context.Context) (Result, error) {
	pinger, err := probing.NewPinger(p.Host)
	if err != nil {
		return Result{}, fmt.Errorf("failed to create pinger: %w", err)
	}

	pinger.Count = p.Count
	pinger.Interval = p.Interval
	pinger.Timeout = p.Timeout
	pinger.SetPrivileged(true)

	err = pinger.Resolve()
	if err != nil {
		return Result{
			Target:    p.Host,
			TimeStamp: time.Now(),
			ProbeType: "ping",
			Success:   false,
			Severity:  SeverityError,
			Message:   fmt.Sprintf("DNS Resolution Failed: %v", err),
		}, nil
	}

	err = pinger.RunWithContext(ctx)
	if err != nil {
		return Result{}, fmt.Errorf("ping failed: %w", err)
	}

	stats := pinger.Statistics()

	data := PingData{
		ResolvedIP:  pinger.IPAddr().String(),
		PacketsSent: stats.PacketsSent,
		PacketsRecv: stats.PacketsRecv,
		PacketLoss:  stats.PacketLoss,
		MinRTT:      stats.MinRtt,
		MaxRTT:      stats.MaxRtt,
		AvgRTT:      stats.AvgRtt,
		StdDevRTT:   stats.StdDevRtt,
	}

	// ── Severity logic ────────────────────────────────────────────────────────
	// Now properly emits SeverityWarning for degraded (but not fully down) hosts.
	// This matches what ping_test.go already asserts.
	var (
		success  bool
		severity Severity
		message  string
	)

	switch {
	case stats.PacketLoss == 100:
		// Total failure — host is unreachable
		success = false
		severity = SeverityError
		message = "Host unreachable"

	case stats.PacketLoss > 0 || stats.AvgRtt > 150*time.Millisecond:
		// Partial loss OR high latency — degraded but alive
		success = true
		severity = SeverityWarning
		message = fmt.Sprintf(
			"Degraded connectivity (loss: %.1f%%, avg: %s)",
			stats.PacketLoss,
			stats.AvgRtt.Round(time.Millisecond),
		)

	default:
		// All packets received, latency within threshold
		success = true
		severity = SeverityOK
		message = fmt.Sprintf(
			"Ping successful (avg: %s, loss: 0%%)",
			stats.AvgRtt.Round(time.Millisecond),
		)
	}
	// ─────────────────────────────────────────────────────────────────────────

	return Result{
		Target:    p.Host,
		TimeStamp: time.Now(),
		ProbeType: "ping",
		PingData:  &data,
		Message:   message,
		Severity:  severity,
		Success:   success,
		Latency:   stats.AvgRtt,
	}, nil
}
