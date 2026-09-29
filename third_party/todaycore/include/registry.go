package include

import (
	"context"

	"github.com/tumgovic/todaycore"
	"github.com/tumgovic/todaycore/adapter"
	"github.com/tumgovic/todaycore/adapter/certificate"
	"github.com/tumgovic/todaycore/adapter/endpoint"
	"github.com/tumgovic/todaycore/adapter/inbound"
	"github.com/tumgovic/todaycore/adapter/outbound"
	"github.com/tumgovic/todaycore/adapter/service"
	C "github.com/tumgovic/todaycore/constant"
	"github.com/tumgovic/todaycore/dns"
	"github.com/tumgovic/todaycore/dns/transport"
	"github.com/tumgovic/todaycore/dns/transport/fakeip"
	"github.com/tumgovic/todaycore/dns/transport/hosts"
	"github.com/tumgovic/todaycore/dns/transport/local"
	"github.com/tumgovic/todaycore/dns/transport/mdns"
	"github.com/tumgovic/todaycore/log"
	"github.com/tumgovic/todaycore/option"
	"github.com/tumgovic/todaycore/protocol/anytls"
	"github.com/tumgovic/todaycore/protocol/block"
	"github.com/tumgovic/todaycore/protocol/bridge"
	"github.com/tumgovic/todaycore/protocol/direct"
	"github.com/tumgovic/todaycore/protocol/group"
	"github.com/tumgovic/todaycore/protocol/http"
	"github.com/tumgovic/todaycore/protocol/masque"
	"github.com/tumgovic/todaycore/protocol/mixed"
	"github.com/tumgovic/todaycore/protocol/naive"
	"github.com/tumgovic/todaycore/protocol/redirect"
	"github.com/tumgovic/todaycore/protocol/shadowsocks"
	"github.com/tumgovic/todaycore/protocol/shadowtls"
	"github.com/tumgovic/todaycore/protocol/snell"
	"github.com/tumgovic/todaycore/protocol/socks"
	"github.com/tumgovic/todaycore/protocol/ssh"
	"github.com/tumgovic/todaycore/protocol/tor"
	"github.com/tumgovic/todaycore/protocol/trojan"
	"github.com/tumgovic/todaycore/protocol/tun"
	"github.com/tumgovic/todaycore/protocol/vless"
	"github.com/tumgovic/todaycore/protocol/vmess"
	"github.com/tumgovic/todaycore/service/api"
	originca "github.com/tumgovic/todaycore/service/origin_ca"
	"github.com/tumgovic/todaycore/service/resolved"
	"github.com/tumgovic/todaycore/service/ssmapi"
	E "github.com/sagernet/sing/common/exceptions"
)

func Context(ctx context.Context) context.Context {
	return box.Context(ctx, InboundRegistry(), OutboundRegistry(), EndpointRegistry(), DNSTransportRegistry(), ServiceRegistry(), CertificateProviderRegistry())
}

func InboundRegistry() *inbound.Registry {
	registry := inbound.NewRegistry()

	tun.RegisterInbound(registry)
	redirect.RegisterRedirect(registry)
	redirect.RegisterTProxy(registry)
	direct.RegisterInbound(registry)

	socks.RegisterInbound(registry)
	http.RegisterInbound(registry)
	mixed.RegisterInbound(registry)

	shadowsocks.RegisterInbound(registry)
	snell.RegisterInbound(registry)
	vmess.RegisterInbound(registry)
	trojan.RegisterInbound(registry)
	naive.RegisterInbound(registry)
	shadowtls.RegisterInbound(registry)
	vless.RegisterInbound(registry)
	anytls.RegisterInbound(registry)

	registerQUICInbounds(registry)
	registerCloudflaredInbound(registry)
	registerTailcatInbound(registry)
	registerStubForRemovedInbounds(registry)

	return registry
}

