package hosts

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type Table struct {
	records map[string]string
}

type Entry struct {
	Domain string
	IP     string
}

func Load(path string) (*Table, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open hosts file: %w", err)
	}
	defer f.Close()

	t := &Table{records: make(map[string]string)}
	s := bufio.NewScanner(f)
	lineNum := 0
	for s.Scan() {
		lineNum++
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if idx := strings.Index(line, "#"); idx >= 0 {
			line = strings.TrimSpace(line[:idx])
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			return nil, fmt.Errorf("hosts parse error line %d", lineNum)
		}

		ip := parts[0]
		for _, host := range parts[1:] {
			h := strings.ToLower(strings.TrimSpace(host))
			if h != "" {
				t.records[h] = ip
			}
		}
	}
	if err := s.Err(); err != nil {
		return nil, fmt.Errorf("scan hosts file: %w", err)
	}

	return t, nil
}

func ParseFile(path string) ([]Entry, map[string]int, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("open hosts file: %w", err)
	}
	defer f.Close()

	entries := make([]Entry, 0, 64)
	counts := make(map[string]int)

	s := bufio.NewScanner(f)
	lineNum := 0
	for s.Scan() {
		lineNum++
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if idx := strings.Index(line, "#"); idx >= 0 {
			line = strings.TrimSpace(line[:idx])
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			return nil, nil, fmt.Errorf("hosts parse error line %d", lineNum)
		}
		ip := parts[0]
		for _, host := range parts[1:] {
			h := strings.ToLower(strings.TrimSpace(host))
			if h == "" {
				continue
			}
			entries = append(entries, Entry{Domain: h, IP: ip})
			counts[h]++
		}
	}
	if err := s.Err(); err != nil {
		return nil, nil, fmt.Errorf("scan hosts file: %w", err)
	}
	return entries, counts, nil
}

func ParseText(text string) ([]Entry, map[string]int, error) {
	entries := make([]Entry, 0, 64)
	counts := make(map[string]int)

	s := bufio.NewScanner(strings.NewReader(text))
	lineNum := 0
	for s.Scan() {
		lineNum++
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if idx := strings.Index(line, "#"); idx >= 0 {
			line = strings.TrimSpace(line[:idx])
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			return nil, nil, fmt.Errorf("hosts parse error line %d", lineNum)
		}
		ip := parts[0]
		for _, host := range parts[1:] {
			h := strings.ToLower(strings.TrimSpace(host))
			if h == "" {
				continue
			}
			entries = append(entries, Entry{Domain: h, IP: ip})
			counts[h]++
		}
	}
	if err := s.Err(); err != nil {
		return nil, nil, fmt.Errorf("scan hosts: %w", err)
	}
	return entries, counts, nil
}

// NormalizeText expands multi-domain lines ("1.2.3.4 a.com b.com") into
// one-domain-per-line format, preserving comments and blank lines.
func NormalizeText(text string) string {
	useCRLF := strings.Contains(text, "\r\n")
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")

	out := make([]string, 0, len(lines))
	for _, raw := range lines {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			out = append(out, raw)
			continue
		}

		var comment string
		working := raw
		if idx := strings.Index(trimmed, "#"); idx > 0 {
			commentStart := strings.Index(raw, "#")
			comment = strings.TrimSpace(raw[commentStart:])
			working = raw[:commentStart]
		}

		parts := strings.Fields(strings.TrimSpace(working))
		if len(parts) < 2 {
			out = append(out, raw)
			continue
		}

		ip := parts[0]
		domains := parts[1:]
		if len(domains) <= 1 {
			out = append(out, raw)
			continue
		}

		for j, d := range domains {
			line := ip + " " + d
			if j == 0 && comment != "" {
				line += " " + comment
			}
			out = append(out, line)
		}
	}

	joined := strings.Join(out, "\n")
	if useCRLF {
		joined = strings.ReplaceAll(joined, "\n", "\r\n")
	}
	return joined
}

// EntriesToMap keeps the first occurrence of each domain,
// matching standard OS hosts resolution behavior (first match wins).
func EntriesToMap(entries []Entry) map[string]string {
	m := make(map[string]string, len(entries))
	for _, e := range entries {
		if _, exists := m[e.Domain]; !exists {
			m[e.Domain] = e.IP
		}
	}
	return m
}

func (t *Table) Resolve(host string) (string, bool) {
	h := strings.ToLower(strings.TrimSpace(host))
	ip, ok := t.records[h]
	return ip, ok
}

func (t *Table) GetAll() map[string]string {
	result := make(map[string]string, len(t.records))
	for k, v := range t.records {
		result[k] = v
	}
	return result
}
