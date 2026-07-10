package main

import (
	"context"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRunRequiresAsset(t *testing.T) {
	err := run(context.Background(), options{appExe: "TraceLog.exe"}, log.New(io.Discard, "", 0))
	if err == nil || !strings.Contains(err.Error(), "missing update asset") {
		t.Fatalf("expected missing asset error, got %v", err)
	}
}

func TestRunRequiresAppExecutable(t *testing.T) {
	err := run(context.Background(), options{assetURL: "https://example.com/app.exe", assetName: "app.exe"}, log.New(io.Discard, "", 0))
	if err == nil || !strings.Contains(err.Error(), "missing app executable") {
		t.Fatalf("expected missing app executable error, got %v", err)
	}
}

func TestWaitForProcessExitWithoutPID(t *testing.T) {
	start := time.Now()
	if err := waitForProcessExit(0, time.Second, log.New(io.Discard, "", 0)); err != nil {
		t.Fatal(err)
	}
	if time.Since(start) > 3*time.Second {
		t.Fatal("wait without pid took too long")
	}
}

func TestNewLoggerPersistsHelperSession(t *testing.T) {
	path := filepath.Join(t.TempDir(), "logs", "update.log")
	logger, closeLog := newLogger(path, "update-session")
	logger.Print("helper diagnostic")
	closeLog()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(content)
	if !strings.Contains(text, "[update-session] [helper]") || !strings.Contains(text, "helper diagnostic") {
		t.Fatalf("unexpected helper log content: %q", text)
	}
}
