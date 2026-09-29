//go:build with_ccm && darwin && !cgo

package include

import (
	"context"

	"github.com/tumgovic/todaycore/adapter"
	"github.com/tumgovic/todaycore/adapter/service"
	C "github.com/tumgovic/todaycore/constant"
	"github.com/tumgovic/todaycore/log"
	"github.com/tumgovic/todaycore/option"
	E "github.com/sagernet/sing/common/exceptions"
)

func registerCCMService(registry *service.Registry) {
	service.Register[option.CCMServiceOptions](registry, C.TypeCCM, func(ctx context.Context, logger log.ContextLogger, tag string, options option.CCMServiceOptions) (adapter.Service, error) {
		return nil, E.New(`CCM requires CGO on darwin, rebuild with CGO_ENABLED=1`)
	})
}
