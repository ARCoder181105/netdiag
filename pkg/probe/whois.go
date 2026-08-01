package probe

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/likexian/whois"
)

type WhoisProber struct {
	Domain  string
	Timeout time.Duration
}

func (w *WhoisProber) Type() string {
	return "whois"
}

func (w *WhoisProber) Probe(ctx context.Context) (Result, error) {
	start := time.Now()

	fail := func(msg string) Result {
		return Result{
			TimeStamp: time.Now(),
			ProbeType: "whois",
			Target:    w.Domain,
			Success:   false,
			Severity:  SeverityError,
			Message:   msg,
			Latency:   time.Since(start),
		}
	}

	client := whois.NewClient()
	if w.Timeout > 0 {
		// SetTimeout bounds each individual hop, and a lookup can chain up to
		// three (IANA -> registry -> registrar). Bound the whole query too, so
		// --timeout means what the user asked for rather than 3x that.
		client.SetTimeout(w.Timeout)

		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, w.Timeout)
		defer cancel()
	}

	// whois.Client has no context-aware API, so run it alongside the context
	// and let the caller's Ctrl+C return promptly. The goroutine finishes on
	// its own once the socket timeout fires.
	type queryResult struct {
		raw string
		err error
	}
	done := make(chan queryResult, 1)

	go func() {
		raw, err := client.Whois(w.Domain)
		done <- queryResult{raw: raw, err: err}
	}()

	select {
	case <-ctx.Done():
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return fail(fmt.Sprintf("WHOIS query timed out after %s", w.Timeout)), nil
		}
		return fail("WHOIS query canceled"), nil

	case res := <-done:
		if res.err != nil {
			return fail(fmt.Sprintf("WHOIS query failed: %v", res.err)), nil
		}

		raw := strings.TrimSpace(res.raw)
		if raw == "" {
			return Result{
				TimeStamp: time.Now(),
				ProbeType: "whois",
				Target:    w.Domain,
				WhoisData: &WhoisData{Raw: raw},
				Success:   true,
				Severity:  SeverityWarning,
				Message:   "WHOIS returned an empty record",
				Latency:   time.Since(start),
			}, nil
		}

		return Result{
			TimeStamp: time.Now(),
			ProbeType: "whois",
			Target:    w.Domain,
			WhoisData: &WhoisData{Raw: raw},
			Success:   true,
			Severity:  SeverityOK,
			Message:   "WHOIS query successful",
			Latency:   time.Since(start),
		}, nil
	}
}
