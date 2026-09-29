//go:build windows

package vpn

import (
	"encoding/binary"
	"net"
	"strconv"
	"syscall"
	"time"
)

// ipUnicastIf is IP_UNICAST_IF: send through this interface regardless of the
// routing table.
const ipUnicastIf = 31

// PingTCP times a TCP handshake with the server itself.
//
// Like PingICMP it has to leave through the physical adapter: a running
// tunnel's userspace stack accepts every connection instantly, so an ordinary
// dial would report about 1 ms for any address at all. The socket is pinned to
// the adapter carrying the default route, the one outside the tunnel.
func PingTCP(host string, port int, timeout time.Duration) (time.Duration, error) {
	ip, err := resolveIPv4(host)
	if err != nil {
		return 0, err
	}
	d := net.Dialer{Timeout: timeout}
	// Loopback is unreachable from a socket pinned to a physical adapter.
	if name := defaultInterface(); name != "" && !ip.IsLoopback() {
		if ifc, err := net.InterfaceByName(name); err == nil {
			d.Control = bindToInterface(ifc.Index)
		}
	}
	start := time.Now()
	conn, err := d.Dial("tcp4", net.JoinHostPort(ip.String(), strconv.Itoa(port)))
	if err != nil {
		return 0, err
	}
	rtt := time.Since(start)
	_ = conn.Close()
	return rtt, nil
}

// bindToInterface sets IP_UNICAST_IF on a socket before it connects. The API
// takes the IPv4 interface index in network byte order.
func bindToInterface(index int) func(network, address string, c syscall.RawConn) error {
	return func(_, _ string, c syscall.RawConn) error {
		var serr error
		err := c.Control(func(fd uintptr) {
			var be [4]byte
			binary.BigEndian.PutUint32(be[:], uint32(index))
			serr = syscall.SetsockoptInt(syscall.Handle(fd), syscall.IPPROTO_IP, ipUnicastIf,
				int(binary.LittleEndian.Uint32(be[:])))
		})
		if err != nil {
			return err
		}
		return serr
	}
}
