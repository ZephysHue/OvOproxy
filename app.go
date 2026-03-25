package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
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

const (
	subscriptionBlockStart = "# >>> Zephy Subscriptions Start"
	subscriptionBlockEnd   = "# <<< Zephy Subscriptions End"
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
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.LoadConfig()
	a.startAllProxies()
	a.syncHostsEnabledState()
	a.startTray()
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
		return fmt.Errorf("profile %s not found", name)
	}
	if !proxyActive {
		return fmt.Errorf("代理端口未启动: %s", proxyErr)
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

	for _, p := range a.profiles {
		if p.Name == name {
			a.mu.Unlock()
			return fmt.Errorf("profile %s already exists", name)
		}
		if p.ListenIP == listenIP && p.Port == port {
			a.mu.Unlock()
			return fmt.Errorf("address %s:%d already in use", listenIP, port)
		}
	}

	hostsFile := filepath.Join("configs", "hosts", name+".hosts")
	hostsPath := filepath.Join(a.exeDir, hostsFile)

	if err := os.MkdirAll(filepath.Dir(hostsPath), 0755); err != nil {
		a.mu.Unlock()
		return err
	}
	if err := os.WriteFile(hostsPath, []byte("# Hosts for "+name+"\n"), 0644); err != nil {
		a.mu.Unlock()
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
		a.mu.Unlock()
		return err
	}
	hostsRules := newProfile.Hosts
	a.mu.Unlock()

	_ = a.proxyManager.StartProxy(name, listenIP, port, hostsRules)
	a.refreshProxyStatus()
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

func (a *App) RefreshProfileSubscriptions(profileName string) error {
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
		return fmt.Errorf("profile %s not found", profileName)
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
	now := time.Now().Format("2006-01-02 15:04:05")
	for _, sub := range subs {
		if !sub.Enabled {
			continue
		}
		text, err := fetchSubscriptionText(sub.URL)
		if err != nil {
			statusByID[sub.ID] = "failed: " + err.Error()
			continue
		}
		entries, _, err := hosts.ParseText(text)
		if err != nil {
			statusByID[sub.ID] = "failed: parse error"
			continue
		}
		remoteEntries = append(remoteEntries, entries...)
		statusByID[sub.ID] = "ok"
	}

	localTextBytes, err := os.ReadFile(hostsPath)
	if err != nil {
		return err
	}
	baseText := splitSubscriptionsBlock(string(localTextBytes))

	localEntries, _, _ := hosts.ParseText(baseText)
	localMap := hosts.EntriesToMap(localEntries)
	remoteDedup := hosts.DedupEntriesKeepLast(remoteEntries)
	finalRemoteLines := make([]string, 0, len(remoteDedup))
	for _, e := range remoteDedup {
		if _, exists := localMap[strings.ToLower(strings.TrimSpace(e.Domain))]; exists {
			continue // local priority
		}
		finalRemoteLines = append(finalRemoteLines, fmt.Sprintf("%s %s", e.IP, e.Domain))
	}

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
		return err
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
	return err
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
