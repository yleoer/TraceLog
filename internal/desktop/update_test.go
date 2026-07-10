package desktop

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		left  string
		right string
		want  int
	}{
		{"v0.2.0", "v0.1.9", 1},
		{"v1.0.0", "1.0.0", 0},
		{"dev", "v0.1.0", -1},
		{"v0.1.0", "dev", 1},
		{"v0.1.0", "v0.1.0", 0},
	}
	for _, test := range tests {
		got := compareVersions(test.left, test.right)
		if got != test.want {
			t.Fatalf("compareVersions(%q, %q) = %d, want %d", test.left, test.right, got, test.want)
		}
	}
}

func TestShouldSkipUpdateCheckForDevVersion(t *testing.T) {
	tests := []struct {
		version string
		want    bool
	}{
		{"", true},
		{"dev", true},
		{"DEV", true},
		{"dev-local", true},
		{"0.1.0-dev", true},
		{"v0.1.0", false},
		{"0.1.0", false},
	}
	for _, test := range tests {
		got := shouldSkipUpdateCheck(test.version)
		if got != test.want {
			t.Fatalf("shouldSkipUpdateCheck(%q) = %t, want %t", test.version, got, test.want)
		}
	}
}

func TestSkippedUpdateInfo(t *testing.T) {
	info := skippedUpdateInfo("dev")

	if !info.Skipped {
		t.Fatal("expected update check to be skipped")
	}
	if info.CurrentVersion != "dev" || info.LatestVersion != "dev" {
		t.Fatalf("expected dev versions, got current=%q latest=%q", info.CurrentVersion, info.LatestVersion)
	}
	if info.HasUpdate {
		t.Fatal("expected no update for skipped dev check")
	}
	if info.Message == "" {
		t.Fatal("expected skip message")
	}
	if info.CheckedAt == "" {
		t.Fatal("expected checked_at timestamp")
	}
}

func TestSelectReleaseAssetForCurrentPlatform(t *testing.T) {
	assets := []githubAsset{
		{Name: "TraceLog-v0.1.0-linux-amd64.deb", BrowserDownloadURL: "linux"},
		{Name: "TraceLog-v0.1.0-macos-universal.dmg", BrowserDownloadURL: "mac"},
		{Name: "TraceLog-v0.1.0-windows-amd64-installer.exe", BrowserDownloadURL: "windows"},
	}

	asset := selectReleaseAsset(assets)

	switch runtime.GOOS {
	case "windows":
		if asset.BrowserDownloadURL != "windows" {
			t.Fatalf("expected windows asset, got %#v", asset)
		}
	case "darwin":
		if asset.BrowserDownloadURL != "mac" {
			t.Fatalf("expected mac asset, got %#v", asset)
		}
	case "linux":
		if asset.BrowserDownloadURL != "linux" {
			t.Fatalf("expected linux asset, got %#v", asset)
		}
	}
}

func TestCachedUpdateInfoForInstall(t *testing.T) {
	originalVersion := AppVersion
	AppVersion = "v0.1.5"
	t.Cleanup(func() { AppVersion = originalVersion })

	app := &App{}
	info := UpdateInfo{
		CurrentVersion: "v0.1.5",
		LatestVersion:  "v0.1.6",
		HasUpdate:      true,
		AssetURL:       "https://example.com/update.exe",
	}
	app.cacheUpdateInfo(info)

	got, ok := app.cachedUpdateInfoForInstall()
	if !ok {
		t.Fatal("expected recently checked update info to be reused")
	}
	if got.AssetURL != info.AssetURL {
		t.Fatalf("expected asset URL %q, got %q", info.AssetURL, got.AssetURL)
	}
}

func TestCachedUpdateInfoForInstallRejectsExpiredInfo(t *testing.T) {
	app := &App{
		cachedUpdateInfo:   UpdateInfo{CurrentVersion: normalizeVersion(AppVersion), LatestVersion: "v9.9.9"},
		updateInfoCachedAt: time.Now().Add(-updateInfoCacheTTL - time.Second),
	}

	if _, ok := app.cachedUpdateInfoForInstall(); ok {
		t.Fatal("expected expired update info to be rejected")
	}
}

func TestPrepareUpdateHelperUsesUniqueTemporaryFiles(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows updater helper is only copied on Windows")
	}
	source := filepath.Join(t.TempDir(), "TraceLogUpdater.exe")
	if err := os.WriteFile(source, []byte("updater"), 0o755); err != nil {
		t.Fatal(err)
	}

	first, err := prepareUpdateHelper(source)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(first)
	second, err := prepareUpdateHelper(source)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(second)

	if first == second {
		t.Fatalf("expected unique helper paths, got %q", first)
	}
	if !strings.HasSuffix(strings.ToLower(first), ".exe") || !strings.HasSuffix(strings.ToLower(second), ".exe") {
		t.Fatalf("expected executable helper paths, got %q and %q", first, second)
	}
}

func TestOpenDesktopUpdateLoggerPersistsSession(t *testing.T) {
	path := filepath.Join(t.TempDir(), "logs", "update.log")
	logger, closeLog, err := openDesktopUpdateLogger(path, "update-session")
	if err != nil {
		t.Fatal(err)
	}
	logger.Print("desktop diagnostic")
	closeLog()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(content)
	if !strings.Contains(text, "[update-session] [desktop]") || !strings.Contains(text, "desktop diagnostic") {
		t.Fatalf("unexpected desktop log content: %q", text)
	}
}
