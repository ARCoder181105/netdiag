package probe

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"net"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
)

// A --fast scan and a connect scan must be indistinguishable to a JSON
// consumer apart from scan_method. Anything else breaks `jq` pipelines that
// were written against the connect scan.
//
// This pins the shared builder both scanners call. It cannot catch a scanner
// that stops calling it — TestSYNScanMatchesConnectScanSchema compares the
// output of the two real Probe methods, but needs CAP_NET_RAW to run.
func TestScanJSONSchemaIsMethodIndependent(t *testing.T) {
	connect := scanResult("example.com", 3, []int{22, 80}, "connect", 250*time.Millisecond)
	syn := scanResult("example.com", 3, []int{22, 80}, "syn", 12*time.Millisecond)

	connectKeys := jsonKeys(t, connect)
	synKeys := jsonKeys(t, syn)

	if !reflect.DeepEqual(connectKeys, synKeys) {
		t.Fatalf("JSON key sets differ:\n connect: %v\n syn:     %v", connectKeys, synKeys)
	}

	if connect.ScanData.ScanMethod != "connect" || syn.ScanData.ScanMethod != "syn" {
		t.Fatalf("scan_method not carried through: connect=%q syn=%q",
			connect.ScanData.ScanMethod, syn.ScanData.ScanMethod)
	}

	// The severity contract is part of the schema's meaning: a scan that found
	// nothing is a warning, not an error, so it still exits 0.
	empty := scanResult("example.com", 3, nil, "syn", time.Millisecond)
	if empty.Severity != SeverityWarning || !empty.Success {
		t.Errorf("empty SYN scan = severity %v success %v, want Warning/true",
			empty.Severity, empty.Success)
	}
}

// jsonKeys returns every key in the marshaled Result, including the keys of
// the nested scan_data object, as a sorted "path" list.
func jsonKeys(t *testing.T, r Result) []string {
	t.Helper()

	raw, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	var keys []string
	for k, v := range decoded {
		keys = append(keys, k)
		if nested, ok := v.(map[string]any); ok {
			for nk := range nested {
				keys = append(keys, k+"."+nk)
			}
		}
	}
	sort.Strings(keys)
	return keys
}

// The worked example from RFC 1071 section 3. This pins the accumulate-and-fold
// loop against a published vector, independent of anything in this package.
func TestInternetChecksumRFC1071(t *testing.T) {
	data := []byte{0x00, 0x01, 0xf2, 0x03, 0xf4, 0xf5, 0xf6, 0xf7}

	if got := internetChecksum(data); got != 0x220d {
		t.Errorf("internetChecksum(RFC 1071 example) = %#04x, want 0x220d", got)
	}
}

// A segment of odd length is padded with a trailing zero byte, so the last
// byte lands in the high half of its word. Dropping it instead would still
// produce a plausible-looking checksum.
func TestInternetChecksumPadsOddLength(t *testing.T) {
	odd := internetChecksum([]byte{0xab})
	padded := internetChecksum([]byte{0xab, 0x00})

	if odd != padded {
		t.Errorf("odd-length checksum %#04x != zero-padded %#04x", odd, padded)
	}
	if odd == internetChecksum([]byte{0x00, 0xab}) {
		t.Error("the trailing byte was placed in the low half of the word, not the high half")
	}
}

// The known-good vector. The expected value was produced by an independent
// implementation in Python (struct.unpack over the pseudo-header plus segment),
// not by running the code under test, so it is evidence rather than a snapshot.
func TestTCPChecksumKnownVector(t *testing.T) {
	segment := synSegment(t, 0)

	got := tcpChecksum(net.ParseIP("192.168.1.10"), net.ParseIP("192.168.1.1"), segment)
	if got != 0x0f9f {
		t.Errorf("tcpChecksum = %#04x, want 0x0f9f", got)
	}
}

