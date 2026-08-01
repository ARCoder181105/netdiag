package probe

import (
	"reflect"
	"testing"
	"time"
)

func TestParsePortRange(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []int
	}{
		{"Single port", "80", []int{80}},
		{"Comma list", "80,443", []int{80, 443}},
		{"Range", "80-82", []int{80, 81, 82}},
		{"Reversed range is normalized", "82-80", []int{80, 81, 82}},
		{"Mixed", "80,443,8080-8081", []int{80, 443, 8080, 8081}},
		{"Whitespace is tolerated", " 80 , 443 ", []int{80, 443}},
		{"Boundaries", "1,65535", []int{1, 65535}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParsePortRange(tt.input)
			if err != nil {
				t.Fatalf("ParsePortRange(%q) returned error: %v", tt.input, err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParsePortRange(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// Malformed input must be rejected, not silently dropped: "abc,80" used to
// scan port 80 and report success.
func TestParsePortRangeRejectsBadInput(t *testing.T) {
	bad := []string{
		"abc",
		"abc,80",
		"80,abc",
		"0",
		"65536",
		"1-65536",
		"",
		"   ",
		"80,,443",
		"80-",
		"-80",
	}

	for _, input := range bad {
		t.Run(input, func(t *testing.T) {
			got, err := ParsePortRange(input)
			if err == nil {
				t.Errorf("ParsePortRange(%q) = %v, want an error", input, got)
			}
		})
	}
}

func TestPortsPerSec(t *testing.T) {
	tests := []struct {
		name    string
		ports   int
		elapsed time.Duration
		want    float64
	}{
		// The old ScanRateMs did integer division by elapsed milliseconds, so
		// this case reported 0.
		{"Slower than one port per ms", 100, time.Second, 100},
		{"Fast scan", 1000, 100 * time.Millisecond, 10000},
		{"Zero elapsed is not a division by zero", 100, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := portsPerSec(tt.ports, tt.elapsed); got != tt.want {
				t.Errorf("portsPerSec(%d, %v) = %v, want %v", tt.ports, tt.elapsed, got, tt.want)
			}
		})
	}
}
