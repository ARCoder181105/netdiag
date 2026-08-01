package probe

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"time"
)

// maxDrainBytes caps how much of a response body is read purely to allow
// connection reuse.
const maxDrainBytes = 64 << 10

type HTTPProber struct {
	URL           string
	Method        string
	Timeout       time.Duration
	SkipTLSVerify bool
}

func (h *HTTPProber) Type() string {
	return "http"
}

func (h *HTTPProber) Probe(ctx context.Context) (Result, error) {
	startTime := time.Now()

	req, err := http.NewRequestWithContext(ctx, h.Method, h.URL, nil)
	if err != nil {
		return Result{}, fmt.Errorf("failed to create request: %w", err)
	}

	redirects := 0

	client := &http.Client{
		Timeout: h.Timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("stopped after 10 redirects")
			}
			redirects = len(via)
			return nil
		},
	}

	if h.SkipTLSVerify {
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return Result{
			TimeStamp: time.Now(),
			ProbeType: "http",
			Target:    h.URL,
			Message:   err.Error(),
			Severity:  SeverityError,
			Success:   false,
			Latency:   time.Since(startTime),
		}, nil
	}
	defer func() { _ = resp.Body.Close() }()

	// Partially drain the body so the connection can be returned to the pool.
	// Bounded: this is a reachability check, and reading a multi-gigabyte body
	// would be charged to the reported latency.
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxDrainBytes))

	latency := time.Since(startTime)

	contentLength := resp.ContentLength
	statusCode := resp.StatusCode

	var tlsIssuer string
	var tlsDaysLeft int
	var tlsValid, certChecked bool

	if resp.TLS != nil && len(resp.TLS.PeerCertificates) > 0 {
		cert := resp.TLS.PeerCertificates[0]
		tlsIssuer = cert.Issuer.CommonName
		tlsDaysLeft = tlsDaysRemaining(cert.NotAfter, time.Now())
		tlsValid = time.Now().Before(cert.NotAfter)
		certChecked = true
	}

	httpData := &HTTPData{
		TLSIssuer:        tlsIssuer,
		Latency:          latency,
		ContentLength:    contentLength,
		StatusCode:       statusCode,
		TLSDaysLeft:      tlsDaysLeft,
		Redirects:        redirects,
		TLSValid:         tlsValid,
		TLSChecked:       certChecked,
		TLSVerifySkipped: resp.TLS != nil && h.SkipTLSVerify,
	}

	severity := SeverityOK
	success := true
	message := fmt.Sprintf("HTTP %d", statusCode)

	if statusCode >= 400 {
		severity = SeverityError
		success = false
	} else if statusCode >= 300 {
		severity = SeverityWarning
	}

	// Only claim expiry when a certificate was actually inspected: a resumed
	// TLS session can report no peer certificates at all.
	if certChecked && !tlsValid {
		severity = SeverityError
		success = false
		message = "Certificate has expired"
	} else if certChecked && tlsDaysLeft < 14 && severity == SeverityOK {
		severity = SeverityWarning
		message = fmt.Sprintf("Certificate expires in %d days", tlsDaysLeft)
	}

	return Result{
		TimeStamp: time.Now(),
		ProbeType: "http",
		Target:    h.URL,
		HTTPData:  httpData,
		Message:   message,
		Severity:  severity,
		Success:   success,
		Latency:   latency,
	}, nil
}

// tlsDaysRemaining is whole days until notAfter, rounded toward zero. Flooring
// is the safe direction for an expiry warning: a cert with 13.9 days left
// reports 13, not 14, so it still trips the "expires soon" threshold.
func tlsDaysRemaining(notAfter, now time.Time) int {
	return int(notAfter.Sub(now).Hours() / 24)
}
