package probe

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"sync"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

// SYNScanner probes TCP ports with half-open (SYN) scanning: it sends a bare
// SYN and reads the reply without ever completing the handshake. A SYN-ACK
// means the port is open, an RST means it is closed, and silence means the
// port is filtered.
//
// This needs a raw socket, so it needs CAP_NET_RAW or root. When the socket is
// refused it falls back to Fallback, the ordinary connect scanner, exactly the
// way the ICMP paths fall back between raw and datagram sockets.
type SYNScanner struct {
	Host string
	// Timeout is how long a single port is given to answer.
	Timeout time.Duration
	// Fallback runs instead when the raw socket is refused. Required.
	Fallback *ConnectScanner
	// Notify receives a one-line diagnostic (the fallback notice). The caller
	// points this at stderr; a probe never writes to stdout itself. nil is
	// silent, which is what tests want.
	Notify func(string)
	Ports  []int
	// Concurrency is the ceiling on in-flight probes. The adaptive limiter
	// starts here and never exceeds it.
	Concurrency int
}

func (s *SYNScanner) Type() string {
	return "scan"
}

// portState is what we learned about a port. The zero value is stateFiltered,
// so a port nothing ever answered for is correctly reported without needing a
// separate "unanswered" bookkeeping pass.
type portState int

const (
	stateFiltered portState = iota
	stateOpen
	stateClosed
)

// tcpProtocolNumber is IPPROTO_TCP, which appears in the checksum pseudo-header.
const tcpProtocolNumber = 6

func (s *SYNScanner) Probe(ctx context.Context) (Result, error) {
	if s.Fallback == nil {
		return Result{}, errors.New("SYN scanner requires a fallback scanner")
	}

	dst, err := net.ResolveIPAddr("ip4", s.Host)
	if err != nil {
		return Result{}, fmt.Errorf("resolve %s: %w", s.Host, err)
	}

	// Ask the kernel which source address it would use for THIS target. Using
	// the address it would use for the internet instead would produce a
	// checksum over the wrong pseudo-header when scanning a host reached
	// through another interface.
	src := preferredIPv4(net.JoinHostPort(dst.IP.String(), "80"))
	if src == nil {
		return Result{}, fmt.Errorf("cannot determine a source address for %s", dst.IP)
	}

	conn, err := net.ListenPacket("ip4:tcp", "0.0.0.0")
	if err != nil {
		if isPrivilegeErr(err) {
			s.notify("raw socket denied: falling back to connect scan. " +
				"Grant the capability with: sudo setcap cap_net_raw+ep $(command -v netdiag)")
			return s.Fallback.Probe(ctx)
		}
		return Result{}, fmt.Errorf("open raw socket: %w", err)
	}
	defer func() { _ = conn.Close() }()

	start := time.Now()

	// One source port for the whole scan. The reply's own source port is the
	// scanned port, so (our port, sequence number) is all the correlation key
	// needs to be.
	srcPort := uint16(33000 + rand.Intn(20000))
	corr := newCorrelator(rand.Uint32(), srcPort, s.Ports)

	// Buffered so the receiver never blocks handing the sender a wake-up.
	replies := make(chan struct{}, 1)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		receiveReplies(conn, dst.IP, corr, replies)
	}()

	sendErr := s.sendAll(ctx, conn, src, dst.IP, corr, replies)

	// Closing the socket is what unblocks the receiver's ReadFrom. Without it
	// a canceled 65k-port scan would sit here until the last read timed out.
	_ = conn.Close()
	wg.Wait()

	if sendErr != nil {
		return Result{}, sendErr
	}

	return scanResult(s.Host, len(s.Ports), corr.openPorts(), "syn", time.Since(start)), nil
}

func (s *SYNScanner) notify(msg string) {
	if s.Notify != nil {
		s.Notify(msg)
	}
}

// inflight is one sent, not-yet-answered probe.
type inflight struct {
	deadline time.Time
	port     int
}

