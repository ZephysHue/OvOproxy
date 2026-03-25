package main

import (
	"embed"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"

	"zephy/internal/singleinstance"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	if os.Getenv("ZEPHY_SKIP_SINGLE_INSTANCE") != "1" {
		acquired, err := singleinstance.Acquire("Global\\ZephyHostsManager_Mutex")
		if err == nil && !acquired {
			println("Another instance is already running.")
			os.Exit(0)
		}
		defer singleinstance.Release()
	}

	app := NewApp()

	err := wails.Run(&options.App{
		Title:     "Multi-Host Proxy",
		Width:     1100,
		Height:    700,
		MinWidth:  900,
		MinHeight: 600,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour:   &options.RGBA{R: 15, G: 23, B: 42, A: 1},
		CSSDragProperty:    "--wails-draggable",
		CSSDragValue:       "drag",
		OnStartup:          app.startup,
		OnBeforeClose:      app.beforeClose,
		OnShutdown:         app.shutdown,
		Frameless:          true,
		HideWindowOnClose:  true,
		StartHidden:        false,
		Windows: &windows.Options{
			WebviewIsTransparent:              false,
			WindowIsTranslucent:               false,
			DisableFramelessWindowDecorations: false,
			BackdropType:                      windows.None,
		},
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
