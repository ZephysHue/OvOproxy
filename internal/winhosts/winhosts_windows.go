//go:build windows

package winhosts

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

const (
	SystemHostsPath = `C:\Windows\System32\drivers\etc\hosts`
	StartMarker     = "# >>> Zephy Managed Start"
	EndMarker       = "# <<< Zephy Managed End"
)

func IsAdmin() (bool, error) {
	// shell32!IsUserAnAdmin
	shell32 := syscall.NewLazyDLL("shell32.dll")
	proc := shell32.NewProc("IsUserAnAdmin")
	r, _, err := proc.Call()
	if r == 0 {
		// err may be "operation completed successfully"
		if err != syscall.Errno(0) {
			return false, err
		}
		return false, nil
	}
	return true, nil
}

func ApplyManagedBlock(lines []string, flushDNS bool) error {
	return updateManagedBlock(lines, flushDNS)
}

func RemoveManagedBlock(flushDNS bool) error {
	return updateManagedBlock(nil, flushDNS)
}

func updateManagedBlock(lines []string, flushDNS bool) error {
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

	updated, changed, err := replaceBlock(orig, lines)
	if err != nil {
		return err
	}
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

func replaceBlock(orig []byte, lines []string) ([]byte, bool, error) {
	// Preserve original line endings
	useCRLF := bytes.Contains(orig, []byte("\r\n"))
	norm := bytes.ReplaceAll(orig, []byte("\r\n"), []byte("\n"))
	text := string(norm)

	startIdx := strings.Index(text, StartMarker)
	endIdx := strings.Index(text, EndMarker)

	if (startIdx >= 0) != (endIdx >= 0) {
		return nil, false, errors.New("system hosts: Zephy 标记块不完整（只有 start 或只有 end）")
	}

	stripBlock := func(s string) string {
		if startIdx < 0 {
			return s
		}
		// find line start of StartMarker
		startLineStart := strings.LastIndex(s[:startIdx], "\n")
		if startLineStart < 0 {
			startLineStart = 0
		} else {
			startLineStart += 1
		}
		// find line end of EndMarker
		endLineEnd := strings.Index(s[endIdx:], "\n")
		if endLineEnd < 0 {
			endLineEnd = len(s)
		} else {
			endLineEnd = endIdx + endLineEnd + 1
		}
		return s[:startLineStart] + s[endLineEnd:]
	}

	base := stripBlock(text)

	// If removing: just return base
	if lines == nil {
		out := base
		if useCRLF {
			out = strings.ReplaceAll(out, "\n", "\r\n")
		}
		if out == string(norm) {
			return orig, false, nil
		}
		return []byte(out), true, nil
	}

	// Build new managed block
	clean := make([]string, 0, len(lines))
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l == "" || strings.HasPrefix(l, "#") {
			continue
		}
		clean = append(clean, l)
	}
	block := StartMarker + "\n" + strings.Join(clean, "\n") + "\n" + EndMarker + "\n"

	// Append at end, preserving existing trailing newline behavior
	out := base
	if out != "" && !strings.HasSuffix(out, "\n") {
		out += "\n"
	}
	out += block

	if useCRLF {
		out = strings.ReplaceAll(out, "\n", "\r\n")
	}
	if out == string(norm) {
		return orig, false, nil
	}
	return []byte(out), true, nil
}

