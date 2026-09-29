//go:build with_ccm && (!darwin || cgo)

package include

import (
	"github.com/tumgovic/todaycore/adapter/service"
	"github.com/tumgovic/todaycore/service/ccm"
)

func registerCCMService(registry *service.Registry) {
	ccm.RegisterService(registry)
}
