package service

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type UploadFile struct {
	Filename string
	Context  string
	Reader   io.Reader
}

const maxImageUploadBytes = 8 << 20
const MaxImageUploadBytes = maxImageUploadBytes

func (s *UploadService) SaveUploadedImage(ctx context.Context, file UploadFile) (UploadedImage, error) {
	_ = ctx
	if s.uploadDir == "" {
		return UploadedImage{}, badRequest("upload storage is not configured")
	}
	if file.Reader == nil {
		return UploadedImage{}, badRequest("image is required")
	}
	data, err := io.ReadAll(io.LimitReader(file.Reader, maxImageUploadBytes+1))
	if err != nil {
		return UploadedImage{}, fmt.Errorf("read uploaded image: %w", err)
	}
	if len(data) == 0 {
		return UploadedImage{}, badRequest("image is required")
	}
	if len(data) > maxImageUploadBytes {
		return UploadedImage{}, badRequest("image must be 8 MB or smaller")
	}
	contentType := http.DetectContentType(data)
	ext, ok := imageExtension(contentType)
	if !ok {
		return UploadedImage{}, badRequest("file must be a png, jpeg, gif, or webp image")
	}
	if err := os.MkdirAll(s.uploadDir, 0o755); err != nil {
		return UploadedImage{}, fmt.Errorf("create upload directory: %w", err)
	}
	filename := uploadFilename(file.Context, file.Filename, time.Now().In(s.loc), ext)
	if err := os.WriteFile(filepath.Join(s.uploadDir, filename), data, 0o644); err != nil {
		return UploadedImage{}, fmt.Errorf("save uploaded image: %w", err)
	}
	return UploadedImage{
		URL:         "/uploads/" + filename,
		Filename:    filename,
		ContentType: contentType,
		Size:        int64(len(data)),
	}, nil
}

func (s *UploadService) UploadedImageDataURL(ctx context.Context, url string) (UploadedImageData, error) {
	_ = ctx
	filename, err := uploadFilenameFromURL(url)
	if err != nil {
		return UploadedImageData{}, err
	}
	if s.uploadDir == "" {
		return UploadedImageData{}, badRequest("upload storage is not configured")
	}
	data, err := os.ReadFile(filepath.Join(s.uploadDir, filename))
	if err != nil {
		return UploadedImageData{}, mapUploadReadError(err)
	}
	contentType := http.DetectContentType(data)
	if _, ok := imageExtension(contentType); !ok {
		return UploadedImageData{}, badRequest("file must be a png, jpeg, gif, or webp image")
	}
	return UploadedImageData{
		URL:     "/uploads/" + filename,
		DataURL: "data:" + contentType + ";base64," + base64.StdEncoding.EncodeToString(data),
	}, nil
}

func (s *UploadService) DeleteUploadedImage(ctx context.Context, url string) (bool, error) {
	filename, err := uploadFilenameFromURL(url)
	if err != nil {
		return false, err
	}
	if s.uploadDir == "" {
		return false, badRequest("upload storage is not configured")
	}
	return s.deleteUploadedImageFile(ctx, filename)
}

func (s *UploadService) deleteUploadedImageFile(ctx context.Context, filename string) (bool, error) {
	normalizedURL := "/uploads/" + filename
	if referenceRepo, ok := s.repo.(UploadRepository); ok {
		referenced, err := referenceRepo.UploadedImageReferenced(ctx, normalizedURL)
		if err != nil {
			return false, fmt.Errorf("check uploaded image references: %w", err)
		}
		if referenced {
			return false, nil
		}
	}
	if err := os.Remove(filepath.Join(s.uploadDir, filename)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return false, fmt.Errorf("delete uploaded image: %w", err)
	}
	return true, nil
}

func (s *UploadService) CleanupUnusedUploadedImages(ctx context.Context) (UploadedImageCleanup, error) {
	var result UploadedImageCleanup
	if s.uploadDir == "" {
		return result, badRequest("upload storage is not configured")
	}
	entries, err := os.ReadDir(s.uploadDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return result, nil
		}
		return result, fmt.Errorf("read upload directory: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		filename := entry.Name()
		if !isUploadedImageFilename(filename) {
			continue
		}
		result.Scanned++
		info, statErr := entry.Info()
		if statErr != nil {
			result.Failed++
			continue
		}
		deleted, deleteErr := s.deleteUploadedImageFile(ctx, filename)
		if deleteErr != nil {
			result.Failed++
			continue
		}
		if deleted {
			result.Deleted++
			result.FreedBytes += info.Size()
			continue
		}
		result.Kept++
	}
	return result, nil
}
