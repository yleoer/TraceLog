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

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	githubLatestReleaseURL     = "https://api.github.com/repos/yleoer/TraceLog/releases/latest"
	updateHTTPTimeout          = 60 * time.Second
	updateInstallerLaunchDelay = 500 * time.Millisecond
	devUpdateMessage           = "开发版本不检查更新"
	updaterExecutableName      = "TraceLogUpdater.exe"
)

var AppVersion = "dev"

type UpdateInfo struct {
	CurrentVersion string `json:"current_version"`
	LatestVersion  string `json:"latest_version"`
	HasUpdate      bool   `json:"has_update"`
	Skipped        bool   `json:"skipped"`
	Message        string `json:"message"`
	CheckedAt      string `json:"checked_at"`
	ReleaseURL     string `json:"release_url"`
	AssetName      string `json:"asset_name"`
	AssetURL       string `json:"asset_url"`
	AssetDigest    string `json:"asset_digest"`
	PublishedAt    string `json:"published_at"`
	ReleaseNotes   string `json:"release_notes"`
}

type UpdateInstallResult struct {
	Started  bool   `json:"started"`
	Path     string `json:"path"`
	Message  string `json:"message"`
	WillQuit bool   `json:"will_quit"`
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
	Digest             string `json:"digest"`
}

func (a *App) GetUpdateInfo() (UpdateInfo, error) {
	if shouldSkipUpdateCheck(AppVersion) {
		return skippedUpdateInfo(AppVersion), nil
	}
	info, err := fetchUpdateInfo(context.Background())
	if err != nil {
		return UpdateInfo{CurrentVersion: normalizeVersion(AppVersion), CheckedAt: nowRFC3339()}, err
	}
	return info, nil
}

func (a *App) InstallUpdate() (UpdateInstallResult, error) {
	if shouldSkipUpdateCheck(AppVersion) {
		return UpdateInstallResult{Message: "开发版本不支持在线升级"}, nil
	}
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

	helperLogPath, err := startUpdateHelper(info)
	if err != nil {
		return UpdateInstallResult{}, err
	}
	willQuit := a.quitSoon()
	message := "TraceLog 将退出，更新助手会下载并安装新版本"
	if !willQuit {
		message = "更新助手已启动，会下载并安装新版本"
	}
	return UpdateInstallResult{Started: true, Path: helperLogPath, Message: message, WillQuit: willQuit}, nil
}

func (a *App) quitSoon() bool {
	if a.ctx == nil {
		return false
	}
	ctx := a.ctx
	go func() {
		time.Sleep(updateInstallerLaunchDelay)
		wailsruntime.Quit(ctx)
	}()
	return true
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
		CheckedAt:      nowRFC3339(),
		ReleaseURL:     release.HTMLURL,
		AssetName:      asset.Name,
		AssetURL:       asset.BrowserDownloadURL,
		AssetDigest:    asset.Digest,
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

func startUpdateHelper(info UpdateInfo) (string, error) {
	helperPath, err := findUpdateHelper()
	if err != nil {
		return "", err
	}
	helperPath, err = prepareUpdateHelper(helperPath)
	if err != nil {
		return "", err
	}
	executablePath, err := os.Executable()
	if err != nil {
		return "", err
	}
	logPath := filepath.Join(os.TempDir(), "TraceLog", "TraceLogUpdater.log")
	_ = os.MkdirAll(filepath.Dir(logPath), 0o755)
	args := []string{
		"--asset-url", info.AssetURL,
		"--asset-name", info.AssetName,
		"--asset-digest", info.AssetDigest,
		"--current-version", info.CurrentVersion,
		"--app-exe", executablePath,
		"--app-pid", fmt.Sprint(os.Getpid()),
		"--log-file", logPath,
	}
	if err := exec.Command(helperPath, args...).Start(); err != nil {
		return "", err
	}
	return logPath, nil
}

func prepareUpdateHelper(source string) (string, error) {
	if runtime.GOOS != "windows" {
		return source, nil
	}
	dir := filepath.Join(os.TempDir(), "TraceLog", "updater")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	target := filepath.Join(dir, fmt.Sprintf("TraceLogUpdater-run-%d.exe", os.Getpid()))
	input, err := os.Open(source)
	if err != nil {
		return "", err
	}
	defer input.Close()
	output, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(output, input); err != nil {
		_ = output.Close()
		_ = os.Remove(target)
		return "", err
	}
	if err := output.Close(); err != nil {
		_ = os.Remove(target)
		return "", err
	}
	return target, nil
}

func findUpdateHelper() (string, error) {
	switch runtime.GOOS {
	case "windows":
		executablePath, err := os.Executable()
		if err != nil {
			return "", err
		}
		path := filepath.Join(filepath.Dir(executablePath), updaterExecutableName)
		if _, err := os.Stat(path); err != nil {
			return "", fmt.Errorf("更新助手不存在，请下载安装包手动升级")
		}
		return path, nil
	case "darwin":
		return "", fmt.Errorf("当前平台暂不支持更新助手，请下载安装包手动升级")
	case "linux":
		return "", fmt.Errorf("当前平台暂不支持更新助手，请下载安装包手动升级")
	default:
		return "", fmt.Errorf("当前平台不支持自动升级")
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

func shouldSkipUpdateCheck(version string) bool {
	return isDevVersion(version)
}

func isDevVersion(version string) bool {
	version = strings.ToLower(normalizeVersion(version))
	version = strings.TrimPrefix(version, "v")
	return version == "dev" || strings.HasPrefix(version, "dev-") || strings.HasSuffix(version, "-dev")
}

func skippedUpdateInfo(version string) UpdateInfo {
	current := normalizeVersion(version)
	return UpdateInfo{
		CurrentVersion: current,
		LatestVersion:  current,
		Skipped:        true,
		Message:        devUpdateMessage,
		CheckedAt:      nowRFC3339(),
	}
}

func nowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}
