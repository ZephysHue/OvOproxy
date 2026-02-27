package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"zephy/internal/config"
	"zephy/internal/hosts"
	"zephy/internal/proxy"
)

type App struct {
	ctx      context.Context
	profiles []ProfileState
	mu       sync.RWMutex
	exeDir   string
}

type ProfileState struct {
	config.Profile
	Running bool              `json:"running"`
	Hosts   map[string]string `json:"hosts"`
	server  *proxy.Server
	cancel  context.CancelFunc
}

type HostEntry struct {
	Domain string `json:"domain"`
	IP     string `json:"ip"`
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
}

func (a *App) beforeClose(ctx context.Context) bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	for i := range a.profiles {
		if a.profiles[i].Running && a.profiles[i].cancel != nil {
			a.profiles[i].cancel()
		}
	}
	return false
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
		hostsPath := p.HostsFile
		if !filepath.IsAbs(hostsPath) {
			hostsPath = filepath.Join(a.exeDir, hostsPath)
		}

		hostsMap := make(map[string]string)
		if table, err := hosts.Load(hostsPath); err == nil {
			hostsMap = table.GetAll()
		}

		a.profiles[i] = ProfileState{
			Profile: p,
			Running: false,
			Hosts:   hostsMap,
		}
	}
	return nil
}

func (a *App) GetProfiles() []ProfileState {
	a.mu.RLock()
	defer a.mu.RUnlock()

	result := make([]ProfileState, len(a.profiles))
	for i, p := range a.profiles {
		result[i] = ProfileState{
			Profile: p.Profile,
			Running: p.Running,
			Hosts:   p.Hosts,
		}
	}
	return result
}

func (a *App) StartProfile(name string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	for i := range a.profiles {
		if a.profiles[i].Name == name {
			if a.profiles[i].Running {
				return fmt.Errorf("profile %s is already running", name)
			}

			hostsPath := a.profiles[i].HostsFile
			if !filepath.IsAbs(hostsPath) {
				hostsPath = filepath.Join(a.exeDir, hostsPath)
			}

			table, err := hosts.Load(hostsPath)
			if err != nil {
				return fmt.Errorf("load hosts failed: %w", err)
			}

			server := proxy.New(a.profiles[i].Profile, table)
			ctx, cancel := context.WithCancel(context.Background())

			go func() {
				_ = server.ListenAndServeWithContext(ctx)
			}()

			a.profiles[i].server = server
			a.profiles[i].cancel = cancel
			a.profiles[i].Running = true
			return nil
		}
	}
	return fmt.Errorf("profile %s not found", name)
}

func (a *App) StopProfile(name string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	for i := range a.profiles {
		if a.profiles[i].Name == name {
			if !a.profiles[i].Running {
				return fmt.Errorf("profile %s is not running", name)
			}

			if a.profiles[i].cancel != nil {
				a.profiles[i].cancel()
			}
			a.profiles[i].server = nil
			a.profiles[i].cancel = nil
			a.profiles[i].Running = false
			return nil
		}
	}
	return fmt.Errorf("profile %s not found", name)
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
					newHosts[e.Domain] = e.IP
				}
			}

			if err := os.WriteFile(hostsPath, []byte(content), 0644); err != nil {
				return err
			}

			a.profiles[i].Hosts = newHosts
			return nil
		}
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
