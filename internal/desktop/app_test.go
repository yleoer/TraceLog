package desktop

import (
	"encoding/base64"
	"net/http"
	"strings"
	"testing"

	"tracelog/internal/service"
)

func TestDecodeFileUploadRejectsOversizedBase64BeforeDecode(t *testing.T) {
	encodedLen := (service.MaxImageUploadBytes + 1 + 2) / 3 * 4
	_, err := decodeFileUpload(FileUpload{
		Name: "huge.png",
		Data: "data:image/png;base64," + strings.Repeat("A", encodedLen),
	})
	if err == nil {
		t.Fatal("expected oversized upload to be rejected")
	}
	appErr, ok := err.(*service.AppError)
	if !ok || appErr.Code != http.StatusBadRequest {
		t.Fatalf("expected bad request AppError, got %#v", err)
	}
}

func TestBase64DecodedLenAccountsForPadding(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString([]byte("hello"))
	if got := base64DecodedLen(encoded); got != 5 {
		t.Fatalf("expected decoded len 5, got %d", got)
	}
}
