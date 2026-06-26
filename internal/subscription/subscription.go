package subscription

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"zephy/internal/hosts"
)

const (
	StartMarker = "# >>> subscription (auto-managed, do not edit)"
	EndMarker   = "# <<< subscription"
)

type Result struct {
	Status     string `json:"status"`
	Message    string `json:"message"`
	LastFetch  string `json:"last_fetch"`
	EntryCount int    `json:"entry_count"`
}

type Options struct {
	URL       string
	HostsPath string
	Client    *http.Client
	Now       func() time.Time
}

func Refresh(opts Options) (Result, []hosts.Entry, bool) {
	now := opts.Now
	if now == nil {
		now = time.Now
	}

	if opts.URL == "" {
		return Result{Status: "error", Message: "subscription URL is not configured"}, nil, false
	}

	client := opts.Client
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}

	resp, err := client.Get(opts.URL)
	if err != nil {
		ts := now().Format(time.RFC3339)
		return Result{Status: "error", Message: fmt.Sprintf("request failed: %v", err), LastFetch: ts}, nil, false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		ts := now().Format(time.RFC3339)
		return Result{Status: "error", Message: fmt.Sprintf("HTTP %d", resp.StatusCode), LastFetch: ts}, nil, false
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024))
	if err != nil {
		ts := now().Format(time.RFC3339)
		return Result{Status: "error", Message: fmt.Sprintf("read failed: %v", err), LastFetch: ts}, nil, false
	}

	entries, _, err := hosts.ParseText(string(body))
	if err != nil {
		ts := now().Format(time.RFC3339)
		return Result{Status: "error", Message: fmt.Sprintf("parse failed: %v", err), LastFetch: ts}, nil, false
	}

	newBody := EntriesBody(entries)
	existing, err := os.ReadFile(opts.HostsPath)
	if err != nil && !os.IsNotExist(err) {
		return Result{Status: "error", Message: fmt.Sprintf("read local file failed: %v", err)}, nil, false
	}
	existingText := string(existing)

	ts := now().Format(time.RFC3339)
	if ExtractBody(existingText) == newBody {
		return Result{Status: "ok", Message: "unchanged", LastFetch: ts, EntryCount: len(entries)}, entries, false
	}

	merged := ReplaceBlock(existingText, opts.URL, newBody)
	if err := os.MkdirAll(filepath.Dir(opts.HostsPath), 0755); err != nil {
		return Result{Status: "error", Message: fmt.Sprintf("create directory failed: %v", err)}, nil, false
	}
	if err := os.WriteFile(opts.HostsPath, []byte(merged), 0644); err != nil {
		return Result{Status: "error", Message: fmt.Sprintf("write failed: %v", err)}, nil, false
	}

	return Result{Status: "ok", Message: "refreshed", LastFetch: ts, EntryCount: len(entries)}, entries, true
}

func EntriesBody(entries []hosts.Entry) string {
	lines := make([]string, len(entries))
	for i, e := range entries {
		lines[i] = fmt.Sprintf("%s %s", e.IP, e.Domain)
	}
	return strings.Join(lines, "\n")
}

func ExtractBody(text string) string {
	idx := blockStart(text)
	if idx < 0 {
		return ""
	}
	start := strings.Index(text[idx:], "\n")
	if start < 0 {
		return ""
	}
	start += idx + 1
	end := strings.Index(text[start:], EndMarker)
	if end < 0 {
		return ""
	}
	return strings.TrimSpace(text[start : start+end])
}

func ReplaceBlock(existing, url, body string) string {
	existing = removeBlock(existing)
	block := StartMarker + " " + url + "\n" + body + "\n" + EndMarker + "\n"
	return strings.TrimRight(existing, "\n") + "\n\n" + block
}

func removeBlock(text string) string {
	startIdx := blockStart(text)
	if startIdx < 0 {
		return text
	}
	endIdx := strings.Index(text[startIdx:], EndMarker)
	if endIdx < 0 {
		return text[:startIdx]
	}
	endIdx += startIdx + len(EndMarker)
	if endIdx < len(text) && text[endIdx] == '\n' {
		endIdx++
	}
	return text[:startIdx] + text[endIdx:]
}

func blockStart(text string) int {
	for _, marker := range []string{StartMarker, "# >>> subscription"} {
		if idx := strings.Index(text, marker); idx >= 0 {
			return idx
		}
	}
	return -1
}
