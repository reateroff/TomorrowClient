//go:build with_tailscale

package include

import (
	"github.com/tumgovic/todaycore/adapter/certificate"
	"github.com/tumgovic/todaycore/adapter/endpoint"
	"github.com/tumgovic/todaycore/adapter/inbound"
	"github.com/tumgovic/todaycore/adapter/outbound"
	"github.com/tumgovic/todaycore/adapter/service"
	"github.com/tumgovic/todaycore/dns"
	"github.com/tumgovic/todaycore/protocol/tailscale"
	"github.com/tumgovic/todaycore/service/derp"
)

func registerTailscaleEndpoint(registry *endpoint.Registry) {
	tailscale.RegisterEndpoint(registry)
}

func registerTailcatInbound(registry *inbound.Registry) {
	tailscale.RegisterTailcatInbound(registry)
}

func registerTailcatOutbound(registry *outbound.Registry) {
	tailscale.RegisterTailcatOutbound(registry)
}

func registerTailscaleTransport(registry *dns.TransportRegistry) {
	tailscale.RegistryTransport(registry)
}

func registerTailscaleCertificateProvider(registry *certificate.Registry) {
	tailscale.RegisterCertificateProvider(registry)
}

func registerDERPService(registry *service.Registry) {
	derp.Register(registry)
}
