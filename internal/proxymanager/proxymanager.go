package proxymanager

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type LogEntry struct {
	Time       time.Time `json:"time"`
	Method     string    `json:"method"`
	Host       string    `json:"host"`
	ResolvedIP string    `json:"resolved_ip"`
	Success    bool      `json:"success"`
	Error      string    `json:"error,omitempty"`
}

type ProxyHandle struct {
	ProfileName string
	ListenAddr  string
	listener    net.Listener
	server      *http.Server
	active      bool
	lastErr     string
	hostsRules  map[string]string
	logs        []LogEntry
	mu          sync.RWMutex
}

func (h *ProxyHandle) IsActive() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.active
}

func (h *ProxyHandle) LastError() string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.lastErr
}

func (h *ProxyHandle) GetHostsRules() map[string]string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	cp := make(map[string]string, len(h.hostsRules))
	for k, v := range h.hostsRules {
		cp[k] = v
	}
	return cp
}

func (h *ProxyHandle) SetHostsRules(rules map[string]string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.hostsRules = make(map[string]string, len(rules))
	for k, v := range rules {
		h.hostsRules[strings.ToLower(k)] = v
	}
}

func (h *ProxyHandle) resolveHost(host string) (string, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	hostname := host
	port := ""
	if idx := strings.LastIndex(host, ":"); idx != -1 {
		hostname = host[:idx]
		port = host[idx:]
	}

	if ip, ok := h.hostsRules[strings.ToLower(hostname)]; ok {
		return ip + port, true
	}
	return host, false
}

func (h *ProxyHandle) addLog(method, host, resolvedIP string, success bool, errMsg string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	entry := LogEntry{
		Time:       time.Now(),
		Method:     method,
		Host:       host,
		ResolvedIP: resolvedIP,
		Success:    success,
		Error:      errMsg,
	}
	h.logs = append(h.logs, entry)
	if len(h.logs) > 500 {
		h.logs = h.logs[len(h.logs)-500:]
	}
}

func (h *ProxyHandle) GetLogs(limit int) []LogEntry {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if limit <= 0 || limit > len(h.logs) {
		limit = len(h.logs)
	}
	start := len(h.logs) - limit
	if start < 0 {
		start = 0
	}
	result := make([]LogEntry, limit)
	copy(result, h.logs[start:])
	return result
}

type Manager struct {
	proxies map[string]*ProxyHandle
	mu      sync.RWMutex
}

func New() *Manager {
	return &Manager{
		proxies: make(map[string]*ProxyHandle),
	}
}

func (m *Manager) StartProxy(profileName, listenIP string, port int, hostsRules map[string]string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	addr := fmt.Sprintf("%s:%d", listenIP, port)

	if existing, ok := m.proxies[profileName]; ok {
		if existing.ListenAddr == addr && existing.IsActive() {
			existing.SetHostsRules(hostsRules)
			return nil
		}
		m.stopProxyLocked(profileName)
	}

	handle := &ProxyHandle{
		ProfileName: profileName,
		ListenAddr:  addr,
		hostsRules:  make(map[string]string),
		logs:        make([]LogEntry, 0),
	}
	handle.SetHostsRules(hostsRules)

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		handle.lastErr = fmt.Sprintf("端口 %d 被占用或无法监听: %v", port, err)
		handle.active = false
		m.proxies[profileName] = handle
		return fmt.Errorf("listen %s: %w", addr, err)
	}

	handle.listener = listener
	handler := &proxyHandler{handle: handle}
	handle.server = &http.Server{
		Handler:      handler,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	handle.active = true
	handle.lastErr = ""

	m.proxies[profileName] = handle

	go func() {
		err := handle.server.Serve(listener)
		handle.mu.Lock()
		if err != nil && err != http.ErrServerClosed {
			handle.lastErr = err.Error()
		}
		handle.active = false
		handle.mu.Unlock()
	}()

	return nil
}

func (m *Manager) UpdateHostsRules(profileName string, rules map[string]string) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if handle, ok := m.proxies[profileName]; ok {
		handle.SetHostsRules(rules)
	}
}

func (m *Manager) StopProxy(profileName string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stopProxyLocked(profileName)
}

