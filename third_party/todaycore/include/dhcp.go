//go:build with_dhcp

package include

import (
	"github.com/tumgovic/todaycore/dns"
	"github.com/tumgovic/todaycore/dns/transport/dhcp"
)

func registerDHCPTransport(registry *dns.TransportRegistry) {
	dhcp.RegisterTransport(registry)
}
