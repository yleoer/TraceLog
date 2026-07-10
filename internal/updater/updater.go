package updater

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const downloadTimeout = 10 * time.Minute

type AssetInfo struct {
	CurrentVersion string
	AssetName      string
	AssetURL       string
	AssetDigest    string
}

func DownloadAsset(ctx context.Context, info AssetInfo, logger *log.Logger) (string, error) {
	startedAt := time.Now()
	logger.Printf("download request started url=%q asset=%q current_version=%q", info.AssetURL, info.AssetName, normalizeVersion(info.CurrentVersion))
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, info.AssetURL, nil)
	if err != nil {
		logger.Printf("download request creation failed: %v", err)
		return "", err
	}
	request.Header.Set("User-Agent", "TraceLog-Updater/"+normalizeVersion(info.CurrentVersion))

	client := &http.Client{Timeout: downloadTimeout}
	response, err := client.Do(request)
	if err != nil {
		logger.Printf("download request failed duration=%s error=%v", time.Since(startedAt), err)
		return "", fmt.Errorf("下载安装包失败: %w", err)
	}
	defer response.Body.Close()
	logger.Printf("download response received status=%q content_length=%d content_type=%q duration=%s", response.Status, response.ContentLength, response.Header.Get("Content-Type"), time.Since(startedAt))
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", fmt.Errorf("下载安装包失败: GitHub 返回 %s", response.Status)
	}

	dir := DownloadDir()
	logger.Printf("download directory resolved path=%q", dir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		logger.Printf("create download directory failed: %v", err)
		return "", err
	}
	targetName := safeFilename(info.AssetName)
	path := filepath.Join(dir, targetName)
	tempPath := path + ".download"
	if err := CleanupDownloads(dir, ""); err != nil {
		logger.Printf("pre-download cleanup warning: %v", err)
	} else {
		logger.Printf("pre-download cleanup completed")
	}
	if err := os.Remove(tempPath); err != nil && !os.IsNotExist(err) {
		logger.Printf("remove stale partial download warning path=%q error=%v", tempPath, err)
	}

	file, err := os.OpenFile(tempPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		logger.Printf("create partial download failed path=%q error=%v", tempPath, err)
		return "", err
	}
	logger.Printf("streaming response to partial file path=%q", tempPath)
	hasher := sha256.New()
	written, err := io.Copy(io.MultiWriter(file, hasher), response.Body)
	if err != nil {
		_ = file.Close()
		_ = os.Remove(tempPath)
		logger.Printf("download stream failed bytes=%d duration=%s error=%v", written, time.Since(startedAt), err)
		return "", err
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(tempPath)
		logger.Printf("close partial download failed path=%q error=%v", tempPath, err)
		return "", err
	}
	actualHash := hasher.Sum(nil)
	logger.Printf("download stream completed bytes=%d sha256=%s duration=%s", written, hex.EncodeToString(actualHash), time.Since(startedAt))
	if err := VerifyAssetDigest(info.AssetDigest, actualHash); err != nil {
		_ = os.Remove(tempPath)
		logger.Printf("asset digest verification failed expected=%q error=%v", info.AssetDigest, err)
		return "", err
	}
	logger.Printf("asset digest verification completed expected=%q", info.AssetDigest)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		_ = os.Remove(tempPath)
		logger.Printf("remove existing installer failed path=%q error=%v", path, err)
		return "", err
	}
	if err := os.Rename(tempPath, path); err != nil {
		_ = os.Remove(tempPath)
		logger.Printf("finalize installer failed source=%q target=%q error=%v", tempPath, path, err)
		return "", err
	}
	logger.Printf("installer finalized path=%q", path)
	if err := CleanupDownloads(dir, targetName); err != nil {
		logger.Printf("post-download cleanup warning: %v", err)
	} else {
		logger.Printf("post-download cleanup completed keep=%q", targetName)
	}
	return path, nil
}

func DownloadDir() string {
	return filepath.Join(os.TempDir(), "TraceLog", "updates")
}

func CleanupDownloads(dir string, keepName string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var firstErr error
	for _, entry := range entries {
		if entry.Name() == keepName {
			continue
		}
		if err := os.RemoveAll(filepath.Join(dir, entry.Name())); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func VerifyAssetDigest(digest string, actualHash []byte) error {
	expected, ok, err := expectedSHA256Digest(digest)
	if err != nil || !ok {
		return err
	}
	actual := hex.EncodeToString(actualHash)
	if actual != expected {
		return fmt.Errorf("安装包校验失败: SHA256 不匹配")
	}
	return nil
}

func expectedSHA256Digest(digest string) (string, bool, error) {
	digest = strings.TrimSpace(strings.ToLower(digest))
	if digest == "" {
		return "", false, nil
	}
	algorithm, value, found := strings.Cut(digest, ":")
	if !found {
		return "", false, fmt.Errorf("安装包校验信息格式无效")
	}
	if algorithm != "sha256" {
		return "", false, nil
	}
	value = strings.TrimSpace(value)
	if len(value) != sha256.Size*2 {
		return "", false, fmt.Errorf("安装包 SHA256 校验信息无效")
	}
	if _, err := hex.DecodeString(value); err != nil {
		return "", false, fmt.Errorf("安装包 SHA256 校验信息无效")
	}
	return value, true, nil
}

func safeFilename(name string) string {
	name = filepath.Base(strings.TrimSpace(name))
	if name == "." || name == string(filepath.Separator) || name == "" {
		return "TraceLog-update"
	}
	return name
}

func normalizeVersion(version string) string {
	version = strings.TrimSpace(version)
	if version == "" {
		return "dev"
	}
	return version
}