// sendAll paces the SYN packets, then waits out the probes still in flight.
func (s *SYNScanner) sendAll(ctx context.Context, conn net.PacketConn, src, dst net.IP, corr *correlator, replies <-chan struct{}) error {
	limiter := newAIMD(s.Concurrency)
	queue := make([]inflight, 0, min(len(s.Ports), 4096))
	head := 0

	// retire releases every probe at the head of the queue that has been
	// answered or has run out of time, and reports the outcome to the limiter.
	// Deadlines are monotonic because every probe gets the same timeout, so the
	// head of the queue is always the next probe to expire.
	retire := func(now time.Time) {
		for head < len(queue) {
			p := queue[head]
			switch {
			case corr.answered(p.port):
				limiter.completed(false)
			case now.After(p.deadline):
				limiter.completed(true)
			default:
				return
			}
			head++
		}
	}

	for _, port := range s.Ports {
		for {
			if err := ctx.Err(); err != nil {
				return nil // a canceled scan reports what it has, like connect scan
			}
			retire(time.Now())
			if len(queue)-head < limiter.limit() {
				break
			}
			waitForRoom(ctx, replies, queue[head].deadline)
		}

		packet, err := buildSYN(src, dst, corr.srcPort, uint16(port), corr.seqFor(port))
		if err != nil {
			return fmt.Errorf("build SYN for port %d: %w", port, err)
		}
		if _, err := conn.WriteTo(packet, &net.IPAddr{IP: dst}); err != nil {
			return fmt.Errorf("send SYN to %s:%d: %w", dst, port, err)
		}
		queue = append(queue, inflight{port: port, deadline: time.Now().Add(s.Timeout)})
	}

	// Everything is sent; wait for the stragglers.
	for head < len(queue) {
		if ctx.Err() != nil {
			return nil
		}
		retire(time.Now())
		if head < len(queue) {
			waitForRoom(ctx, replies, queue[head].deadline)
		}
	}
	return nil
}

// aimd is additive-increase, multiplicative-decrease pacing for in-flight
// probes. Every window of completed probes either had a timeout in it, in
// which case the network is being pushed too hard and the limit halves, or was
// clean, in which case the limit creeps up by one.
//
// A dropped SYN is indistinguishable from a filtered port, so a scan of mostly
// filtered ports will back off even on a healthy network. That is the right
// trade: it is the same evidence the network would give under real congestion.
type aimd struct {
	max     int
	current int
	done    int
	lost    int
	mu      sync.Mutex
}

// aimdWindow is how many completed probes a pacing decision is made over.
const aimdWindow = 64

func newAIMD(maxInFlight int) *aimd {
	if maxInFlight < 1 {
		maxInFlight = 1
	}
	return &aimd{max: maxInFlight, current: maxInFlight}
}

func (a *aimd) limit() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.current
}

// completed records the outcome of one probe and adjusts the limit at the end
// of each window.
func (a *aimd) completed(timedOut bool) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.done++
	if timedOut {
		a.lost++
	}
	if a.done < aimdWindow {
		return
	}

	switch {
	case a.lost > 0:
		a.current = max(1, a.current/2)
	case a.current < a.max:
		a.current++
	}
	a.done, a.lost = 0, 0
}

// waitForRoom blocks until a reply arrives, the oldest probe expires, or the
// scan is canceled.
func waitForRoom(ctx context.Context, replies <-chan struct{}, deadline time.Time) {
	timer := time.NewTimer(time.Until(deadline))
	defer timer.Stop()
	select {
	case <-replies:
	case <-timer.C:
	case <-ctx.Done():
	}
}

// receiveReplies reads TCP segments off the raw socket until it is closed,
// classifying the ones that belong to this scan. Go strips the IPv4 header on
// ip4: sockets, so what arrives here starts at the TCP header.
func receiveReplies(conn net.PacketConn, dst net.IP, corr *correlator, replies chan<- struct{}) {
	buf := make([]byte, 1500)
	var tcp layers.TCP

	for {
		n, addr, err := conn.ReadFrom(buf)
		if err != nil {
			return // the socket was closed: the scan is over
		}

		ipAddr, ok := addr.(*net.IPAddr)
		if !ok || !ipAddr.IP.Equal(dst) {
			continue
		}
		if err := tcp.DecodeFromBytes(buf[:n], gopacket.NilDecodeFeedback); err != nil {
			continue
		}
		if !corr.resolve(&tcp) {
			continue
		}

		select {
		case replies <- struct{}{}:
		default:
		}
	}
}

// correlator maps replies back to the probes that caused them. Matching on
// arrival order would be wrong: replies come back out of order, and any other
// TCP traffic on the box lands on this raw socket too.
type correlator struct {
	state   map[int]portState
	seqBase uint32
	srcPort uint16
	mu      sync.Mutex
}

func newCorrelator(seqBase uint32, srcPort uint16, ports []int) *correlator {
	state := make(map[int]portState, len(ports))
	for _, p := range ports {
		state[p] = stateFiltered
	}
	return &correlator{seqBase: seqBase, srcPort: srcPort, state: state}
}

