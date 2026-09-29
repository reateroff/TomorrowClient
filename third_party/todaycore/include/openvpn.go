//go:build with_openvpn

package include

import (
	"github.com/tumgovic/todaycore/adapter/endpoint"
	"github.com/tumgovic/todaycore/dns"
	"github.com/tumgovic/todaycore/protocol/openvpn"
)

func registerOpenVPNEndpoints(registry *endpoint.Registry) {
	openvpn.RegisterEndpoint(registry)
}

func registerOpenVPNDNSTransport(registry *dns.TransportRegistry) {
	openvpn.RegisterDNSTransport(registry)
}
