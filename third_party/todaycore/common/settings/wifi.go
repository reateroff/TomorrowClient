package settings

import (
	"context"

	"github.com/tumgovic/todaycore/adapter"
)

type WIFIMonitor interface {
	ReadWIFIState(ctx context.Context) adapter.WIFIState
	Start() error
	Close() error
}
