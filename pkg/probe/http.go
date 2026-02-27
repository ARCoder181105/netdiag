package probe

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"time"
)

type HTTPProber struct {
	URL           string
	Method        string
	Timeout       time.Duration
	SkipTLSVerify bool
}

func (p *HTTPProber) Type() string {
	return "http"
}

func (p *HTTPProber) Probe(ctx context.Context) (Result, error) {
	startTime := time.Now()

	req, err := http.NewRequestWithContext(ctx, p.Method, p.URL, nil)
	if err != nil {
		return Result{}, fmt.Errorf("failed to create request: %w", err)
	}

	redirects := 0

	client := &http.Client{
		Timeout: p.Timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("stopped after 10 redirects")
			}
			redirects = len(via)
			return nil
		},
	}

	if p.SkipTLSVerify {
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return Result{
			TimeStamp: time.Now(),
			ProbeType: "http",
			Target:    p.URL,
			Message:   err.Error(),
			Severity:  SeverityError,
			Success:   false,
		}, nil
	}
	defer resp.Body.Close()

	latency := time.Since(startTime)

	contentLength := resp.ContentLength
	statusCode := resp.StatusCode

	var tlsIssuer string
	var tlsDaysLeft int
	var tlsValid bool

	if resp.TLS != nil && len(resp.TLS.PeerCertificates) > 0 {
		cert := resp.TLS.PeerCertificates[0]
		tlsIssuer = cert.Issuer.CommonName
		tlsDaysLeft = int(time.Until(cert.NotAfter).Hours() / 24)
		tlsValid = time.Now().Before(cert.NotAfter)
	}

	httpData := &HTTPData{
		TLSIssuer:     tlsIssuer,
		Latency:       latency,
		ContentLength: contentLength,
		StatusCode:    statusCode,
		TLSDaysLeft:   tlsDaysLeft,
		Redirects:     redirects,
		TLSValid:      tlsValid,
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

	if tlsDaysLeft > 0 && tlsDaysLeft < 14 {
		severity = SeverityWarning
		message = fmt.Sprintf("Certificate expires in %d days", tlsDaysLeft)
	}

	return Result{
		TimeStamp: time.Now(),
		ProbeType: "http",
		Target:    p.URL,
		HTTPData:  httpData,
		Message:   message,
		Severity:  severity,
		Success:   success,
		Latency:   latency,
	}, nil
}