func OutboundRegistry() *outbound.Registry {
	registry := outbound.NewRegistry()

	direct.RegisterOutbound(registry)
	bridge.RegisterOutbound(registry)

	block.RegisterOutbound(registry)

	group.RegisterSelector(registry)
	group.RegisterURLTest(registry)

	socks.RegisterOutbound(registry)
	http.RegisterOutbound(registry)
	shadowsocks.RegisterOutbound(registry)
	snell.RegisterOutbound(registry)
	vmess.RegisterOutbound(registry)
	trojan.RegisterOutbound(registry)
	registerNaiveOutbound(registry)
	tor.RegisterOutbound(registry)
	ssh.RegisterOutbound(registry)
	shadowtls.RegisterOutbound(registry)
	vless.RegisterOutbound(registry)
	anytls.RegisterOutbound(registry)

	registerQUICOutbounds(registry)
	registerTailcatOutbound(registry)
	registerStubForRemovedOutbounds(registry)

	return registry
}

func EndpointRegistry() *endpoint.Registry {
	registry := endpoint.NewRegistry()

	registerWireGuardEndpoint(registry)
	registerOpenConnectEndpoint(registry)
	registerOpenVPNEndpoints(registry)
	masque.RegisterEndpoint(registry)
	registerTailscaleEndpoint(registry)

	return registry
}

func DNSTransportRegistry() *dns.TransportRegistry {
	registry := dns.NewTransportRegistry()

	transport.RegisterTCP(registry)
	transport.RegisterUDP(registry)
	transport.RegisterTLS(registry)
	transport.RegisterHTTPS(registry)
	hosts.RegisterTransport(registry)
	local.RegisterTransport(registry)
	mdns.RegisterTransport(registry)
	fakeip.RegisterTransport(registry)
	resolved.RegisterTransport(registry)

	registerQUICTransports(registry)
	registerDHCPTransport(registry)
	registerTailscaleTransport(registry)
	registerOpenConnectDNSTransport(registry)
	registerOpenVPNDNSTransport(registry)

	return registry
}

func ServiceRegistry() *service.Registry {
	registry := service.NewRegistry()

	api.RegisterService(registry)
	resolved.RegisterService(registry)
	ssmapi.RegisterService(registry)

	registerQUICServices(registry)
	registerDERPService(registry)
	registerCCMService(registry)
	registerOCMService(registry)
	registerOOMKillerService(registry)
	registerUSBIPServices(registry)

	return registry
}

func CertificateProviderRegistry() *certificate.Registry {
	registry := certificate.NewRegistry()

	registerACMECertificateProvider(registry)
	registerTailscaleCertificateProvider(registry)
	originca.RegisterCertificateProvider(registry)

	return registry
}

func registerStubForRemovedInbounds(registry *inbound.Registry) {
	inbound.Register[option.ShadowsocksInboundOptions](registry, C.TypeShadowsocksR, func(ctx context.Context, router adapter.Router, logger log.ContextLogger, tag string, options option.ShadowsocksInboundOptions) (adapter.Inbound, error) {
		return nil, E.New("ShadowsocksR is deprecated and removed in sing-box 1.6.0")
	})
}

func registerStubForRemovedOutbounds(registry *outbound.Registry) {
	outbound.Register[option.ShadowsocksROutboundOptions](registry, C.TypeShadowsocksR, func(ctx context.Context, router adapter.Router, logger log.ContextLogger, tag string, options option.ShadowsocksROutboundOptions) (adapter.Outbound, error) {
		return nil, E.New("ShadowsocksR is deprecated and removed in sing-box 1.6.0")
	})
	outbound.Register[option.StubOptions](registry, C.TypeWireGuard, func(ctx context.Context, router adapter.Router, logger log.ContextLogger, tag string, options option.StubOptions) (adapter.Outbound, error) {
		return nil, E.New("WireGuard outbound is deprecated in sing-box 1.11.0 and removed in sing-box 1.13.0, use WireGuard endpoint instead")
	})
}
