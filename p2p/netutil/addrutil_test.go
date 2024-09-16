package netutil

import (
	"math/rand"
	"net"
	"net/netip"
	"path/filepath"
	"testing"
)

// customNetAddr is a custom implementation of net.Addr for testing purposes.
type customNetAddr struct{}

func (c *customNetAddr) Network() string { return "custom" }
func (c *customNetAddr) String() string  { return "custom" }

func TestAddrAddr(t *testing.T) {
	tempDir := t.TempDir()
	tests := []struct {
		name string
		addr net.Addr
		want netip.Addr
	}{
		{
			name: "IPAddr IPv4",
			addr: &net.IPAddr{IP: net.ParseIP("192.0.2.1")},
			want: netip.MustParseAddr("192.0.2.1"),
		},
		{
			name: "IPAddr IPv6",
			addr: &net.IPAddr{IP: net.ParseIP("2001:db8::1")},
			want: netip.MustParseAddr("2001:db8::1"),
		},
		{
			name: "TCPAddr IPv4",
			addr: &net.TCPAddr{IP: net.ParseIP("192.0.2.1"), Port: 8080},
			want: netip.MustParseAddr("192.0.2.1"),
		},
		{
			name: "TCPAddr IPv6",
			addr: &net.TCPAddr{IP: net.ParseIP("2001:db8::1"), Port: 8080},
			want: netip.MustParseAddr("2001:db8::1"),
		},
		{
			name: "UDPAddr IPv4",
			addr: &net.UDPAddr{IP: net.ParseIP("192.0.2.1"), Port: 8080},
			want: netip.MustParseAddr("192.0.2.1"),
		},
		{
			name: "UDPAddr IPv6",
			addr: &net.UDPAddr{IP: net.ParseIP("2001:db8::1"), Port: 8080},
			want: netip.MustParseAddr("2001:db8::1"),
		},
		{
			name: "Unsupported Addr type",
			addr: &net.UnixAddr{Name: filepath.Join(tempDir, "test.sock"), Net: "unix"},
			want: netip.Addr{},
		},
		{
			name: "Nil input",
			addr: nil,
			want: netip.Addr{},
		},
		{
			name: "Custom net.Addr implementation",
			addr: &customNetAddr{},
			want: netip.Addr{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := AddrAddr(tt.addr); got != tt.want {
				t.Errorf("AddrAddr() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIPToAddr(t *testing.T) {
	tests := []struct {
		name string
		ip   net.IP
		want netip.Addr
	}{
		{
			name: "IPv4",
			ip:   net.ParseIP("192.0.2.1"),
			want: netip.MustParseAddr("192.0.2.1"),
		},
		{
			name: "IPv6",
			ip:   net.ParseIP("2001:db8::1"),
			want: netip.MustParseAddr("2001:db8::1"),
		},
		{
			name: "Invalid IP",
			ip:   net.IP{1, 2, 3},
			want: netip.Addr{},
		},
		{
			name: "Invalid IP (5 octets)",
			ip:   net.IP{192, 0, 2, 1, 1},
			want: netip.Addr{},
		},
		{
			name: "IPv4-mapped IPv6",
			ip:   net.ParseIP("::ffff:192.0.2.1"),
			want: netip.MustParseAddr("192.0.2.1"),
		},
		{
			name: "Nil input",
			ip:   nil,
			want: netip.Addr{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IPToAddr(tt.ip); got != tt.want {
				t.Errorf("IPToAddr() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRandomAddr(t *testing.T) {
	// Use a fixed seed for reproducibility
	rng := rand.New(rand.NewSource(42))

	// Test IPv4 generation
	t.Run("IPv4", func(t *testing.T) {
		addr := RandomAddr(rng, true)
		if !addr.Is4() {
			t.Errorf("Expected IPv4 address, got %v", addr)
		}
	})

	// Test IPv6 generation
	t.Run("IPv6", func(t *testing.T) {
		addr := RandomAddr(rng, false)
		if !addr.Is6() {
			t.Errorf("Expected IPv6 address, got %v", addr)
		}
	})

	// Test random choice between IPv4 and IPv6
	t.Run("Random choice", func(t *testing.T) {
		ipv4Count := 0
		ipv6Count := 0
		iterations := 1000

		for i := 0; i < iterations; i++ {
			addr := RandomAddr(rng, false)
			if addr.Is4() {
				ipv4Count++
			} else if addr.Is6() {
				ipv6Count++
			} else {
				t.Errorf("Invalid address generated: %v", addr)
			}
		}

		if ipv4Count == 0 || ipv6Count == 0 {
			t.Errorf("Expected mix of IPv4 and IPv6 addresses, got %d IPv4 and %d IPv6", ipv4Count, ipv6Count)
		}
	})

	// Test randomness of generated addresses
	t.Run("Randomness check", func(t *testing.T) {
		addresses := make(map[string]bool)
		for i := 0; i < 1000; i++ {
			addr := RandomAddr(rng, false)
			addresses[addr.String()] = true
		}
		if len(addresses) < 990 {
			t.Errorf("Expected close to 1000 unique addresses, got %d", len(addresses))
		}
	})

	// Test different seeds produce different results
	t.Run("Different seeds", func(t *testing.T) {
		rng1 := rand.New(rand.NewSource(42))
		rng2 := rand.New(rand.NewSource(43))
		addr1 := RandomAddr(rng1, false)
		addr2 := RandomAddr(rng2, false)
		if addr1 == addr2 {
			t.Errorf("Expected different addresses for different seeds, got %v and %v", addr1, addr2)
		}
	})
}
