package probe

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/showwin/speedtest-go/speedtest"
)

type SpeedTestProber struct {
	ServerID string
	NoUpload bool
}

func (s *SpeedTestProber) Type() string {
	return "speedtest"
}

func (s *SpeedTestProber) Probe(ctx context.Context) (Result, error) {
	start := time.Now()

	// Fetch user info
	user, err := speedtest.FetchUserInfo()
	if err != nil {
		return Result{
			TimeStamp: time.Now(),
			ProbeType: "speedtest",
			Target:    "internet",
			Success:   false,
			Severity:  SeverityError,
			Message:   fmt.Sprintf("Failed to fetch user info: %v", err),
		}, nil
	}

	// Fetch servers
	serverList, err := speedtest.FetchServers()
	if err != nil {
		return Result{
			TimeStamp: time.Now(),
			ProbeType: "speedtest",
			Target:    "internet",
			Success:   false,
			Severity:  SeverityError,
			Message:   fmt.Sprintf("Failed to fetch server list: %v", err),
		}, nil
	}

	var targets speedtest.Servers

	if s.ServerID != "" {
		id, convErr := strconv.Atoi(s.ServerID)
		if convErr != nil {
			return Result{
				TimeStamp: time.Now(),
				ProbeType: "speedtest",
				Target:    "internet",
				Success:   false,
				Severity:  SeverityError,
				Message:   "Invalid server ID format",
			}, nil
		}

		targets, err = serverList.FindServer([]int{id})
		if err != nil || len(targets) == 0 {
			return Result{
				TimeStamp: time.Now(),
				ProbeType: "speedtest",
				Target:    "internet",
				Success:   false,
				Severity:  SeverityError,
				Message:   "Server not found",
			}, nil
		}
	} else {
		targets, err = serverList.FindServer([]int{})
		if err != nil || len(targets) == 0 {
			return Result{
				TimeStamp: time.Now(),
				ProbeType: "speedtest",
				Target:    "internet",
				Success:   false,
				Severity:  SeverityError,
				Message:   "No servers found",
			}, nil
		}
	}

	target := targets[0]

	// Ping
	if err := target.PingTest(nil); err != nil {
		return Result{
			TimeStamp: time.Now(),
			ProbeType: "speedtest",
			Target:    "internet",
			Success:   false,
			Severity:  SeverityError,
			Message:   fmt.Sprintf("Ping test failed: %v", err),
		}, nil
	}

	// Download
	if err := target.DownloadTest(); err != nil {
		return Result{
			TimeStamp: time.Now(),
			ProbeType: "speedtest",
			Target:    "internet",
			Success:   false,
			Severity:  SeverityError,
			Message:   fmt.Sprintf("Download test failed: %v", err),
		}, nil
	}

	// Upload (optional)
	if !s.NoUpload {
		if err := target.UploadTest(); err != nil {
			return Result{
				TimeStamp: time.Now(),
				ProbeType: "speedtest",
				Target:    "internet",
				Success:   false,
				Severity:  SeverityError,
				Message:   fmt.Sprintf("Upload test failed: %v", err),
			}, nil
		}
	}

	// Convert to Mbps
	downloadMbps := (float64(target.DLSpeed) * 8) / 1_000_000
	uploadMbps := 0.0
	if !s.NoUpload {
		uploadMbps = (float64(target.ULSpeed) * 8) / 1_000_000
	}

	data := &SpeedTestData{
		ISP:          user.String(),
		PublicIP:     user.IP,
		ServerName:   target.Name,
		Country:      target.Country,
		Sponsor:      target.Sponsor,
		DistanceKm:   target.Distance,
		PingMs:       float64(target.Latency.Milliseconds()),
		DownloadMbps: downloadMbps,
		UploadMbps:   uploadMbps,
	}

	return Result{
		TimeStamp:     time.Now(),
		ProbeType:     "speedtest",
		Target:        "internet",
		SpeedTestData: data,
		Success:       true,
		Severity:      SeverityOK,
		Message:       "Speed test completed successfully",
		Latency:       time.Since(start),
	}, nil
}
