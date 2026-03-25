package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
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
	ctx          context.Context
	profiles     []ProfileState
	mu           sync.RWMutex
	exeDir       string
	allowQuit    bool
	proxyManager *proxymanager.Manager
	refreshState map[string]*subscriptionRefreshRuntime
	refreshTasks map[string]chan struct{}
	refreshBusy  map[string]bool
	refreshHist  map[string][]SubscriptionRefreshReport
	auditLogs    []AuditLogEntry
}

type ProfileState struct {
	config.Profile
	Running           bool              `json:"running"`
	Hosts             map[string]string `json:"hosts"`
	DuplicateDomains  []DuplicateDomain `json:"duplicate_domains"`
	SystemHostsActive bool              `json:"system_hosts_active"`
	ProxyActive       bool              `json:"proxy_active"`
	ProxyError        string            `json:"proxy_error"`
}

type HostEntry struct {
	Domain string `json:"domain"`
	IP     string `json:"ip"`
}

type DuplicateDomain struct {
	Domain string `json:"domain"`
	Count  int    `json:"count"`
}

type BackupInfo struct {
	FileName string `json:"file_name"`
	Path     string `json:"path"`
	Size     int64  `json:"size"`
	Modified string `json:"modified"`
}

type AuditLogEntry struct {
	Time    string `json:"time"`
	Action  string `json:"action"`
	Profile string `json:"profile"`
	Detail  string `json:"detail"`
	Success bool   `json:"success"`
}

type SubscriptionConflictPreview struct {
	SubID    string                     `json:"sub_id"`
	SubName  string                     `json:"sub_name"`
	Domains  []string                   `json:"domains"`
	Items    []SubscriptionConflictItem `json:"items"`
	Total    int                        `json:"total"`
	Truncate bool                       `json:"truncate"`
}

type SubscriptionConflictItem struct {
	Domain   string `json:"domain"`
	LocalIP  string `json:"local_ip"`
	RemoteIP string `json:"remote_ip"`
}

type SubscriptionRefreshFailure struct {
	SubID   string `json:"sub_id"`
	SubName string `json:"sub_name"`
	Reason  string `json:"reason"`
}

type SubscriptionRefreshReport struct {
	Time         string                       `json:"time"`
	Source       string                       `json:"source"`
	Success      bool                         `json:"success"`
	EnabledTotal int                          `json:"enabled_total"`
	SuccessTotal int                          `json:"success_total"`
	FailedTotal  int                          `json:"failed_total"`
	AddedTotal   int                          `json:"added_total"`
	ConflictDiff int                          `json:"conflict_diff"`
	ConflictSame int                          `json:"conflict_same"`
	Failures     []SubscriptionRefreshFailure `json:"failures"`
}

type subscriptionRefreshRuntime struct {
	FailCount    int
	NextDelaySec int
}

const (
	subscriptionBlockStart = "# >>> Zephy Subscriptions Start"
	subscriptionBlockEnd   = "# <<< Zephy Subscriptions End"
)

const (
	conflictResolveKeepLocal  = "keep_local"
	conflictResolveUseRemote  = "use_remote"
	conflictResolvePreviewCap = 30
	minAutoRefreshIntervalSec = 30
	defaultAutoBackoffSec     = 900
	defaultAutoHistoryLimit   = 20
)

func NewApp() *App {
	exeDir := "."
	if exe, err := os.Executable(); err == nil {
		exeDir = filepath.Dir(exe)
	}
	return &App{
		profiles:     []ProfileState{},
		exeDir:       exeDir,
		proxyManager: proxymanager.New(),
		refreshState: map[string]*subscriptionRefreshRuntime{},
		refreshTasks: map[string]chan struct{}{},
		refreshBusy:  map[string]bool{},
		refreshHist:  map[string][]SubscriptionRefreshReport{},
		auditLogs:    []AuditLogEntry{},
	}
}

func (a *App) addAudit(action, profile, detail string, success bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	entry := AuditLogEntry{
		Time:    time.Now().Format("2006-01-02 15:04:05"),
		Action:  action,
		Profile: profile,
		Detail:  detail,
		Success: success,
	}
	a.auditLogs = append(a.auditLogs, entry)
	if len(a.auditLogs) > 500 {
		a.auditLogs = a.auditLogs[len(a.auditLogs)-500:]
	}
}

func (a *App) GetAuditLogs(limit int) []AuditLogEntry {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if limit <= 0 {
		limit = 100
	}
	if len(a.auditLogs) == 0 {
		return []AuditLogEntry{}
	}
	if limit > len(a.auditLogs) {
		limit = len(a.auditLogs)
	}
	out := make([]AuditLogEntry, limit)
	copy(out, a.auditLogs[len(a.auditLogs)-limit:])
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}

