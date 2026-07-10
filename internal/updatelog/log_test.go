package updatelog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenAppendsComponentsForSameSession(t *testing.T) {
	path := filepath.Join(t.TempDir(), "logs", Filename)
	logger, closer, err := Open(path, "session-1", "desktop")
	if err != nil {
		t.Fatal(err)
	}
	logger.Print("update started")
	if err := closer.Close(); err != nil {
		t.Fatal(err)
	}
	logger, closer, err = Open(path, "session-1", "helper")
	if err != nil {
		t.Fatal(err)
	}
	logger.Print("download started")
	if err := closer.Close(); err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(content)
	if !strings.Contains(text, "[session-1] [desktop]") || !strings.Contains(text, "update started") ||
		!strings.Contains(text, "[session-1] [helper]") || !strings.Contains(text, "download started") {
		t.Fatalf("unexpected log content: %q", text)
	}
}

func TestPathUsesApplicationDataDirectory(t *testing.T) {
	dataDir := filepath.Join("testdata", "TraceLog")
	want := filepath.Join(dataDir, "logs", Filename)
	if got := Path(dataDir); got != want {
		t.Fatalf("Path(%q) = %q, want %q", dataDir, got, want)
	}
}
