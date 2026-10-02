// TomorrowClient — минималистичный VPN-клиент для Windows.
//
// # Copyright (C) 2026 reater
//
// This program is free software: you can redistribute it and/or modify it
// under the terms of the GNU Affero General Public License as published by the
// Free Software Foundation, either version 3 of the License, or (at your
// option) any later version.
//
// This program is distributed in the hope that it will be useful, but WITHOUT
// ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or
// FITNESS FOR A PARTICULAR PURPOSE. See the GNU Affero General Public License
// for more details. You should have received a copy of it along with this
// program; if not, see <https://www.gnu.org/licenses/>.
//
// Additional terms apply under AGPL-3.0 section 7, covering the project name
// and logo only. See LICENSE-ADDITIONAL-TERMS.md.
package main

import (
	"context"
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
	setWindowsAppIdentity()
	app := NewApp()

	err := wails.Run(&options.App{
		Title:     "TomorrowClient",
		Width:     1120,
		Height:    760,
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
		OnDomReady: func(_ context.Context) {
			if err := syncWindowsWindowIcon(); err != nil {
				println("Window icon:", err.Error())
			}
		},
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
