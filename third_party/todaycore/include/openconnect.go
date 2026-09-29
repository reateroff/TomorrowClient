//go:build with_openconnect

package include

import (
	"github.com/tumgovic/todaycore/adapter/endpoint"
	"github.com/tumgovic/todaycore/dns"
	"github.com/tumgovic/todaycore/protocol/openconnect"
)

func registerOpenConnectEndpoint(registry *endpoint.Registry) {
	openconnect.RegisterEndpoint(registry)
}

func registerOpenConnectDNSTransport(registry *dns.TransportRegistry) {
	openconnect.RegisterDNSTransport(registry)
}
