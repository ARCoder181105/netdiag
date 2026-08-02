package cmd

import (
	"testing"
	"time"
)

// pro-bing's --timeout bounds the whole run, not one packet. The default must
// exceed count*interval or the default run gets cut short and reports false
// packet loss on a healthy host.
func TestPingDefaultTimeoutCoversDefaultRun(t *testing.T) {
	minRequired := interval * time.Duration(count)
	if timeout <= minRequired {
		t.Fatalf("default --timeout %s does not exceed default count(%d)*interval(%s) = %s",
			timeout, count, interval, minRequired)
	}
}
