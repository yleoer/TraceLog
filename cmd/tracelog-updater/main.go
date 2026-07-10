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
	"strings"
	"time"

	"tracelog/internal/updatelog"
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
	sessionID      string
}

func main() {
	opts := parseOptions()
	logger, closeLog := newLogger(opts.logFile, opts.sessionID)

	exitCode := 0
	if err := run(context.Background(), opts, logger); err != nil {
		logger.Printf("update failed: %v", err)
		exitCode = 1
	}
	closeLog()
	os.Exit(exitCode)
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
	flag.StringVar(&opts.sessionID, "session-id", "", "update log session id")
	flag.Parse()
	if opts.sessionID == "" {
		opts.sessionID = updatelog.NewSessionID()
	}
	return opts
}

func run(ctx context.Context, opts options, logger *log.Logger) error {
	if opts.assetURL == "" || opts.assetName == "" {
		return fmt.Errorf("missing update asset")
	}
	if opts.appExe == "" {
		return fmt.Errorf("missing app executable path")
	}
	logger.Printf("updater started helper_pid=%d platform=%s/%s app=%q app_pid=%d current_version=%q asset=%q digest=%q url=%q", os.Getpid(), runtime.GOOS, runtime.GOARCH, opts.appExe, opts.appPID, opts.currentVersion, opts.assetName, opts.assetDigest, opts.assetURL)

	logger.Printf("download phase started")
	path, err := updater.DownloadAsset(ctx, updater.AssetInfo{
		CurrentVersion: opts.currentVersion,
		AssetName:      opts.assetName,
		AssetURL:       opts.assetURL,
		AssetDigest:    opts.assetDigest,
	}, logger)
	if err != nil {
		return err
	}
	logger.Printf("download phase completed installer=%q", path)

	if err := waitForProcessExit(opts.appPID, defaultWaitTimeout, logger); err != nil {
		return err
	}

	if err := runInstaller(ctx, path, logger); err != nil {
		return err
	}

	if err := updater.CleanupDownloads(updater.DownloadDir(), ""); err != nil {
		logger.Printf("download cleanup warning: %v", err)
	} else {
		logger.Printf("download cleanup completed")
	}
	cleanupSelf(logger)
	pid, err := restartApp(opts.appExe)
	if err != nil {
		logger.Printf("restart failed: %v", err)
		logger.Printf("update installation completed but automatic restart failed")
	} else {
		logger.Printf("restart completed app=%q pid=%d", opts.appExe, pid)
		logger.Printf("update completed successfully")
	}
	return nil
}

func runInstaller(ctx context.Context, path string, logger *log.Logger) error {
	ctx, cancel := context.WithTimeout(ctx, installerTimeout)
	defer cancel()
	startedAt := time.Now()
	logger.Printf("installer phase started path=%q timeout=%s", path, installerTimeout)

	var command *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		script := "$process = Start-Process -FilePath $args[0] -Wait -PassThru; if ($null -ne $process.ExitCode) { Write-Output ('installer_exit_code=' + $process.ExitCode); exit $process.ExitCode }"
		command = exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-WindowStyle", "Hidden", "-ExecutionPolicy", "Bypass", "-Command", script, path)
	case "darwin":
		command = exec.CommandContext(ctx, "open", path)
	case "linux":
		command = exec.CommandContext(ctx, "xdg-open", path)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
	output, err := command.CombinedOutput()
	logger.Printf("installer command completed duration=%s output=%q", time.Since(startedAt), strings.TrimSpace(string(output)))
	if err != nil {
		logger.Printf("installer phase failed: %v", err)
		return fmt.Errorf("run installer: %w", err)
	}
	logger.Printf("installer phase completed duration=%s", time.Since(startedAt))
	return nil
}

func cleanupSelf(logger *log.Logger) {
	if runtime.GOOS != "windows" {
		logger.Printf("helper self-cleanup skipped platform=%s", runtime.GOOS)
		return
	}
	path, err := os.Executable()
	if err != nil {
		logger.Printf("helper self-cleanup path resolution failed: %v", err)
		return
	}
	logger.Printf("helper self-cleanup scheduling path=%q", path)
	script := "Start-Sleep -Seconds 3; Remove-Item -LiteralPath $args[0] -Force -ErrorAction SilentlyContinue"
	command := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-WindowStyle", "Hidden", "-ExecutionPolicy", "Bypass", "-Command", script, path)
	if err := command.Start(); err != nil {
		logger.Printf("cleanup self failed: %v", err)
		return
	}
	logger.Printf("helper self-cleanup process created pid=%d", command.Process.Pid)
	_ = command.Process.Release()
}

func restartApp(path string) (int, error) {
	if _, err := os.Stat(path); err != nil {
		return 0, err
	}
	command := exec.Command(path)
	command.Dir = filepath.Dir(path)
	if err := command.Start(); err != nil {
		return 0, err
	}
	pid := command.Process.Pid
	_ = command.Process.Release()
	return pid, nil
}

func waitForProcessExit(pid int, timeout time.Duration, logger *log.Logger) error {
	startedAt := time.Now()
	logger.Printf("wait for app exit started pid=%d timeout=%s", pid, timeout)
	if pid <= 0 || runtime.GOOS != "windows" {
		time.Sleep(2 * time.Second)
		logger.Printf("wait for app exit used fallback delay duration=%s", time.Since(startedAt))
		return nil
	}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		running, err := windowsProcessRunning(pid)
		if err != nil {
			logger.Printf("wait for app exit process query failed pid=%d error=%v", pid, err)
			return err
		}
		if !running {
			logger.Printf("app process exited pid=%d duration=%s", pid, time.Since(startedAt))
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	logger.Printf("wait for app exit timed out pid=%d duration=%s", pid, time.Since(startedAt))
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

func newLogger(path string, sessionID string) (*log.Logger, func()) {
	if path == "" {
		return log.New(os.Stderr, fmt.Sprintf("[%s] [helper] ", sessionID), log.LstdFlags|log.Lmicroseconds|log.LUTC), func() {}
	}
	logger, closer, err := updatelog.Open(path, sessionID, "helper")
	if err != nil {
		fallback := log.New(os.Stderr, fmt.Sprintf("[%s] [helper] ", sessionID), log.LstdFlags|log.Lmicroseconds|log.LUTC)
		fallback.Printf("open update log failed path=%q error=%v", path, err)
		return fallback, func() {}
	}
	return logger, func() { _ = closer.Close() }
}
