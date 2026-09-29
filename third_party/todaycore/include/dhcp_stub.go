//go:build !with_dhcp

package include

import (
	"context"

	"github.com/tumgovic/todaycore/adapter"
	C "github.com/tumgovic/todaycore/constant"
	"github.com/tumgovic/todaycore/dns"
	"github.com/tumgovic/todaycore/log"
	"github.com/tumgovic/todaycore/option"
	E "github.com/sagernet/sing/common/exceptions"
)

func registerDHCPTransport(registry *dns.TransportRegistry) {
	dns.RegisterTransport[option.DHCPDNSServerOptions](registry, C.DNSTypeDHCP, func(ctx context.Context, logger log.ContextLogger, tag string, options option.DHCPDNSServerOptions) (adapter.DNSTransport, error) {
		return nil, E.New(`DHCP is not included in this build, rebuild with -tags with_dhcp`)
	})
}
