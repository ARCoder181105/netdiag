package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/ARCoder181105/netdiag/pkg/logger"
	"github.com/ARCoder181105/netdiag/pkg/output"
	"github.com/ARCoder181105/netdiag/pkg/probe"
)

// Exit codes. Documented in the README; scripts depend on them.
const (
	exitOK      = 0 // probe ran and the target is healthy
	exitProbe   = 1 // probe ran but the target is unhealthy or unreachable
	exitUsage   = 2 // bad flags, bad arguments, or unusable configuration
	exitRuntime = 3 // the probe itself could not run
)

// probeOpts customizes how a single probe run is reported.
type probeOpts struct {
	// Banner is printed before the probe starts, for probes slow enough that
	// the user needs to know something is happening.
	Banner string

	// LogAttrs contributes probe-specific key/value pairs to the success log
	// line. Called only when the probe succeeded.
	LogAttrs func(probe.Result) []any

	// Render prints the human-readable output. Skipped entirely in --json mode.
	Render func(probe.Result)
}

// runProbe executes one prober end to end: signal-aware context, error
// normalization, structured logging, --json handling, severity-colored
// summary and exit code. Every command routes through it so these behaviors
// cannot drift apart per command.
func runProbe(p probe.Prober, target string, o probeOpts) {
	ctx, stop := signalContext()
	defer stop()

	if o.Banner != "" && !jsonOutput {
		output.PrintInfo(o.Banner)
	}

	result, err := p.Probe(ctx)

	// A returned error means the probe could not run at all (no privileges, a
	// malformed request). That is a different failure from a probe that ran and
	// found the target unhealthy, and it gets its own exit code.
	code := exitProbe
	if err != nil {
		result = probe.ErrorResult(p.Type(), target, err)
		code = exitRuntime
	}

	logResult(result, o.LogAttrs)
	reportResult(result, o.Render, code)
}

// signalContext returns a context canceled on Ctrl+C or SIGTERM, so a long
// scan or trace stops promptly instead of running to completion.
func signalContext() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
}

// logResult writes one structured log line per probe.
func logResult(result probe.Result, attrs func(probe.Result) []any) {
	if result.Success {
		fields := []any{
			"target", result.Target,
			"probe_type", result.ProbeType,
			"severity", result.Severity.String(),
			"latency_ms", float64(result.Latency.Microseconds()) / 1000.0,
		}
		if attrs != nil {
			fields = append(fields, attrs(result)...)
		}
		logger.Log.Info("probe completed", fields...)
		return
	}

	logger.Log.Error("probe failed",
		"target", result.Target,
		"probe_type", result.ProbeType,
		"error", result.Message,
	)
}

// reportResult prints the result and exits, using failCode when the probe did
// not come back healthy.
func reportResult(result probe.Result, render func(probe.Result), failCode int) {
	if jsonOutput {
		output.PrintJSON(result)
		exitFor(result, failCode)
	}

	if render != nil {
		render(result)
	}
	output.PrintBySeverity(result.Severity, result.Message)

	exitFor(result, failCode)
}

// exitFor terminates with the code matching the probe outcome. A healthy
// target exits 0; anything else exits non-zero so `netdiag http x && deploy`
// behaves the way a shell user expects.
func exitFor(result probe.Result, failCode int) {
	if result.Success && result.Severity != probe.SeverityError {
		os.Exit(exitOK)
	}
	os.Exit(failCode)
}

// exitForAll terminates with the worst outcome across several results, so
// `netdiag ping a b c` fails if any host failed. failCode distinguishes "ran
// and found a problem" from "could not run".
func exitForAll(results []probe.Result, failCode int) {
	for _, r := range results {
		if !r.Success || r.Severity == probe.SeverityError {
			os.Exit(failCode)
		}
	}
	os.Exit(exitOK)
}

// failUsage reports an invalid argument or flag and exits with exitUsage.
func failUsage(msg string) {
	output.PrintError(msg)
	os.Exit(exitUsage)
}
