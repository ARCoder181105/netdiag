// Package probe provides types and utilities for network diagnostics.
package probe

import (
	"context"
	"time"
)

// Severity represents the level of a probe result.
type Severity int

// SeverityOK indicates that the probe result is OK.
const (
	SeverityOK Severity = iota
	SeverityWarning
	SeverityError
	SeverityUnknown
)

func (s Severity) String() string {
	switch s {
	case SeverityOK:
		return "OK"
	case SeverityWarning:
		return "Warning"
	case SeverityError:
		return "Error"
	case SeverityUnknown:
		return "Unknown"
	default:
		return "Unknown"
	}
}

// Result represents the outcome of a network probe.
// It contains metadata about the probe and optional payloads for specific probe types.
type Result struct {
	// Identity
	TimeStamp time.Time `json:"timestamp"`
	ProbeType string    `json:"probe_type"`
	Target    string    `json:"target"`

	// Payloads
	PingData      *PingData      `json:"ping_data,omitempty"`
	ScanData      *ScanData      `json:"scan_data,omitempty"`
	TraceData     *TraceData     `json:"trace_data,omitempty"`
	HTTPData      *HTTPData      `json:"http_data,omitempty"`
	DNSData       *DNSData       `json:"dns_data,omitempty"`
	DiscoverData  *DiscoverData  `json:"discover_data,omitempty"`
	SpeedTestData *SpeedTestData `json:"speedtest_data,omitempty"`
	WhoisData     *WhoisData     `json:"whois_data,omitempty"`

	// Outcome
	Message  string   `json:"message"`
	Severity Severity `json:"severity"`
	Success  bool     `json:"success"`

	// Timing
	Latency time.Duration `json:"latency"`
}

// PingData contains statistics and results from a ping probe.
type PingData struct {
	ResolvedIP  string        `json:"resolved_ip"`
	PacketsSent int           `json:"packets_sent"`
	PacketsRecv int           `json:"packets_recv"`
	PacketLoss  float64       `json:"packet_loss"`
	MinRTT      time.Duration `json:"min_rtt"`
	MaxRTT      time.Duration `json:"max_rtt"`
	AvgRTT      time.Duration `json:"avg_rtt"`
	StdDevRTT   time.Duration `json:"stdDev_rtt"`
}

// ScanData contains information about a port scan probe.
type ScanData struct {
	ScanRateMs int64  `json:"scan_rate_ms"`
	TotalPorts int    `json:"total_ports"`
	OpenPorts  []int  `json:"open_ports"`
	ScanMethod string `json:"scan_method"`
}

// DiscoverDevice represents a single active device found on the network.
type DiscoverDevice struct {
	IP       string        `json:"ip"`
	HostName string        `json:"host_name"`
	Latency  time.Duration `json:"latency"`
}

// DiscoverData contains the results of a local network sweep.
type DiscoverData struct {
	LocalIP string           `json:"local_ip"`
	Prefix  string           `json:"prefix"`
	Devices []DiscoverDevice `json:"devices"`
}

// WhoisData contains raw WHOIS response.
type WhoisData struct {
	Raw string `json:"raw"`
}

type TraceHop struct {
	IP        string        `json:"ip"`
	HostName  string        `json:"host_name"`
	RTT       time.Duration `json:"rtt"`
	HopNumber int           `json:"hop_number"`
	Timeout   bool          `json:"timeout"`
}

// TraceData contains the sequence of hops from a traceroute probe.
type TraceData struct {
	Hops []TraceHop `json:"hops"`
}

// DNSRecord represents a single DNS record.
type DNSRecord struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// DNSData contains the results of a DNS lookup probe.
type DNSData struct {
	Server  string      `json:"server"`
	Records []DNSRecord `json:"records"`
}

// HTTPData contains the results of an HTTP probe.
type HTTPData struct {
	TLSIssuer     string        `json:"tls_issuer"`
	Latency       time.Duration `json:"latency"`
	ContentLength int64         `json:"content_length"`
	StatusCode    int           `json:"status_code"`
	TLSDaysLeft   int           `json:"tls_days_left"`
	Redirects     int           `json:"redirects"`
	TLSValid      bool          `json:"tls_valid"`
}

// SpeedTestData contains the results of an internet speed test.
type SpeedTestData struct {
	ISP          string  `json:"isp"`
	PublicIP     string  `json:"public_ip"`
	ServerName   string  `json:"server_name"`
	Country      string  `json:"country"`
	Sponsor      string  `json:"sponsor"`
	DistanceKm   float64 `json:"distance_km"`
	PingMs       float64 `json:"ping_ms"`
	DownloadMbps float64 `json:"download_mbps"`
	UploadMbps   float64 `json:"upload_mbps,omitempty"`
}

// Prober defines the interface that all network probes must implement.
type Prober interface {
	Probe(ctx context.Context) (Result, error)
	Type() string
}

// IsAnomaly returns true if the probe result indicates an anomalous state.
// This is a stub for Phase 4 (Anomaly Detection & Notifications).
func (r *Result) IsAnomaly() bool {
	// Simple heuristic for now: anything that isn't OK or failed entirely.
	return r.Severity == SeverityWarning || r.Severity == SeverityError || !r.Success
}