func (a *App) RelaunchAsAdmin() error {
	admin, err := winhosts.IsAdmin()
	if err != nil {
		a.addAudit("admin.relaunch", "", "admin check failed: "+err.Error(), false)
		return err
	}
	if admin {
		a.addAudit("admin.relaunch", "", "already admin", true)
		return nil
	}

	exe, err := os.Executable()
	if err != nil {
		a.addAudit("admin.relaunch", "", "resolve executable failed: "+err.Error(), false)
		return err
	}
	quotedExe := strings.ReplaceAll(exe, "'", "''")
	psInner := fmt.Sprintf("Start-Sleep -Milliseconds 800; Start-Process -FilePath '%s'", quotedExe)
	cmd := exec.Command("powershell", "-NoProfile", "-Command",
		fmt.Sprintf("Start-Process powershell -Verb RunAs -ArgumentList '-NoProfile -Command \"%s\"'", psInner))
	if err := cmd.Start(); err != nil {
		a.addAudit("admin.relaunch", "", "launch failed: "+err.Error(), false)
		return err
	}
	a.addAudit("admin.relaunch", "", "requested elevation and relaunch", true)
	go func() {
		time.Sleep(300 * time.Millisecond)
		if a.ctx != nil {
			runtime.Quit(a.ctx)
		}
	}()
	return nil
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.LoadConfig()
	a.startTray()
	go func() {
		a.startAllProxies()
		a.syncHostsEnabledState()
		a.refreshAutoSchedulers()
		if a.ctx != nil {
			runtime.EventsEmit(a.ctx, "profiles:changed")
		}
	}()
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
	a.stopAllAutoSchedulers()
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
		a.refreshTasks = map[string]chan struct{}{}
		a.refreshState = map[string]*subscriptionRefreshRuntime{}
		a.refreshBusy = map[string]bool{}
		a.refreshHist = map[string][]SubscriptionRefreshReport{}
		return nil
	}

	a.profiles = make([]ProfileState, len(cfg.Profiles))
	for i, p := range cfg.Profiles {
		normalizeRefreshSettings(&p.SubscriptionRefresh)
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
			DuplicateDomains:  p.DuplicateDomains,
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
	for i := range a.profiles {
		if a.profiles[i].Name != profileName {
			continue
		}
		hostsPath := a.profiles[i].HostsFile
		if !filepath.IsAbs(hostsPath) {
			hostsPath = filepath.Join(a.exeDir, hostsPath)
		}
		if err := os.MkdirAll(filepath.Dir(hostsPath), 0755); err != nil {
			a.mu.Unlock()
			a.addAudit("hosts.save_text", profileName, "mkdir failed: "+err.Error(), false)
			return err
		}
		if err := os.WriteFile(hostsPath, []byte(text), 0644); err != nil {
			a.mu.Unlock()
			a.addAudit("hosts.save_text", profileName, "write failed: "+err.Error(), false)
			return err
		}
		a.refreshProfileHostsLocked(i)
		a.mu.Unlock()
		a.addAudit("hosts.save_text", profileName, "saved hosts text", true)
		return nil
	}
	a.mu.Unlock()
	a.addAudit("hosts.save_text", profileName, "profile not found", false)
	return fmt.Errorf("profile %s not found", profileName)
}

func (a *App) IsAdmin() (bool, error) {
	return winhosts.IsAdmin()
}