// Checksumming a segment that already carries its own checksum must come out
// to zero. This catches pseudo-header field-order and length mistakes without
// needing any magic number.
func TestTCPChecksumOfCompleteSegmentIsZero(t *testing.T) {
	src, dst := net.ParseIP("192.168.1.10"), net.ParseIP("192.168.1.1")
	segment := synSegment(t, 0)
	binary.BigEndian.PutUint16(segment[16:18], tcpChecksum(src, dst, segment))

	// Summing a segment that carries its own checksum gives all ones, whose
	// complement is zero. This is how a receiver verifies a segment.
	if got := tcpChecksum(src, dst, segment); got != 0x0000 {
		t.Errorf("checksum over a complete segment = %#04x, want 0x0000", got)
	}
}

// The addresses are not transmitted inside the TCP header, so a checksum that
// silently ignored the pseudo-header would pass every test above. Changing only
// the destination address must change the result.
func TestTCPChecksumCoversThePseudoHeader(t *testing.T) {
	segment := synSegment(t, 0)
	src := net.ParseIP("192.168.1.10")

	base := tcpChecksum(src, net.ParseIP("192.168.1.1"), segment)

	if other := tcpChecksum(src, net.ParseIP("192.168.1.2"), segment); other == base {
		t.Error("changing the destination address did not change the checksum")
	}
	if other := tcpChecksum(net.ParseIP("10.0.0.1"), net.ParseIP("192.168.1.1"), segment); other == base {
		t.Error("changing the source address did not change the checksum")
	}
}

// buildSYN must emit a 20-byte header whose checksum validates, and must not
// let gopacket overwrite it.
func TestBuildSYNWritesAValidChecksum(t *testing.T) {
	src, dst := net.ParseIP("192.168.1.10"), net.ParseIP("192.168.1.1")

	packet, err := buildSYN(src, dst, 54321, 80, 0x11223344)
	if err != nil {
		t.Fatalf("buildSYN: %v", err)
	}
	if len(packet) != 20 {
		t.Fatalf("packet length = %d, want 20", len(packet))
	}
	if got := binary.BigEndian.Uint16(packet[16:18]); got != 0x0f9f {
		t.Errorf("serialized checksum = %#04x, want 0x0f9f", got)
	}

	var tcp layers.TCP
	if err := tcp.DecodeFromBytes(packet, gopacket.NilDecodeFeedback); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !tcp.SYN || tcp.ACK || tcp.RST {
		t.Errorf("flags = SYN:%v ACK:%v RST:%v, want SYN only", tcp.SYN, tcp.ACK, tcp.RST)
	}
	if tcp.Seq != 0x11223344 || uint16(tcp.SrcPort) != 54321 || uint16(tcp.DstPort) != 80 {
		t.Errorf("header = sport %d dport %d seq %#08x", tcp.SrcPort, tcp.DstPort, tcp.Seq)
	}
}

// synSegment is the reference segment shared by the checksum vectors:
// 54321 -> 80, seq 0x11223344, window 1024, SYN, checksum field zero.
func synSegment(t *testing.T, checksum uint16) []byte {
	t.Helper()

	segment := make([]byte, 20)
	binary.BigEndian.PutUint16(segment[0:2], 54321)
	binary.BigEndian.PutUint16(segment[2:4], 80)
	binary.BigEndian.PutUint32(segment[4:8], 0x11223344)
	segment[12] = 5 << 4 // data offset, no options
	segment[13] = 0x02   // SYN
	binary.BigEndian.PutUint16(segment[14:16], 1024)
	binary.BigEndian.PutUint16(segment[16:18], checksum)
	return segment
}

