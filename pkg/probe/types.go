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

// Result represents the outcome of a network probe.
// It contains metadata about the probe and optional payloads for specific probe types.
type Result struct {
	// Identity
	TimeStamp time.Time `json:"timestamp"`
	ProbeType string    `json:"probe_type"`
	Target    string    `json:"target"`

	// Payloads
	PingData  *PingData  `json:"ping_data,omitempty"`
	ScanData  *ScanData  `json:"scan_data,omitempty"`
	TraceData *TraceData `json:"trace_data,omitempty"`
	HTTPData  *HTTPData  `json:"http_data,omitempty"`

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
	Jitter      time.Duration `json:"jitter"`
}

// ScanData contains information about a port scan probe.
type ScanData struct {
	ScanRateMs int64  `json:"scan_rate_ms"`
	OpenPorts  []int  `json:"open_ports"`
	TotalPorts int    `json:"total_ports"`
	ScanMethod string `json:"scan_method"`
}

// TraceHop represents a single hop in a traceroute probe.
type TraceHop struct {
	HopNumber int           `json:"hop_number"`
	IP        string        `json:"ip"`
	HostName  string        `json:"host_name"`
	RTT       time.Duration `json:"rtt"`
	Timeout   bool          `json:"timeout"`
}

// TraceData contains the sequence of hops from a traceroute probe.
type TraceData struct {
	Hops []TraceHop `json:"hops"`
}

// HTTPData contains the results of an HTTP probe.
type HTTPData struct {
	StatusCode    int           `json:"status_code"`
	Latency       time.Duration `json:"latency"`
	TLSValid      bool          `json:"tls_valid"`
	TLSDaysLeft   int           `json:"tls_days_left"`
	TLSIssuer     string        `json:"tls_issuer"`
	Redirects     int           `json:"redirects"`
	ContentLength int64         `json:"content_length"`
}

// Prober defines the interface that all network probes must implement.
type Prober interface {
	Probe(ctx context.Context) (Result, error)
	Type() string
}