// StartProfile == 启用 Profile：写入系统 hosts 的该 Profile 标记块
// 默认单启用策略：启用新 Profile 时自动禁用其他 enabled 的 Profile
func (a *App) StartProfile(name string) error {
	a.mu.RLock()
	var hostsPath string
	var proxyActive bool
	var proxyErr string
	for i := range a.profiles {
		if a.profiles[i].Name == name {
			hostsPath = a.profiles[i].HostsFile
			proxyActive = a.profiles[i].ProxyActive
			proxyErr = a.profiles[i].ProxyError
			break
		}
	}
	exeDir := a.exeDir
	a.mu.RUnlock()

	if hostsPath == "" {
		a.addAudit("hosts.enable_profile", name, "profile not found", false)
		return fmt.Errorf("profile %s not found", name)
	}
	if !proxyActive {
		a.addAudit("hosts.enable_profile", name, "proxy not active: "+proxyErr, false)
		return fmt.Errorf("代理端口未启动: %s", proxyErr)
	}
	if !filepath.IsAbs(hostsPath) {
		hostsPath = filepath.Join(exeDir, hostsPath)
	}

	text, err := os.ReadFile(hostsPath)
	if err != nil {
		a.addAudit("hosts.enable_profile", name, "read profile hosts failed: "+err.Error(), false)
		return fmt.Errorf("read profile hosts: %w", err)
	}

	entries, _, err := hosts.ParseText(string(text))
	if err != nil {
		a.addAudit("hosts.enable_profile", name, "parse profile hosts failed: "+err.Error(), false)
		return fmt.Errorf("parse profile hosts: %w", err)
	}

	dedup := hosts.DedupEntriesKeepLast(entries)
	lines := make([]string, 0, len(dedup))
	for _, e := range dedup {
		lines = append(lines, fmt.Sprintf("%s %s", e.IP, e.Domain))
	}

	a.mu.RLock()
	var otherEnabled []string
	for i := range a.profiles {
		if a.profiles[i].SystemHostsActive && a.profiles[i].Name != name {
			otherEnabled = append(otherEnabled, a.profiles[i].Name)
		}
	}
	a.mu.RUnlock()

	for _, other := range otherEnabled {
		_ = winhosts.RemoveProfileBlock(other, false)
	}

	if err := winhosts.ApplyProfileBlock(name, lines, true); err != nil {
		a.addAudit("hosts.enable_profile", name, "apply profile block failed: "+err.Error(), false)
		return err
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	for i := range a.profiles {
		a.profiles[i].SystemHostsActive = (a.profiles[i].Name == name)
		a.profiles[i].Running = (a.profiles[i].Name == name)
	}
	go a.addAudit("hosts.enable_profile", name, "enabled profile block", true)
	return nil
}

// StopProfile == 关闭 Profile：移除该 Profile 的 hosts 标记块
func (a *App) StopProfile(name string) error {
	if err := winhosts.RemoveProfileBlock(name, true); err != nil {
		a.addAudit("hosts.disable_profile", name, "remove profile block failed: "+err.Error(), false)
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
	go a.addAudit("hosts.disable_profile", name, "disabled profile block", true)
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

	for _, p := range a.profiles {
		if p.Name == name {
			a.mu.Unlock()
			a.addAudit("profile.add", name, "duplicate name", false)
			return fmt.Errorf("profile %s already exists", name)
		}
		if p.ListenIP == listenIP && p.Port == port {
			a.mu.Unlock()
			a.addAudit("profile.add", name, "duplicate listen address", false)
			return fmt.Errorf("address %s:%d already in use", listenIP, port)
		}
	}

	hostsFile := filepath.Join("configs", "hosts", name+".hosts")
	hostsPath := filepath.Join(a.exeDir, hostsFile)

	if err := os.MkdirAll(filepath.Dir(hostsPath), 0755); err != nil {
		a.mu.Unlock()
		a.addAudit("profile.add", name, "create hosts dir failed: "+err.Error(), false)
		return err
	}
	if err := os.WriteFile(hostsPath, []byte("# Hosts for "+name+"\n"), 0644); err != nil {
		a.mu.Unlock()
		a.addAudit("profile.add", name, "create hosts file failed: "+err.Error(), false)
		return err
	}

	newProfile := ProfileState{
		Profile: config.Profile{
			Name:     name,
			ListenIP: listenIP,
			Port:     port,
			HostsFile: hostsFile,
			SubscriptionRefresh: config.SubscriptionRefreshSettings{
				AutoEnabled:       false,
				IntervalSeconds:   600,
				MaxBackoffSeconds: defaultAutoBackoffSec,
				HistoryLimit:      defaultAutoHistoryLimit,
			},
		},
		Running: false,
		Hosts:   make(map[string]string),
	}
	a.profiles = append(a.profiles, newProfile)

	if err := a.saveConfig(); err != nil {
		a.mu.Unlock()
		a.addAudit("profile.add", name, "save config failed: "+err.Error(), false)
		return err
	}
	hostsRules := newProfile.Hosts
	a.mu.Unlock()

	_ = a.proxyManager.StartProxy(name, listenIP, port, hostsRules)
	a.refreshProxyStatus()
	a.refreshAutoSchedulers()
	a.addAudit("profile.add", name, fmt.Sprintf("added profile %s:%d", listenIP, port), true)
	return nil
}

func (a *App) DeleteProfile(name string) error {
	a.mu.Lock()

	for i := range a.profiles {
		if a.profiles[i].Name == name {
			if a.profiles[i].Running || a.profiles[i].SystemHostsActive {
				a.mu.Unlock()
				a.addAudit("profile.delete", name, "profile is active", false)
				return fmt.Errorf("cannot delete running profile")
			}
			a.profiles = append(a.profiles[:i], a.profiles[i+1:]...)
			err := a.saveConfig()
			a.mu.Unlock()

			a.proxyManager.StopProxy(name)
			a.refreshAutoSchedulers()
			if err != nil {
				a.addAudit("profile.delete", name, "save config failed: "+err.Error(), false)
			} else {
				a.addAudit("profile.delete", name, "profile deleted", true)
			}
			return err
		}
	}
	a.mu.Unlock()
	a.addAudit("profile.delete", name, "profile not found", false)
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
		if err := a.saveConfig(); err != nil {
			go a.addAudit("profile.rename", oldName, "save config failed: "+err.Error(), false)
			return err
		}
		go a.addAudit("profile.rename", newName, "renamed from "+oldName, true)
		return nil
	}
	go a.addAudit("profile.rename", oldName, "profile not found", false)
	return fmt.Errorf("profile %s not found", oldName)
}

func (a *App) UpdateHosts(profileName string, entries []HostEntry) error {
	a.mu.Lock()
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
				a.mu.Unlock()
				a.addAudit("hosts.update_entries", profileName, "write failed: "+err.Error(), false)
				return err
			}

			a.profiles[i].Hosts = newHosts
			a.refreshProfileHostsLocked(i)
			a.mu.Unlock()
			a.addAudit("hosts.update_entries", profileName, fmt.Sprintf("updated %d entries", len(newHosts)), true)
			return nil
		}
	}
	a.mu.Unlock()
	a.addAudit("hosts.update_entries", profileName, "profile not found", false)
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

type ProxyLogEntry struct {
	Time       string `json:"time"`
	Method     string `json:"method"`
	Host       string `json:"host"`
	ResolvedIP string `json:"resolved_ip"`
	Success    bool   `json:"success"`
	Error      string `json:"error,omitempty"`
}

func (a *App) GetProxyLogs(profileName string, limit int) []ProxyLogEntry {
	logs := a.proxyManager.GetLogs(profileName, limit)
	result := make([]ProxyLogEntry, len(logs))
	for i, l := range logs {
		result[i] = ProxyLogEntry{
			Time:       l.Time.Format("15:04:05"),
			Method:     l.Method,
			Host:       l.Host,
			ResolvedIP: l.ResolvedIP,
			Success:    l.Success,
			Error:      l.Error,
		}
	}
	return result
}

func (a *App) GetProfileSubscriptions(profileName string) ([]config.Subscription, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	for i := range a.profiles {
		if a.profiles[i].Name == profileName {
			subs := make([]config.Subscription, len(a.profiles[i].Subscriptions))
			copy(subs, a.profiles[i].Subscriptions)
			return subs, nil
		}
	}
	return nil, fmt.Errorf("profile %s not found", profileName)
}

