package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"zephy/internal/config"
	"zephy/internal/hosts"
	"zephy/internal/winhosts"
)

type App struct {
	ctx      context.Context
	profiles []ProfileState
	mu       sync.RWMutex
	exeDir   string
	allowQuit bool
}

type ProfileState struct {
	config.Profile
	Running          bool              `json:"running"`
	Hosts            map[string]string `json:"hosts"`
	DuplicateDomains []DuplicateDomain `json:"duplicate_domains"`
	SystemHostsActive bool             `json:"system_hosts_active"`
}

type HostEntry struct {
	Domain string `json:"domain"`
	IP     string `json:"ip"`
}

type DuplicateDomain struct {
	Domain string `json:"domain"`
	Count  int    `json:"count"`
}

func NewApp() *App {
	exeDir := "."
	if exe, err := os.Executable(); err == nil {
		exeDir = filepath.Dir(exe)
	}
	return &App{
		profiles: []ProfileState{},
		exeDir:   exeDir,
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.LoadConfig()
	a.startTray()
}

func (a *App) beforeClose(ctx context.Context) bool {
	a.mu.RLock()
	allow := a.allowQuit
	a.mu.RUnlock()
	if allow {
		return false
	}
	runtime.WindowHide(ctx)
	return true
}

func (a *App) shutdown(ctx context.Context) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// no-op for hosts mode
}

func (a *App) getConfigPath() string {
	return filepath.Join(a.exeDir, "configs", "proxy_profiles.json")
}

func (a *App) LoadConfig() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	configPath := a.getConfigPath()
	cfg, err := config.Load(configPath)
	if err != nil {
		a.profiles = []ProfileState{}
		return nil
	}

	a.profiles = make([]ProfileState, len(cfg.Profiles))
	for i, p := range cfg.Profiles {
		a.profiles[i] = ProfileState{
			Profile: p,
			Running: false,
			Hosts:   make(map[string]string),
		}
		a.refreshProfileHostsLocked(i)
	}
	return nil
}

func (a *App) refreshProfileHostsLocked(i int) {
	hostsPath := a.profiles[i].HostsFile
	if !filepath.IsAbs(hostsPath) {
		hostsPath = filepath.Join(a.exeDir, hostsPath)
	}

	data, err := os.ReadFile(hostsPath)
	if err != nil {
		a.profiles[i].Hosts = make(map[string]string)
		a.profiles[i].DuplicateDomains = nil
		return
	}

	entries, counts, err := hosts.ParseText(string(data))
	if err != nil {
		a.profiles[i].Hosts = make(map[string]string)
		a.profiles[i].DuplicateDomains = nil
		return
	}

	a.profiles[i].Hosts = hosts.EntriesToMap(entries)

	dups := make([]DuplicateDomain, 0)
	for domain, c := range counts {
		if c > 1 {
			dups = append(dups, DuplicateDomain{Domain: domain, Count: c})
		}
	}
	sort.Slice(dups, func(a1, b1 int) bool { return dups[a1].Domain < dups[b1].Domain })
	a.profiles[i].DuplicateDomains = dups
}

func (a *App) GetProfiles() []ProfileState {
	a.mu.RLock()
	defer a.mu.RUnlock()

	result := make([]ProfileState, len(a.profiles))
	for i, p := range a.profiles {
		result[i] = ProfileState{
			Profile:         p.Profile,
			Running:         p.Running,
			Hosts:           p.Hosts,
			DuplicateDomains: p.DuplicateDomains,
			SystemHostsActive: p.SystemHostsActive,
		}
	}
	return result
}

func (a *App) GetHostsText(profileName string) (string, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	for i := range a.profiles {
		if a.profiles[i].Name != profileName {
			continue
		}
		hostsPath := a.profiles[i].HostsFile
		if !filepath.IsAbs(hostsPath) {
			hostsPath = filepath.Join(a.exeDir, hostsPath)
		}
		data, err := os.ReadFile(hostsPath)
		if err != nil {
			if os.IsNotExist(err) {
				return "", nil
			}
			return "", err
		}
		return string(data), nil
	}
	return "", fmt.Errorf("profile %s not found", profileName)
}

