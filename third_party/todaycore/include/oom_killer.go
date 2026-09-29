package include

import (
	"github.com/tumgovic/todaycore/adapter/service"
	"github.com/tumgovic/todaycore/service/oomkiller"
)

func registerOOMKillerService(registry *service.Registry) {
	oomkiller.RegisterService(registry)
}
