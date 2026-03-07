package probe

import (
	"testing"
)

func TestSeverityString(t *testing.T) {
	tests := []struct {
		name     string
		severity Severity
		expected string
	}{
		{"OK", SeverityOK, "OK"},
		{"Warning", SeverityWarning, "Warning"},
		{"Error", SeverityError, "Error"},
		{"Unknown", SeverityUnknown, "Unknown"},
		{"Out of Bounds", Severity(999), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.severity.String(); got != tt.expected {
				t.Errorf("Severity.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}