func (a *App) SetHostsText(profileName string, text string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	for i := range a.profiles {
		if a.profiles[i].Name != profileName {
			continue
		}
		hostsPath := a.profiles[i].HostsFile
		if !filepath.IsAbs(hostsPath) {
			hostsPath = filepath.Join(a.exeDir, hostsPath)
		}
		if err := os.MkdirAll(filepath.Dir(hostsPath), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(hostsPath, []byte(text), 0644); err != nil {
			return err
		}
		a.refreshProfileHostsLocked(i)
		return nil
	}
	return fmt.Errorf("profile %s not found", profileName)
}

func (a *App) IsAdmin() (bool, error) {
	return winhosts.IsAdmin()
}

// StartProfile == 启用 Profile：写入系统 hosts 的 Zephy 管理块
func (a *App) StartProfile(name string) error {
	a.mu.RLock()
	var hostsPath string
	for i := range a.profiles {
		if a.profiles[i].Name == name {
			hostsPath = a.profiles[i].HostsFile
			break
		}
	}
	exeDir := a.exeDir
	a.mu.RUnlock()
	if hostsPath == "" {
		return fmt.Errorf("profile %s not found", name)
	}
	if !filepath.IsAbs(hostsPath) {
		hostsPath = filepath.Join(exeDir, hostsPath)
	}

	text, err := os.ReadFile(hostsPath)
	if err != nil {
		return fmt.Errorf("read profile hosts: %w", err)
	}

	entries, _, err := hosts.ParseText(string(text))
	if err != nil {
		return fmt.Errorf("parse profile hosts: %w", err)
	}
	// Dedup keep-last, then materialize as "ip domain"
	dedup := hosts.DedupEntriesKeepLast(entries)
	lines := make([]string, 0, len(dedup))
	for _, e := range dedup {
		lines = append(lines, fmt.Sprintf("%s %s", e.IP, e.Domain))
	}

	if err := winhosts.ApplyManagedBlock(lines, true); err != nil {
		return err
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	for i := range a.profiles {
		a.profiles[i].SystemHostsActive = (a.profiles[i].Name == name)
		a.profiles[i].Running = (a.profiles[i].Name == name)
	}
	return nil
}

// StopProfile == 关闭 Profile：移除 Zephy 管理块
func (a *App) StopProfile(name string) error {
	if err := winhosts.RemoveManagedBlock(true); err != nil {
		return err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	for i := range a.profiles {
		a.profiles[i].SystemHostsActive = false
		a.profiles[i].Running = false
	}
	return nil
}

func (a *App) AddProfile(name, listenIP string, port int) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	for _, p := range a.profiles {
		if p.Name == name {
			return fmt.Errorf("profile %s already exists", name)
		}
		if p.ListenIP == listenIP && p.Port == port {
			return fmt.Errorf("address %s:%d already in use", listenIP, port)
		}
	}

	hostsFile := filepath.Join("configs", "hosts", name+".hosts")
	hostsPath := filepath.Join(a.exeDir, hostsFile)

	if err := os.MkdirAll(filepath.Dir(hostsPath), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(hostsPath, []byte("# Hosts for "+name+"\n"), 0644); err != nil {
		return err
	}

	newProfile := ProfileState{
		Profile: config.Profile{
			Name:      name,
			ListenIP:  listenIP,
			Port:      port,
			HostsFile: hostsFile,
		},
		Running: false,
		Hosts:   make(map[string]string),
	}
	a.profiles = append(a.profiles, newProfile)

	return a.saveConfig()
}

func (a *App) DeleteProfile(name string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	for i := range a.profiles {
		if a.profiles[i].Name == name {
			if a.profiles[i].Running {
				return fmt.Errorf("cannot delete running profile")
			}
			a.profiles = append(a.profiles[:i], a.profiles[i+1:]...)
			return a.saveConfig()
		}
	}
	return fmt.Errorf("profile %s not found", name)
}

func (a *App) RenameProfile(oldName, newName string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	oldName = strings.TrimSpace(oldName)
	newName = strings.TrimSpace(newName)
	if oldName == "" || newName == "" {
		return fmt.Errorf("name is required")
	}
	if oldName == newName {
		return nil
	}
	for _, p := range a.profiles {
		if p.Name == newName {
			return fmt.Errorf("profile %s already exists", newName)
		}
	}

	for i := range a.profiles {
		if a.profiles[i].Name != oldName {
			continue
		}
		if a.profiles[i].Running {
			return fmt.Errorf("cannot rename running profile")
		}

		// Attempt to rename default hosts file path if it follows configs/hosts/<name>.hosts
		oldHostsRel := filepath.Join("configs", "hosts", oldName+".hosts")
		newHostsRel := filepath.Join("configs", "hosts", newName+".hosts")

		if filepath.Clean(a.profiles[i].HostsFile) == filepath.Clean(oldHostsRel) {
			oldHostsAbs := filepath.Join(a.exeDir, oldHostsRel)
			newHostsAbs := filepath.Join(a.exeDir, newHostsRel)
			_ = os.MkdirAll(filepath.Dir(newHostsAbs), 0755)
			// Rename if exists; if not, we'll just update the path
			if _, err := os.Stat(oldHostsAbs); err == nil {
				_ = os.Rename(oldHostsAbs, newHostsAbs)
			}
			a.profiles[i].HostsFile = newHostsRel
		}

		a.profiles[i].Name = newName
		a.refreshProfileHostsLocked(i)
		return a.saveConfig()
	}
	return fmt.Errorf("profile %s not found", oldName)
}

func (a *App) UpdateHosts(profileName string, entries []HostEntry) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	for i := range a.profiles {
		if a.profiles[i].Name == profileName {
			hostsPath := a.profiles[i].HostsFile
			if !filepath.IsAbs(hostsPath) {
				hostsPath = filepath.Join(a.exeDir, hostsPath)
			}

			content := fmt.Sprintf("# Hosts for %s\n", profileName)
			newHosts := make(map[string]string)
			for _, e := range entries {
				if e.Domain != "" && e.IP != "" {
					content += fmt.Sprintf("%s %s\n", e.IP, e.Domain)
					newHosts[strings.ToLower(strings.TrimSpace(e.Domain))] = strings.TrimSpace(e.IP)
				}
			}

			if err := os.WriteFile(hostsPath, []byte(content), 0644); err != nil {
				return err
			}

			a.profiles[i].Hosts = newHosts
			a.refreshProfileHostsLocked(i)
			return nil
		}
	}
	return fmt.Errorf("profile %s not found", profileName)
}

// NOTE: 禁止修改系统代理/注册表：已移除相关功能

func (a *App) ImportHostsFromDialog(profileName string) error {
	selection, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Import hosts",
		Filters: []runtime.FileFilter{
			{DisplayName: "Hosts/TXT", Pattern: "*.hosts;*.txt"},
			{DisplayName: "All files", Pattern: "*.*"},
		},
	})
	if err != nil {
		return err
	}
	if selection == "" {
		return nil
	}
	data, err := os.ReadFile(selection)
	if err != nil {
		return err
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	for i := range a.profiles {
		if a.profiles[i].Name != profileName {
			continue
		}
		hostsPath := a.profiles[i].HostsFile
		if !filepath.IsAbs(hostsPath) {
			hostsPath = filepath.Join(a.exeDir, hostsPath)
		}
		if err := os.MkdirAll(filepath.Dir(hostsPath), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(hostsPath, data, 0644); err != nil {
			return err
		}
		a.refreshProfileHostsLocked(i)
		return nil
	}
	return fmt.Errorf("profile %s not found", profileName)
}

func (a *App) ExportHostsToDialog(profileName string) error {
	var hostsPath string
	a.mu.RLock()
	for i := range a.profiles {
		if a.profiles[i].Name != profileName {
			continue
		}
		hostsPath = a.profiles[i].HostsFile
		if !filepath.IsAbs(hostsPath) {
			hostsPath = filepath.Join(a.exeDir, hostsPath)
		}
		break
	}
	a.mu.RUnlock()

	if hostsPath == "" {
		return fmt.Errorf("profile %s not found", profileName)
	}

	savePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Export hosts",
		DefaultFilename: profileName + ".hosts",
		Filters: []runtime.FileFilter{
			{DisplayName: "Hosts/TXT", Pattern: "*.hosts;*.txt"},
		},
	})
	if err != nil {
		return err
	}
	if savePath == "" {
		return nil
	}

	data, err := os.ReadFile(hostsPath)
	if err != nil {
		return err
	}
	return os.WriteFile(savePath, data, 0644)
}

