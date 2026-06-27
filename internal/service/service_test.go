package service

import (
	"bytes"
	"context"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

func TestSaveUploadedImageStoresPNG(t *testing.T) {
	dir := t.TempDir()
	svc := New(nil, nil, nil, dir)

	image, err := svc.SaveUploadedImage(context.Background(), UploadFile{
		Filename: "note.png",
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
	if _, err := os.Stat(filepath.Join(dir, image.Filename)); err != nil {
		t.Fatal(err)
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

var testPNG = []byte{
	0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
	0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
	0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
	0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4,
	0x89,
}
