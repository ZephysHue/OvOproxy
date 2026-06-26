package subscription

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func fixedNow() time.Time {
	return time.Date(2026, 6, 26, 10, 0, 0, 0, time.UTC)
}

func testServer(status int, body string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
}

func TestRefreshCreatesSubscriptionBlockWhenFileMissing(t *testing.T) {
	server := testServer(http.StatusOK, "1.1.1.1 example.com\n2.2.2.2 api.example.com\n")
	defer server.Close()

	hostsPath := filepath.Join(t.TempDir(), "nested", "remote.hosts")
	result, entries, changed := Refresh(Options{
		URL:       server.URL,
		HostsPath: hostsPath,
		Client:    server.Client(),
		Now:       fixedNow,
	})

	if result.Status != "ok" || !changed || len(entries) != 2 {
		t.Fatalf("unexpected result=%+v changed=%v entries=%d", result, changed, len(entries))
	}
	data, err := os.ReadFile(hostsPath)
	if err != nil {
		t.Fatalf("read hosts file: %v", err)
	}
	text := string(data)
	for _, want := range []string{
		"# >>> subscription (auto-managed, do not edit) " + server.URL,
		"1.1.1.1 example.com",
		"2.2.2.2 api.example.com",
		"# <<< subscription",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("hosts file missing %q in:\n%s", want, text)
		}
	}
}

func TestRefreshSkipsWriteWhenSubscriptionBodyIsUnchanged(t *testing.T) {
	server := testServer(http.StatusOK, "1.1.1.1 example.com\n")
	defer server.Close()

	hostsPath := filepath.Join(t.TempDir(), "remote.hosts")
	original := "# manual\n\n# >>> subscription (auto-managed, do not edit) " + server.URL + "\n1.1.1.1 example.com\n# <<< subscription\n"
	if err := os.WriteFile(hostsPath, []byte(original), 0644); err != nil {
		t.Fatal(err)
	}

	result, entries, changed := Refresh(Options{
		URL:       server.URL,
		HostsPath: hostsPath,
		Client:    server.Client(),
		Now:       fixedNow,
	})

	if result.Status != "ok" || changed || len(entries) != 1 {
		t.Fatalf("unexpected result=%+v changed=%v entries=%d", result, changed, len(entries))
	}
	data, err := os.ReadFile(hostsPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != original {
		t.Fatalf("unchanged body should not rewrite file\nwant:\n%s\ngot:\n%s", original, string(data))
	}
}

func TestRefreshReplacesOldSubscriptionBlockAndPreservesManualLines(t *testing.T) {
	server := testServer(http.StatusOK, "2.2.2.2 new.example.com\n")
	defer server.Close()

	hostsPath := filepath.Join(t.TempDir(), "remote.hosts")
	original := "127.0.0.1 localhost\n\n# >>> subscription old\n1.1.1.1 old.example.com\n# <<< subscription\n# tail\n"
	if err := os.WriteFile(hostsPath, []byte(original), 0644); err != nil {
		t.Fatal(err)
	}

	result, _, changed := Refresh(Options{
		URL:       server.URL,
		HostsPath: hostsPath,
		Client:    server.Client(),
		Now:       fixedNow,
	})

	if result.Status != "ok" || !changed {
		t.Fatalf("unexpected result=%+v changed=%v", result, changed)
	}
	data, err := os.ReadFile(hostsPath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.Contains(text, "127.0.0.1 localhost") || !strings.Contains(text, "# tail") {
		t.Fatalf("manual lines were not preserved:\n%s", text)
	}
	if strings.Contains(text, "old.example.com") || !strings.Contains(text, "2.2.2.2 new.example.com") {
		t.Fatalf("subscription block was not replaced:\n%s", text)
	}
}

func TestRefreshReturnsErrorStatusForHTTPFailure(t *testing.T) {
	server := testServer(http.StatusBadGateway, "bad gateway")
	defer server.Close()

	result, entries, changed := Refresh(Options{
		URL:       server.URL,
		HostsPath: filepath.Join(t.TempDir(), "remote.hosts"),
		Client:    server.Client(),
		Now:       fixedNow,
	})

	if result.Status != "error" || result.Message != "HTTP 502" || result.LastFetch == "" {
		t.Fatalf("unexpected result=%+v", result)
	}
	if changed || entries != nil {
		t.Fatalf("failure should not change file or return entries")
	}
}

func TestRefreshReturnsErrorStatusForParseFailure(t *testing.T) {
	server := testServer(http.StatusOK, "not-a-valid-host-line\n")
	defer server.Close()

	result, entries, changed := Refresh(Options{
		URL:       server.URL,
		HostsPath: filepath.Join(t.TempDir(), "remote.hosts"),
		Client:    server.Client(),
		Now:       fixedNow,
	})

	if result.Status != "error" || !strings.Contains(result.Message, "parse") {
		t.Fatalf("unexpected result=%+v", result)
	}
	if changed || entries != nil {
		t.Fatalf("parse failure should not change file or return entries")
	}
}
