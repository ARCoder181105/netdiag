package probe

import (
	"testing"
	"time"
)

// TestTLSDaysRemaining exercises the real helper used by HTTPProber.Probe.
// The previous version computed math.Round in the test while production
// truncated, so the test asserted behavior the code did not have.
func TestTLSDaysRemaining(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		notAfter time.Time
		expected int
	}{
		{"10 days left", now.AddDate(0, 0, 10), 10},
		{"10 days minus an hour rounds down", now.AddDate(0, 0, 10).Add(-time.Hour), 9},
		{"expires in an hour", now.Add(time.Hour), 0},
		{"expires exactly now", now, 0},
		{"expired yesterday", now.AddDate(0, 0, -1), -1},
		{"long lived", now.AddDate(1, 0, 0), 365},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tlsDaysRemaining(tt.notAfter, now); got != tt.expected {
				t.Errorf("tlsDaysRemaining() = %d days, want %d", got, tt.expected)
			}
		})
	}
}

// A certificate with just under 14 days left must still trip the "expires
// soon" threshold in Probe; flooring is what makes that true.
func TestTLSDaysRemainingFloorsTowardExpiry(t *testing.T) {
	now := time.Now()
	notAfter := now.Add(14*24*time.Hour - time.Minute)

	if got := tlsDaysRemaining(notAfter, now); got >= 14 {
		t.Errorf("tlsDaysRemaining() = %d, want < 14 so the expiry warning fires", got)
	}
}
