package probe

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	probing "github.com/prometheus-community/pro-bing"
)

// ICMPMode reports which socket type a ping used.
type ICMPMode string

const (
	// ICMPPrivileged uses raw ICMP sockets, requiring root or CAP_NET_RAW.
	ICMPPrivileged ICMPMode = "privileged"
	// ICMPUnprivileged uses datagram (UDP) ICMP sockets, which need no special
	// privileges on macOS, on Linux when net.ipv4.ping_group_range permits it,
	// and on Windows.
	ICMPUnprivileged ICMPMode = "unprivileged"
)

// icmpMode caches the socket type that worked, so a process running hundreds of
// pings (discover sweeps a whole subnet) pays the fallback cost at most once.
// Atomic because discover pings 50 hosts concurrently, and the goroutine that
// discovers the working mode writes it while the others are reading.
var (
	icmpModeOnce sync.Once
	icmpMode     atomic.Value // ICMPMode
)

// currentICMPMode returns the cached mode, initializing it on first use.
func currentICMPMode() ICMPMode {
	icmpModeOnce.Do(func() { icmpMode.Store(defaultICMPMode()) })
	mode, _ := icmpMode.Load().(ICMPMode)
	return mode
}

// defaultICMPMode picks the mode to try first. Windows has no unprivileged ICMP
// datagram socket at all, so raw is the only option there.
func defaultICMPMode() ICMPMode {
	if runtime.GOOS == "windows" {
		return ICMPPrivileged
	}
	if os.Geteuid() == 0 {
		return ICMPPrivileged
	}
	return ICMPUnprivileged
}

// RunPinger runs a configured pinger, transparently retrying with the other
// socket type if the first attempt fails on privileges. configure receives a
// fresh pinger and sets Count/Timeout/Interval before the run.
//
// This exists because hardcoding SetPrivileged(true) made every ping fail for
// users without CAP_NET_RAW, which is the default state of a `go install`ed
// binary on Linux.
func RunPinger(ctx context.Context, host string, configure func(*probing.Pinger)) (*probing.Statistics, ICMPMode, error) {
	mode := currentICMPMode()

	stats, err := runPingerWith(ctx, host, mode, configure)
	if err == nil {
		return stats, mode, nil
	}

	// Only a privilege failure is worth retrying: an unreachable host or a bad
	// name fails identically on both socket types.
	if ctx.Err() != nil || !isPrivilegeErr(err) {
		return nil, mode, err
	}

	other := ICMPPrivileged
	if mode == ICMPPrivileged {
		other = ICMPUnprivileged
	}

	stats, retryErr := runPingerWith(ctx, host, other, configure)
	if retryErr != nil {
		if !isPrivilegeErr(retryErr) {
			// The retry failed for an unrelated reason; the permission advice
			// would be misleading, so report what actually went wrong.
			return nil, mode, retryErr
		}
		// Neither socket type is available. Tell the user how to fix it
		// instead of surfacing a bare "socket: permission denied".
		return nil, mode, permissionError()
	}

	icmpMode.Store(other)
	return stats, other, nil
}

// permissionError explains how to grant ICMP access on the current platform.
func permissionError() error {
	switch runtime.GOOS {
	case "linux":
		return fmt.Errorf(
			"ICMP is not permitted for this user. Either grant the binary the capability:\n" +
				"    sudo setcap cap_net_raw+ep $(command -v netdiag)\n" +
				"or allow unprivileged ICMP for your group:\n" +
				"    sudo sysctl -w net.ipv4.ping_group_range=\"0 2147483647\"")
	case "darwin":
		return fmt.Errorf("ICMP is not permitted for this user. Try running with sudo")
	case "windows":
		return fmt.Errorf("ICMP is not permitted. Run the terminal as Administrator")
	default:
		return fmt.Errorf("ICMP is not permitted for this user")
	}
}

func runPingerWith(ctx context.Context, host string, mode ICMPMode, configure func(*probing.Pinger)) (*probing.Statistics, error) {
	pinger, err := probing.NewPinger(host)
	if err != nil {
		return nil, err
	}

	pinger.SetPrivileged(mode == ICMPPrivileged)
	configure(pinger)

	if err := pinger.RunWithContext(ctx); err != nil {
		return nil, err
	}
	return pinger.Statistics(), nil
}

// isPrivilegeErr reports whether err is the kernel refusing a raw socket.
func isPrivilegeErr(err error) bool {
	return errors.Is(err, os.ErrPermission)
}

// ResolveHost resolves host to an IP without sending any packets, so a name
// failure can be reported distinctly from an unreachable host.
func ResolveHost(host string, timeout time.Duration) (string, error) {
	pinger, err := probing.NewPinger(host)
	if err != nil {
		return "", err
	}
	pinger.Timeout = timeout
	if err := pinger.Resolve(); err != nil {
		return "", err
	}
	return pinger.IPAddr().String(), nil
}
