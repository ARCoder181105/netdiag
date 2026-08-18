package probe

import (
	"encoding/json"
	"reflect"
	"sort"
	"testing"
	"time"
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

	for range 63 {
		limiter.completed(true)
	}
	if got := limiter.limit(); got != 64 {
		t.Fatalf("limit changed mid-window: %d, want 64", got)
	}

	limiter.completed(true)
	if got := limiter.limit(); got != 32 {
		t.Fatalf("limit after completing the window = %d, want 32", got)
	}
}

func TestAIMDNeverReachesZero(t *testing.T) {
	limiter := newAIMD(4)
	for range 20 {
		completeWindow(limiter, aimdWindow)
	}
	if got := limiter.limit(); got != 1 {
		t.Fatalf("limit after sustained loss = %d, want 1 (a scan must still make progress)", got)
	}
}

// completeWindow reports one full window of probes, of which lost timed out.
func completeWindow(a *aimd, lost int) {
	for i := range aimdWindow {
		a.completed(i < lost)
	}
}
