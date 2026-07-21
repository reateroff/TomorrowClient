package main

import (
	_ "embed"
	stdruntime "runtime"

	"github.com/energye/systray"
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"TomorrowClient/internal/model"
)

//go:embed build/windows/icon.ico
var trayIcon []byte

// setupTray starts the system-tray icon on its own OS-locked goroutine.
// systray.Run creates the tray window AND pumps its Windows message loop —
// systray.Register alone only builds the icon and leaves message dispatch to
// the caller, so tray clicks would never fire under Wails. The loop and the
// window it services must live on the same locked OS thread.
func (a *App) setupTray() {
	go func() {
		stdruntime.LockOSThread()
		systray.Run(a.onTrayReady, nil)
	}()
}

// onTrayReady builds the tray icon and its right-click menu.
func (a *App) onTrayReady() {
	systray.SetIcon(trayIcon)
	systray.SetTitle("TomorrowClient")
	systray.SetTooltip("TomorrowClient")

	// Left-click restores the window; right-click shows the menu. The right
	// handler is set explicitly — on Windows the automatic menu only pops for
	// a nil handler, and setting it ourselves is the reliable path.
	systray.SetOnClick(func(systray.IMenu) { a.showWindow() })
	systray.SetOnRClick(func(menu systray.IMenu) { _ = menu.ShowMenu() })

	mOpen := systray.AddMenuItem("Открыть окно", "")
	a.mToggle = systray.AddMenuItem("Подключить", "")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Выход", "")

	mOpen.Click(func() { a.showWindow() })
	a.mToggle.Click(func() { a.toggleFromTray() })
	mQuit.Click(func() { runtime.Quit(a.ctx) })

	// Reflect the current connection state on the toggle item immediately.
	a.updateTrayStatus(a.engine.Status())
}

// showWindow brings the window back from the tray/minimized state.
func (a *App) showWindow() {
	if a.ctx == nil {
		return
	}
	runtime.WindowShow(a.ctx)
}

// toggleFromTray connects or disconnects using the active profile.
func (a *App) toggleFromTray() {
	if a.engine.Status().State == model.StateConnected {
		a.Disconnect()
		return
	}
	_ = a.Connect("")
}

// updateTrayStatus relabels the toggle menu item to match the connection state.
func (a *App) updateTrayStatus(s model.Status) {
	if a.mToggle == nil {
		return
	}
	if s.State == model.StateConnected {
		a.mToggle.SetTitle("Отключить")
		systray.SetTooltip("TomorrowClient — подключено")
	} else {
		a.mToggle.SetTitle("Подключить")
		systray.SetTooltip("TomorrowClient — отключено")
	}
}
