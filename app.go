package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"zephy/internal/config"
	"zephy/internal/hosts"
	"zephy/internal/proxymanager"
	"zephy/internal/winhosts"
)

type App struct {
	ctx               context.Context
	profiles          []ProfileState
	mu                sync.RWMutex
	exeDir            string
	allowQuit         bool
	proxyManager      *proxymanager.Manager
	subscriptionCancel context.CancelFunc
	refreshMu         sync.Mutex // 串行化 RefreshSubscription
	configDirty       bool
	configSaveCancel  context.CancelFunc
}

type ProfileState struct {
	config.Profile
	Running           bool              `json:"running"`
	Hosts             map[string]string `json:"hosts"`
	SystemHostsActive bool              `json:"system_hosts_active"`
	ProxyActive       bool              `json:"proxy_active"`
	ProxyError        string            `json:"proxy_error"`
}

type BackupInfo struct {
	FileName string `json:"file_name"`
	Path     string `json:"path"`
	Size     int64  `json:"size"`
	Modified string `json:"modified"`
}

type SubscriptionResult struct {
	Status     string `json:"status"`      // "ok" / "error"
	Message    string `json:"message"`
	LastFetch  string `json:"last_fetch"`
	EntryCount int    `json:"entry_count"`
}

func NewApp() *App {
	exeDir := "."
	if exe, err := os.Executable(); err == nil {
		exeDir = filepath.Dir(exe)
	}
	return &App{
		profiles:     []ProfileState{},
		exeDir:       exeDir,
		proxyManager: proxymanager.New(),
	}
}

func (a *App) RelaunchAsAdmin() error {
	admin, err := winhosts.IsAdmin()
	if err != nil {
		return err
	}
	if admin {
		return nil
	}

	exe, err := os.Executable()
	if err != nil {
		return err
	}
	quotedExe := strings.ReplaceAll(exe, "'", "''")
	psInner := fmt.Sprintf("Start-Sleep -Milliseconds 800; Start-Process -FilePath '%s'", quotedExe)
	cmd := exec.Command("powershell", "-NoProfile", "-Command",
		fmt.Sprintf("Start-Process powershell -Verb RunAs -ArgumentList '-NoProfile -Command \"%s\"'", psInner))
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() {
		time.Sleep(300 * time.Millisecond)
		if a.ctx != nil {
			runtime.Quit(a.ctx)
		}
	}()
	return nil
}

// initConfigDir 首次运行时从 configs.example 复制默认配置
func (a *App) initConfigDir() {
	cfgDir := filepath.Join(a.exeDir, "configs")
	if _, err := os.Stat(cfgDir); err == nil {
		return // 已有配置，不覆盖
	}
	exampleDir := filepath.Join(a.exeDir, "configs.example")
	if _, err := os.Stat(exampleDir); err != nil {
		return // 没有模板，跳过
	}
	_ = os.MkdirAll(cfgDir, 0755)
	entries, err := os.ReadDir(exampleDir)
	if err != nil {
		return
	}
	for _, e := range entries {
		src := filepath.Join(exampleDir, e.Name())
		dst := filepath.Join(cfgDir, e.Name())
		if e.IsDir() {
			_ = copyDir(src, dst)
		} else {
			data, err := os.ReadFile(src)
			if err != nil {
				continue
			}
			_ = os.WriteFile(dst, data, 0644)
		}
	}
}

func copyDir(src, dst string) error {
	_ = os.MkdirAll(dst, 0755)
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, e := range entries {
		srcPath := filepath.Join(src, e.Name())
		dstPath := filepath.Join(dst, e.Name())
		if e.IsDir() {
			_ = copyDir(srcPath, dstPath)
		} else {
			data, err := os.ReadFile(srcPath)
			if err != nil {
				continue
			}
			_ = os.WriteFile(dstPath, data, 0644)
		}
	}
	return nil
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.initConfigDir()
	if err := a.LoadConfig(); err != nil {
		runtime.LogError(ctx, "LoadConfig: "+err.Error())
	}
	a.startTray()
	go func() {
		a.startAllProxies()
		a.syncHostsEnabledState()
		if a.ctx != nil {
			runtime.EventsEmit(a.ctx, "profiles:changed")
		}
	}()
	a.startSubscriptionAutoRefresh()
	a.startConfigBatcher()
}