// seqFor is the sequence number sent to a port. Deriving it from the port
// means a reply carries its own proof of which probe it answers: a SYN-ACK
// acknowledges exactly seqFor(port)+1.
func (c *correlator) seqFor(port int) uint32 {
	return c.seqBase + uint32(port)
}

// resolve records a reply and reports whether it belonged to this scan.
func (c *correlator) resolve(t *layers.TCP) bool {
	if uint16(t.DstPort) != c.srcPort {
		return false
	}

	state, ok := classifySYNResponse(t)
	if !ok {
		return false
	}

	port := int(t.SrcPort)

	// An acknowledgement that does not acknowledge our SYN is somebody else's
	// packet that happened to reach the same local port.
	if t.ACK && t.Ack != c.seqFor(port)+1 {
		return false
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	current, tracked := c.state[port]
	if !tracked || current != stateFiltered {
		return false // not a port we scanned, or already decided
	}
	c.state[port] = state
	return true
}

func (c *correlator) answered(port int) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.state[port] != stateFiltered
}

func (c *correlator) openPorts() []int {
	c.mu.Lock()
	defer c.mu.Unlock()

	var open []int
	for port, state := range c.state {
		if state == stateOpen {
			open = append(open, port)
		}
	}
	return open
}

// classifySYNResponse maps a reply to a port state. The bool reports whether
// this was a reply to a SYN at all; anything else (a stray ACK, a FIN from an
// unrelated connection) is not evidence about the port.
func classifySYNResponse(t *layers.TCP) (portState, bool) {
	switch {
	case t.SYN && t.ACK:
		return stateOpen, true
	case t.RST:
		return stateClosed, true
	default:
		return stateFiltered, false
	}
}

// buildSYN serializes one SYN segment. gopacket lays out the header; the
// checksum is computed here rather than by gopacket's ComputeChecksums option,
// which would hide the pseudo-header arithmetic.
func buildSYN(src, dst net.IP, srcPort, dstPort uint16, seq uint32) ([]byte, error) {
	tcp := &layers.TCP{
		SrcPort:    layers.TCPPort(srcPort),
		DstPort:    layers.TCPPort(dstPort),
		Seq:        seq,
		SYN:        true,
		Window:     1024,
		DataOffset: 5, // 20 bytes, no options
	}

	buf := gopacket.NewSerializeBuffer()
	// Both options off on purpose: FixLengths would recompute DataOffset and
	// ComputeChecksums would write its own checksum over ours.
	if err := tcp.SerializeTo(buf, gopacket.SerializeOptions{}); err != nil {
		return nil, err
	}

	segment := buf.Bytes()
	// The checksum field must be zero while the checksum is computed; gopacket
	// wrote tcp.Checksum, which is still its zero value here.
	binary.BigEndian.PutUint16(segment[16:18], tcpChecksum(src, dst, segment))
	return segment, nil
}

// tcpChecksum computes the TCP checksum of a segment as defined by RFC 793.
//
// The checksum covers a 12-byte IPv4 pseudo-header that is never transmitted —
// source address, destination address, a zero byte, the protocol number, and
// the TCP length — followed by the segment itself. Including the addresses is
// what makes a segment delivered to the wrong host detectable. The segment's
// own checksum field must be zero on the way in.
func tcpChecksum(src, dst net.IP, segment []byte) uint16 {
	var pseudo [12]byte
	copy(pseudo[0:4], src.To4())
	copy(pseudo[4:8], dst.To4())
	pseudo[8] = 0
	pseudo[9] = tcpProtocolNumber
	binary.BigEndian.PutUint16(pseudo[10:12], uint16(len(segment)))

	return internetChecksum(pseudo[:], segment)
}

// internetChecksum is the 16-bit ones-complement checksum of RFC 1071: sum the
// data as big-endian 16-bit words, fold the carries back in, and complement.
//
// Each chunk is padded independently, so only the final chunk may have an odd
// length.
func internetChecksum(chunks ...[]byte) uint16 {
	var sum uint32

	for _, chunk := range chunks {
		i := 0
		for ; i+1 < len(chunk); i += 2 {
			sum += uint32(chunk[i])<<8 | uint32(chunk[i+1])
		}
		if i < len(chunk) {
			// An odd trailing byte is padded with a zero byte on the right.
			sum += uint32(chunk[i]) << 8
		}
	}

	// Fold the carries out of the high half until none are left.
	for sum>>16 != 0 {
		sum = sum&0xffff + sum>>16
	}

	return ^uint16(sum)
}
