package probe

import (
	"reflect"
	"testing"
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
		// Your code successfully auto-swaps reversed ranges!
		{"Reversed range", "82-80", []int{80, 81, 82}},
		// Your code gracefully ignores invalid input, returning nil
		{"Invalid input", "abc", nil},
		{"Out of range", "70000", nil},
		{"Mixed", "80,443,8080-8081", []int{80, 443, 8080, 8081}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParsePortRange(tt.input)

			// Safely handle nil vs empty slice comparisons
			if len(got) == 0 && len(tt.want) == 0 {
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParsePortRange() = %v, want %v", got, tt.want)
			}
		})
	}
}
