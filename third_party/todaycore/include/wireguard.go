//go:build with_wireguard

package include

import (
	"github.com/tumgovic/todaycore/adapter/endpoint"
	"github.com/tumgovic/todaycore/protocol/wireguard"
)

func registerWireGuardEndpoint(registry *endpoint.Registry) {
	wireguard.RegisterEndpoint(registry)
}
