package desktop

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"tracelog/internal/updatelog"
	"tracelog/internal/updater"
)

const (
	githubLatestReleaseURL      = "https://api.github.com/repos/yleoer/TraceLog/releases/latest"
	updateURLOverrideEnv        = "TRACELOG_UPDATE_URL"
	updateHTTPTimeout           = 60 * time.Second
	updateDownloadTimeout       = 10 * time.Minute
	updateHelperReadyTimeout    = 5 * time.Second
	updateHelperReadySignal     = "ready"
	updateInfoCacheTTL          = 15 * time.Minute
	devUpdateMessage            = "开发版本不检查更新"
	updaterExecutableName       = "TraceLogUpdater.exe"
	updateHelperReadyFilePrefix = "TraceLogUpdater-ready-"
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
	sessionID := updatelog.NewSessionID()
	logPath := updatelog.Path(a.cfg.DataDir)
	logger, closeLog, logErr := openDesktopUpdateLogger(logPath, sessionID)
	defer closeLog()
	if logErr != nil {
		logger.Printf("update check log unavailable: %v", logErr)
	}
	logger.Printf("update check requested current=%q platform=%s/%s pid=%d", normalizeVersion(AppVersion), runtime.GOOS, runtime.GOARCH, os.Getpid())
	if shouldSkipUpdateCheck(AppVersion) {
		logger.Printf("update check skipped: %s", devUpdateMessage)
		return skippedUpdateInfo(AppVersion), nil
	}
	ctx, cancel := a.longCallContext()
	defer cancel()
	info, err := fetchUpdateInfo(ctx, logger)
	if err != nil {
		logger.Printf("update check failed: %v", err)
		return UpdateInfo{CurrentVersion: normalizeVersion(AppVersion), CheckedAt: nowRFC3339()}, err
	}
	a.cacheUpdateInfo(info)
	logger.Printf("update check completed current=%q latest=%q has_update=%t asset=%q", info.CurrentVersion, info.LatestVersion, info.HasUpdate, info.AssetName)
	return info, nil
}

func (a *App) InstallUpdate() (UpdateInstallResult, error) {
	sessionID := updatelog.NewSessionID()
	logPath := updatelog.Path(a.cfg.DataDir)
	logger, closeLog, err := openDesktopUpdateLogger(logPath, sessionID)
	defer closeLog()
	if err != nil {
		return updateInstallFailure("无法创建升级日志", err, logPath), nil
	}
	logger.Printf("update install requested current=%q platform=%s/%s app_pid=%d", normalizeVersion(AppVersion), runtime.GOOS, runtime.GOARCH, os.Getpid())
	if shouldSkipUpdateCheck(AppVersion) {
		logger.Printf("update install rejected: development version")
		return UpdateInstallResult{Path: logPath, Message: "开发版本不支持在线升级"}, nil
	}
	info, ok := a.cachedUpdateInfoForInstall()
	logger.Printf("cached update info available=%t", ok)
	if !ok {
		ctx, cancel := a.longCallContext()
		defer cancel()
		info, err = fetchUpdateInfo(ctx, logger)
		if err != nil {
			logger.Printf("update info refresh failed: %v", err)
			return updateInstallFailure("无法获取最新版本信息", err, logPath), nil
		}
		a.cacheUpdateInfo(info)
	}
	logger.Printf("update candidate current=%q latest=%q has_update=%t asset=%q digest=%q url=%q", info.CurrentVersion, info.LatestVersion, info.HasUpdate, info.AssetName, info.AssetDigest, info.AssetURL)
	if !info.HasUpdate {
		logger.Printf("update install stopped: current version is latest")
		return UpdateInstallResult{Path: logPath, Message: "当前已是最新版本"}, nil
	}
	if info.AssetURL == "" {
		logger.Printf("update install stopped: no matching release asset")
		return UpdateInstallResult{Path: logPath, Message: "当前平台没有可用安装包，请从 Release 页面手动下载"}, nil
	}

	logger.Printf("desktop download phase started asset=%q", info.AssetName)
	downloadCtx, cancel := a.callContext(updateDownloadTimeout)
	defer cancel()
	installerPath, err := updater.DownloadAsset(downloadCtx, updater.AssetInfo{
		CurrentVersion: info.CurrentVersion,
		AssetName:      info.AssetName,
		AssetURL:       info.AssetURL,
		AssetDigest:    info.AssetDigest,
	}, logger)
	if err != nil {
		logger.Printf("desktop download phase failed: %v", err)
		return updateInstallFailure("下载安装包失败", err, logPath), nil
	}
	logger.Printf("desktop download phase completed installer=%q", installerPath)

	helperLogPath, err := startUpdateHelper(installerPath, logPath, sessionID, logger)
	if err != nil {
		logger.Printf("update helper launch failed: %v", err)
		return updateInstallFailure("无法启动更新助手", err, logPath), nil
	}
	logger.Printf("update helper is ready log=%q frontend_quit_required=true", helperLogPath)
	return UpdateInstallResult{
		Started:  true,
		Path:     helperLogPath,
		Message:  "安装包已下载并验证，更新助手已就绪，TraceLog 即将关闭并开始安装",
		WillQuit: true,
	}, nil
}

