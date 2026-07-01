package desktop

import (
	"runtime"
	"testing"
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
