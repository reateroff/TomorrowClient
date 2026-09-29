//go:build with_quic

package include

import (
	"github.com/tumgovic/todaycore/adapter/inbound"
	"github.com/tumgovic/todaycore/adapter/outbound"
	"github.com/tumgovic/todaycore/adapter/service"
	"github.com/tumgovic/todaycore/dns"
	"github.com/tumgovic/todaycore/dns/transport/quic"
	"github.com/tumgovic/todaycore/protocol/hysteria"
	"github.com/tumgovic/todaycore/protocol/hysteria2"
	_ "github.com/tumgovic/todaycore/protocol/naive/quic"
	"github.com/tumgovic/todaycore/protocol/tuic"
	_ "github.com/tumgovic/todaycore/transport/v2rayquic"
)

func registerQUICInbounds(registry *inbound.Registry) {
	hysteria.RegisterInbound(registry)
	tuic.RegisterInbound(registry)
	hysteria2.RegisterInbound(registry)
}

func registerQUICOutbounds(registry *outbound.Registry) {
	hysteria.RegisterOutbound(registry)
	tuic.RegisterOutbound(registry)
	hysteria2.RegisterOutbound(registry)
}

func registerQUICTransports(registry *dns.TransportRegistry) {
	quic.RegisterTransport(registry)
	quic.RegisterHTTP3Transport(registry)
}

func registerQUICServices(registry *service.Registry) {
	hysteria2.RegisterRealmService(registry)
}