func (a *App) AddProfileSubscription(profileName, subName, subURL string) error {
	subName = strings.TrimSpace(subName)
	subURL = strings.TrimSpace(subURL)
	if subURL == "" {
		return fmt.Errorf("subscription url is required")
	}
	u, err := url.Parse(subURL)
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return fmt.Errorf("subscription url must be valid https url")
	}
	if subName == "" {
		subName = u.Host
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	for i := range a.profiles {
		if a.profiles[i].Name != profileName {
			continue
		}
		for _, s := range a.profiles[i].Subscriptions {
			if strings.EqualFold(s.URL, subURL) {
				return fmt.Errorf("subscription already exists")
			}
		}
		id := strconv.FormatInt(time.Now().UnixNano(), 10)
		a.profiles[i].Subscriptions = append(a.profiles[i].Subscriptions, config.Subscription{
			ID:      id,
			Name:    subName,
			URL:     subURL,
			Enabled: true,
		})
		return a.saveConfig()
	}
	return fmt.Errorf("profile %s not found", profileName)
}

func (a *App) RemoveProfileSubscription(profileName, subID string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	for i := range a.profiles {
		if a.profiles[i].Name != profileName {
			continue
		}
		for j := range a.profiles[i].Subscriptions {
			if a.profiles[i].Subscriptions[j].ID == subID {
				a.profiles[i].Subscriptions = append(a.profiles[i].Subscriptions[:j], a.profiles[i].Subscriptions[j+1:]...)
				return a.saveConfig()
			}
		}
		return fmt.Errorf("subscription not found")
	}
	return fmt.Errorf("profile %s not found", profileName)
}

func (a *App) SetProfileSubscriptionEnabled(profileName, subID string, enabled bool) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	for i := range a.profiles {
		if a.profiles[i].Name != profileName {
			continue
		}
		for j := range a.profiles[i].Subscriptions {
			if a.profiles[i].Subscriptions[j].ID == subID {
				a.profiles[i].Subscriptions[j].Enabled = enabled
				return a.saveConfig()
			}
		}
		return fmt.Errorf("subscription not found")
	}
	return fmt.Errorf("profile %s not found", profileName)
}

func (a *App) SetAllProfileSubscriptionsEnabled(profileName string, enabled bool) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	for i := range a.profiles {
		if a.profiles[i].Name != profileName {
			continue
		}
		changed := false
		for j := range a.profiles[i].Subscriptions {
			if a.profiles[i].Subscriptions[j].Enabled != enabled {
				a.profiles[i].Subscriptions[j].Enabled = enabled
				changed = true
			}
		}
		if !changed {
			return nil
		}
		return a.saveConfig()
	}
	return fmt.Errorf("profile %s not found", profileName)
}

func (a *App) UpdateProfileSubscription(profileName, subID, subName, subURL string) error {
	subName = strings.TrimSpace(subName)
	subURL = strings.TrimSpace(subURL)
	if subURL == "" {
		return fmt.Errorf("subscription url is required")
	}
	u, err := url.Parse(subURL)
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return fmt.Errorf("subscription url must be valid https url")
	}
	if subName == "" {
		subName = u.Host
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	for i := range a.profiles {
		if a.profiles[i].Name != profileName {
			continue
		}
		target := -1
		for j := range a.profiles[i].Subscriptions {
			if a.profiles[i].Subscriptions[j].ID == subID {
				target = j
				break
			}
		}
		if target < 0 {
			return fmt.Errorf("subscription not found")
		}
		for j := range a.profiles[i].Subscriptions {
			if j == target {
				continue
			}
			if strings.EqualFold(a.profiles[i].Subscriptions[j].URL, subURL) {
				return fmt.Errorf("subscription already exists")
			}
		}
		a.profiles[i].Subscriptions[target].Name = subName
		a.profiles[i].Subscriptions[target].URL = subURL
		return a.saveConfig()
	}
	return fmt.Errorf("profile %s not found", profileName)
}

func normalizeRefreshSettings(s *config.SubscriptionRefreshSettings) {
	if s.IntervalSeconds <= 0 {
		s.IntervalSeconds = 600
	}
	if s.MaxBackoffSeconds <= 0 {
		s.MaxBackoffSeconds = defaultAutoBackoffSec
	}
	if s.HistoryLimit <= 0 {
		s.HistoryLimit = defaultAutoHistoryLimit
	}
	if s.IntervalSeconds < minAutoRefreshIntervalSec {
		s.IntervalSeconds = minAutoRefreshIntervalSec
	}
	if s.MaxBackoffSeconds < s.IntervalSeconds {
		s.MaxBackoffSeconds = s.IntervalSeconds
	}
	if s.HistoryLimit < 5 {
		s.HistoryLimit = 5
	}
}

func (a *App) GetSubscriptionRefreshSettings(profileName string) (config.SubscriptionRefreshSettings, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	for i := range a.profiles {
		if a.profiles[i].Name != profileName {
			continue
		}
		s := a.profiles[i].SubscriptionRefresh
		normalizeRefreshSettings(&s)
		return s, nil
	}
	return config.SubscriptionRefreshSettings{}, fmt.Errorf("profile %s not found", profileName)
}

