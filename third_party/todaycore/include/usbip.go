//go:build with_usbip && (linux || (darwin && cgo) || windows)

package include

import (
	"github.com/tumgovic/todaycore/adapter/service"
	"github.com/tumgovic/todaycore/service/usbip"
)

func registerUSBIPServices(registry *service.Registry) {
	usbip.RegisterService(registry)
}
