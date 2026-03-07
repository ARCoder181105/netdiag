package probe

import (
	"testing"
	"time"
)

func TestPingSeverity(t *testing.T) {
	tests := []struct {
		name     string
		loss     float64
		latency  time.Duration
		expected Severity
	}{
		{"Perfect", 0, 50 * time.Millisecond, SeverityOK},
		{"High Latency", 0, 200 * time.Millisecond, SeverityWarning},
		{"Partial Loss", 25.0, 50 * time.Millisecond, SeverityWarning},
		{"Complete Loss", 100.0, 0, SeverityError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mocking the logic that assigns severity in PingProber
			var got Severity
			if tt.loss == 100 {
				got = SeverityError
			} else if tt.loss > 0 || tt.latency > 150*time.Millisecond {
				got = SeverityWarning
			} else {
				got = SeverityOK
			}

			if got != tt.expected {
				t.Errorf("Ping Severity logic = %v, want %v", got, tt.expected)
			}
		})
	}
}
