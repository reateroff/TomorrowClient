//go:build with_quic

package v2rayquic

import "github.com/tumgovic/todaycore/transport/v2ray"

func init() {
	v2ray.RegisterQUICConstructor(NewServer, NewClient)
}
