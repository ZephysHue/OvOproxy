package main

import (
	_ "embed"
	"sync"

	"github.com/getlantern/systray"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed assets/tray.ico
var trayIcon []byte

var trayOnce sync.Once

func (a *App) startTray() {
	trayOnce.Do(func() {
		go systray.Run(func() {
			systray.SetIcon(trayIcon)
			systray.SetTitle("MultiHostProxy")
			systray.SetTooltip("Multi-Host Proxy")

			showItem := systray.AddMenuItem("Show", "Show window")
			hideItem := systray.AddMenuItem("Hide", "Hide window")
			systray.AddSeparator()
			quitItem := systray.AddMenuItem("Quit", "Quit application")

			go func() {
				for {
					select {
					case <-showItem.ClickedCh:
						runtime.WindowShow(a.ctx)
						runtime.WindowUnminimise(a.ctx)
						runtime.WindowSetAlwaysOnTop(a.ctx, false)
					case <-hideItem.ClickedCh:
						runtime.WindowHide(a.ctx)
					case <-quitItem.ClickedCh:
						a.mu.Lock()
						a.allowQuit = true
						a.mu.Unlock()
						runtime.Quit(a.ctx)
						systray.Quit()
						return
					}
				}
			}()
		}, func() {})
	})
}