func (a *App) startAllProxies() {
	a.mu.RLock()
	profiles := make([]ProfileState, len(a.profiles))
	copy(profiles, a.profiles)
	a.mu.RUnlock()

	for _, p := range profiles {
		_ = a.proxyManager.StartProxy(p.Name, p.ListenIP, p.Port, p.Hosts)
	}
	a.refreshProxyStatus()
}

func (a *App) refreshProxyStatus() {
	a.mu.Lock()
	defer a.mu.Unlock()
	status := a.proxyManager.GetAllStatus()
	for i := range a.profiles {
		if s, ok := status[a.profiles[i].Name]; ok {
			a.profiles[i].ProxyActive = s.Active
			a.profiles[i].ProxyError = s.LastErr
		} else {
			a.profiles[i].ProxyActive = false
			a.profiles[i].ProxyError = ""
		}
	}
}

func (a *App) syncHostsEnabledState() {
	enabled, err := winhosts.GetEnabledProfiles()
	if err != nil {
		return
	}
	enabledSet := make(map[string]bool)
	for _, id := range enabled {
		enabledSet[id] = true
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	for i := range a.profiles {
		a.profiles[i].SystemHostsActive = enabledSet[a.profiles[i].Name]
		a.profiles[i].Running = a.profiles[i].SystemHostsActive
	}
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
	if a.subscriptionCancel != nil {
		a.subscriptionCancel()
	}
	if a.configSaveCancel != nil {
		a.configSaveCancel()
	}
	a.configFlush()
	a.proxyManager.StopAll()

	admin, _ := winhosts.IsAdmin()
	if admin {
		_ = winhosts.RemoveAllZephyBlocks(true)
	}
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
		if os.IsNotExist(err) {
			return nil // 首次运行，无配置文件
		}
		return fmt.Errorf("load config: %w", err)
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
		return
	}

	entries, _, err := hosts.ParseText(string(data))
	if err != nil {
		a.profiles[i].Hosts = make(map[string]string)
		return
	}

	a.profiles[i].Hosts = hosts.EntriesToMap(entries)
	a.proxyManager.UpdateHostsRules(a.profiles[i].Name, a.profiles[i].Hosts)
}

