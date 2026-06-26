package proxymanager

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"
)

func freePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen free port: %v", err)
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port
}

func proxyClient(t *testing.T, listenAddr string) *http.Client {
	t.Helper()
	proxyURL, err := url.Parse("http://" + listenAddr)
	if err != nil {
		t.Fatal(err)
	}
	return &http.Client{
		Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)},
		Timeout:   5 * time.Second,
	}
}

func TestManagerProxiesMappedHTTPHost(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.Host, "example.test:") {
			t.Fatalf("expected original host header, got %q", r.Host)
		}
		_, _ = w.Write([]byte("mapped"))
	}))
	defer backend.Close()

	backendURL, err := url.Parse(backend.URL)
	if err != nil {
		t.Fatal(err)
	}

	manager := New()
	port := freePort(t)
	if err := manager.StartProxy("mapped", "127.0.0.1", port, map[string]string{"example.test": "127.0.0.1"}); err != nil {
		t.Fatalf("start proxy: %v", err)
	}
	defer manager.StopAll()

	resp, err := proxyClient(t, net.JoinHostPort("127.0.0.1", itoa(port))).Get("http://example.test:" + backendURL.Port() + "/hello")
	if err != nil {
		t.Fatalf("proxy request: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "mapped" {
		t.Fatalf("unexpected body %q", string(body))
	}

	logs := manager.GetLogs("mapped", 1)
	if len(logs) != 1 || !logs[0].Success || logs[0].ResolvedIP == "(direct)" {
		t.Fatalf("expected mapped success log, got %+v", logs)
	}
}

func TestManagerProxiesUnmappedHTTPHostDirectly(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("direct"))
	}))
	defer backend.Close()

	manager := New()
	port := freePort(t)
	if err := manager.StartProxy("direct", "127.0.0.1", port, nil); err != nil {
		t.Fatalf("start proxy: %v", err)
	}
	defer manager.StopAll()

	resp, err := proxyClient(t, net.JoinHostPort("127.0.0.1", itoa(port))).Get(backend.URL)
	if err != nil {
		t.Fatalf("proxy request: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "direct" {
		t.Fatalf("unexpected body %q", string(body))
	}

	logs := manager.GetLogs("direct", 1)
	if len(logs) != 1 || !logs[0].Success || logs[0].ResolvedIP != "(direct)" {
		t.Fatalf("expected direct success log, got %+v", logs)
	}
}

func TestManagerReportsPortInUse(t *testing.T) {
	manager := New()
	port := freePort(t)
	if err := manager.StartProxy("one", "127.0.0.1", port, nil); err != nil {
		t.Fatalf("start first proxy: %v", err)
	}
	defer manager.StopAll()

	err := manager.StartProxy("two", "127.0.0.1", port, nil)
	if err == nil {
		t.Fatal("expected second proxy on same port to fail")
	}
	active, lastErr := manager.GetStatus("two")
	if active || lastErr == "" || !strings.Contains(lastErr, "port") && !strings.Contains(lastErr, "端口") {
		t.Fatalf("unexpected status active=%v lastErr=%q", active, lastErr)
	}
}

func itoa(v int) string {
	return strconv.Itoa(v)
}
