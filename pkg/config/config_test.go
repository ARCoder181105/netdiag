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

// Isolate from the real home directory: an actual ~/.netdiag.yaml would
// override the default. Load resets viper's env/config state itself, so the
// sibling test's NETDIAG_SCAN_DEFAULT_TIMEOUT env var does not leak here.
func TestLoadFallsBackToDefault(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())

	if err := Load(); err != nil {
		t.Fatalf("Load(): %v", err)
	}

	if got := AppConfig.Scan.DefaultTimeout; got != "1s" {
		t.Errorf("Scan.DefaultTimeout = %q, want the built-in default %q", got, "1s")
	}
}
