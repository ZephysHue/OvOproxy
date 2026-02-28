package hosts

import (
	"bufio"
	"fmt"
	"os"
	"sort"
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

func DedupTextKeepLast(text string) (string, int) {
	// Determine line ending style
	useCRLF := strings.Contains(text, "\r\n")
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")

	lastLine := make(map[string]int)
	counts := make(map[string]int)

	parseMapping := func(raw string) (ip string, domains []string, comment string, ok bool) {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			return "", nil, "", false
		}
		commentIdx := strings.Index(raw, "#")
		working := raw
		if commentIdx >= 0 {
			working = raw[:commentIdx]
			comment = raw[commentIdx:]
		}
		working = strings.TrimSpace(working)
		parts := strings.Fields(working)
		if len(parts) < 2 {
			return "", nil, "", false
		}
		ip = parts[0]
		for _, d := range parts[1:] {
			h := strings.ToLower(strings.TrimSpace(d))
			if h == "" {
				continue
			}
			domains = append(domains, h)
		}
		if len(domains) == 0 {
			return "", nil, "", false
		}
		return ip, domains, comment, true
	}

	// Pass 1: record last occurrence per domain
	for i, raw := range lines {
		_, domains, _, ok := parseMapping(raw)
		if !ok {
			continue
		}
		for _, d := range domains {
			lastLine[d] = i
			counts[d]++
		}
	}

	dupDomains := 0
	for _, c := range counts {
		if c > 1 {
			dupDomains++
		}
	}

	// Pass 2: build new lines
	out := make([]string, 0, len(lines))
	for i, raw := range lines {
		ip, domains, comment, ok := parseMapping(raw)
		if !ok {
			out = append(out, raw)
			continue
		}
		kept := make([]string, 0, len(domains))
		for _, d := range domains {
			if lastLine[d] == i {
				kept = append(kept, d)
			}
		}
		if len(kept) == 0 {
			// Drop this mapping line entirely
			continue
		}
		newLine := ip + " " + strings.Join(kept, " ")
		if comment != "" {
			if !strings.HasPrefix(comment, "#") {
				comment = "#" + comment
			}
			newLine = newLine + " " + strings.TrimSpace(comment)
		}
		out = append(out, newLine)
	}

	joined := strings.Join(out, "\n")
	if useCRLF {
		joined = strings.ReplaceAll(joined, "\n", "\r\n")
	}
	return joined, dupDomains
}

func DedupEntriesKeepLast(entries []Entry) []Entry {
	seen := make(map[string]struct{}, len(entries))
	out := make([]Entry, 0, len(entries))
	for i := len(entries) - 1; i >= 0; i-- {
		d := entries[i].Domain
		if _, ok := seen[d]; ok {
			continue
		}
		seen[d] = struct{}{}
		out = append(out, entries[i])
	}
	// reverse to keep natural order (based on last occurrences)
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}

func EntriesToMap(entries []Entry) map[string]string {
	m := make(map[string]string, len(entries))
	for _, e := range entries {
		m[e.Domain] = e.IP
	}
	return m
}

func DuplicateDomains(counts map[string]int) []string {
	dups := make([]string, 0)
	for d, c := range counts {
		if c > 1 {
			dups = append(dups, d)
		}
	}
	sort.Strings(dups)
	return dups
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
