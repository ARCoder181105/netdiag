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

	// Run the pinger synchronously
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
	var (
		success  = stats.PacketsRecv > 0
		severity = SeverityError
		message  = "Host unreachable"
	)

	if success {
		severity = SeverityOK
		message = "Ping successful"
	}

	result := Result{
		Target:    p.Host,
		TimeStamp: time.Now(),
		ProbeType: "ping",
		PingData:  &data,
		Message:   message,
		Severity:  severity,
		Success:   success,
		Latency:   stats.AvgRtt,
	}

	return result, nil
}