func (a *App) UpdateSubscriptionRefreshSettings(profileName string, autoEnabled bool, intervalSeconds int, maxBackoffSeconds int, historyLimit int) error {
	next := config.SubscriptionRefreshSettings{
		AutoEnabled:       autoEnabled,
		IntervalSeconds:   intervalSeconds,
		MaxBackoffSeconds: maxBackoffSeconds,
		HistoryLimit:      historyLimit,
	}
	normalizeRefreshSettings(&next)

	a.mu.Lock()
	for i := range a.profiles {
		if a.profiles[i].Name != profileName {
			continue
		}
		a.profiles[i].SubscriptionRefresh = next
		if err := a.saveConfig(); err != nil {
			a.mu.Unlock()
			a.addAudit("subscriptions.update_refresh_settings", profileName, "save config failed: "+err.Error(), false)
			return err
		}
		a.mu.Unlock()
		a.addAudit("subscriptions.update_refresh_settings", profileName, fmt.Sprintf("auto=%v interval=%ds", next.AutoEnabled, next.IntervalSeconds), true)
		go a.refreshAutoSchedulers()
		return nil
	}
	a.mu.Unlock()
	a.addAudit("subscriptions.update_refresh_settings", profileName, "profile not found", false)
	return fmt.Errorf("profile %s not found", profileName)
}

func (a *App) GetSubscriptionRefreshHistory(profileName string, limit int) ([]SubscriptionRefreshReport, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if limit <= 0 {
		limit = defaultAutoHistoryLimit
	}
	if _, ok := a.findProfileLocked(profileName); !ok {
		return nil, fmt.Errorf("profile %s not found", profileName)
	}
	h := a.refreshHist[profileName]
	if len(h) == 0 {
		return []SubscriptionRefreshReport{}, nil
	}
	if limit > len(h) {
		limit = len(h)
	}
	out := make([]SubscriptionRefreshReport, limit)
	copy(out, h[len(h)-limit:])
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out, nil
}

func (a *App) IsSubscriptionRefreshRunning(profileName string) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.refreshBusy[profileName]
}

func (a *App) findProfileLocked(profileName string) (*ProfileState, bool) {
	for i := range a.profiles {
		if a.profiles[i].Name == profileName {
			return &a.profiles[i], true
		}
	}
	return nil, false
}

func (a *App) appendRefreshHistoryLocked(profileName string, report SubscriptionRefreshReport) {
	p, ok := a.findProfileLocked(profileName)
	if !ok {
		return
	}
	limit := p.SubscriptionRefresh.HistoryLimit
	if limit <= 0 {
		limit = defaultAutoHistoryLimit
	}
	h := append(a.refreshHist[profileName], report)
	if len(h) > limit {
		h = h[len(h)-limit:]
	}
	a.refreshHist[profileName] = h
}

func (a *App) computeBackoffDelaySeconds(failCount int, maxBackoffSec int) int {
	if failCount <= 0 {
		return minAutoRefreshIntervalSec
	}
	base := minAutoRefreshIntervalSec
	for i := 1; i < failCount; i++ {
		base *= 2
		if base >= maxBackoffSec {
			base = maxBackoffSec
			break
		}
	}
	if base > maxBackoffSec {
		base = maxBackoffSec
	}
	jitter := base / 5
	if jitter <= 0 {
		return base
	}
	seeded := rand.New(rand.NewSource(time.Now().UnixNano()))
	delta := seeded.Intn((2*jitter)+1) - jitter
	n := base + delta
	if n < minAutoRefreshIntervalSec {
		n = minAutoRefreshIntervalSec
	}
	if n > maxBackoffSec {
		n = maxBackoffSec
	}
	return n
}

func (a *App) executeSubscriptionRefresh(profileName, source string, include map[string]struct{}) (SubscriptionRefreshReport, error) {
	a.mu.Lock()
	if a.refreshBusy[profileName] {
		h := a.refreshHist[profileName]
		a.mu.Unlock()
		if len(h) > 0 {
			last := h[len(h)-1]
			return last, nil
		}
		return SubscriptionRefreshReport{
			Time:    time.Now().Format("2006-01-02 15:04:05"),
			Source:  source,
			Success: false,
		}, nil
	}
	a.refreshBusy[profileName] = true
	a.mu.Unlock()

	defer func() {
		a.mu.Lock()
		a.refreshBusy[profileName] = false
		a.mu.Unlock()
	}()

	report, err := a.refreshProfileSubscriptionsWithFilter(profileName, include)
	report.Source = source
	report.Success = err == nil && report.FailedTotal == 0
	if err != nil && len(report.Failures) == 0 {
		report.Failures = append(report.Failures, SubscriptionRefreshFailure{
			SubName: profileName,
			Reason:  err.Error(),
		})
	}

	a.mu.Lock()
	a.appendRefreshHistoryLocked(profileName, report)
	if st, ok := a.refreshState[profileName]; ok {
		if report.Success {
			st.FailCount = 0
			p, has := a.findProfileLocked(profileName)
			if has {
				st.NextDelaySec = p.SubscriptionRefresh.IntervalSeconds
			}
		} else {
			st.FailCount++
			p, has := a.findProfileLocked(profileName)
			if has {
				st.NextDelaySec = a.computeBackoffDelaySeconds(st.FailCount, p.SubscriptionRefresh.MaxBackoffSeconds)
			}
		}
	}
	a.mu.Unlock()

	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "subscriptions:historyChanged", profileName)
		runtime.EventsEmit(a.ctx, "profiles:changed")
	}
	a.addAudit(
		"subscriptions.refresh."+source,
		profileName,
		fmt.Sprintf("ok=%v enabled=%d success=%d failed=%d added=%d", report.Success, report.EnabledTotal, report.SuccessTotal, report.FailedTotal, report.AddedTotal),
		err == nil,
	)
	return report, err
}

