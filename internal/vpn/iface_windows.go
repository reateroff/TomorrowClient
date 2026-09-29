//go:build windows

package vpn

import (
	"math"
	"net"
	"strings"

	"golang.org/x/sys/windows"
	"golang.zx2c4.com/wireguard/windows/tunnel/winipcfg"

	"TomorrowClient/internal/model"
)

// defaultInterface returns the friendly name of the adapter carrying the IPv4
// default route, skipping the app's own TUN adapter — the physical network the
// proxy's own connections must leave through. Empty when there is none.
//
// sing-box finds this itself (auto_detect_interface). Xray and mihomo are told
// explicitly and bind their sockets to it, so that with the tunnel's routes in
// place their traffic to the server still bypasses the tunnel.
func defaultInterface() string {
	rows, err := winipcfg.GetIPForwardTable2(windows.AF_INET)
	if err != nil {
		return ""
	}
	best, bestMetric := "", uint32(math.MaxUint32)
	for i := range rows {
		r := &rows[i]
		if r.DestinationPrefix.PrefixLength != 0 {
			continue
		}
		iface, err := r.InterfaceLUID.Interface()
		if err != nil || iface.OperStatus != winipcfg.IfOperStatusUp {
			continue
		}
		alias := iface.Alias()
		if alias == activeTunName.Load() {
			continue
		}
		// Windows ranks routes by route metric plus interface metric.
		metric := r.Metric
		if ipif, err := r.InterfaceLUID.IPInterface(windows.AF_INET); err == nil {
			metric += ipif.Metric
		}
		if metric < bestMetric {
			best, bestMetric = alias, metric
		}
	}
	return best
}

// bindInterface is the adapter a core should bind the connection to p's
// server to. Loopback servers — a local proxy chained in front — get none:
// binding a socket to a physical adapter makes loopback unreachable from it.
func bindInterface(p model.Profile) string {
	host := strings.Trim(p.Address, "[]")
	if strings.EqualFold(host, "localhost") {
		return ""
	}
	if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
		return ""
	}
	return defaultInterface()
}
