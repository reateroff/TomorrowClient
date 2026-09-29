//go:build with_naive_outbound

package include

import (
	"github.com/tumgovic/todaycore/adapter/outbound"
	"github.com/tumgovic/todaycore/protocol/naive"
)

func registerNaiveOutbound(registry *outbound.Registry) {
	naive.RegisterOutbound(registry)
}
