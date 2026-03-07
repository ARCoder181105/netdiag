package probe

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/likexian/whois"
)

type WhoisProber struct {
	Domain string
}

func (w *WhoisProber) Type() string {
	return "whois"
}

func (w *WhoisProber) Probe(ctx context.Context) (Result, error) {
	start := time.Now()

	raw, err := whois.Whois(w.Domain)
	if err != nil {
		return Result{
			TimeStamp: time.Now(),
			ProbeType: "whois",
			Target:    w.Domain,
			Success:   false,
			Severity:  SeverityError,
			Message:   fmt.Sprintf("WHOIS query failed: %v", err),
		}, nil
	}

	data := &WhoisData{
		Raw: strings.TrimSpace(raw),
	}

	return Result{
		TimeStamp: time.Now(),
		ProbeType: "whois",
		Target:    w.Domain,
		WhoisData: data,
		Success:   true,
		Severity:  SeverityOK,
		Message:   "WHOIS query successful",
		Latency:   time.Since(start),
	}, nil
}
