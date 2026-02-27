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
