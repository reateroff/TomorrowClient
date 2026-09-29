//go:build !with_ocm

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

func registerOCMService(registry *service.Registry) {
	service.Register[option.OCMServiceOptions](registry, C.TypeOCM, func(ctx context.Context, logger log.ContextLogger, tag string, options option.OCMServiceOptions) (adapter.Service, error) {
		return nil, E.New(`OCM is not included in this build, rebuild with -tags with_ocm`)
	})
}
