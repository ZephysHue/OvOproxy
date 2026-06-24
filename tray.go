package main

import (
	_ "embed"
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

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
			systray.SetTitle("OvOproxy")
			systray.SetTooltip("OvOproxy / Hosts 管理")

			// 显示主窗口放在最顶部
			showItem := systray.AddMenuItem("显示主窗口 / Show", "Show main window")
			systray.AddSeparator()

			currentItem := systray.AddMenuItem("当前启用 / Active: (none)", "Current enabled profile")
			currentItem.Disable()
			systray.AddSeparator()

			hideItem := systray.AddMenuItem("隐藏窗口 / Hide", "Hide window")
			refreshUIItem := systray.AddMenuItem("刷新界面 / Refresh UI", "Refresh profile list in UI")
			openConfigDirItem := systray.AddMenuItem("打开配置目录 / Open Config Folder", "Open configs folder")
			systray.AddSeparator()
			disableAllItem := systray.AddMenuItem("禁用全部 Hosts / Disable All", "Disable all hosts profiles")
			systray.AddSeparator()

			type trayProfile struct {
				Name   string
				Active bool
			}
			getProfilesSorted := func() []trayProfile {
				a.mu.RLock()
				out := make([]trayProfile, 0, len(a.profiles))
				for _, p := range a.profiles {
					out = append(out, trayProfile{Name: p.Name, Active: p.SystemHostsActive})
				}
				a.mu.RUnlock()
				sort.Slice(out, func(i, j int) bool {
					if out[i].Active != out[j].Active {
						return out[i].Active && !out[j].Active
					}
					return out[i].Name < out[j].Name
				})
				return out
			}

			// 每个 profile 一个切换项
			profileToggleItems := make(map[string]*systray.MenuItem)
			for _, p := range getProfilesSorted() {
				title := fmt.Sprintf("  %s", p.Name)
				if p.Active {
					title = fmt.Sprintf("● %s", p.Name)
				}
				profileToggleItems[p.Name] = systray.AddMenuItem(title, "Toggle this profile")
			}

			updateTrayStatus := func() {
				a.mu.RLock()
				activeSet := map[string]bool{}
				activeNames := make([]string, 0)
				for _, p := range a.profiles {
					if p.SystemHostsActive {
						activeSet[p.Name] = true
						activeNames = append(activeNames, p.Name)
					}
				}
				a.mu.RUnlock()

				sort.Strings(activeNames)
				if len(activeNames) == 0 {
					currentItem.SetTitle("当前启用 / Active: (none)")
				} else {
					currentItem.SetTitle(fmt.Sprintf("当前启用 / Active: %s", strings.Join(activeNames, ", ")))
				}
				for name, item := range profileToggleItems {
					if activeSet[name] {
						item.SetTitle(fmt.Sprintf("● %s", name))
					} else {
						item.SetTitle(fmt.Sprintf("  %s", name))
					}
				}
			}

			updateTrayStatus()

			// 点击切换
			for name, item := range profileToggleItems {
				profileName := name
				profileItem := item
				go func() {
					for range profileItem.ClickedCh {
						a.mu.RLock()
						var active bool
						for _, p := range a.profiles {
							if p.Name == profileName {
								active = p.SystemHostsActive
								break
							}
						}
						a.mu.RUnlock()
						if active {
							_ = a.StopProfile(profileName)
						} else {
							_ = a.StartProfile(profileName)
						}
						runtime.EventsEmit(a.ctx, "profiles:changed")
						updateTrayStatus()
					}
				}()
			}
			systray.AddSeparator()
			quitItem := systray.AddMenuItem("退出程序 / Quit", "Quit application")

			stopTicker := make(chan struct{})
			go func() {
				ticker := time.NewTicker(2 * time.Second)
				defer ticker.Stop()
				for {
					select {
					case <-ticker.C:
						updateTrayStatus()
					case <-stopTicker:
						return
					}
				}
			}()

			go func() {
				for {
					select {
					case <-showItem.ClickedCh:
						runtime.WindowShow(a.ctx)
						runtime.WindowUnminimise(a.ctx)
						runtime.WindowSetAlwaysOnTop(a.ctx, true)
						runtime.WindowSetAlwaysOnTop(a.ctx, false)
					case <-hideItem.ClickedCh:
						runtime.WindowHide(a.ctx)
					case <-refreshUIItem.ClickedCh:
						runtime.EventsEmit(a.ctx, "profiles:changed")
					case <-openConfigDirItem.ClickedCh:
						_ = exec.Command("explorer", filepath.Join(a.exeDir, "configs")).Start()
					case <-disableAllItem.ClickedCh:
						_ = a.StopAllProfiles()
						runtime.EventsEmit(a.ctx, "profiles:changed")
						updateTrayStatus()
					case <-quitItem.ClickedCh:
						a.mu.Lock()
						a.allowQuit = true
						a.mu.Unlock()
						close(stopTicker)
						runtime.Quit(a.ctx)
						systray.Quit()
						return
					}
				}
			}()
		}, func() {})
	})
}
