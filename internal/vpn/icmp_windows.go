//go:build windows

package vpn

import (
	"context"
	"fmt"
	"net"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// ICMP echo through the Windows helper API rather than a raw socket. A raw
// socket would need hand-rolled checksums and reply matching; IcmpSendEcho2Ex
// is the same path ping.exe takes.
var (
	modIphlp            = windows.NewLazySystemDLL("iphlpapi.dll")
	procIcmpCreateFile  = modIphlp.NewProc("IcmpCreateFile")
	procIcmpCloseH      = modIphlp.NewProc("IcmpCloseHandle")
	procIcmpSendEcho    = modIphlp.NewProc("IcmpSendEcho")
	procIcmpSendEcho2Ex = modIphlp.NewProc("IcmpSendEcho2Ex")
)

// ipOptionInformation mirrors IP_OPTION_INFORMATION.
type ipOptionInformation struct {
	TTL         uint8
	TOS         uint8
	Flags       uint8
	OptionsSize uint8
	OptionsData uintptr
}

// icmpEchoReply mirrors ICMP_ECHO_REPLY. RoundTripTime is in milliseconds.
type icmpEchoReply struct {
	Address       uint32
	Status        uint32
	RoundTripTime uint32
	DataSize      uint16
	Reserved      uint16
	Data          uintptr
	Options       ipOptionInformation
}

const (
	icmpTimeout  = 3 * time.Second
	icmpPayload  = 32 // bytes, what ping.exe sends by default
	icmpSamples  = 3  // best of, once the host has proven it answers
	ipStatusGood = 0  // IP_SUCCESS
)

// Interface types that are never the machine's real link to the internet.
const (
	ifTypeLoopback    = 24  // IF_TYPE_SOFTWARE_LOOPBACK
	ifTypePropVirtual = 53  // IF_TYPE_PROP_VIRTUAL — what WinTun adapters report
	ifTypeTunnel      = 131 // IF_TYPE_TUNNEL
)

// PingICMP measures the round trip to a host with ICMP echo.
//
// The echo is sent with the physical adapter's address as its source, and that
// is the whole point. A running tunnel installs a default route with metric 0,
// so an ordinary echo goes into the TUN, where the core's userspace network
// stack answers it locally: every server then reports about 1 ms and a TTL of
// 128, the machine's own initial value. Binding the source steers the packet
// out of the real adapter, and the same host that "replied" in 1 ms through the
// tunnel answers in 30 ms for real.
//
// A host that filters ICMP returns an error even though it may work perfectly
// as a proxy — callers must treat that as "unknown latency", not as a failure.
//
// IPv6-only hosts are not supported: they need the separate Icmp6 API.
func PingICMP(host string) (time.Duration, error) {
	ip, err := resolveIPv4(host)
	if err != nil {
		return 0, err
	}

	// Missing source: the machine has no ordinary adapter with a gateway. Fall
	// back to an unbound echo, which at least reports something, with the
	// caveat that a tunnel may answer it instead of the server.
	src, _ := physicalIPv4()

	h, _, _ := procIcmpCreateFile.Call()
	if h == 0 || h == uintptr(windows.InvalidHandle) {
		return 0, fmt.Errorf("IcmpCreateFile failed")
	}
	defer procIcmpCloseH.Call(h)

	best := time.Duration(0)
	for i := range icmpSamples {
		rtt, err := echoOnce(h, src, ip)
		if err != nil {
			// The first sample decides: a host that ignores the first echo is
			// not going to answer the rest, and waiting out three timeouts for
			// every dead server would make a group scan crawl.
			if i == 0 {
				return 0, err
			}
			break
		}
		if best == 0 || rtt < best {
			best = rtt
		}
	}
	return best, nil
}

// echoOnce sends one echo request, from src when it is set.
func echoOnce(handle uintptr, src, dest net.IP) (time.Duration, error) {
	d := ipv4ToU32(dest)
	if d == 0 {
		return 0, fmt.Errorf("not an IPv4 address")
	}

	req := make([]byte, icmpPayload)
	// Reply buffer: the struct, the echoed payload, and room for an ICMP error
	// message, as the API documents.
	reply := make([]byte, unsafe.Sizeof(icmpEchoReply{})+icmpPayload+8)

	var n uintptr
	if src != nil {
		n, _, _ = procIcmpSendEcho2Ex.Call(
			handle,
			0, 0, 0, // no completion event, no APC
			uintptr(ipv4ToU32(src)),
			uintptr(d),
			uintptr(unsafe.Pointer(&req[0])),
			uintptr(uint16(len(req))),
			0, // no IP options
			uintptr(unsafe.Pointer(&reply[0])),
			uintptr(uint32(len(reply))),
			uintptr(uint32(icmpTimeout/time.Millisecond)),
		)
	} else {
		n, _, _ = procIcmpSendEcho.Call(
			handle,
			uintptr(d),
			uintptr(unsafe.Pointer(&req[0])),
			uintptr(uint16(len(req))),
			0,
			uintptr(unsafe.Pointer(&reply[0])),
			uintptr(uint32(len(reply))),
			uintptr(uint32(icmpTimeout/time.Millisecond)),
		)
	}
	if n == 0 {
		return 0, fmt.Errorf("no ICMP reply")
	}

	r := (*icmpEchoReply)(unsafe.Pointer(&reply[0]))
	if r.Status != ipStatusGood {
		return 0, fmt.Errorf("ICMP status %d", r.Status)
	}
	// A sub-millisecond reply reports 0; clamp so the UI never shows "0 мс".
	rtt := time.Duration(r.RoundTripTime) * time.Millisecond
	if rtt == 0 {
		rtt = time.Millisecond
	}
	return rtt, nil
}

// physicalIPv4 returns the IPv4 address of the adapter that carries real
// traffic, to be used as an echo's source.
//
// Tunnels are excluded by interface type rather than by name: WinTun adapters
// report IF_TYPE_PROP_VIRTUAL, and matching on a name would only ever recognise
// this app's own tunnel while any other VPN on the machine kept intercepting
// the measurement. Candidates must also own a gateway, which drops VMware-style
// host-only adapters, and a routable address, which drops link-local ones.
func physicalIPv4() (net.IP, error) {
	adapters, err := adapterAddresses()
	if err != nil {
		return nil, err
	}

	var (
		best   net.IP
		metric uint32
	)
	for a := adapters; a != nil; a = a.Next {
		switch {
		case a.OperStatus != windows.IfOperStatusUp,
			a.FirstGatewayAddress == nil,
			a.IfType == ifTypeLoopback,
			a.IfType == ifTypePropVirtual,
			a.IfType == ifTypeTunnel:
			continue
		}

		for ua := a.FirstUnicastAddress; ua != nil; ua = ua.Next {
			ip := ua.Address.IP()
			if ip == nil {
				continue
			}
			v4 := ip.To4()
			if v4 == nil || v4.IsLinkLocalUnicast() || v4.IsLoopback() {
				continue
			}
			if best == nil || a.Ipv4Metric < metric {
				best, metric = v4, a.Ipv4Metric
			}
			break
		}
	}
	if best == nil {
		return nil, fmt.Errorf("no ordinary adapter with a gateway")
	}
	return best, nil
}

// adapterAddresses returns the adapter table, growing the buffer until it fits.
func adapterAddresses() (*windows.IpAdapterAddresses, error) {
	// INCLUDE_GATEWAYS is required: without it FirstGatewayAddress is always
	// nil and every adapter looks gateway-less.
	const flags = windows.GAA_FLAG_INCLUDE_GATEWAYS |
		windows.GAA_FLAG_SKIP_ANYCAST |
		windows.GAA_FLAG_SKIP_MULTICAST |
		windows.GAA_FLAG_SKIP_DNS_SERVER

	size := uint32(32 * 1024)
	for range 4 {
		buf := make([]byte, size)
		addrs := (*windows.IpAdapterAddresses)(unsafe.Pointer(&buf[0]))
		err := windows.GetAdaptersAddresses(windows.AF_UNSPEC, flags, 0, addrs, &size)
		if err == nil {
			return addrs, nil
		}
		if err != windows.ERROR_BUFFER_OVERFLOW {
			return nil, err
		}
		// size now holds the required length; loop and retry with it.
	}
	return nil, fmt.Errorf("adapter table kept growing")
}

// ipv4ToU32 packs an IPv4 address the way the ICMP helper API expects it.
func ipv4ToU32(ip net.IP) uint32 {
	v4 := ip.To4()
	if v4 == nil {
		return 0
	}
	return uint32(v4[0]) | uint32(v4[1])<<8 | uint32(v4[2])<<16 | uint32(v4[3])<<24
}

// resolveIPv4 resolves a host to its first IPv4 address.
func resolveIPv4(host string) (net.IP, error) {
	if ip := net.ParseIP(host); ip != nil {
		if v4 := ip.To4(); v4 != nil {
			return v4, nil
		}
		return nil, fmt.Errorf("IPv6 addresses are not supported by the ICMP probe")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ips, err := net.DefaultResolver.LookupIP(ctx, "ip4", host)
	if err != nil {
		return nil, err
	}
	if len(ips) == 0 {
		return nil, fmt.Errorf("no IPv4 address for %s", host)
	}
	return ips[0], nil
}
