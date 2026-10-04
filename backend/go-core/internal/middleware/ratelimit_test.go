package middleware

import (
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"
)

// TestResolveClientIP_NoTrustedProxy proves that when TRUSTED_PROXY_IPS is
// unset (or empty), the limiter keys strictly on the TCP peer and ignores any
// client-supplied X-Forwarded-For header.
func TestResolveClientIP_NoTrustedProxy(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	r.RemoteAddr = "203.0.113.9:4321"
	// Spoofed header must be ignored — without a trusted proxy there is no
	// legitimate source for this value.
	r.Header.Set("X-Forwarded-For", "198.51.100.7")

	if got := resolveClientIP(r, nil); got != "203.0.113.9" {
		t.Fatalf("expected peer 203.0.113.9, got %q", got)
	}
}

// TestResolveClientIP_UntrustedPeerIgnoresHeader proves that even when a
// trusted proxy list exists, a request arriving directly from an untrusted IP
// (e.g. hitting the API port bypassing nginx) cannot spoof its way to a fresh
// rate-limit bucket.
func TestResolveClientIP_UntrustedPeerIgnoresHeader(t *testing.T) {
	trusted := parseTrustedProxies("127.0.0.1,10.0.0.0/8")

	r := httptest.NewRequest(http.MethodPost, "/", nil)
	r.RemoteAddr = "203.0.113.5:5000"
	r.Header.Set("X-Forwarded-For", "1.2.3.4") // spoofed

	if got := resolveClientIP(r, trusted); got != "203.0.113.5" {
		t.Fatalf("expected untrusted peer 203.0.113.5, got %q", got)
	}
}

// TestResolveClientIP_TrustedProxyWalksChain proves that behind a trusted
// proxy the chain is walked right-to-left, skipping trusted hops, until the
// first untrusted (original client) address is found.
func TestResolveClientIP_TrustedProxyWalksChain(t *testing.T) {
	trusted := parseTrustedProxies("127.0.0.1,10.0.0.0/8")

	r := httptest.NewRequest(http.MethodPost, "/", nil)
	r.RemoteAddr = "127.0.0.1:8081"
	r.Header.Set("X-Forwarded-For", "198.51.100.7, 10.0.0.4")

	if got := resolveClientIP(r, trusted); got != "198.51.100.7" {
		t.Fatalf("expected original client 198.51.100.7, got %q", got)
	}
}

// TestResolveClientIP_TrustedProxyNoHeader proves that a request arriving
// without X-Forwarded-For from a trusted proxy falls back to the proxy itself.
func TestResolveClientIP_TrustedProxyNoHeader(t *testing.T) {
	trusted := parseTrustedProxies("127.0.0.1")

	r := httptest.NewRequest(http.MethodPost, "/", nil)
	r.RemoteAddr = "127.0.0.1:8081"

	if got := resolveClientIP(r, trusted); got != "127.0.0.1" {
		t.Fatalf("expected fallback to proxy peer 127.0.0.1, got %q", got)
	}
}

// TestResolveClientIP_AllHopsTrusted proves that when every hop is a trusted
// proxy (no original client visible), the resolver falls back to the peer
// instead of returning a fabricated address.
func TestResolveClientIP_AllHopsTrusted(t *testing.T) {
	trusted := parseTrustedProxies("127.0.0.1,10.0.0.0/8")

	r := httptest.NewRequest(http.MethodPost, "/", nil)
	r.RemoteAddr = "10.0.0.1:8081"
	r.Header.Set("X-Forwarded-For", "10.0.0.4, 127.0.0.1")

	if got := resolveClientIP(r, trusted); got != "10.0.0.1" {
		t.Fatalf("expected fallback to peer 10.0.0.1, got %q", got)
	}
}

// TestResolveClientIP_IPv6Peer proves the port/bracket stripping works for
// IPv6 loopback peers.
func TestResolveClientIP_IPv6Peer(t *testing.T) {
	trusted := parseTrustedProxies("::1")

	r := httptest.NewRequest(http.MethodPost, "/", nil)
	r.RemoteAddr = "[::1]:8081"
	r.Header.Set("X-Forwarded-For", "2001:db8::5")

	if got := resolveClientIP(r, trusted); got != "2001:db8::5" {
		t.Fatalf("expected original client 2001:db8::5, got %q", got)
	}
}

// TestParseTrustedProxies covers plain IPs, CIDRs, whitespace, and malformed
// entries being skipped (fail closed).
func TestParseTrustedProxies(t *testing.T) {
	trusted := parseTrustedProxies("127.0.0.1, 10.0.0.0/8, not-an-ip, ::1")

	want := []netip.Addr{
		netip.MustParseAddr("127.0.0.1"),
		netip.MustParseAddr("10.255.255.255"),
		netip.MustParseAddr("::1"),
	}
	if len(trusted) != 3 {
		t.Fatalf("expected 3 trusted prefixes, got %d", len(trusted))
	}
	for _, addr := range want {
		if !isTrustedIP(addr.String(), trusted) {
			t.Errorf("expected %s to be trusted", addr)
		}
	}
	if isTrustedIP("198.51.100.7", trusted) {
		t.Error("198.51.100.7 must not be trusted")
	}
}
