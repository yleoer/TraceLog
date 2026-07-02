package service

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func imageExtension(contentType string) (string, bool) {
	switch contentType {
	case "image/png":
		return ".png", true
	case "image/jpeg":
		return ".jpg", true
	case "image/gif":
		return ".gif", true
	case "image/webp":
		return ".webp", true
	default:
		return "", false
	}
}

func uploadFilenameFromURL(url string) (string, error) {
	name := strings.TrimPrefix(strings.TrimSpace(url), "/uploads/")
	if name == "" || name == url || strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return "", badRequest("invalid upload url")
	}
	return name, nil
}

func isUploadedImageFilename(filename string) bool {
	if filename != filepath.Base(filename) {
		return false
	}
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp":
		return true
	default:
		return false
	}
}

func uploadFilename(context string, originalFilename string, uploadedAt time.Time, ext string) string {
	prefix := uploadContextSlug(context)
	if prefix == "" {
		prefix = "upload"
	}
	base := uploadBaseName(originalFilename)
	original := uploadContextSlug(strings.TrimSuffix(base, filepath.Ext(base)))
	if original == "" {
		return fmt.Sprintf("%s-%s-%s%s", prefix, uploadedAt.Format("20060102T150405"), randomHex(4), ext)
	}
	return fmt.Sprintf("%s-%s-%s-%s%s", prefix, uploadedAt.Format("20060102T150405"), original, randomHex(4), ext)
}

func uploadBaseName(filename string) string {
	filename = strings.TrimSpace(strings.ReplaceAll(filename, "\\", "/"))
	return filepath.Base(filename)
}

func uploadContextSlug(context string) string {
	context = strings.ToLower(strings.TrimSpace(context))
	var b strings.Builder
	lastDash := false
	for _, r := range context {
		isToken := r >= 'a' && r <= 'z' || r >= '0' && r <= '9'
		if isToken {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if b.Len() > 0 && !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

func mapUploadReadError(err error) error {
	if errors.Is(err, os.ErrNotExist) {
		return &AppError{Code: http.StatusNotFound, Message: "uploaded image not found", Err: ErrNotFound}
	}
	return fmt.Errorf("read uploaded image: %w", err)
}

func randomHex(bytesCount int) string {
	buffer := make([]byte, bytesCount)
	if _, err := rand.Read(buffer); err != nil {
		return fmt.Sprint(time.Now().UnixNano())
	}
	return hex.EncodeToString(buffer)
}