func (a *App) GetProfiles() []ProfileState {
	a.refreshProxyStatus()

	a.mu.RLock()
	defer a.mu.RUnlock()

	result := make([]ProfileState, len(a.profiles))
	for i, p := range a.profiles {
		result[i] = ProfileState{
			Profile:           p.Profile,
			Running:           p.Running,
			Hosts:             p.Hosts,
			SystemHostsActive: p.SystemHostsActive,
			ProxyActive:       p.ProxyActive,
			ProxyError:        p.ProxyError,
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
		text = hosts.NormalizeText(text)
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

// StartProfile == 启用 Profile：写入系统 hosts 的该 Profile 标记块
// 支持多 Profile 同时启用：不再自动禁用其他已启用配置
func (a *App) StartProfile(name string) error {
	a.mu.RLock()
	var hostsPath string
	var proxyActive bool
	var proxyErr string
	var isRemote bool
	for i := range a.profiles {
		if a.profiles[i].Name == name {
			hostsPath = a.profiles[i].HostsFile
			proxyActive = a.profiles[i].ProxyActive
			proxyErr = a.profiles[i].ProxyError
			isRemote = a.profiles[i].Type == "remote"
			break
		}
	}
	exeDir := a.exeDir
	a.mu.RUnlock()

	if !proxyActive {
		return fmt.Errorf("代理端口未启动: %s", proxyErr)
	}

	// remote 类型：hosts 文件在首次刷新时创建，这里兜底生成路径
	if isRemote && hostsPath == "" {
		hostsPath = filepath.Join(exeDir, "configs", "hosts", name+".hosts")
		a.mu.Lock()
		for i := range a.profiles {
			if a.profiles[i].Name == name {
				a.profiles[i].HostsFile = filepath.Join("configs", "hosts", name+".hosts")
				break
			}
		}
		a.mu.Unlock()
	}
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

	lines := make([]string, 0, len(entries))
	for _, e := range entries {
		lines = append(lines, fmt.Sprintf("%s %s", e.IP, e.Domain))
	}

	if err := winhosts.ApplyProfileBlock(name, lines, true); err != nil {
		return err
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	found := false
	for i := range a.profiles {
		if a.profiles[i].Name == name {
			a.profiles[i].SystemHostsActive = true
			a.profiles[i].Running = true
			found = true
			break
		}
	}
	if !found {
		// 写入期间 profile 被删除，清理孤儿块
		_ = winhosts.RemoveProfileBlock(name, true)
	}
	return nil
}

// StopProfile == 关闭 Profile：移除该 Profile 的 hosts 标记块
func (a *App) StopProfile(name string) error {
	if err := winhosts.RemoveProfileBlock(name, true); err != nil {
		return err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	for i := range a.profiles {
		if a.profiles[i].Name == name {
			a.profiles[i].SystemHostsActive = false
			a.profiles[i].Running = false
		}
	}
	return nil
}

func (a *App) StopAllProfiles() error {
	admin, err := winhosts.IsAdmin()
	if err != nil {
		return err
	}
	if !admin {
		return fmt.Errorf("需要管理员权限才能修改系统 hosts 文件")
	}
	if err := winhosts.RemoveAllZephyBlocks(true); err != nil {
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

	if err := a.saveConfig(); err != nil {
		return err
	}
	hostsRules := newProfile.Hosts
	listenIPCopy := listenIP
	portCopy := port
	nameCopy := name

	// 在锁外启动代理（避免持锁执行网络操作）
	a.mu.Unlock()
	_ = a.proxyManager.StartProxy(nameCopy, listenIPCopy, portCopy, hostsRules)
	a.refreshProxyStatus()
	a.mu.Lock()

	return nil
}

func (a *App) AddRemoteProfile(name, listenIP string, port int, url string, interval int) error {
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

	newProfile := ProfileState{
		Profile: config.Profile{
			Type:                 "remote",
			Name:                 name,
			ListenIP:             listenIP,
			Port:                 port,
			HostsFile:            hostsFile,
			SubscriptionURL:      url,
			SubscriptionInterval: interval,
			SubscriptionEnabled:  true,
		},
		Running: false,
		Hosts:   make(map[string]string),
	}
	a.profiles = append(a.profiles, newProfile)

	if err := a.saveConfig(); err != nil {
		return err
	}

	nameCopy := name
	listenIPCopy := listenIP
	portCopy := port

	a.mu.Unlock()
	_ = a.proxyManager.StartProxy(nameCopy, listenIPCopy, portCopy, nil)
	a.refreshProxyStatus()
	a.mu.Lock()

	// 立即拉取一次订阅
	go func() {
		_, _ = a.RefreshSubscription(nameCopy)
	}()

	return nil
}

func (a *App) DeleteProfile(name string) error {
	a.mu.Lock()

	for i := range a.profiles {
		if a.profiles[i].Name == name {
			if a.profiles[i].Running || a.profiles[i].SystemHostsActive {
				a.mu.Unlock()
				return fmt.Errorf("cannot delete running profile")
			}
			a.profiles = append(a.profiles[:i], a.profiles[i+1:]...)
			err := a.saveConfig()
			a.mu.Unlock()

			a.proxyManager.StopProxy(name)
			return err
		}
	}
	a.mu.Unlock()
	return fmt.Errorf("profile %s not found", name)
}

func (a *App) RenameProfile(oldName, newName string) error {
	a.mu.Lock()

	oldName = strings.TrimSpace(oldName)
	newName = strings.TrimSpace(newName)
	if oldName == "" || newName == "" {
		a.mu.Unlock()
		return fmt.Errorf("name is required")
	}
	if oldName == newName {
		a.mu.Unlock()
		return nil
	}
	for _, p := range a.profiles {
		if p.Name == newName {
			a.mu.Unlock()
			return fmt.Errorf("profile %s already exists", newName)
		}
	}

	for i := range a.profiles {
		if a.profiles[i].Name != oldName {
			continue
		}
		wasActive := a.profiles[i].SystemHostsActive
		listenIP := a.profiles[i].ListenIP
		port := a.profiles[i].Port
		hostsRules := make(map[string]string, len(a.profiles[i].Hosts))
		for k, v := range a.profiles[i].Hosts {
			hostsRules[k] = v
		}

		// Attempt to rename default hosts file path if it follows configs/hosts/<name>.hosts
		oldHostsRel := filepath.Join("configs", "hosts", oldName+".hosts")
		newHostsRel := filepath.Join("configs", "hosts", newName+".hosts")

		if filepath.Clean(a.profiles[i].HostsFile) == filepath.Clean(oldHostsRel) {
			oldHostsAbs := filepath.Join(a.exeDir, oldHostsRel)
			newHostsAbs := filepath.Join(a.exeDir, newHostsRel)
			_ = os.MkdirAll(filepath.Dir(newHostsAbs), 0755)
			if _, err := os.Stat(oldHostsAbs); err == nil {
				_ = os.Rename(oldHostsAbs, newHostsAbs)
			}
			a.profiles[i].HostsFile = newHostsRel
		}

		a.profiles[i].Name = newName
		a.refreshProfileHostsLocked(i)
		if err := a.saveConfig(); err != nil {
			a.mu.Unlock()
			return err
		}
		a.mu.Unlock()

		a.proxyManager.StopProxy(oldName)
		_ = a.proxyManager.StartProxy(newName, listenIP, port, hostsRules)
		a.refreshProxyStatus()

		if wasActive {
			_ = winhosts.RemoveProfileBlock(oldName, false)
			if err := a.StartProfile(newName); err != nil {
				return err
			}
		}

		return nil
	}
	a.mu.Unlock()
	return fmt.Errorf("profile %s not found", oldName)
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

	home, _ := os.UserHomeDir()
	desktopDir := filepath.Join(home, "Desktop")

	savePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:            "Export hosts",
		DefaultDirectory: desktopDir,
		DefaultFilename:  profileName + ".txt",
		Filters: []runtime.FileFilter{
			{DisplayName: "Text Files (*.txt)", Pattern: "*.txt"},
			{DisplayName: "All files", Pattern: "*.*"},
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

func (a *App) GetProxyAddress(profileName string) (string, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	for _, p := range a.profiles {
		if p.Name == profileName {
			return fmt.Sprintf("%s:%d", p.ListenIP, p.Port), nil
		}
	}
	return "", fmt.Errorf("profile %s not found", profileName)
}

func (a *App) profileHostsPath(profileName string) (string, error) {
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
		return hostsPath, nil
	}
	return "", fmt.Errorf("profile %s not found", profileName)
}

func (a *App) backupDir(profileName string) string {
	return filepath.Join(a.exeDir, "configs", "backups", profileName)
}

func (a *App) CreateHostsBackup(profileName string) (string, error) {
	hostsPath, err := a.profileHostsPath(profileName)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(hostsPath)
	if err != nil {
		return "", err
	}
	dir := a.backupDir(profileName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	name := "snapshot_" + time.Now().Format("20060102150405") + ".hosts"
	dst := filepath.Join(dir, name)
	if err := os.WriteFile(dst, data, 0644); err != nil {
		return "", err
	}
	return name, nil
}

func (a *App) ListHostsBackups(profileName string) ([]BackupInfo, error) {
	dir := a.backupDir(profileName)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []BackupInfo{}, nil
		}
		return nil, err
	}
	out := make([]BackupInfo, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		out = append(out, BackupInfo{
			FileName: e.Name(),
			Path:     filepath.Join(dir, e.Name()),
			Size:     info.Size(),
			Modified: info.ModTime().Format("2006-01-02 15:04:05"),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Modified > out[j].Modified })
	return out, nil
}

func (a *App) RestoreHostsBackup(profileName, fileName string) error {
	hostsPath, err := a.profileHostsPath(profileName)
	if err != nil {
		return err
	}
	src := filepath.Join(a.backupDir(profileName), fileName)
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if err := os.WriteFile(hostsPath, data, 0644); err != nil {
		return err
	}

	a.mu.Lock()
	for i := range a.profiles {
		if a.profiles[i].Name == profileName {
			a.refreshProfileHostsLocked(i)
			break
		}
	}
	active := false
	for i := range a.profiles {
		if a.profiles[i].Name == profileName {
			active = a.profiles[i].SystemHostsActive
			break
		}
	}
	a.mu.Unlock()

	if active {
		return a.StartProfile(profileName)
	}
	return nil
}

func (a *App) ClearHostsEntries(profileName string) error {
	return a.SetHostsText(profileName, "")
}

func (a *App) ResetHostsTemplate(profileName string) error {
	text := `# This file is managed by OvOproxy
# Add entries in the format: IP DOMAIN
# Example:
# 120.92.124.158 account.wps.cn
`
	return a.SetHostsText(profileName, text)
}

// --- Subscription methods ---

const subStartMarker = "# >>> subscription (auto-managed, do not edit)"
const subEndMarker = "# <<< subscription"

func (a *App) SetSubscription(name, url string, interval int) error {
	a.mu.Lock()
	found := false
	for i := range a.profiles {
		if a.profiles[i].Name == name {
			a.profiles[i].SubscriptionURL = url
			a.profiles[i].SubscriptionInterval = interval
			a.profiles[i].SubscriptionEnabled = true
			found = true
			break
		}
	}
	if !found {
		a.mu.Unlock()
		return fmt.Errorf("profile %s not found", name)
	}
	if err := a.saveConfig(); err != nil {
		a.mu.Unlock()
		return err
	}
	a.mu.Unlock()

	// 立即触发一次刷新
	_, _ = a.RefreshSubscription(name)
	return nil
}

func (a *App) RemoveSubscription(name string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	for i := range a.profiles {
		if a.profiles[i].Name == name {
			a.profiles[i].SubscriptionURL = ""
			a.profiles[i].SubscriptionInterval = 0
			a.profiles[i].SubscriptionEnabled = false
			return a.saveConfig()
		}
	}
	return fmt.Errorf("profile %s not found", name)
}

func (a *App) RefreshSubscription(name string) (SubscriptionResult, error) {
	var url string
	var hostsFile string
	var isActive bool
	a.mu.RLock()
	for i := range a.profiles {
		if a.profiles[i].Name == name {
			url = a.profiles[i].SubscriptionURL
			hostsFile = a.profiles[i].HostsFile
			isActive = a.profiles[i].SystemHostsActive
			break
		}
	}
	exeDir := a.exeDir
	a.mu.RUnlock()

	a.refreshMu.Lock()
	defer a.refreshMu.Unlock()

	if url == "" {
		return SubscriptionResult{Status: "error", Message: "未设置订阅 URL"}, nil
	}
	if !filepath.IsAbs(hostsFile) {
		hostsFile = filepath.Join(exeDir, hostsFile)
	}

	// 拉取远程内容
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		now := time.Now().Format(time.RFC3339)
		a.updateLastFetch(name, now)
		return SubscriptionResult{Status: "error", Message: fmt.Sprintf("请求失败: %v", err), LastFetch: now}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		now := time.Now().Format(time.RFC3339)
		a.updateLastFetch(name, now)
		return SubscriptionResult{Status: "error", Message: fmt.Sprintf("HTTP %d", resp.StatusCode), LastFetch: now}, nil
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024)) // 10MB limit
	if err != nil {
		now := time.Now().Format(time.RFC3339)
		a.updateLastFetch(name, now)
		return SubscriptionResult{Status: "error", Message: fmt.Sprintf("读取失败: %v", err), LastFetch: now}, nil
	}

	entries, _, err := hosts.ParseText(string(body))
	if err != nil {
		now := time.Now().Format(time.RFC3339)
		a.updateLastFetch(name, now)
		return SubscriptionResult{Status: "error", Message: fmt.Sprintf("解析失败: %v", err), LastFetch: now}, nil
	}

	// 构建新条目行
	newLines := make([]string, len(entries))
	for i, e := range entries {
		newLines[i] = fmt.Sprintf("%s %s", e.IP, e.Domain)
	}
	newBody := strings.Join(newLines, "\n")

	// 读取现有 hosts，提取旧订阅条目行用于比对
	existing, err := os.ReadFile(hostsFile)
	if err != nil && !os.IsNotExist(err) {
		return SubscriptionResult{Status: "error", Message: fmt.Sprintf("读取本地文件失败: %v", err)}, nil
	}
	existingStr := string(existing)

	// 查找并提取旧订阅块中的条目行
	oldBody := extractSubBody(existingStr)

	// 内容无变化则跳过写盘
	if oldBody == newBody {
		now := time.Now().Format(time.RFC3339)
		a.updateLastFetch(name, now)
		return SubscriptionResult{Status: "ok", Message: "无变化", LastFetch: now, EntryCount: len(entries)}, nil
	}

	// 构建新订阅块
	var subLines []string
	subLines = append(subLines, subStartMarker+" "+url)
	subLines = append(subLines, newLines...)
	subLines = append(subLines, subEndMarker)
	subBlock := strings.Join(subLines, "\n") + "\n"

	// 移除旧订阅块
	startIdx := subBlockStart(existingStr)
	if startIdx >= 0 {
		endIdx := strings.Index(existingStr[startIdx:], subEndMarker)
		if endIdx >= 0 {
			endIdx += startIdx + len(subEndMarker)
			if endIdx < len(existingStr) && existingStr[endIdx] == '\n' {
				endIdx++
			}
			existingStr = existingStr[:startIdx] + existingStr[endIdx:]
		} else {
			existingStr = existingStr[:startIdx]
		}
	}

	// 追加新订阅块
	existingStr = strings.TrimRight(existingStr, "\n") + "\n\n" + subBlock

	if err := os.MkdirAll(filepath.Dir(hostsFile), 0755); err != nil {
		return SubscriptionResult{Status: "error", Message: fmt.Sprintf("创建目录失败: %v", err)}, nil
	}
	if err := os.WriteFile(hostsFile, []byte(existingStr), 0644); err != nil {
		return SubscriptionResult{Status: "error", Message: fmt.Sprintf("写入失败: %v", err)}, nil
	}

	// 更新内存中的 hosts map + 代理规则
	a.mu.Lock()
	for i := range a.profiles {
		if a.profiles[i].Name == name {
			a.refreshProfileHostsLocked(i)
			break
		}
	}
	a.mu.Unlock()

	now := time.Now().Format(time.RFC3339)
	a.updateLastFetch(name, now)

	// 如果已启用，重新写入系统 hosts
	if isActive {
		_ = a.StartProfile(name)
	}

	return SubscriptionResult{
		Status:     "ok",
		Message:    "刷新成功",
		LastFetch:  now,
		EntryCount: len(entries),
	}, nil
}

// 提取 hosts 文件中订阅块的条目内容（去掉标记行），用于比对
func extractSubBody(text string) string {
	idx := subBlockStart(text)
	if idx < 0 {
		return ""
	}
	start := strings.Index(text[idx:], "\n")
	if start < 0 {
		return ""
	}
	start += idx + 1
	end := strings.Index(text[start:], subEndMarker)
	if end < 0 {
		return ""
	}
	return strings.TrimSpace(text[start : start+end])
}

// 查找订阅块起始位置，兼容新旧标记
func subBlockStart(text string) int {
	for _, marker := range []string{subStartMarker, "# >>> subscription"} {
		if idx := strings.Index(text, marker); idx >= 0 {
			return idx
		}
	}
	return -1
}

func (a *App) updateLastFetch(name, timestamp string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	for i := range a.profiles {
		if a.profiles[i].Name == name {
			a.profiles[i].SubscriptionLastFetch = timestamp
			a.configDirty = true
			return
		}
	}
}

func (a *App) startSubscriptionAutoRefresh() {
	ctx, cancel := context.WithCancel(context.Background())
	a.subscriptionCancel = cancel

	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		lastRefresh := make(map[string]time.Time)

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				a.mu.RLock()
				type subInfo struct {
					name     string
					interval int
				}
				var subs []subInfo
				for _, p := range a.profiles {
					if p.SubscriptionEnabled && p.SubscriptionInterval > 0 && p.SubscriptionURL != "" {
						subs = append(subs, subInfo{name: p.Name, interval: p.SubscriptionInterval})
					}
				}
				a.mu.RUnlock()

				for _, s := range subs {
					if last, ok := lastRefresh[s.name]; ok {
						if time.Since(last) < time.Duration(s.interval)*time.Second {
							continue
						}
					}
					_, _ = a.RefreshSubscription(s.name)
					lastRefresh[s.name] = time.Now()
				}
			}
		}
	}()
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

func (a *App) startConfigBatcher() {
	ctx, cancel := context.WithCancel(context.Background())
	a.configSaveCancel = cancel

	go func() {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				a.configFlush()
			}
		}
	}()
}

func (a *App) configFlush() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if !a.configDirty {
		return
	}
	a.configDirty = false
	_ = a.saveConfig()
}
