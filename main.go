package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := NewApp()

	err := wails.Run(&options.App{
		Title:     "TomorrowClient",
		Width:     840,
		Height:    560,
		MinWidth:  760,
		MinHeight: 520,
		// Frameless so we can draw our own dark title bar in React.
		Frameless: true,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 10, G: 10, B: 12, A: 1},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		// A second launch (e.g. from a desktop shortcut while hidden in the
		// tray) just re-shows the running window instead of starting a copy.
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId: "tomorrowclient-single-instance",
			OnSecondInstanceLaunch: func(options.SecondInstanceData) {
				runtime.WindowShow(app.ctx)
			},
		},
		// Keep the browser's right-click context menu off in production so the
		// window behaves like a native app.
		EnableDefaultContextMenu: false,
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			// Lock the webview down: no browser zoom controls or pinch-zoom.
			IsZoomControlEnabled: false,
			ZoomFactor:           1.0,
			DisablePinchZoom:     true,
		},
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