func (a *App) stopAllAutoSchedulers() {
	a.mu.Lock()
	defer a.mu.Unlock()
	for name, ch := range a.refreshTasks {
		close(ch)
		delete(a.refreshTasks, name)
		delete(a.refreshState, name)
		delete(a.refreshBusy, name)
	}
}

func (a *App) refreshAutoSchedulers() {
	a.mu.Lock()
	defer a.mu.Unlock()

	desired := map[string]bool{}
	for i := range a.profiles {
		normalizeRefreshSettings(&a.profiles[i].SubscriptionRefresh)
		if a.profiles[i].SubscriptionRefresh.AutoEnabled {
			desired[a.profiles[i].Name] = true
			if _, ok := a.refreshState[a.profiles[i].Name]; !ok {
				a.refreshState[a.profiles[i].Name] = &subscriptionRefreshRuntime{
					FailCount:    0,
					NextDelaySec: a.profiles[i].SubscriptionRefresh.IntervalSeconds,
				}
			}
		}
	}
	for name, ch := range a.refreshTasks {
		if !desired[name] {
			close(ch)
			delete(a.refreshTasks, name)
			delete(a.refreshState, name)
			delete(a.refreshBusy, name)
		}
	}
	for name := range desired {
		if _, ok := a.refreshTasks[name]; ok {
			continue
		}
		stopCh := make(chan struct{})
		a.refreshTasks[name] = stopCh
		go a.autoRefreshLoop(name, stopCh)
	}
}

func (a *App) autoRefreshLoop(profileName string, stopCh <-chan struct{}) {
	for {
		a.mu.RLock()
		p, ok := a.findProfileLocked(profileName)
		if !ok {
			a.mu.RUnlock()
			return
		}
		settings := p.SubscriptionRefresh
		normalizeRefreshSettings(&settings)
		st := a.refreshState[profileName]
		delay := settings.IntervalSeconds
		if st != nil && st.NextDelaySec > 0 {
			delay = st.NextDelaySec
		}
		a.mu.RUnlock()

		timer := time.NewTimer(time.Duration(delay) * time.Second)
		select {
		case <-stopCh:
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			return
		case <-timer.C:
		}

		a.mu.RLock()
		p2, ok2 := a.findProfileLocked(profileName)
		if !ok2 || !p2.SubscriptionRefresh.AutoEnabled {
			a.mu.RUnlock()
			return
		}
		a.mu.RUnlock()
		_, _ = a.executeSubscriptionRefresh(profileName, "auto", nil)
	}
}

