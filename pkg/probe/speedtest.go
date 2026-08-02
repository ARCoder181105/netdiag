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

	fail := func(format string, args ...any) (Result, error) {
		return Result{
			TimeStamp: time.Now(),
			ProbeType: "speedtest",
			Target:    "internet",
			Success:   false,
			Severity:  SeverityError,
			Message:   fmt.Sprintf(format, args...),
			Latency:   time.Since(start),
		}, nil
	}

	user, err := speedtest.FetchUserInfoContext(ctx)
	if err != nil {
		return fail("Failed to fetch user info: %v", err)
	}

	serverList, err := speedtest.FetchServerListContext(ctx)
	if err != nil {
		return fail("Failed to fetch server list: %v", err)
	}

	var ids []int
	if s.ServerID != "" {
		id, convErr := strconv.Atoi(s.ServerID)
		if convErr != nil {
			return fail("Invalid server ID %q: must be a number", s.ServerID)
		}
		ids = []int{id}
	}

	targets, err := serverList.FindServer(ids)
	if err != nil {
		return fail("Failed to select a speedtest server: %v", err)
	}
	if len(targets) == 0 {
		if s.ServerID != "" {
			return fail("Server %s not found", s.ServerID)
		}
		return fail("No speedtest servers found")
	}

	target := targets[0]

	if err := target.PingTestContext(ctx, nil); err != nil {
		return fail("Ping test failed: %v", err)
	}

	if err := target.DownloadTestContext(ctx); err != nil {
		return fail("Download test failed: %v", err)
	}

	uploadMbps := 0.0
	if !s.NoUpload {
		if err := target.UploadTestContext(ctx); err != nil {
			return fail("Upload test failed: %v", err)
		}
		uploadMbps = toMbps(float64(target.ULSpeed))
	}

	data := &SpeedTestData{
		ISP:          user.Isp,
		PublicIP:     user.IP,
		ServerName:   target.Name,
		Country:      target.Country,
		Sponsor:      target.Sponsor,
		DistanceKm:   target.Distance,
		PingMs:       float64(target.Latency.Microseconds()) / 1000.0,
		DownloadMbps: toMbps(float64(target.DLSpeed)),
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

// toMbps converts bytes per second to megabits per second.
func toMbps(bytesPerSec float64) float64 {
	return (bytesPerSec * 8) / 1_000_000
}