func TestClassifySYNResponse(t *testing.T) {
	tests := []struct {
		name  string
		flags layers.TCP
		want  portState
		ok    bool
	}{
		{"SYN-ACK is open", layers.TCP{SYN: true, ACK: true}, stateOpen, true},
		{"RST is closed", layers.TCP{RST: true}, stateClosed, true},
		{"RST-ACK is closed", layers.TCP{RST: true, ACK: true}, stateClosed, true},
		{"a bare SYN is our own outbound packet", layers.TCP{SYN: true}, stateFiltered, false},
		{"a bare ACK says nothing", layers.TCP{ACK: true}, stateFiltered, false},
		{"a FIN belongs to another connection", layers.TCP{FIN: true, ACK: true}, stateFiltered, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := classifySYNResponse(&tt.flags)
			if got != tt.want || ok != tt.ok {
				t.Errorf("classifySYNResponse() = (%v, %v), want (%v, %v)", got, ok, tt.want, tt.ok)
			}
		})
	}
}

// A raw socket receives every TCP segment on the machine. Correlation is the
// only thing separating this scan's replies from unrelated traffic, so each
// rejection below is a real misclassification that would otherwise happen.
func TestCorrelatorRejectsForeignReplies(t *testing.T) {
	const (
		ourPort = 45678
		scanned = 80
		seqBase = 1000
	)
	corr := newCorrelator(seqBase, ourPort, []int{scanned})
	goodAck := corr.seqFor(scanned) + 1

	rejected := []struct {
		name  string
		reply layers.TCP
	}{
		{"addressed to a different local port", layers.TCP{
			SrcPort: scanned, DstPort: ourPort + 1, SYN: true, ACK: true, Ack: goodAck,
		}},
		{"acknowledges a sequence number we never sent", layers.TCP{
			SrcPort: scanned, DstPort: ourPort, SYN: true, ACK: true, Ack: goodAck + 7,
		}},
		{"comes from a port this scan never probed", layers.TCP{
			SrcPort: 443, DstPort: ourPort, SYN: true, ACK: true, Ack: corr.seqFor(443) + 1,
		}},
		{"is our own outbound SYN looped back", layers.TCP{
			SrcPort: ourPort, DstPort: scanned, SYN: true, Seq: corr.seqFor(scanned),
		}},
	}

	for _, tt := range rejected {
		t.Run(tt.name, func(t *testing.T) {
			reply := tt.reply
			if corr.resolve(&reply) {
				t.Error("accepted a reply that does not belong to this scan")
			}
			if corr.answered(scanned) {
				t.Fatal("a foreign reply resolved the scanned port")
			}
		})
	}

	// The matching reply is accepted, once.
	good := layers.TCP{SrcPort: scanned, DstPort: ourPort, SYN: true, ACK: true, Ack: goodAck}
	if !corr.resolve(&good) {
		t.Fatal("rejected the reply to our own SYN")
	}
	if !corr.answered(scanned) || !reflect.DeepEqual(corr.openPorts(), []int{scanned}) {
		t.Fatalf("open ports = %v, want [%d]", corr.openPorts(), scanned)
	}

	// A second reply for a decided port must not flip it: an open port that is
	// later RST by our own kernel is still open.
	late := layers.TCP{SrcPort: scanned, DstPort: ourPort, RST: true, ACK: true, Ack: goodAck}
	if corr.resolve(&late) {
		t.Error("a late RST overwrote a decided port")
	}
	if !reflect.DeepEqual(corr.openPorts(), []int{scanned}) {
		t.Errorf("open ports after a late RST = %v, want [%d]", corr.openPorts(), scanned)
	}
}

// An unanswered port is filtered, not open. The zero value carries this, which
// is easy to break by switching the iota order.
func TestCorrelatorUnansweredPortsAreFiltered(t *testing.T) {
	corr := newCorrelator(1000, 45678, []int{22, 80, 443})

	if got := corr.openPorts(); got != nil {
		t.Errorf("open ports before any reply = %v, want none", got)
	}
	if corr.answered(22) {
		t.Error("an unprobed port reported itself answered")
	}
}

