//go:build windows

package vpn

import (
	"testing"
)

// The measurement must survive a tunnel holding the default route. Any TUN on
// the machine answers echo requests from its own userspace stack in about 1 ms,
// so a result at or below that means the packet never left the machine.
func TestPingICMPBypassesTunnels(t *testing.T) {
	src, err := physicalIPv4()
	if err != nil {
		t.Skipf("no physical adapter to bind to: %v", err)
	}
	t.Logf("source address: %s", src)

	for _, host := range []string{"1.1.1.1", "8.8.8.8"} {
		d, err := PingICMP(host)
		if err != nil {
			t.Errorf("%s: %v", host, err)
			continue
		}
		ms := d.Milliseconds()
		t.Logf("%-10s %d ms", host, ms)
		if ms <= 1 {
			t.Errorf("%s answered in %d ms — that is a local TUN reply, not the host", host, ms)
		}
	}
}

// A host that does not exist must fail rather than be answered by a tunnel.
func TestPingICMPRejectsUnroutable(t *testing.T) {
	if _, err := physicalIPv4(); err != nil {
		t.Skipf("no physical adapter to bind to: %v", err)
	}
	if d, err := PingICMP("192.0.2.1"); err == nil {
		t.Errorf("TEST-NET-1 answered in %v", d)
	}
}
