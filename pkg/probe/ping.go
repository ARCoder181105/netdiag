package probe

import (
	"context"
	"fmt"
	"time"

	probing "github.com/prometheus-community/pro-bing"
)

// highLatencyThreshold is the average RTT above which a reachable host is
// still reported as degraded.
const highLatencyThreshold = 150 * time.Millisecond

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
	start := time.Now()

	resolvedIP, err := ResolveHost(p.Host, p.Timeout)
	if err != nil {
		return Result{
			Target:    p.Host,
			TimeStamp: time.Now(),
			ProbeType: "ping",
			Success:   false,
			Severity:  SeverityError,
			Message:   fmt.Sprintf("DNS Resolution Failed: %v", err),
			Latency:   time.Since(start),
		}, nil
	}

	stats, _, err := RunPinger(ctx, p.Host, func(pinger *probing.Pinger) {
		pinger.Count = p.Count
		pinger.Interval = p.Interval
		pinger.Timeout = p.Timeout
	})
	if err != nil {
		return Result{}, err
	}

	data := PingData{
		ResolvedIP:  resolvedIP,
		PacketsSent: stats.PacketsSent,
		PacketsRecv: stats.PacketsRecv,
		PacketLoss:  stats.PacketLoss,
		MinRTT:      stats.MinRtt,
		MaxRTT:      stats.MaxRtt,
		AvgRTT:      stats.AvgRtt,
		StdDevRTT:   stats.StdDevRtt,
	}

	success, severity, message := pingSeverity(stats.PacketLoss, stats.AvgRtt)

	// pro-bing computes PacketLoss as (sent-recv)/sent, so a run interrupted
	// before the first packet yields NaN, which compares false against every
	// threshold and would otherwise be classified as a healthy ping.
	if stats.PacketsSent == 0 {
		success, severity, message = false, SeverityError, "No packets were sent"
	}

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

// pingSeverity classifies a ping outcome. Kept as a pure function so the
// classification can be tested without sending packets.
func pingSeverity(loss float64, avgRTT time.Duration) (success bool, severity Severity, message string) {
	switch {
	case loss >= 100:
		// Total failure — host is unreachable.
		return false, SeverityError, "Host unreachable"

	case loss > 0 || avgRTT > highLatencyThreshold:
		// Partial loss OR high latency — degraded but alive.
		return true, SeverityWarning, fmt.Sprintf(
			"Degraded connectivity (loss: %.1f%%, avg: %s)",
			loss, avgRTT.Round(time.Millisecond),
		)

	default:
		return true, SeverityOK, fmt.Sprintf(
			"Ping successful (avg: %s, loss: 0%%)",
			avgRTT.Round(time.Millisecond),
		)
	}
}
