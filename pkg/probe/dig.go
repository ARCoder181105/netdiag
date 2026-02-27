package probe

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"
)

type DigProber struct {
	Host       string
	Server     string // Optional custom DNS server (e.g., "8.8.8.8")
	RecordType string // "A", "MX", "TXT", "NS", "CNAME"
	Timeout    time.Duration
}

func (d *DigProber) Type() string {
	return "dns"
}

func (d *DigProber) Probe(ctx context.Context) (Result, error) {

	start := time.Now()

	var resolver *net.Resolver

	// Custom DNS server support
	if d.Server != "" {

		dialer := &net.Dialer{
			Timeout: d.Timeout,
		}

		serverAddr := d.Server
		if !strings.Contains(serverAddr, ":") {
			serverAddr = net.JoinHostPort(serverAddr, "53")
		}

		resolver = &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				return dialer.DialContext(ctx, network, serverAddr)
			},
		}

	} else {
		resolver = net.DefaultResolver
	}

	recordType := strings.ToUpper(d.RecordType)
	if recordType == "" {
		recordType = "A"
	}

	var records []DNSRecord
	var err error

	switch recordType {

	case "A":
		var ips []net.IPAddr
		ips, err = resolver.LookupIPAddr(ctx, d.Host)
		if err == nil {
			for _, ip := range ips {
				if ip.IP.To4() != nil {
					records = append(records, DNSRecord{
						Type:  "A",
						Value: ip.IP.String(),
					})
				}
			}
		}

	case "MX":
		var mxs []*net.MX
		mxs, err = resolver.LookupMX(ctx, d.Host)
		if err == nil {
			for _, mx := range mxs {
				records = append(records, DNSRecord{
					Type:  "MX",
					Value: fmt.Sprintf("%d %s", mx.Pref, mx.Host),
				})
			}
		}

	case "TXT":
		var txts []string
		txts, err = resolver.LookupTXT(ctx, d.Host)
		if err == nil {
			for _, txt := range txts {
				records = append(records, DNSRecord{
					Type:  "TXT",
					Value: txt,
				})
			}
		}

	case "NS":
		var nss []*net.NS
		nss, err = resolver.LookupNS(ctx, d.Host)
		if err == nil {
			for _, ns := range nss {
				records = append(records, DNSRecord{
					Type:  "NS",
					Value: ns.Host,
				})
			}
		}

	case "CNAME":
		var cname string
		cname, err = resolver.LookupCNAME(ctx, d.Host)
		if err == nil {
			records = append(records, DNSRecord{
				Type:  "CNAME",
				Value: cname,
			})
		}

	default:
		return Result{
			TimeStamp: time.Now(),
			ProbeType: "dns",
			Target:    d.Host,
			Success:   false,
			Severity:  SeverityError,
			Message:   "Unsupported record type",
		}, nil
	}

	// Graceful DNS failure
	if err != nil {
		return Result{
			TimeStamp: time.Now(),
			ProbeType: "dns",
			Target:    d.Host,
			Success:   false,
			Severity:  SeverityError,
			Message:   fmt.Sprintf("DNS lookup failed: %v", err),
		}, nil
	}

	// 0-record polish
	severity := SeverityOK
	if len(records) == 0 {
		severity = SeverityWarning
	}

	data := &DNSData{
		Server:  d.Server,
		Records: records,
	}

	return Result{
		TimeStamp: time.Now(),
		ProbeType: "dns",
		Target:    d.Host,
		DNSData:   data,
		Success:   true,
		Severity:  severity,
		Message:   fmt.Sprintf("Found %d %s record(s)", len(records), recordType),
		Latency:   time.Since(start),
	}, nil
}
