package config

import "testing"

// The README documents NETDIAG_SCAN_DEFAULT_TIMEOUT. Viper only maps that to
// the nested scan.default_timeout key when an env key replacer is installed, so
// without one the documented override silently did nothing.
func TestLoadReadsNestedKeyFromEnvironment(t *testing.T) {
	t.Setenv("NETDIAG_SCAN_DEFAULT_TIMEOUT", "2s")

	if err := Load(); err != nil {
		t.Fatalf("Load(): %v", err)
	}

	if got := AppConfig.Scan.DefaultTimeout; got != "2s" {
		t.Errorf("Scan.DefaultTimeout = %q, want %q", got, "2s")
	}
}

func TestLoadFallsBackToDefault(t *testing.T) {
	if err := Load(); err != nil {
		t.Fatalf("Load(): %v", err)
	}

	if AppConfig.Scan.DefaultTimeout == "" {
		t.Error("Scan.DefaultTimeout is empty; the built-in default was not applied")
	}
}
