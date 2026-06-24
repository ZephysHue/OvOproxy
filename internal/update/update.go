package update

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

const apiURL = "https://api.github.com/repos/ZephysHue/OvOproxy/releases/latest"

type ReleaseInfo struct {
	TagName     string    `json:"tag_name"`
	Name        string    `json:"name"`
	Body        string    `json:"body"`
	HTMLURL     string    `json:"html_url"`
	PublishedAt time.Time `json:"published_at"`
	Assets      []Asset   `json:"assets"`
}

type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

type CheckResult struct {
	HasUpdate   bool
	Current     string
	Latest      string
	Changelog   string
	DownloadURL string
}

var (
	currentVersion = "dev"
	latestRelease  *ReleaseInfo
)

func SetVersion(v string) {
	currentVersion = v
}

func CheckForUpdate() (*CheckResult, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("API 请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API 返回 HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1*1024*1024))
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	var release ReleaseInfo
	if err := json.Unmarshal(body, &release); err != nil {
		return nil, fmt.Errorf("解析 JSON 失败: %w", err)
	}

	if release.TagName == "" {
		return nil, fmt.Errorf("release 无 tag_name")
	}

	latestRelease = &release

	result := &CheckResult{
		Current:   currentVersion,
		Latest:    release.TagName,
		Changelog: release.Body,
	}

	if currentVersion == "dev" {
		result.HasUpdate = false
		return result, nil
	}

	if release.TagName != currentVersion {
		result.HasUpdate = true
		// 优先找 .exe 资产
		for _, a := range release.Assets {
			ext := filepath.Ext(a.Name)
			if ext == ".exe" || ext == ".EXE" {
				result.DownloadURL = a.BrowserDownloadURL
				break
			}
		}
	}

	return result, nil
}

func DownloadUpdate(downloadURL string) (string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("获取 exe 路径失败: %w", err)
	}
	exeDir := filepath.Dir(exePath)
	tmpPath := filepath.Join(exeDir, "OvOproxy.exe.new")

	client := &http.Client{Timeout: 10 * time.Minute}
	resp, err := client.Get(downloadURL)
	if err != nil {
		return "", fmt.Errorf("下载失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("下载返回 HTTP %d", resp.StatusCode)
	}

	f, err := os.Create(tmpPath)
	if err != nil {
		return "", fmt.Errorf("创建文件失败: %w", err)
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	if err != nil {
		os.Remove(tmpPath)
		return "", fmt.Errorf("写入文件失败: %w", err)
	}

	return tmpPath, nil
}

func GetCurrentVersion() string {
	return currentVersion
}

func ApplyAndRestart() error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("获取 exe 路径失败: %w", err)
	}
	exeDir := filepath.Dir(exePath)
	newExe := filepath.Join(exeDir, "OvOproxy.exe.new")

	batPath := filepath.Join(os.TempDir(), "ovoproxy_update.bat")
	bat := fmt.Sprintf(`@echo off
timeout /t 2 /nobreak >nul
move /y "%s" "%s"
start "" "%s"
del "%%~f0"
`, escapePath(newExe), escapePath(exePath), escapePath(exePath))

	if err := os.WriteFile(batPath, []byte(bat), 0644); err != nil {
		return fmt.Errorf("写入 update.bat 失败: %w", err)
	}

	cmd := exec.Command("cmd", "/c", batPath)
	cmd.SysProcAttr = getDetachAttr()
	return cmd.Start()
}

func escapePath(p string) string {
	return p
}