func (a *App) cacheUpdateInfo(info UpdateInfo) {
	a.updateMu.Lock()
	defer a.updateMu.Unlock()
	a.cachedUpdateInfo = info
	a.updateInfoCachedAt = time.Now()
}

func (a *App) cachedUpdateInfoForInstall() (UpdateInfo, bool) {
	a.updateMu.RLock()
	defer a.updateMu.RUnlock()
	if a.updateInfoCachedAt.IsZero() || time.Since(a.updateInfoCachedAt) > updateInfoCacheTTL {
		return UpdateInfo{}, false
	}
	info := a.cachedUpdateInfo
	if info.CurrentVersion != normalizeVersion(AppVersion) || info.LatestVersion == "" {
		return UpdateInfo{}, false
	}
	return info, true
}

func updateInstallFailure(action string, err error, logPath string) UpdateInstallResult {
	return UpdateInstallResult{
		Path:    logPath,
		Message: fmt.Sprintf("%s：%v。请从 Release 页面手动下载安装包。升级日志：%s", action, err, logPath),
	}
}

func fetchUpdateInfo(ctx context.Context, logger *log.Logger) (UpdateInfo, error) {
	startedAt := time.Now()
	releaseURL := latestReleaseURL()
	logger.Printf("release request started method=%s url=%q", http.MethodGet, releaseURL)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, releaseURL, nil)
	if err != nil {
		logger.Printf("release request creation failed: %v", err)
		return UpdateInfo{}, err
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("User-Agent", "TraceLog-Updater/"+normalizeVersion(AppVersion))

	client := &http.Client{Timeout: updateHTTPTimeout}
	response, err := client.Do(request)
	if err != nil {
		logger.Printf("release request failed duration=%s error=%v", time.Since(startedAt), err)
		return UpdateInfo{}, fmt.Errorf("检查更新失败: %w", err)
	}
	defer response.Body.Close()
	logger.Printf("release response received status=%q content_length=%d duration=%s", response.Status, response.ContentLength, time.Since(startedAt))
	if response.StatusCode == http.StatusNotFound {
		return UpdateInfo{}, fmt.Errorf("尚未发布 GitHub Release")
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return UpdateInfo{}, fmt.Errorf("检查更新失败: GitHub 返回 %s", response.Status)
	}

	var release githubRelease
	if err := json.NewDecoder(response.Body).Decode(&release); err != nil {
		logger.Printf("release response decode failed: %v", err)
		return UpdateInfo{}, fmt.Errorf("解析更新信息失败: %w", err)
	}
	asset := selectReleaseAsset(release.Assets)
	logger.Printf("release response decoded tag=%q assets=%d selected_asset=%q published_at=%q", release.TagName, len(release.Assets), asset.Name, release.PublishedAt)
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

func openDesktopUpdateLogger(path string, sessionID string) (*log.Logger, func(), error) {
	logger, closer, err := updatelog.Open(path, sessionID, "desktop")
	if err != nil {
		return updatelog.Discard(sessionID, "desktop"), func() {}, err
	}
	return logger, func() { _ = closer.Close() }, nil
}

func startUpdateHelper(installerPath string, logPath string, sessionID string, logger *log.Logger) (string, error) {
	logger.Printf("locating update helper name=%q", updaterExecutableName)
	helperPath, err := findUpdateHelper()
	if err != nil {
		return "", err
	}
	logger.Printf("update helper found source=%q", helperPath)
	helperPath, err = prepareUpdateHelper(helperPath)
	if err != nil {
		return "", err
	}
	logger.Printf("update helper copied target=%q", helperPath)
	executablePath, err := os.Executable()
	if err != nil {
		return "", err
	}
	logger.Printf("current executable resolved path=%q", executablePath)
	readyPath, err := newUpdateHelperReadyPath()
	if err != nil {
		return "", err
	}
	defer os.Remove(readyPath)
	args := []string{
		"--installer-path", installerPath,
		"--app-exe", executablePath,
		"--app-pid", fmt.Sprint(os.Getpid()),
		"--log-file", logPath,
		"--session-id", sessionID,
		"--ready-file", readyPath,
	}
	command := exec.Command(helperPath, args...)
	configureUpdateHelperCommand(command)
	logger.Printf("starting update helper executable=%q installer=%q app_pid=%d log=%q ready=%q", helperPath, installerPath, os.Getpid(), logPath, readyPath)
	if err := command.Start(); err != nil {
		return "", err
	}
	logger.Printf("update helper process created pid=%d", command.Process.Pid)
	_ = command.Process.Release()
	if err := waitForUpdateHelperReady(readyPath, updateHelperReadyTimeout); err != nil {
		return "", err
	}
	logger.Printf("update helper readiness acknowledged path=%q", readyPath)
	return logPath, nil
}

func latestReleaseURL() string {
	if value := strings.TrimSpace(os.Getenv(updateURLOverrideEnv)); value != "" {
		return value
	}
	return githubLatestReleaseURL
}

func newUpdateHelperReadyPath() (string, error) {
	dir := filepath.Join(os.TempDir(), "TraceLog", "updater")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	file, err := os.CreateTemp(dir, updateHelperReadyFilePrefix+"*.signal")
	if err != nil {
		return "", err
	}
	path := file.Name()
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		return "", err
	}
	if err := os.Remove(path); err != nil {
		return "", err
	}
	return path, nil
}

func waitForUpdateHelperReady(path string, timeout time.Duration) error {
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		content, err := os.ReadFile(path)
		if err == nil {
			if strings.TrimSpace(string(content)) == updateHelperReadySignal {
				return nil
			}
			return fmt.Errorf("更新助手就绪信号无效")
		}
		if !os.IsNotExist(err) {
			return fmt.Errorf("读取更新助手就绪信号: %w", err)
		}

		select {
		case <-timer.C:
			return fmt.Errorf("更新助手未在 %s 内就绪", timeout)
		case <-ticker.C:
		}
	}
}

func prepareUpdateHelper(source string) (string, error) {
	if runtime.GOOS != "windows" {
		return source, nil
	}
	dir := filepath.Join(os.TempDir(), "TraceLog", "updater")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	input, err := os.Open(source)
	if err != nil {
		return "", err
	}
	defer input.Close()
	output, err := os.CreateTemp(dir, "TraceLogUpdater-run-*.exe")
	if err != nil {
		return "", err
	}
	target := output.Name()
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