func TestAIMDHalvesOnLossAndCreepsBackUp(t *testing.T) {
	limiter := newAIMD(64)

	if got := limiter.limit(); got != 64 {
		t.Fatalf("initial limit = %d, want 64", got)
	}

	// A window containing a single timeout halves the limit.
	completeWindow(limiter, 1)
	if got := limiter.limit(); got != 32 {
		t.Fatalf("limit after a lossy window = %d, want 32", got)
	}

	// Loss again: multiplicative decrease compounds.
	completeWindow(limiter, 1)
	if got := limiter.limit(); got != 16 {
		t.Fatalf("limit after a second lossy window = %d, want 16", got)
	}

	// Clean windows recover additively — one per window, not all at once.
	completeWindow(limiter, 0)
	if got := limiter.limit(); got != 17 {
		t.Fatalf("limit after a clean window = %d, want 17 (additive increase)", got)
	}

	// And it never climbs past the ceiling the user asked for.
	for range 200 {
		completeWindow(limiter, 0)
	}
	if got := limiter.limit(); got != 64 {
		t.Fatalf("limit after many clean windows = %d, want the ceiling 64", got)
	}
}

// The limit must not move until a full window has been observed; reacting to
// every single timeout would collapse concurrency on the first filtered port.
//
// The window size is written out as a literal 64 on purpose. Deriving it from
// aimdWindow would make this test agree with any value of the constant, which
// would pin nothing at all.
func TestAIMDWaitsForAFullWindow(t *testing.T) {
	limiter := newAIMD(64)

	// 62 timeouts and one reply: partial loss, so this window will back off —
	// but not until its 64th probe completes.
	for i := range 63 {
		limiter.completed(i < 62)
	}
	if got := limiter.limit(); got != 64 {
		t.Fatalf("limit changed mid-window: %d, want 64", got)
	}

	limiter.completed(true)
	if got := limiter.limit(); got != 32 {
		t.Fatalf("limit after completing the window = %d, want 32", got)
	}
}

// Sustained partial loss must stop at the floor, not at one probe in flight.
func TestAIMDStopsAtTheFloor(t *testing.T) {
	limiter := newAIMD(128)

	for range 20 {
		completeWindow(limiter, 1)
	}

	if got := limiter.limit(); got != 8 {
		t.Fatalf("limit after sustained partial loss = %d, want the floor of 8", got)
	}
}

// A user who asked for less concurrency than the floor gets what they asked
// for, and the limit never drops below it — least of all to zero, which would
// stall the scan outright.
func TestAIMDFloorNeverExceedsRequestedConcurrency(t *testing.T) {
	limiter := newAIMD(4)

	for range 20 {
		completeWindow(limiter, 1)
	}

	if got := limiter.limit(); got != 4 {
		t.Fatalf("limit after sustained partial loss = %d, want 4 (the requested concurrency)", got)
	}
}

// A window where nothing at all answered means the range is filtered or the
// host is down, not that the network is congested. Backing off there recovers
// no replies and only stretches the scan: measured, halving on total silence
// made a filtered 65,535-port scan about eight times slower than the connect
// scan it exists to beat.
func TestAIMDHoldsWhenNothingAnswers(t *testing.T) {
	limiter := newAIMD(100)

	for range 20 {
		completeWindow(limiter, aimdWindow) // every probe timed out
	}

	if got := limiter.limit(); got != 100 {
		t.Fatalf("limit after total silence = %d, want 100 (unchanged)", got)
	}

	// But a window that mixes replies with timeouts is congestion, and must
	// still back off.
	completeWindow(limiter, aimdWindow-1)
	if got := limiter.limit(); got != 50 {
		t.Fatalf("limit after a mostly-lost but not silent window = %d, want 50", got)
	}
}

// requireRawSocket skips the test unless this process can actually open a raw
// socket. Without the skip the SYN scanner would quietly fall back to the
// connect scanner and the test would "pass" having exercised nothing.
func requireRawSocket(t *testing.T) {
	t.Helper()

	conn, err := net.ListenPacket("ip4:tcp", "0.0.0.0")
	if err != nil {
		t.Skipf("no raw socket access (%v); run with CAP_NET_RAW to exercise the SYN path", err)
	}
	_ = conn.Close()
}

