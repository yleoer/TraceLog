package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"tracelog/internal/updater"
)

const (
	defaultWaitTimeout = 90 * time.Second
	installerTimeout   = 30 * time.Minute
)

type options struct {
	assetURL       string
	assetName      string
	assetDigest    string
	currentVersion string
	appExe         string
	appPID         int
	logFile        string
}

func main() {
	opts := parseOptions()
	logger, closeLog := newLogger(opts.logFile)
	defer closeLog()

	if err := run(context.Background(), opts, logger); err != nil {
		logger.Printf("update failed: %v", err)
		os.Exit(1)
	}
}

func parseOptions() options {
	var opts options
	flag.StringVar(&opts.assetURL, "asset-url", "", "release asset download URL")
	flag.StringVar(&opts.assetName, "asset-name", "", "release asset filename")
	flag.StringVar(&opts.assetDigest, "asset-digest", "", "release asset digest")
	flag.StringVar(&opts.currentVersion, "current-version", "dev", "current app version")
	flag.StringVar(&opts.appExe, "app-exe", "", "TraceLog executable path")
	flag.IntVar(&opts.appPID, "app-pid", 0, "TraceLog process id")
	flag.StringVar(&opts.logFile, "log-file", "", "updater log file")
	flag.Parse()
	return opts
}

func run(ctx context.Context, opts options, logger *log.Logger) error {
	if opts.assetURL == "" || opts.assetName == "" {
		return fmt.Errorf("missing update asset")
	}
	if opts.appExe == "" {
		return fmt.Errorf("missing app executable path")
	}
	logger.Printf("updater started, app=%q pid=%d asset=%q", opts.appExe, opts.appPID, opts.assetName)

	path, err := updater.DownloadAsset(ctx, updater.AssetInfo{
		CurrentVersion: opts.currentVersion,
		AssetName:      opts.assetName,
		AssetURL:       opts.assetURL,
		AssetDigest:    opts.assetDigest,
	})
	if err != nil {
		return err
	}
	logger.Printf("downloaded installer: %s", path)

	if err := waitForProcessExit(opts.appPID, defaultWaitTimeout); err != nil {
		return err
	}
	logger.Printf("app process exited")

	if err := runInstaller(ctx, path); err != nil {
		return err
	}
	logger.Printf("installer finished")

	_ = updater.CleanupDownloads(updater.DownloadDir(), "")
	cleanupSelf(logger)
	if err := restartApp(opts.appExe); err != nil {
		logger.Printf("restart failed: %v", err)
	}
	return nil
}

func runInstaller(ctx context.Context, path string) error {
	ctx, cancel := context.WithTimeout(ctx, installerTimeout)
	defer cancel()

	var command *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		script := "$process = Start-Process -FilePath $args[0] -Wait -PassThru; if ($null -ne $process.ExitCode) { exit $process.ExitCode }"
		command = exec.CommandContext(ctx, "powershell", "-NoProfile", "-WindowStyle", "Hidden", "-ExecutionPolicy", "Bypass", "-Command", script, path)
	case "darwin":
		command = exec.CommandContext(ctx, "open", path)
	case "linux":
		command = exec.CommandContext(ctx, "xdg-open", path)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
	if err := command.Run(); err != nil {
		return fmt.Errorf("run installer: %w", err)
	}
	return nil
}

func cleanupSelf(logger *log.Logger) {
	if runtime.GOOS != "windows" {
		return
	}
	path, err := os.Executable()
	if err != nil {
		return
	}
	script := "Start-Sleep -Seconds 3; Remove-Item -LiteralPath $args[0] -Force -ErrorAction SilentlyContinue"
	if err := exec.Command("powershell", "-NoProfile", "-WindowStyle", "Hidden", "-ExecutionPolicy", "Bypass", "-Command", script, path).Start(); err != nil {
		logger.Printf("cleanup self failed: %v", err)
	}
}

func restartApp(path string) error {
	if _, err := os.Stat(path); err != nil {
		return err
	}
	command := exec.Command(path)
	command.Dir = filepath.Dir(path)
	return command.Start()
}

func waitForProcessExit(pid int, timeout time.Duration) error {
	if pid <= 0 || runtime.GOOS != "windows" {
		time.Sleep(2 * time.Second)
		return nil
	}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		running, err := windowsProcessRunning(pid)
		if err != nil {
			return err
		}
		if !running {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("TraceLog 仍在运行，请关闭后重试")
}

func windowsProcessRunning(pid int) (bool, error) {
	command := exec.Command("powershell", "-NoProfile", "-Command", "if (Get-Process -Id $args[0] -ErrorAction SilentlyContinue) { exit 0 } else { exit 1 }", fmt.Sprint(pid))
	err := command.Run()
	if err == nil {
		return true, nil
	}
	if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
		return false, nil
	}
	return false, err
}

func newLogger(path string) (*log.Logger, func()) {
	if path == "" {
		return log.New(os.Stderr, "", log.LstdFlags), func() {}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return log.New(os.Stderr, "", log.LstdFlags), func() {}
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return log.New(os.Stderr, "", log.LstdFlags), func() {}
	}
	return log.New(file, "", log.LstdFlags), func() { _ = file.Close() }
}
