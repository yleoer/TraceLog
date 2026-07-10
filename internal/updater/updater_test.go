package updater

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCleanupDownloadsKeepsOnlyCurrentAsset(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"TraceLog-v0.1.3-windows-amd64-installer.exe":          "old",
		"TraceLog-v0.1.4-windows-amd64-installer.exe":          "current",
		"TraceLog-v0.1.4-windows-amd64-installer.exe.download": "partial",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	if err := CleanupDownloads(dir, "TraceLog-v0.1.4-windows-amd64-installer.exe"); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(dir, "TraceLog-v0.1.4-windows-amd64-installer.exe")); err != nil {
		t.Fatalf("expected current installer to remain: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "TraceLog-v0.1.3-windows-amd64-installer.exe")); !os.IsNotExist(err) {
		t.Fatalf("expected old installer to be removed, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "TraceLog-v0.1.4-windows-amd64-installer.exe.download")); !os.IsNotExist(err) {
		t.Fatalf("expected partial download to be removed, got %v", err)
	}
}

func TestExpectedSHA256Digest(t *testing.T) {
	hash := sha256.Sum256([]byte("installer"))
	digest := "sha256:" + hex.EncodeToString(hash[:])

	got, ok, err := expectedSHA256Digest(digest)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected sha256 digest to be used")
	}
	if got != hex.EncodeToString(hash[:]) {
		t.Fatalf("expected %q, got %q", hex.EncodeToString(hash[:]), got)
	}
}

func TestVerifyAssetDigestAllowsMissingOrDifferentAlgorithm(t *testing.T) {
	hash := sha256.Sum256([]byte("installer"))
	if err := VerifyAssetDigest("", hash[:]); err != nil {
		t.Fatalf("expected missing digest to be allowed: %v", err)
	}
	if err := VerifyAssetDigest("sha512:abc", hash[:]); err != nil {
		t.Fatalf("expected non-sha256 digest to be ignored: %v", err)
	}
}

func TestVerifyAssetDigestRejectsMismatch(t *testing.T) {
	hash := sha256.Sum256([]byte("installer"))
	other := sha256.Sum256([]byte("other"))
	if err := VerifyAssetDigest("sha256:"+hex.EncodeToString(other[:]), hash[:]); err == nil {
		t.Fatal("expected digest mismatch error")
	}
}

func TestDownloadAssetLogsTransferAndDigest(t *testing.T) {
	content := []byte("installer")
	hash := sha256.Sum256(content)
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Content-Type", "application/octet-stream")
		_, _ = response.Write(content)
	}))
	defer server.Close()

	tempDir := t.TempDir()
	t.Setenv("TEMP", tempDir)
	t.Setenv("TMP", tempDir)
	t.Setenv("TMPDIR", tempDir)
	var output bytes.Buffer
	logger := log.New(&output, "", 0)

	path, err := DownloadAsset(context.Background(), AssetInfo{
		CurrentVersion: "v0.1.6",
		AssetName:      "TraceLog-v0.1.7-windows-amd64-installer.exe",
		AssetURL:       server.URL,
		AssetDigest:    "sha256:" + hex.EncodeToString(hash[:]),
	}, logger)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(path)

	downloaded, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(downloaded, content) {
		t.Fatalf("unexpected downloaded content: %q", downloaded)
	}
	logs := output.String()
	for _, want := range []string{"download request started", "download response received", "download stream completed", "asset digest verification completed", "installer finalized"} {
		if !strings.Contains(logs, want) {
			t.Fatalf("expected log %q in %q", want, logs)
		}
	}
}
