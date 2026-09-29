//go:build !(darwin || dragonfly || freebsd || netbsd || openbsd)

package listener

import (
	C "github.com/tumgovic/todaycore/constant"
)

func UDPSocketBufferSize() int {
	return C.UDPSocketBufferSize
}
