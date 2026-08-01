package probe

import (
	"context"
	"fmt"
	"net"
	"os"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

const tracePayload = "NETDIAG_TRACE"

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

	fail := func(msg string) (Result, error) {
		return Result{
			Target:    t.Host,
			TimeStamp: time.Now(),
			ProbeType: "trace",
			Success:   false,
			Severity:  SeverityError,
			Message:   msg,
			Latency:   time.Since(startTime),
		}, nil
	}

	// Graceful DNS failure
	destAddr, err := net.ResolveIPAddr("ip4", t.Host)
	if err != nil {
		return fail(fmt.Sprintf("DNS Resolution Failed: %v", err))
	}

	conn, err := net.ListenPacket("ip4:1", "0.0.0.0")
	if err != nil {
		return fail("Permission denied: Traceroute requires root/sudo privileges")
	}
	defer func() { _ = conn.Close() }()

	p := ipv4.NewPacketConn(conn)
	defer func() { _ = p.Close() }()

	icmpConn, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		return fail("Permission denied: Traceroute requires root/sudo privileges")
	}
	defer func() { _ = icmpConn.Close() }()

	// Identify our probes so replies to other pingers on this host are not
	// mistaken for ours. A single fixed ID collided with every other trace.
	id := os.Getpid() & 0xffff

	var (
		hops     []TraceHop
		reached  bool
		anyReply bool
	)

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
				ID:   id,
				Seq:  ttl,
				Data: []byte(tracePayload),
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

		peer, isReply, ok := readTraceReply(icmpConn, id, ttl, t.Timeout)
		rtt := time.Since(startHop)

		hop := TraceHop{HopNumber: ttl, RTT: rtt}

		if !ok {
			hop.Timeout = true
			hop.IP = "*"
			hops = append(hops, hop)
			continue
		}

		anyReply = true
		hop.IP = peer
		hop.HostName = resolveHostname(peer)
		hops = append(hops, hop)

		// Stop early if destination reached
		if isReply && peer == destAddr.String() {
			reached = true
			break
		}
	}

	severity, message := traceOutcome(reached, anyReply, len(hops))

	return Result{
		TimeStamp: time.Now(),
		ProbeType: "trace",
		Target:    t.Host,
		TraceData: &TraceData{Hops: hops},
		Message:   message,
		Severity:  severity,
		Success:   true,
		Latency:   time.Since(startTime),
	}, nil
}

// readTraceReply waits up to timeout for an ICMP reply belonging to this trace.
// Packets addressed to other processes are skipped rather than attributed to
// the current hop. It returns the responding peer, whether the reply was an
// echo reply (destination reached) and whether anything matched at all.
func readTraceReply(conn *icmp.PacketConn, id, seq int, timeout time.Duration) (peer string, isEchoReply, ok bool) {
	deadline := time.Now().Add(timeout)
	buf := make([]byte, 1500)

	for {
		if time.Now().After(deadline) {
			return "", false, false
		}
		if err := conn.SetReadDeadline(deadline); err != nil {
			return "", false, false
		}

		n, from, err := conn.ReadFrom(buf)
		if err != nil {
			return "", false, false
		}

		parsed, err := icmp.ParseMessage(ipv4.ICMPTypeEchoReply.Protocol(), buf[:n])
		if err != nil {
			continue
		}

		switch body := parsed.Body.(type) {
		case *icmp.Echo:
			// Our own echo reply from the destination.
			if parsed.Type == ipv4.ICMPTypeEchoReply && body.ID == id && body.Seq == seq {
				return from.String(), true, true
			}

		case *icmp.TimeExceeded:
			// An intermediate router quotes the IP header plus the first bytes
			// of our original packet; the echo we sent is inside it.
			if echoMatches(body.Data, id, seq) {
				return from.String(), false, true
			}
		}
		// Anything else belongs to another process. Keep reading.
	}
}

// echoMatches reports whether the quoted payload of an ICMP error carries the
// echo request we sent.
func echoMatches(quoted []byte, id, seq int) bool {
	header, err := ipv4.ParseHeader(quoted)
	if err != nil || len(quoted) < header.Len {
		return false
	}

	inner, err := icmp.ParseMessage(ipv4.ICMPTypeEchoReply.Protocol(), quoted[header.Len:])
	if err != nil {
		return false
	}

	echo, ok := inner.Body.(*icmp.Echo)
	return ok && echo.ID == id && echo.Seq == seq
}

// traceOutcome classifies a completed trace. Previously every trace reported
// SeverityOK, including one where no hop ever answered.
func traceOutcome(reached, anyReply bool, hopCount int) (Severity, string) {
	switch {
	case reached:
		return SeverityOK, fmt.Sprintf("Trace complete: destination reached in %d hops", hopCount)
	case !anyReply:
		return SeverityError, "No hops responded (ICMP may be filtered on this network)"
	default:
		return SeverityWarning, fmt.Sprintf("Destination not reached after %d hops", hopCount)
	}
}