func fetchSubscriptionText(subURL string) (string, error) {
	client := &http.Client{Timeout: 12 * time.Second}
	req, err := http.NewRequest(http.MethodGet, subURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "ZephyHosts/1.0")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("http status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func splitSubscriptionsBlock(text string) string {
	start := strings.Index(text, subscriptionBlockStart)
	end := strings.Index(text, subscriptionBlockEnd)
	if start < 0 || end < 0 || end < start {
		return text
	}
	endLine := strings.Index(text[end:], "\n")
	if endLine >= 0 {
		end = end + endLine + 1
	} else {
		end = len(text)
	}
	return strings.TrimRight(text[:start]+text[end:], "\r\n")
}

func removeManagedBlock(text, startMarker, endMarker string) string {
	start := strings.Index(text, startMarker)
	end := strings.Index(text, endMarker)
	if start < 0 || end < 0 || end < start {
		return text
	}
	endLine := strings.Index(text[end:], "\n")
	if endLine >= 0 {
		end = end + endLine + 1
	} else {
		end = len(text)
	}
	return strings.TrimRight(text[:start]+text[end:], "\r\n")
}

func conflictResolveStartMarker(subID string) string {
	return "# >>> Zephy Conflict Resolve:" + subID
}

func conflictResolveEndMarker(subID string) string {
	return "# <<< Zephy Conflict Resolve:" + subID
}

func (a *App) buildSubscriptionConflictPreview(profileName, subID string) (SubscriptionConflictPreview, error) {
	preview := SubscriptionConflictPreview{
		SubID:   subID,
		Domains: []string{},
		Items:   []SubscriptionConflictItem{},
	}
	a.mu.RLock()
	profileIdx := -1
	var target config.Subscription
	for i := range a.profiles {
		if a.profiles[i].Name != profileName {
			continue
		}
		profileIdx = i
		for _, s := range a.profiles[i].Subscriptions {
			if s.ID == subID {
				target = s
				break
			}
		}
		break
	}
	if profileIdx < 0 {
		a.mu.RUnlock()
		return preview, fmt.Errorf("profile %s not found", profileName)
	}
	if target.ID == "" {
		a.mu.RUnlock()
		return preview, fmt.Errorf("subscription not found")
	}
	hostsPath := a.profiles[profileIdx].HostsFile
	if !filepath.IsAbs(hostsPath) {
		hostsPath = filepath.Join(a.exeDir, hostsPath)
	}
	a.mu.RUnlock()

	preview.SubName = target.Name
	remoteText, err := fetchSubscriptionText(target.URL)
	if err != nil {
		return preview, err
	}
	remoteEntries, _, err := hosts.ParseText(remoteText)
	if err != nil {
		return preview, err
	}
	remoteEntries = hosts.DedupEntriesKeepLast(remoteEntries)

	localTextBytes, err := os.ReadFile(hostsPath)
	if err != nil {
		return preview, err
	}
	baseText := splitSubscriptionsBlock(string(localTextBytes))
	localEntries, _, _ := hosts.ParseText(baseText)
	localMap := hosts.EntriesToMap(localEntries)

	conflicts := make([]SubscriptionConflictItem, 0)
	seen := map[string]struct{}{}
	for _, e := range remoteEntries {
		d := strings.ToLower(strings.TrimSpace(e.Domain))
		if d == "" {
			continue
		}
		localIP, ok := localMap[d]
		if !ok || localIP == "" || localIP == e.IP {
			continue
		}
		if _, exists := seen[d]; exists {
			continue
		}
		seen[d] = struct{}{}
		conflicts = append(conflicts, SubscriptionConflictItem{
			Domain:   d,
			LocalIP:  localIP,
			RemoteIP: e.IP,
		})
	}
	sort.Slice(conflicts, func(i, j int) bool { return conflicts[i].Domain < conflicts[j].Domain })
	preview.Total = len(conflicts)
	if len(conflicts) > conflictResolvePreviewCap {
		preview.Items = conflicts[:conflictResolvePreviewCap]
		preview.Truncate = true
	} else {
		preview.Items = conflicts
	}
	preview.Domains = make([]string, 0, len(preview.Items))
	for _, it := range preview.Items {
		preview.Domains = append(preview.Domains, it.Domain)
	}
	return preview, nil
}

func (a *App) refreshProfileSubscriptionsWithFilter(profileName string, include map[string]struct{}) (SubscriptionRefreshReport, error) {
	report := SubscriptionRefreshReport{
		Time:     time.Now().Format("2006-01-02 15:04:05"),
		Failures: []SubscriptionRefreshFailure{},
	}
	a.mu.RLock()
	profileIdx := -1
	for i := range a.profiles {
		if a.profiles[i].Name == profileName {
			profileIdx = i
			break
		}
	}
	if profileIdx < 0 {
		a.mu.RUnlock()
		return report, fmt.Errorf("profile %s not found", profileName)
	}
	hostsPath := a.profiles[profileIdx].HostsFile
	if !filepath.IsAbs(hostsPath) {
		hostsPath = filepath.Join(a.exeDir, hostsPath)
	}
	subs := make([]config.Subscription, len(a.profiles[profileIdx].Subscriptions))
	copy(subs, a.profiles[profileIdx].Subscriptions)
	a.mu.RUnlock()

	remoteEntries := make([]hosts.Entry, 0)
	statusByID := make(map[string]string)
	now := report.Time
	for _, sub := range subs {
		if !sub.Enabled {
			continue
		}
		if include != nil {
			if _, ok := include[sub.ID]; !ok {
				continue
			}
		}
		report.EnabledTotal++
		text, err := fetchSubscriptionText(sub.URL)
		if err != nil {
			reason := "failed: " + err.Error()
			statusByID[sub.ID] = reason
			report.FailedTotal++
			report.Failures = append(report.Failures, SubscriptionRefreshFailure{
				SubID:   sub.ID,
				SubName: sub.Name,
				Reason:  reason,
			})
			continue
		}
		entries, _, err := hosts.ParseText(text)
		if err != nil {
			reason := "failed: parse error"
			statusByID[sub.ID] = reason
			report.FailedTotal++
			report.Failures = append(report.Failures, SubscriptionRefreshFailure{
				SubID:   sub.ID,
				SubName: sub.Name,
				Reason:  reason,
			})
			continue
		}
		remoteEntries = append(remoteEntries, entries...)
		statusByID[sub.ID] = "ok"
		report.SuccessTotal++
	}

	localTextBytes, err := os.ReadFile(hostsPath)
	if err != nil {
		return report, err
	}
	baseText := splitSubscriptionsBlock(string(localTextBytes))

	localEntries, _, _ := hosts.ParseText(baseText)
	localMap := hosts.EntriesToMap(localEntries)
	remoteDedup := hosts.DedupEntriesKeepLast(remoteEntries)
	finalRemoteLines := make([]string, 0, len(remoteDedup))
	for _, e := range remoteDedup {
		domain := strings.ToLower(strings.TrimSpace(e.Domain))
		if domain == "" {
			continue
		}
		if localIP, exists := localMap[domain]; exists {
			if strings.TrimSpace(localIP) == strings.TrimSpace(e.IP) {
				report.ConflictSame++
			} else {
				report.ConflictDiff++
			}
			continue // local priority
		}
		finalRemoteLines = append(finalRemoteLines, fmt.Sprintf("%s %s", e.IP, e.Domain))
	}
	report.AddedTotal = len(finalRemoteLines)

	newText := baseText
	if len(finalRemoteLines) > 0 {
		block := subscriptionBlockStart + "\n" +
			"# Auto merged from subscriptions (manual refresh)\n" +
			strings.Join(finalRemoteLines, "\n") + "\n" +
			subscriptionBlockEnd + "\n"
		if strings.TrimSpace(newText) != "" {
			newText += "\n"
		}
		newText += block
	}

	if err := os.WriteFile(hostsPath, []byte(newText), 0644); err != nil {
		return report, err
	}

	a.mu.Lock()
	for i := range a.profiles {
		if a.profiles[i].Name != profileName {
			continue
		}
		for j := range a.profiles[i].Subscriptions {
			if s, ok := statusByID[a.profiles[i].Subscriptions[j].ID]; ok {
				a.profiles[i].Subscriptions[j].LastStatus = s
				a.profiles[i].Subscriptions[j].LastUpdated = now
			}
		}
		a.refreshProfileHostsLocked(i)
		break
	}
	err = a.saveConfig()
	a.mu.Unlock()
	return report, err
}

func (a *App) RefreshProfileSubscriptions(profileName string) error {
	_, err := a.executeSubscriptionRefresh(profileName, "manual", nil)
	return err
}

func (a *App) RefreshProfileSubscriptionsWithReport(profileName string) (SubscriptionRefreshReport, error) {
	return a.executeSubscriptionRefresh(profileName, "manual", nil)
}

func (a *App) RefreshSingleProfileSubscription(profileName, subID string) error {
	a.mu.RLock()
	found := false
	for i := range a.profiles {
		if a.profiles[i].Name != profileName {
			continue
		}
		for _, s := range a.profiles[i].Subscriptions {
			if s.ID == subID {
				found = true
				break
			}
		}
		break
	}
	a.mu.RUnlock()
	if !found {
		return fmt.Errorf("subscription not found")
	}
	_, err := a.executeSubscriptionRefresh(profileName, "manual-single", nil)
	return err
}

func (a *App) RetryFailedProfileSubscriptions(profileName string) (SubscriptionRefreshReport, error) {
	a.mu.RLock()
	found := false
	failedIDs := map[string]struct{}{}
	for i := range a.profiles {
		if a.profiles[i].Name != profileName {
			continue
		}
		found = true
		for _, s := range a.profiles[i].Subscriptions {
			if strings.HasPrefix(strings.ToLower(strings.TrimSpace(s.LastStatus)), "failed:") {
				failedIDs[s.ID] = struct{}{}
			}
		}
		break
	}
	a.mu.RUnlock()
	if !found {
		return SubscriptionRefreshReport{}, fmt.Errorf("profile %s not found", profileName)
	}
	if len(failedIDs) == 0 {
		return SubscriptionRefreshReport{
			Time:    time.Now().Format("2006-01-02 15:04:05"),
			Source:  "retry",
			Success: true,
			Failures: []SubscriptionRefreshFailure{},
		}, nil
	}
	return a.executeSubscriptionRefresh(profileName, "retry", failedIDs)
}

func (a *App) PreviewSubscriptionConflicts(profileName, subID string) (SubscriptionConflictPreview, error) {
	return a.buildSubscriptionConflictPreview(profileName, subID)
}

func (a *App) ResolveSubscriptionConflicts(profileName, subID, strategy string) error {
	switch strategy {
	case conflictResolveKeepLocal:
		a.addAudit("subscriptions.resolve_conflicts", profileName, "keep_local", true)
		return nil
	case conflictResolveUseRemote:
	default:
		a.addAudit("subscriptions.resolve_conflicts", profileName, "invalid strategy: "+strategy, false)
		return fmt.Errorf("invalid strategy: %s", strategy)
	}

	preview, err := a.buildSubscriptionConflictPreview(profileName, subID)
	if err != nil {
		a.addAudit("subscriptions.resolve_conflicts", profileName, "build preview failed: "+err.Error(), false)
		return err
	}
	if preview.Total == 0 {
		a.addAudit("subscriptions.resolve_conflicts", profileName, "no conflicts", true)
		return nil
	}

	a.mu.RLock()
	hostsPath := ""
	for i := range a.profiles {
		if a.profiles[i].Name == profileName {
			hostsPath = a.profiles[i].HostsFile
			break
		}
	}
	if hostsPath == "" {
		a.mu.RUnlock()
		a.addAudit("subscriptions.resolve_conflicts", profileName, "profile not found", false)
		return fmt.Errorf("profile %s not found", profileName)
	}
	if !filepath.IsAbs(hostsPath) {
		hostsPath = filepath.Join(a.exeDir, hostsPath)
	}
	a.mu.RUnlock()

	textBytes, err := os.ReadFile(hostsPath)
	if err != nil {
		a.addAudit("subscriptions.resolve_conflicts", profileName, "read hosts failed: "+err.Error(), false)
		return err
	}
	baseText := splitSubscriptionsBlock(string(textBytes))
	baseText = removeManagedBlock(baseText, conflictResolveStartMarker(subID), conflictResolveEndMarker(subID))

	lines := make([]string, 0, len(preview.Items))
	for _, it := range preview.Items {
		lines = append(lines, fmt.Sprintf("%s %s", it.RemoteIP, it.Domain))
	}
	sort.Strings(lines)
	block := conflictResolveStartMarker(subID) + "\n" +
		"# Auto resolved: use_remote\n" +
		strings.Join(lines, "\n") + "\n" +
		conflictResolveEndMarker(subID) + "\n"

	newBase := strings.TrimRight(baseText, "\r\n")
	if newBase != "" {
		newBase += "\n"
	}
	newBase += block
	if err := os.WriteFile(hostsPath, []byte(newBase), 0644); err != nil {
		a.addAudit("subscriptions.resolve_conflicts", profileName, "write hosts failed: "+err.Error(), false)
		return err
	}
	if err := a.RefreshProfileSubscriptions(profileName); err != nil {
		a.addAudit("subscriptions.resolve_conflicts", profileName, "refresh failed: "+err.Error(), false)
		return err
	}
	a.addAudit("subscriptions.resolve_conflicts", profileName, fmt.Sprintf("use_remote applied %d items", len(preview.Items)), true)
	return nil
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
	text := `# This file is managed by Zephy
# Add entries in the format: IP DOMAIN
# Example:
# 120.92.124.158 account.wps.cn
`
	return a.SetHostsText(profileName, text)
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