func (m *Manager) stopProxyLocked(profileName string) {
	handle, ok := m.proxies[profileName]
	if !ok {
		return
	}
	if handle.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		_ = handle.server.Shutdown(ctx)
		cancel()
	}
	if handle.listener != nil {
		_ = handle.listener.Close()
	}
	handle.mu.Lock()
	handle.active = false
	handle.mu.Unlock()
	delete(m.proxies, profileName)
}

func (m *Manager) StopAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for name := range m.proxies {
		m.stopProxyLocked(name)
	}
}

func (m *Manager) GetStatus(profileName string) (active bool, lastErr string) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	handle, ok := m.proxies[profileName]
	if !ok {
		return false, ""
	}
	return handle.IsActive(), handle.LastError()
}

func (m *Manager) GetAllStatus() map[string]struct {
	Active  bool
	LastErr string
} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make(map[string]struct {
		Active  bool
		LastErr string
	})
	for name, handle := range m.proxies {
		result[name] = struct {
			Active  bool
			LastErr string
		}{
			Active:  handle.IsActive(),
			LastErr: handle.LastError(),
		}
	}
	return result
}

func (m *Manager) GetLogs(profileName string, limit int) []LogEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()
	handle, ok := m.proxies[profileName]
	if !ok {
		return nil
	}
	return handle.GetLogs(limit)
}

type proxyHandler struct {
	handle *ProxyHandle
}

func (h *proxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodConnect {
		h.handleConnect(w, r)
	} else {
		h.handleHTTP(w, r)
	}
}

func (h *proxyHandler) handleConnect(w http.ResponseWriter, r *http.Request) {
	originalHost := r.Host
	resolvedAddr, mapped := h.handle.resolveHost(r.Host)

	destConn, err := net.DialTimeout("tcp", resolvedAddr, 10*time.Second)
	if err != nil {
		h.handle.addLog("CONNECT", originalHost, resolvedAddr, false, err.Error())
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		h.handle.addLog("CONNECT", originalHost, resolvedAddr, false, "Hijacking not supported")
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		destConn.Close()
		return
	}

	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		h.handle.addLog("CONNECT", originalHost, resolvedAddr, false, err.Error())
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		destConn.Close()
		return
	}

	_, _ = clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))

	if mapped {
		h.handle.addLog("CONNECT", originalHost, resolvedAddr, true, "")
	} else {
		h.handle.addLog("CONNECT", originalHost, "(direct)", true, "")
	}

	go transfer(destConn, clientConn)
	go transfer(clientConn, destConn)
}

func (h *proxyHandler) handleHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Host == "" {
		r.URL.Host = r.Host
	}
	if r.URL.Scheme == "" {
		r.URL.Scheme = "http"
	}

	originalHost := r.URL.Host
	resolvedHost, mapped := h.handle.resolveHost(originalHost)

	resolvedURL := *r.URL
	resolvedURL.Host = resolvedHost

	outReq, err := http.NewRequest(r.Method, resolvedURL.String(), r.Body)
	if err != nil {
		h.handle.addLog(r.Method, originalHost, resolvedHost, false, err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	copyHeader(outReq.Header, r.Header)
	outReq.Host = originalHost
	outReq.Header.Set("Host", originalHost)
	outReq.Header.Del("Proxy-Connection")
	outReq.Header.Del("Proxy-Authenticate")
	outReq.Header.Del("Proxy-Authorization")

	hopHeaders := []string{
		"Connection",
		"Keep-Alive",
		"Transfer-Encoding",
		"TE",
		"Trailer",
		"Upgrade",
	}
	for _, hdr := range hopHeaders {
		outReq.Header.Del(hdr)
	}

	client := &http.Client{
		Timeout: 60 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.Do(outReq)
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "no such host") {
			errMsg = "DNS lookup failed: " + originalHost
		}
		if mapped {
			h.handle.addLog(r.Method, originalHost, resolvedHost, false, errMsg)
		} else {
			h.handle.addLog(r.Method, originalHost, "(direct)", false, errMsg)
		}
		http.Error(w, errMsg, http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	if mapped {
		h.handle.addLog(r.Method, originalHost, resolvedHost, true, "")
	} else {
		h.handle.addLog(r.Method, originalHost, "(direct)", true, "")
	}

	copyHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func transfer(dst, src net.Conn) {
	defer dst.Close()
	defer src.Close()
	_, _ = io.Copy(dst, src)
}