func (a *App) DedupHosts(profileName string) error {
	var hostsPath string
	a.mu.RLock()
	for i := range a.profiles {
		if a.profiles[i].Name != profileName {
			continue
		}
		hostsPath = a.profiles[i].HostsFile
		if !filepath.IsAbs(hostsPath) {
			hostsPath = filepath.Join(a.exeDir, hostsPath)
		}
		break
	}
	a.mu.RUnlock()
	if hostsPath == "" {
		return fmt.Errorf("profile %s not found", profileName)
	}

	orig, err := os.ReadFile(hostsPath)
	if err != nil {
		return err
	}
	updated, _ := hosts.DedupTextKeepLast(string(orig))
	if err := os.WriteFile(hostsPath, []byte(updated), 0644); err != nil {
		return err
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	for i := range a.profiles {
		if a.profiles[i].Name != profileName {
			continue
		}
		a.refreshProfileHostsLocked(i)
		return nil
	}
	return fmt.Errorf("profile %s not found", profileName)
}

func (a *App) saveConfig() error {
	profiles := make([]config.Profile, len(a.profiles))
	for i, p := range a.profiles {
		profiles[i] = p.Profile
	}

	cfg := config.File{Profiles: profiles}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	configPath := a.getConfigPath()
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return err
	}
	return os.WriteFile(configPath, data, 0644)
}
