package probe

import (
	"testing"
	"time"
)

// TestPingSeverity exercises the real classifier used by PingProber.Probe.
// It previously re-implemented the same switch inside the test, so it passed
// no matter what ping.go did.
func TestPingSeverity(t *testing.T) {
	tests := []struct {
		name        string
		loss        float64
		latency     time.Duration
		wantSuccess bool
		want        Severity
	}{
		{"Perfect", 0, 50 * time.Millisecond, true, SeverityOK},
		{"At latency threshold", 0, 150 * time.Millisecond, true, SeverityOK},
		{"High latency", 0, 200 * time.Millisecond, true, SeverityWarning},
		{"Partial loss", 25.0, 50 * time.Millisecond, true, SeverityWarning},
		{"Partial loss and high latency", 25.0, 400 * time.Millisecond, true, SeverityWarning},
		{"Complete loss", 100.0, 0, false, SeverityError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			success, got, message := pingSeverity(tt.loss, tt.latency)

			if got != tt.want {
				t.Errorf("pingSeverity(%v, %v) severity = %v, want %v", tt.loss, tt.latency, got, tt.want)
			}
			if success != tt.wantSuccess {
				t.Errorf("pingSeverity(%v, %v) success = %v, want %v", tt.loss, tt.latency, success, tt.wantSuccess)
			}
			if message == "" {
				t.Error("pingSeverity returned an empty message")
			}
		})
	}
}
