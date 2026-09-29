//go:build with_ocm

package include

import (
	"github.com/tumgovic/todaycore/adapter/service"
	"github.com/tumgovic/todaycore/service/ocm"
)

func registerOCMService(registry *service.Registry) {
	ocm.RegisterService(registry)
}