// synScannerFor builds a scanner aimed at loopback with a fallback that would
// be obvious in the results if it were ever used.
func synScannerFor(ports []int) *SYNScanner {
	connect := &ConnectScanner{Host: "127.0.0.1", Ports: ports, Timeout: time.Second, Concurrency: 16}
	return &SYNScanner{
		Host:        "127.0.0.1",
		Ports:       ports,
		Timeout:     time.Second,
		Concurrency: 16,
		Fallback:    connect,
	}
}

// The end-to-end check: a listener this test opens itself must come back open,
// and a port nothing is listening on must not.
func TestSYNScanFindsAListenerOnLoopback(t *testing.T) {
	if testing.Short() {
		t.Skip("sends packets")
	}
	requireRawSocket(t)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer func() { _ = listener.Close() }()
	openPort := listener.Addr().(*net.TCPAddr).Port

	// A second listener, closed immediately, gives us a port that is almost
	// certainly free — and therefore should answer RST rather than SYN-ACK.
	spare, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	closedPort := spare.Addr().(*net.TCPAddr).Port
	_ = spare.Close()

	result, err := synScannerFor([]int{openPort, closedPort}).Probe(context.Background())
	if err != nil {
		t.Fatalf("Probe: %v", err)
	}
	if result.ScanData.ScanMethod != "syn" {
		t.Fatalf("scan method = %q, want syn (the scanner fell back)", result.ScanData.ScanMethod)
	}
	if !reflect.DeepEqual(result.ScanData.OpenPorts, []int{openPort}) {
		t.Errorf("open ports = %v, want [%d] (closed port was %d)",
			result.ScanData.OpenPorts, openPort, closedPort)
	}
}

// The schema promise, checked against the two real Probe implementations
// rather than the shared builder they happen to call today.
func TestSYNScanMatchesConnectScanSchema(t *testing.T) {
	if testing.Short() {
		t.Skip("sends packets")
	}
	requireRawSocket(t)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer func() { _ = listener.Close() }()
	ports := []int{listener.Addr().(*net.TCPAddr).Port}

	scanner := synScannerFor(ports)

	synResult, err := scanner.Probe(context.Background())
	if err != nil {
		t.Fatalf("SYN probe: %v", err)
	}
	connectResult, err := scanner.Fallback.Probe(context.Background())
	if err != nil {
		t.Fatalf("connect probe: %v", err)
	}

	if !reflect.DeepEqual(jsonKeys(t, synResult), jsonKeys(t, connectResult)) {
		t.Errorf("JSON key sets differ:\n syn:     %v\n connect: %v",
			jsonKeys(t, synResult), jsonKeys(t, connectResult))
	}
	if !reflect.DeepEqual(synResult.ScanData.OpenPorts, connectResult.ScanData.OpenPorts) {
		t.Errorf("the two methods disagree about which ports are open: syn %v, connect %v",
			synResult.ScanData.OpenPorts, connectResult.ScanData.OpenPorts)
	}
}

// Ctrl+C during a big scan must return promptly instead of running to
// completion — the reason cmd/run.go hands every probe a signal-aware context.
func TestSYNScanStopsOnCancel(t *testing.T) {
	if testing.Short() {
		t.Skip("sends packets")
	}
	requireRawSocket(t)

	ports := make([]int, 0, 20000)
	for p := 20000; p < 40000; p++ {
		ports = append(ports, p)
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(150 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	if _, err := synScannerFor(ports).Probe(ctx); err != nil {
		t.Fatalf("Probe: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 3*time.Second {
		t.Errorf("canceled scan took %s to return", elapsed)
	}
}

// completeWindow reports one full window of probes, of which lost timed out.
func completeWindow(a *aimd, lost int) {
	for i := range aimdWindow {
		a.completed(i < lost)
	}
}
