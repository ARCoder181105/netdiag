package probe

import (
	"net"
	"reflect"
	"testing"
)

func mustCIDR(t *testing.T, cidr string) *net.IPNet {
	t.Helper()
	_, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		t.Fatalf("ParseCIDR(%q): %v", cidr, err)
	}
	return ipnet
}

// hostAddresses used to be a string split on the last dot, which assumed every
// local network was a /24.
func TestHostAddresses(t *testing.T) {
	tests := []struct {
		name          string
		cidr          string
		skip          string
		limit         int
		wantCount     int
		wantFirst     string
		wantLast      string
		wantTruncated bool
	}{
		{
			name: "/24 excludes network, broadcast and self",
			cidr: "192.168.1.0/24", skip: "192.168.1.10", limit: 1024,
			wantCount: 253, wantFirst: "192.168.1.1", wantLast: "192.168.1.254",
		},
		{
			name: "/30 has two usable hosts",
			cidr: "10.0.0.0/30", skip: "", limit: 1024,
			wantCount: 2, wantFirst: "10.0.0.1", wantLast: "10.0.0.2",
		},
		{
			name: "/28 has fourteen usable hosts",
			cidr: "172.16.5.0/28", skip: "", limit: 1024,
			wantCount: 14, wantFirst: "172.16.5.1", wantLast: "172.16.5.14",
		},
		{
			name: "/16 is truncated at the limit",
			cidr: "10.1.0.0/16", skip: "", limit: 1024,
			wantCount: 1024, wantFirst: "10.1.0.1", wantTruncated: true,
		},
		{
			// RFC 3021: a /31 has no network or broadcast address, so both
			// addresses are usable. The old ones >= 31 branch probed only one.
			name: "/31 has two usable hosts",
			cidr: "10.0.0.4/31", skip: "", limit: 1024,
			wantCount: 2, wantFirst: "10.0.0.4", wantLast: "10.0.0.5",
		},
		{
			name: "/32 probes the address itself",
			cidr: "192.168.1.7/32", skip: "", limit: 1024,
			wantCount: 1, wantFirst: "192.168.1.7", wantLast: "192.168.1.7",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var skip net.IP
			if tt.skip != "" {
				skip = net.ParseIP(tt.skip).To4()
			}

			hosts, truncated := hostAddresses(mustCIDR(t, tt.cidr), skip, tt.limit)

			if len(hosts) != tt.wantCount {
				t.Fatalf("got %d hosts, want %d", len(hosts), tt.wantCount)
			}
			if truncated != tt.wantTruncated {
				t.Errorf("truncated = %v, want %v", truncated, tt.wantTruncated)
			}
			if hosts[0] != tt.wantFirst {
				t.Errorf("first host = %s, want %s", hosts[0], tt.wantFirst)
			}
			if tt.wantLast != "" && hosts[len(hosts)-1] != tt.wantLast {
				t.Errorf("last host = %s, want %s", hosts[len(hosts)-1], tt.wantLast)
			}

			for _, h := range hosts {
				if tt.skip != "" && h == tt.skip {
					t.Errorf("host list contains the local address %s", h)
				}
			}
		})
	}
}

// Devices must sort numerically, so .9 comes before .10.
func TestCompareIPv4Ordering(t *testing.T) {
	got := []string{"192.168.1.10", "192.168.1.9", "192.168.1.100", "192.168.1.1"}

	// Insertion sort using the comparator under test.
	for i := 1; i < len(got); i++ {
		for j := i; j > 0 && compareIPv4(got[j], got[j-1]) < 0; j-- {
			got[j], got[j-1] = got[j-1], got[j]
		}
	}

	want := []string{"192.168.1.1", "192.168.1.9", "192.168.1.10", "192.168.1.100"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("sorted = %v, want %v", got, want)
	}
}
