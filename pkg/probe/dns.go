package probe

import (
	"context"
	"net"
	"strings"
	"time"
)

// resolveHostname does reverse DNS with a short bound of its own: a slow or
// unresponsive PTR server must not stall the sweep or trace calling it.
func resolveHostname(ip string) string {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	names, err := net.DefaultResolver.LookupAddr(ctx, ip)
	if err == nil && len(names) > 0 {
		return strings.TrimSuffix(names[0], ".")
	}
	return "(Unknown)"
}
