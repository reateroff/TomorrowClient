//go:build with_cloudflared

package include

import (
	"github.com/tumgovic/todaycore/adapter/inbound"
	"github.com/tumgovic/todaycore/protocol/cloudflare"
)

func registerCloudflaredInbound(registry *inbound.Registry) {
	cloudflare.RegisterInbound(registry)
}
