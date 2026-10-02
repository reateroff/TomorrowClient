//go:build windows

package vpn

import (
	"errors"
	"strings"
	"testing"

	"TomorrowClient/internal/model"
)

func TestCoreStageErrorIdentifiesSelectedCoreAndFront(t *testing.T) {
	cause := errors.New("adapter unavailable")
	for _, core := range []model.Core{model.CoreXray, model.CoreMihomo} {
		err := coreStageError(core, "TUN/DNS/маршрутизация (sing-box)", cause)
		if !errors.Is(err, cause) {
			t.Fatal("original cause was lost")
		}
		if !strings.Contains(strings.ToLower(err.Error()), string(core)) || !strings.Contains(err.Error(), "(sing-box)") {
			t.Fatalf("missing core/front context: %v", err)
		}
	}
	if coreStageError(model.CoreXray, "запуск ядра", nil) != nil {
		t.Fatal("nil cause should remain nil")
	}
}
