package probe

import (
	"context"
	"errors"
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
		hops        []TraceHop
		reached     bool
		anyReply    bool
		unreachable string
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

		reply, ok, readErr := readTraceReply(ctx, icmpConn, id, ttl, t.Timeout)
		rtt := time.Since(startHop)

		if readErr != nil {
			// The socket itself failed (or the trace was canceled) rather than
			// simply timing out. Treating this as a normal timeout would render
			// every remaining hop as "*" even though nothing was ever sent.
			break
		}

		hop := TraceHop{HopNumber: ttl}

		if !ok {
			// Leave RTT zero: nothing was measured, and the full timeout would
			// otherwise be reported as this hop's latency in --json output.
			hop.Timeout = true
			hop.IP = "*"
			hops = append(hops, hop)
			continue
		}

		anyReply = true
		hop.RTT = rtt
		hop.IP = reply.Peer
		hop.HostName = resolveHostname(reply.Peer)
		hops = append(hops, hop)

		// Stop early if destination reached
		if reply.EchoReply && reply.Peer == destAddr.String() {
			reached = true
			break
		}

		// A router told us it will not forward any further. Continuing would
		// just burn MaxHops timeouts against a path that has already ended.
		if reply.Unreachable != "" {
			unreachable = reply.Unreachable
			break
		}
	}

	severity, message := traceOutcome(reached, anyReply, len(hops), unreachable)

	return Result{
		TimeStamp: time.Now(),
		ProbeType: "trace",
		Target:    t.Host,
		TraceData: &TraceData{Hops: hops},
		Message:   message,
		Severity:  severity,
		Success:   severity != SeverityError,
		Latency:   time.Since(startTime),
	}, nil
}

// traceReply is one ICMP response matched to one of our probes.
type traceReply struct {
	Peer string
	// EchoReply is set when the destination itself answered.
	EchoReply bool
	// Unreachable holds the reason when a router refused to forward further.
	Unreachable string
}

// readTraceReply waits up to timeout for an ICMP reply belonging to this trace,
// clamped to ctx's deadline if it is sooner. Packets addressed to other
// processes are skipped rather than attributed to the current hop. ok reports
// whether anything matched at all; a non-nil err means the read failed for a
// reason other than the deadline expiring (socket error, cancellation) and the
// TTL loop should stop rather than record a timeout hop.
func readTraceReply(ctx context.Context, conn *icmp.PacketConn, id, seq int, timeout time.Duration) (reply traceReply, ok bool, err error) {
	deadline := time.Now().Add(timeout)
	if d, hasDeadline := ctx.Deadline(); hasDeadline && d.Before(deadline) {
		deadline = d
	}
	buf := make([]byte, 1500)

	for {
		if err := ctx.Err(); err != nil {
			return traceReply{}, false, err
		}
		if time.Now().After(deadline) {
			return traceReply{}, false, nil
		}
		if err := conn.SetReadDeadline(deadline); err != nil {
			return traceReply{}, false, err
		}

		n, from, readErr := conn.ReadFrom(buf)
		if readErr != nil {
			if errors.Is(readErr, os.ErrDeadlineExceeded) {
				return traceReply{}, false, nil
			}
			return traceReply{}, false, readErr
		}

		parsed, parseErr := icmp.ParseMessage(ipv4.ICMPTypeEchoReply.Protocol(), buf[:n])
		if parseErr != nil {
			continue
		}

		switch body := parsed.Body.(type) {
		case *icmp.Echo:
			// Our own echo reply from the destination.
			if parsed.Type == ipv4.ICMPTypeEchoReply && body.ID == id && body.Seq == seq {
				return traceReply{Peer: from.String(), EchoReply: true}, true, nil
			}

		case *icmp.TimeExceeded:
			// An intermediate router quotes the IP header plus the first bytes
			// of our original packet; the echo we sent is inside it.
			if echoMatches(body.Data, id, seq) {
				return traceReply{Peer: from.String()}, true, nil
			}

		case *icmp.DstUnreach:
			// Same quoting rule as TimeExceeded, but the path ends here.
			if echoMatches(body.Data, id, seq) {
				return traceReply{
					Peer:        from.String(),
					Unreachable: unreachReason(parsed.Code),
				}, true, nil
			}
		}
		// Anything else belongs to another process. Keep reading.
	}
}

// unreachReason names an ICMP destination-unreachable code (RFC 792).
func unreachReason(code int) string {
	switch code {
	case 0:
		return "network unreachable"
	case 1:
		return "host unreachable"
	case 2:
		return "protocol unreachable"
	case 3:
		return "port unreachable"
	case 9, 10, 13:
		return "administratively prohibited"
	default:
		return fmt.Sprintf("destination unreachable (code %d)", code)
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
func traceOutcome(reached, anyReply bool, hopCount int, unreachable string) (Severity, string) {
	switch {
	case reached:
		return SeverityOK, fmt.Sprintf("Trace complete: destination reached in %d hops", hopCount)
	case unreachable != "":
		return SeverityWarning, fmt.Sprintf("Trace stopped at hop %d: %s", hopCount, unreachable)
	case !anyReply:
		return SeverityError, "No hops responded (ICMP may be filtered on this network)"
	default:
		return SeverityWarning, fmt.Sprintf("Destination not reached after %d hops", hopCount)
	}
}
