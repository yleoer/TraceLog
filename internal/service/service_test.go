package service

import (
	"bytes"
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSaveUploadedImageStoresPNG(t *testing.T) {
	dir := t.TempDir()
	svc := New(nil, nil, nil, dir)

	image, err := svc.SaveUploadedImage(context.Background(), UploadFile{
		Filename: "note.png",
		Context:  "issue-GCS-45000-comment",
		Reader:   bytes.NewReader(testPNG),
	})
	if err != nil {
		t.Fatal(err)
	}

	if image.URL == "" || image.Filename == "" {
		t.Fatalf("expected upload response fields, got %#v", image)
	}
	if image.ContentType != "image/png" {
		t.Fatalf("expected image/png, got %q", image.ContentType)
	}
	if !strings.HasPrefix(image.Filename, "issue-gcs-45000-comment-") || !strings.Contains(image.Filename, "-note-") {
		t.Fatalf("expected context prefix in filename, got %q", image.Filename)
	}
	if _, err := os.Stat(filepath.Join(dir, image.Filename)); err != nil {
		t.Fatal(err)
	}
}

func TestUploadFilenameSanitizesContext(t *testing.T) {
	name := uploadFilename("../Issue GCS-45000 评论!@#", "..\\截图 01.png", time.Date(2026, 6, 30, 15, 30, 12, 0, time.UTC), ".png")
	if !strings.HasPrefix(name, "issue-gcs-45000-20260630T153012-01-") {
		t.Fatalf("expected sanitized context and timestamp, got %q", name)
	}
	if strings.Contains(name, "..") || strings.Contains(name, "/") || strings.Contains(name, "\\") {
		t.Fatalf("expected safe filename, got %q", name)
	}
}

func TestSaveUploadedImageRejectsNonImage(t *testing.T) {
	svc := New(nil, nil, nil, t.TempDir())

	_, err := svc.SaveUploadedImage(context.Background(), UploadFile{
		Filename: "note.txt",
		Reader:   bytes.NewBufferString("plain text"),
	})
	if err == nil {
		t.Fatal("expected error")
	}
	appErr, ok := err.(*AppError)
	if !ok || appErr.Code != http.StatusBadRequest {
		t.Fatalf("expected bad request AppError, got %#v", err)
	}
}

func TestUploadedImageDataURLReturnsDataURL(t *testing.T) {
	dir := t.TempDir()
	svc := New(nil, nil, nil, dir)

	image, err := svc.SaveUploadedImage(context.Background(), UploadFile{
		Filename: "note.png",
		Reader:   bytes.NewReader(testPNG),
	})
	if err != nil {
		t.Fatal(err)
	}

	data, err := svc.UploadedImageDataURL(context.Background(), image.URL)
	if err != nil {
		t.Fatal(err)
	}
	if data.URL != image.URL {
		t.Fatalf("expected url %q, got %q", image.URL, data.URL)
	}
	if !strings.HasPrefix(data.DataURL, "data:image/png;base64,") {
		t.Fatalf("expected png data url, got %q", data.DataURL)
	}
}

func TestUploadedImageDataURLRejectsInvalidURL(t *testing.T) {
	svc := New(nil, nil, nil, t.TempDir())

	_, err := svc.UploadedImageDataURL(context.Background(), "../note.png")
	if err == nil {
		t.Fatal("expected error")
	}
	appErr, ok := err.(*AppError)
	if !ok || appErr.Code != http.StatusBadRequest {
		t.Fatalf("expected bad request AppError, got %#v", err)
	}
}

func TestDeleteUploadedImageRemovesUnreferencedFile(t *testing.T) {
	dir := t.TempDir()
	svc := New(nil, nil, nil, dir)

	image, err := svc.SaveUploadedImage(context.Background(), UploadFile{
		Filename: "note.png",
		Reader:   bytes.NewReader(testPNG),
	})
	if err != nil {
		t.Fatal(err)
	}

	deleted, err := svc.DeleteUploadedImage(context.Background(), image.URL)
	if err != nil {
		t.Fatal(err)
	}
	if !deleted {
		t.Fatal("expected image to be deleted")
	}
	if _, err := os.Stat(filepath.Join(dir, image.Filename)); !os.IsNotExist(err) {
		t.Fatalf("expected uploaded file to be removed, got %v", err)
	}
}

func TestDeleteUploadedImageKeepsReferencedFile(t *testing.T) {
	dir := t.TempDir()
	svc := New(nil, nil, nil, dir)

	image, err := svc.SaveUploadedImage(context.Background(), UploadFile{
		Filename: "note.png",
		Reader:   bytes.NewReader(testPNG),
	})
	if err != nil {
		t.Fatal(err)
	}
	svc.repo = uploadReferenceRepo{url: image.URL}

	deleted, err := svc.DeleteUploadedImage(context.Background(), image.URL)
	if err != nil {
		t.Fatal(err)
	}
	if deleted {
		t.Fatal("expected referenced image to be kept")
	}
	if _, err := os.Stat(filepath.Join(dir, image.Filename)); err != nil {
		t.Fatal(err)
	}
}

func TestDeleteUploadedImageRejectsInvalidURL(t *testing.T) {
	svc := New(nil, nil, nil, t.TempDir())

	_, err := svc.DeleteUploadedImage(context.Background(), "../note.png")
	if err == nil {
		t.Fatal("expected error")
	}
	appErr, ok := err.(*AppError)
	if !ok || appErr.Code != http.StatusBadRequest {
		t.Fatalf("expected bad request AppError, got %#v", err)
	}
}

func TestCleanupUnusedUploadedImagesRemovesOnlyUnreferencedImages(t *testing.T) {
	dir := t.TempDir()
	svc := New(nil, nil, nil, dir)

	unused, err := svc.SaveUploadedImage(context.Background(), UploadFile{
		Filename: "unused.png",
		Reader:   bytes.NewReader(testPNG),
	})
	if err != nil {
		t.Fatal(err)
	}
	referenced, err := svc.SaveUploadedImage(context.Background(), UploadFile{
		Filename: "referenced.png",
		Reader:   bytes.NewReader(testPNG),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("keep me"), 0o644); err != nil {
		t.Fatal(err)
	}
	svc.repo = uploadReferenceRepo{url: referenced.URL}

	result, err := svc.CleanupUnusedUploadedImages(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result.Scanned != 2 || result.Deleted != 1 || result.Kept != 1 || result.Failed != 0 {
		t.Fatalf("unexpected cleanup result: %#v", result)
	}
	if result.FreedBytes <= 0 {
		t.Fatalf("expected freed bytes, got %#v", result)
	}
	if _, err := os.Stat(filepath.Join(dir, unused.Filename)); !os.IsNotExist(err) {
		t.Fatalf("expected unused image to be removed, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, referenced.Filename)); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "notes.txt")); err != nil {
		t.Fatal(err)
	}
}

func TestCleanupUnusedUploadedImagesMissingDirectory(t *testing.T) {
	svc := New(nil, nil, nil, filepath.Join(t.TempDir(), "missing"))

	result, err := svc.CleanupUnusedUploadedImages(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result != (UploadedImageCleanup{}) {
		t.Fatalf("expected empty cleanup result, got %#v", result)
	}
}

func TestBucketByDayGroupsCommentsByReference(t *testing.T) {
	days := []time.Time{time.Date(2026, 6, 23, 0, 0, 0, 0, time.UTC)}
	comments := []DayComment{
		{Source: "issue", RefID: 1, RefKey: "GCS-1", RefTitle: "Issue one", EventType: "created", ContentMD: "", HappenedAt: "2026-06-23T09:00:00Z"},
		{Source: "issue", RefID: 1, RefKey: "GCS-1", RefTitle: "Issue one", EventType: "note", ContentMD: "comment", HappenedAt: "2026-06-23T10:00:00Z"},
		{Source: "temp_task", RefID: 2, RefTitle: "Task one", EventType: "deleted", ContentMD: "删除临时需求", HappenedAt: "2026-06-23T11:00:00Z"},
	}

	result := bucketByDay(time.UTC, days, comments, nil)

	if len(result) != 1 || len(result[0].Activities) != 2 {
		t.Fatalf("expected two grouped activities, got %#v", result)
	}
	if len(result[0].Activities[0].Comments) != 2 {
		t.Fatalf("expected issue comments grouped together, got %#v", result[0].Activities[0])
	}
	if result[0].Activities[1].Comments[0].EventType != "deleted" {
		t.Fatalf("expected deleted temp task activity, got %#v", result[0].Activities[1])
	}
}

type uploadReferenceRepo struct {
	Repository
	url string
}

func (repo uploadReferenceRepo) UploadedImageReferenced(_ context.Context, url string) (bool, error) {
	return url == repo.url, nil
}

var testPNG = []byte{
	0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
	0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
	0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
	0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4,
	0x89,
}
