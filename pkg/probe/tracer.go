package probe

import (
	"context"
	"fmt"
	"net"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

type TraceProber struct {
	Host    string
	MaxHops int
	Timeout time.Duration
}

func (t *TraceProber) Type() string {
	return "trace"
}

func (t *TraceProber) Probe(ctx context.Context) (Result, error) {

	startTime := time.Now()

	// Graceful DNS failure
	destAddr, err := net.ResolveIPAddr("ip4", t.Host)
	if err != nil {
		return Result{
			Target:    t.Host,
			TimeStamp: time.Now(),
			ProbeType: "trace",
			Success:   false,
			Severity:  SeverityError,
			Message:   fmt.Sprintf("DNS Resolution Failed: %v", err),
		}, nil
	}

	conn, err := net.ListenPacket("ip4:1", "0.0.0.0")
	if err != nil {
		return Result{
			Target:    t.Host,
			TimeStamp: time.Now(),
			ProbeType: "trace",
			Success:   false,
			Severity:  SeverityError,
			Message:   "Permission denied: Traceroute requires root/sudo privileges",
		}, nil
	}
	defer func() { _ = conn.Close() }()

	p := ipv4.NewPacketConn(conn)
	defer func() { _ = p.Close() }()

	icmpConn, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		return Result{
			Target:    t.Host,
			TimeStamp: time.Now(),
			ProbeType: "trace",
			Success:   false,
			Severity:  SeverityError,
			Message:   "Permission denied: Traceroute requires root/sudo privileges",
		}, nil
	}
	defer func() { _ = icmpConn.Close() }()

	var hops []TraceHop

	for ttl := 1; ttl <= t.MaxHops; ttl++ {

		// Graceful context cancellation → return partial trace
		if ctx.Err() != nil {
			break
		}

		if err := p.SetTTL(ttl); err != nil {
			break
		}

		msg := icmp.Message{
			Type: ipv4.ICMPTypeEcho,
			Code: 0,
			Body: &icmp.Echo{
				ID:   1234,
				Seq:  ttl,
				Data: []byte("NETDIAG_TRACE"),
			},
		}

		msgBytes, err := msg.Marshal(nil)
		if err != nil {
			break
		}

		startHop := time.Now()

		if _, writeErr := p.WriteTo(msgBytes, nil, destAddr); writeErr != nil {
			break
		}

		reply := make([]byte, 1500)
		_ = icmpConn.SetReadDeadline(time.Now().Add(t.Timeout))

		n, peer, err := icmpConn.ReadFrom(reply)
		rtt := time.Since(startHop)

		hop := TraceHop{
			HopNumber: ttl,
			RTT:       rtt,
		}

		// Timeout case
		if err != nil {
			hop.Timeout = true
			hop.IP = "*"
			hops = append(hops, hop)
			continue
		}

		parsedMsg, err := icmp.ParseMessage(1, reply[:n])
		if err != nil {
			hop.Timeout = true
			hop.IP = "*"
			hops = append(hops, hop)
			continue
		}

		ipAddr := peer.String()
		hop.IP = ipAddr

		names, err := net.LookupAddr(ipAddr)
		if err == nil && len(names) > 0 {
			hop.HostName = names[0]
		}

		hops = append(hops, hop)

		// Stop early if destination reached
		if parsedMsg.Type == ipv4.ICMPTypeEchoReply &&
			ipAddr == destAddr.String() {
			break
		}
	}

	traceData := &TraceData{
		Hops: hops,
	}

	return Result{
		TimeStamp: time.Now(),
		ProbeType: "trace",
		Target:    t.Host,
		TraceData: traceData,
		Message:   "Trace complete",
		Severity:  SeverityOK,
		Success:   true,
		Latency:   time.Since(startTime),
	}, nil
}
