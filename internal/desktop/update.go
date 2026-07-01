package desktop

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	githubLatestReleaseURL = "https://api.github.com/repos/yleoer/TraceLog/releases/latest"
	updateHTTPTimeout      = 60 * time.Second
)

var AppVersion = "dev"

type UpdateInfo struct {
	CurrentVersion string `json:"current_version"`
	LatestVersion  string `json:"latest_version"`
	HasUpdate      bool   `json:"has_update"`
	ReleaseURL     string `json:"release_url"`
	AssetName      string `json:"asset_name"`
	AssetURL       string `json:"asset_url"`
	PublishedAt    string `json:"published_at"`
	ReleaseNotes   string `json:"release_notes"`
}

type UpdateInstallResult struct {
	Started bool   `json:"started"`
	Path    string `json:"path"`
	Message string `json:"message"`
}

type githubRelease struct {
	TagName     string        `json:"tag_name"`
	HTMLURL     string        `json:"html_url"`
	Body        string        `json:"body"`
	PublishedAt string        `json:"published_at"`
	Assets      []githubAsset `json:"assets"`
}

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

func (a *App) GetUpdateInfo() (UpdateInfo, error) {
	info, err := fetchUpdateInfo(context.Background())
	if err != nil {
		return UpdateInfo{CurrentVersion: normalizeVersion(AppVersion)}, err
	}
	return info, nil
}

func (a *App) InstallUpdate() (UpdateInstallResult, error) {
	info, err := fetchUpdateInfo(context.Background())
	if err != nil {
		return UpdateInstallResult{}, err
	}
	if !info.HasUpdate {
		return UpdateInstallResult{Message: "当前已是最新版本"}, nil
	}
	if info.AssetURL == "" {
		return UpdateInstallResult{}, fmt.Errorf("当前平台没有可用安装包")
	}

	path, err := downloadUpdateAsset(context.Background(), info)
	if err != nil {
		return UpdateInstallResult{}, err
	}
	if err := startInstaller(path); err != nil {
		return UpdateInstallResult{}, err
	}
	return UpdateInstallResult{Started: true, Path: path, Message: "安装程序已启动，请按提示完成升级"}, nil
}

func fetchUpdateInfo(ctx context.Context) (UpdateInfo, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, githubLatestReleaseURL, nil)
	if err != nil {
		return UpdateInfo{}, err
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("User-Agent", "TraceLog-Updater/"+normalizeVersion(AppVersion))

	client := &http.Client{Timeout: updateHTTPTimeout}
	response, err := client.Do(request)
	if err != nil {
		return UpdateInfo{}, fmt.Errorf("检查更新失败: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNotFound {
		return UpdateInfo{}, fmt.Errorf("尚未发布 GitHub Release")
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return UpdateInfo{}, fmt.Errorf("检查更新失败: GitHub 返回 %s", response.Status)
	}

	var release githubRelease
	if err := json.NewDecoder(response.Body).Decode(&release); err != nil {
		return UpdateInfo{}, fmt.Errorf("解析更新信息失败: %w", err)
	}
	asset := selectReleaseAsset(release.Assets)
	current := normalizeVersion(AppVersion)
	latest := normalizeVersion(release.TagName)
	return UpdateInfo{
		CurrentVersion: current,
		LatestVersion:  latest,
		HasUpdate:      compareVersions(latest, current) > 0,
		ReleaseURL:     release.HTMLURL,
		AssetName:      asset.Name,
		AssetURL:       asset.BrowserDownloadURL,
		PublishedAt:    release.PublishedAt,
		ReleaseNotes:   release.Body,
	}, nil
}

func selectReleaseAsset(assets []githubAsset) githubAsset {
	keywords := releaseAssetKeywords()
	for _, asset := range assets {
		name := strings.ToLower(asset.Name)
		matched := true
		for _, keyword := range keywords {
			if !strings.Contains(name, keyword) {
				matched = false
				break
			}
		}
		if matched {
			return asset
		}
	}
	return githubAsset{}
}

func releaseAssetKeywords() []string {
	switch runtime.GOOS {
	case "windows":
		return []string{"windows", "amd64", "installer.exe"}
	case "darwin":
		return []string{"macos", ".dmg"}
	case "linux":
		return []string{"linux", "amd64", ".deb"}
	default:
		return []string{runtime.GOOS}
	}
}

func downloadUpdateAsset(ctx context.Context, info UpdateInfo) (string, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, info.AssetURL, nil)
	if err != nil {
		return "", err
	}
	request.Header.Set("User-Agent", "TraceLog-Updater/"+info.CurrentVersion)

	client := &http.Client{Timeout: 10 * time.Minute}
	response, err := client.Do(request)
	if err != nil {
		return "", fmt.Errorf("下载安装包失败: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", fmt.Errorf("下载安装包失败: GitHub 返回 %s", response.Status)
	}

	dir := filepath.Join(os.TempDir(), "TraceLog", "updates")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, safeFilename(info.AssetName))
	file, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	if _, err := io.Copy(file, response.Body); err != nil {
		return "", err
	}
	return path, nil
}

func startInstaller(path string) error {
	switch runtime.GOOS {
	case "windows":
		return exec.Command(path).Start()
	case "darwin":
		return exec.Command("open", path).Start()
	case "linux":
		return exec.Command("xdg-open", path).Start()
	default:
		return fmt.Errorf("当前平台不支持自动启动安装包")
	}
}

func normalizeVersion(version string) string {
	version = strings.TrimSpace(version)
	if version == "" {
		return "dev"
	}
	return version
}

func compareVersions(left string, right string) int {
	left = strings.TrimPrefix(normalizeVersion(left), "v")
	right = strings.TrimPrefix(normalizeVersion(right), "v")
	if left == right {
		return 0
	}
	if left == "dev" {
		return -1
	}
	if right == "dev" {
		return 1
	}
	leftParts := versionParts(left)
	rightParts := versionParts(right)
	maxParts := len(leftParts)
	if len(rightParts) > maxParts {
		maxParts = len(rightParts)
	}
	for i := 0; i < maxParts; i++ {
		leftValue := partAt(leftParts, i)
		rightValue := partAt(rightParts, i)
		if leftValue > rightValue {
			return 1
		}
		if leftValue < rightValue {
			return -1
		}
	}
	return strings.Compare(left, right)
}

func versionParts(version string) []int {
	tokens := strings.FieldsFunc(version, func(r rune) bool {
		return r == '.' || r == '-' || r == '+'
	})
	parts := make([]int, 0, len(tokens))
	for _, token := range tokens {
		value := 0
		for _, r := range token {
			if r < '0' || r > '9' {
				break
			}
			value = value*10 + int(r-'0')
		}
		parts = append(parts, value)
	}
	return parts
}

func partAt(parts []int, index int) int {
	if index >= len(parts) {
		return 0
	}
	return parts[index]
}

func safeFilename(name string) string {
	name = filepath.Base(strings.TrimSpace(name))
	if name == "." || name == string(filepath.Separator) || name == "" {
		return "TraceLog-update"
	}
	return name
}
