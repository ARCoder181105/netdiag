package probe

import (
	"math"
	"testing"
	"time"
)

func TestTLSDaysRemaining(t *testing.T) {
	tests := []struct {
		name     string
		expiry   time.Time
		expected int
	}{
		{"10 Days Left", time.Now().AddDate(0, 0, 10), 10},
		{"Expired Yesterday", time.Now().AddDate(0, 0, -1), -1},
		{"Expires Today", time.Now(), 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			duration := time.Until(tt.expiry)
			// Use math.Round to prevent microsecond truncation (e.g., 9.9999 days becoming 9)
			got := int(math.Round(duration.Hours() / 24))

			if got != tt.expected {
				t.Errorf("TLS calculation = %v days, want %v days", got, tt.expected)
			}
		})
	}
}
