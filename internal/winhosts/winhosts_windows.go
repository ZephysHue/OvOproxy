//go:build windows

package winhosts

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"
)

const (
	SystemHostsPath = `C:\Windows\System32\drivers\etc\hosts`
	startMarkerPrefix = "# >>> Zephy Profile:"
	endMarkerPrefix   = "# <<< Zephy Profile:"
)

func profileStartMarker(profileId string) string {
	return startMarkerPrefix + profileId
}

func profileEndMarker(profileId string) string {
	return endMarkerPrefix + profileId
}

func IsAdmin() (bool, error) {
	shell32 := syscall.NewLazyDLL("shell32.dll")
	proc := shell32.NewProc("IsUserAnAdmin")
	r, _, err := proc.Call()
	if r == 0 {
		if err != syscall.Errno(0) {
			return false, err
		}
		return false, nil
	}
	return true, nil
}

func ApplyProfileBlock(profileId string, lines []string, flushDNS bool) error {
	return updateProfileBlock(profileId, lines, flushDNS)
}

func RemoveProfileBlock(profileId string, flushDNS bool) error {
	return updateProfileBlock(profileId, nil, flushDNS)
}

func RemoveAllZephyBlocks(flushDNS bool) error {
	admin, err := IsAdmin()
	if err != nil {
		return fmt.Errorf("admin check failed: %w", err)
	}
	if !admin {
		return fmt.Errorf("需要管理员权限才能修改系统 hosts 文件")
	}

	orig, err := os.ReadFile(SystemHostsPath)
	if err != nil {
		return fmt.Errorf("read system hosts: %w", err)
	}

	useCRLF := bytes.Contains(orig, []byte("\r\n"))
	text := string(bytes.ReplaceAll(orig, []byte("\r\n"), []byte("\n")))

	cleaned := removeAllProfileBlocks(text)

	if cleaned == text {
		return nil
	}

	backupPath, err := backupHosts(orig)
	if err != nil {
		return err
	}

	out := cleaned
	if useCRLF {
		out = strings.ReplaceAll(out, "\n", "\r\n")
	}

	if err := os.WriteFile(SystemHostsPath, []byte(out), 0644); err != nil {
		_ = os.WriteFile(SystemHostsPath, orig, 0644)
		return fmt.Errorf("write system hosts failed (restored backup=%s): %w", backupPath, err)
	}

	if flushDNS {
		_ = exec.Command("ipconfig", "/flushdns").Run()
	}
	return nil
}

func removeAllProfileBlocks(text string) string {
	lines := strings.Split(text, "\n")
	var result []string
	inBlock := false
	var currentProfileId string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, startMarkerPrefix) {
			inBlock = true
			currentProfileId = strings.TrimPrefix(trimmed, startMarkerPrefix)
			continue
		}
		if inBlock && strings.HasPrefix(trimmed, endMarkerPrefix) {
			endId := strings.TrimPrefix(trimmed, endMarkerPrefix)
			if endId == currentProfileId {
				inBlock = false
				currentProfileId = ""
				continue
			}
		}
		if !inBlock {
			result = append(result, line)
		}
	}
	return strings.Join(result, "\n")
}

func GetEnabledProfiles() ([]string, error) {
	data, err := os.ReadFile(SystemHostsPath)
	if err != nil {
		return nil, err
	}
	text := string(bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n")))

	var ids []string
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, startMarkerPrefix) {
			id := strings.TrimPrefix(trimmed, startMarkerPrefix)
			if id != "" {
				ids = append(ids, id)
			}
		}
	}
	return ids, nil
}

func updateProfileBlock(profileId string, lines []string, flushDNS bool) error {
	admin, err := IsAdmin()
	if err != nil {
		return fmt.Errorf("admin check failed: %w", err)
	}
	if !admin {
		return fmt.Errorf("需要管理员权限才能修改系统 hosts 文件")
	}

	orig, err := os.ReadFile(SystemHostsPath)
	if err != nil {
		return fmt.Errorf("read system hosts: %w", err)
	}

	updated, changed := replaceProfileBlock(orig, profileId, lines)
	if !changed {
		return nil
	}

	backupPath, err := backupHosts(orig)
	if err != nil {
		return err
	}

	if err := os.WriteFile(SystemHostsPath, updated, 0644); err != nil {
		_ = os.WriteFile(SystemHostsPath, orig, 0644)
		return fmt.Errorf("write system hosts failed (restored backup=%s): %w", backupPath, err)
	}

	if flushDNS {
		_ = exec.Command("ipconfig", "/flushdns").Run()
	}
	return nil
}

func backupHosts(orig []byte) (string, error) {
	dir := filepath.Dir(SystemHostsPath)
	ts := time.Now().Format("20060102150405")
	backup := filepath.Join(dir, fmt.Sprintf("hosts.bak_%s", ts))
	if err := os.WriteFile(backup, orig, 0644); err != nil {
		return "", fmt.Errorf("backup hosts failed: %w", err)
	}
	return backup, nil
}

func replaceProfileBlock(orig []byte, profileId string, lines []string) ([]byte, bool) {
	useCRLF := bytes.Contains(orig, []byte("\r\n"))
	text := string(bytes.ReplaceAll(orig, []byte("\r\n"), []byte("\n")))

	startMarker := profileStartMarker(profileId)
	endMarker := profileEndMarker(profileId)

	profileRegex := regexp.MustCompile(
		`(?m)^` + regexp.QuoteMeta(startMarker) + `\r?\n[\s\S]*?` + regexp.QuoteMeta(endMarker) + `\r?\n?`,
	)
	base := profileRegex.ReplaceAllString(text, "")

	if lines == nil {
		out := base
		if useCRLF {
			out = strings.ReplaceAll(out, "\n", "\r\n")
		}
		if out == string(bytes.ReplaceAll(orig, []byte("\r\n"), []byte("\n"))) {
			return orig, false
		}
		return []byte(out), true
	}

	clean := make([]string, 0, len(lines))
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l == "" || strings.HasPrefix(l, "#") {
			continue
		}
		clean = append(clean, l)
	}
	if len(clean) == 0 {
		out := base
		if useCRLF {
			out = strings.ReplaceAll(out, "\n", "\r\n")
		}
		return []byte(out), base != text
	}

	block := startMarker + "\n" + strings.Join(clean, "\n") + "\n" + endMarker + "\n"

	out := base
	if out != "" && !strings.HasSuffix(out, "\n") {
		out += "\n"
	}
	out += block

	if useCRLF {
		out = strings.ReplaceAll(out, "\n", "\r\n")
	}

	return []byte(out), true
}

func ApplyManagedBlock(lines []string, flushDNS bool) error {
	return ApplyProfileBlock("default", lines, flushDNS)
}

func RemoveManagedBlock(flushDNS bool) error {
	return RemoveProfileBlock("default", flushDNS)
}

