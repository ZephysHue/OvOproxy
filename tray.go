package main

import (
	_ "embed"
	"fmt"
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
			disableAllItem := systray.AddMenuItem("Disable Hosts", "Disable all hosts profiles")
			systray.AddSeparator()
			profileItems := make(map[string]*systray.MenuItem)
			a.mu.RLock()
			for _, p := range a.profiles {
				profileItems[p.Name] = systray.AddMenuItem(fmt.Sprintf("Enable: %s", p.Name), "Enable this profile")
			}
			a.mu.RUnlock()
			for name, item := range profileItems {
				profileName := name
				profileItem := item
				go func() {
					for range profileItem.ClickedCh {
						_ = a.StartProfile(profileName)
						runtime.EventsEmit(a.ctx, "profiles:changed")
					}
				}()
			}
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
					case <-disableAllItem.ClickedCh:
						_ = a.StopAllProfiles()
						runtime.EventsEmit(a.ctx, "profiles:changed")
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

