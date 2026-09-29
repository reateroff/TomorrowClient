package v2ray

import (
	"context"

	"github.com/tumgovic/todaycore/adapter"
	"github.com/tumgovic/todaycore/common/tls"
	C "github.com/tumgovic/todaycore/constant"
	"github.com/tumgovic/todaycore/option"
	"github.com/tumgovic/todaycore/transport/v2rayhttp"
	"github.com/tumgovic/todaycore/transport/v2rayhttpupgrade"
	"github.com/tumgovic/todaycore/transport/v2raywebsocket"
	"github.com/tumgovic/todaycore/transport/v2rayxhttp"
	E "github.com/sagernet/sing/common/exceptions"
	"github.com/sagernet/sing/common/logger"
	M "github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"
)

type (
	ServerConstructor[O any] func(ctx context.Context, logger logger.ContextLogger, options O, tlsConfig tls.ServerConfig, handler adapter.V2RayServerTransportHandler) (adapter.V2RayServerTransport, error)
	ClientConstructor[O any] func(ctx context.Context, dialer N.Dialer, serverAddr M.Socksaddr, options O, tlsConfig tls.Config) (adapter.V2RayClientTransport, error)
)

func NewServerTransport(ctx context.Context, logger logger.ContextLogger, options option.V2RayTransportOptions, tlsConfig tls.ServerConfig, handler adapter.V2RayServerTransportHandler) (adapter.V2RayServerTransport, error) {
	if options.Type == "" {
		return nil, nil
	}
	switch options.Type {
	case C.V2RayTransportTypeHTTP:
		return v2rayhttp.NewServer(ctx, logger, options.HTTPOptions, tlsConfig, handler)
	case C.V2RayTransportTypeWebsocket:
		return v2raywebsocket.NewServer(ctx, logger, options.WebsocketOptions, tlsConfig, handler)
	case C.V2RayTransportTypeQUIC:
		if tlsConfig == nil {
			return nil, C.ErrTLSRequired
		}
		return NewQUICServer(ctx, logger, options.QUICOptions, tlsConfig, handler)
	case C.V2RayTransportTypeGRPC:
		return NewGRPCServer(ctx, logger, options.GRPCOptions, tlsConfig, handler)
	case C.V2RayTransportTypeHTTPUpgrade:
		return v2rayhttpupgrade.NewServer(ctx, logger, options.HTTPUpgradeOptions, tlsConfig, handler)
	case C.V2RayTransportTypeXHTTP:
		// Client-side only by design.
		return nil, E.New("xhttp inbound is not implemented; use xhttp in outbounds only")
	default:
		return nil, E.New("unknown transport type: " + options.Type)
	}
}

func NewClientTransport(ctx context.Context, dialer N.Dialer, serverAddr M.Socksaddr, options option.V2RayTransportOptions, tlsConfig tls.Config) (adapter.V2RayClientTransport, error) {
	if options.Type == "" {
		return nil, nil
	}
	switch options.Type {
	case C.V2RayTransportTypeHTTP:
		return v2rayhttp.NewClient(ctx, dialer, serverAddr, options.HTTPOptions, tlsConfig)
	case C.V2RayTransportTypeGRPC:
		return NewGRPCClient(ctx, dialer, serverAddr, options.GRPCOptions, tlsConfig)
	case C.V2RayTransportTypeWebsocket:
		return v2raywebsocket.NewClient(ctx, dialer, serverAddr, options.WebsocketOptions, tlsConfig)
	case C.V2RayTransportTypeQUIC:
		if tlsConfig == nil {
			return nil, C.ErrTLSRequired
		}
		return NewQUICClient(ctx, dialer, serverAddr, options.QUICOptions, tlsConfig)
	case C.V2RayTransportTypeHTTPUpgrade:
		return v2rayhttpupgrade.NewClient(ctx, dialer, serverAddr, options.HTTPUpgradeOptions, tlsConfig)
	case C.V2RayTransportTypeXHTTP:
		return v2rayxhttp.NewClient(ctx, dialer, serverAddr, options.XHTTPOptions, tlsConfig)
	default:
		return nil, E.New("unknown transport type: " + options.Type)
	}
}
