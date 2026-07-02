package desktop

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	runtimeapp "tracelog/internal/app"
	"tracelog/internal/config"
	"tracelog/internal/db"
	"tracelog/internal/service"
)

const (
	desktopCallTimeout     = 2 * time.Minute
	desktopLongCallTimeout = 5 * time.Minute
)

type App struct {
	ctx        context.Context
	cfg        config.Config
	migrations fs.FS
	database   *sql.DB
	service    *service.Service
	uploadDir  string
}

type FileUpload struct {
	Name    string `json:"name"`
	Data    string `json:"data"`
	Context string `json:"context"`
}

type SaveResult struct {
	Path     string `json:"path"`
	Canceled bool   `json:"canceled"`
}

func NewApp(cfg config.Config, migrations fs.FS) *App {
	return &App{
		cfg:        cfg,
		migrations: migrations,
		uploadDir:  filepath.Join(cfg.DataDir, "uploads"),
	}
}

func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	database, err := db.Open(a.cfg)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	if err := db.MigrateFS(database, a.migrations, "db/migrations"); err != nil {
		_ = database.Close()
		log.Fatalf("migrate database: %v", err)
	}
	runtime, err := runtimeapp.NewRuntime(a.cfg, database)
	if err != nil {
		_ = database.Close()
		log.Fatalf("create runtime: %v", err)
	}
	a.database = database
	a.service = runtime.Service
	a.uploadDir = runtime.UploadDir
}

func (a *App) Shutdown(ctx context.Context) {
	_ = ctx
	if a.database != nil {
		_ = a.database.Close()
	}
}

func (a *App) callContext(timeout time.Duration) (context.Context, context.CancelFunc) {
	base := a.ctx
	if base == nil {
		base = context.Background()
	}
	return context.WithTimeout(base, timeout)
}

func (a *App) defaultCallContext() (context.Context, context.CancelFunc) {
	return a.callContext(desktopCallTimeout)
}

func (a *App) longCallContext() (context.Context, context.CancelFunc) {
	return a.callContext(desktopLongCallTimeout)
}

func (a *App) Dashboard() (service.Dashboard, error) {
	ctx, cancel := a.defaultCallContext()
	defer cancel()
	return a.service.Dashboard(ctx)
}

func (a *App) Today() (service.TodayWorkflow, error) {
	ctx, cancel := a.defaultCallContext()
	defer cancel()
	return a.service.TodayWorkflow(ctx)
}

func (a *App) GetSettings() (service.AppSettings, error) {
	ctx, cancel := a.defaultCallContext()
	defer cancel()
	return a.service.GetSettings(ctx)
}

func (a *App) UpdateSettings(settings service.AppSettings) (service.AppSettings, error) {
	ctx, cancel := a.longCallContext()
	defer cancel()
	return a.service.UpdateSettings(ctx, settings)
}

func (a *App) ListTimeWorkItems() ([]service.TimeWorkItem, error) {
	ctx, cancel := a.defaultCallContext()
	defer cancel()
	return a.service.ListTimeWorkItems(ctx)
}

func (a *App) GetTimeWeek(week string) (service.TimeWeekView, error) {
	ctx, cancel := a.defaultCallContext()
	defer cancel()
	return a.service.GetTimeWeek(ctx, week)
}

func (a *App) RefreshTimeWeek(week string) (service.TimeWeekView, error) {
	ctx, cancel := a.longCallContext()
	defer cancel()
	return a.service.RefreshTimeWeek(ctx, week)
}

func (a *App) LogTempoTime(request service.LogTimeRequest) (service.LogTimeResult, error) {
	ctx, cancel := a.longCallContext()
	defer cancel()
	return a.service.LogTempoTime(ctx, request)
}

func (a *App) ListIssues(filter service.IssueFilter) ([]service.Issue, error) {
	ctx, cancel := a.defaultCallContext()
	defer cancel()
	return a.service.ListIssues(ctx, filter)
}

func (a *App) ImportJiraIssue(jiraKey string) (service.Issue, error) {
	ctx, cancel := a.longCallContext()
	defer cancel()
	return a.service.ImportJiraIssue(ctx, jiraKey)
}

func (a *App) CreateIssue(issue service.Issue) (service.Issue, error) {
	ctx, cancel := a.defaultCallContext()
	defer cancel()
	return a.service.CreateIssue(ctx, issue)
}

func (a *App) GetIssue(jiraKey string) (service.Issue, error) {
	ctx, cancel := a.defaultCallContext()
	defer cancel()
	return a.service.GetIssue(ctx, jiraKey)
}

func (a *App) UpdateIssue(jiraKey string, issue service.Issue) (service.Issue, error) {
	ctx, cancel := a.defaultCallContext()
	defer cancel()
	return a.service.UpdateIssue(ctx, jiraKey, issue)
}

func (a *App) DeleteIssue(jiraKey string) (map[string]bool, error) {
	ctx, cancel := a.defaultCallContext()
	defer cancel()
	return ok(a.service.DeleteIssue(ctx, jiraKey))
}

func (a *App) GenerateIssueSummary(jiraKey string) (service.IssueSummaryResponse, error) {
	ctx, cancel := a.longCallContext()
	defer cancel()
	return a.service.GenerateIssueSummary(ctx, jiraKey)
}

func (a *App) ListIssueEvents(jiraKey string) ([]service.IssueEvent, error) {
	ctx, cancel := a.defaultCallContext()
	defer cancel()
	return a.service.ListIssueEvents(ctx, jiraKey)
}

func (a *App) CreateIssueEvent(jiraKey string, event service.IssueEvent) (service.IssueEvent, error) {
	ctx, cancel := a.defaultCallContext()
	defer cancel()
	return a.service.CreateIssueEvent(ctx, jiraKey, event)
}

func (a *App) UpdateIssueEvent(id int64, event service.IssueEvent) (service.IssueEvent, error) {
	ctx, cancel := a.defaultCallContext()
	defer cancel()
	return a.service.UpdateIssueEvent(ctx, id, event)
}

func (a *App) DeleteIssueEvent(id int64) (map[string]bool, error) {
	ctx, cancel := a.defaultCallContext()
	defer cancel()
	return ok(a.service.DeleteIssueEvent(ctx, id))
}

func (a *App) ListIssueTodos(jiraKey string, includeDone bool) ([]service.IssueTodo, error) {
	ctx, cancel := a.defaultCallContext()
	defer cancel()
	return a.service.ListIssueTodos(ctx, jiraKey, includeDone)
}

func (a *App) CreateIssueTodo(jiraKey string, todo service.IssueTodo) (service.IssueTodo, error) {
	ctx, cancel := a.defaultCallContext()
	defer cancel()
	return a.service.CreateIssueTodo(ctx, jiraKey, todo)
}

func (a *App) UpdateIssueTodo(id int64, todo service.IssueTodo) (service.IssueTodo, error) {
	ctx, cancel := a.defaultCallContext()
	defer cancel()
	return a.service.UpdateIssueTodo(ctx, id, todo)
}

func (a *App) DeleteIssueTodo(id int64) (map[string]bool, error) {
	ctx, cancel := a.defaultCallContext()
	defer cancel()
	return ok(a.service.DeleteIssueTodo(ctx, id))
}

func (a *App) UploadImage(file FileUpload) (service.UploadedImage, error) {
	data, err := decodeFileUpload(file)
	if err != nil {
		return service.UploadedImage{}, err
	}
	ctx, cancel := a.defaultCallContext()
	defer cancel()
	return a.service.SaveUploadedImage(ctx, service.UploadFile{
		Filename: file.Name,
		Context:  file.Context,
		Reader:   bytes.NewReader(data),
	})
}

func (a *App) GetUploadedImageDataURL(url string) (service.UploadedImageData, error) {
	ctx, cancel := a.defaultCallContext()
	defer cancel()
	return a.service.UploadedImageDataURL(ctx, url)
}

func (a *App) DeleteUploadedImage(url string) (map[string]bool, error) {
	ctx, cancel := a.defaultCallContext()
	defer cancel()
	deleted, err := a.service.DeleteUploadedImage(ctx, url)
	if err != nil {
		return nil, err
	}
	return map[string]bool{"ok": true, "deleted": deleted}, nil
}

func (a *App) CleanupUnusedUploadedImages() (service.UploadedImageCleanup, error) {
	ctx, cancel := a.longCallContext()
	defer cancel()
	return a.service.CleanupUnusedUploadedImages(ctx)
}

func (a *App) ListTempTasks(filter service.TempTaskFilter) ([]service.TempTask, error) {
	ctx, cancel := a.defaultCallContext()
	defer cancel()
	return a.service.ListTempTasks(ctx, filter)
}

func (a *App) CreateTempTask(task service.TempTask) (service.TempTask, error) {
	ctx, cancel := a.defaultCallContext()
	defer cancel()
	return a.service.CreateTempTask(ctx, task)
}

func (a *App) GetTempTask(id int64) (service.TempTask, error) {
	ctx, cancel := a.defaultCallContext()
	defer cancel()
	return a.service.GetTempTask(ctx, id)
}

func (a *App) UpdateTempTask(id int64, task service.TempTask) (service.TempTask, error) {
	ctx, cancel := a.defaultCallContext()
	defer cancel()
	return a.service.UpdateTempTask(ctx, id, task)
}

func (a *App) DeleteTempTask(id int64) (map[string]bool, error) {
	ctx, cancel := a.defaultCallContext()
	defer cancel()
	return ok(a.service.DeleteTempTask(ctx, id))
}

func (a *App) ListTempTaskEvents(id int64) ([]service.TempTaskEvent, error) {
	ctx, cancel := a.defaultCallContext()
	defer cancel()
	return a.service.ListTempTaskEvents(ctx, id)
}

func (a *App) CreateTempTaskEvent(id int64, event service.TempTaskEvent) (service.TempTaskEvent, error) {
	ctx, cancel := a.defaultCallContext()
	defer cancel()
	return a.service.CreateTempTaskEvent(ctx, id, event)
}

func (a *App) UpdateTempTaskEvent(id int64, event service.TempTaskEvent) (service.TempTaskEvent, error) {
	ctx, cancel := a.defaultCallContext()
	defer cancel()
	return a.service.UpdateTempTaskEvent(ctx, id, event)
}

func (a *App) DeleteTempTaskEvent(id int64) (map[string]bool, error) {
	ctx, cancel := a.defaultCallContext()
	defer cancel()
	return ok(a.service.DeleteTempTaskEvent(ctx, id))
}

func (a *App) CreateDayEntry(entry service.DayEntry) (service.DayEntry, error) {
	ctx, cancel := a.defaultCallContext()
	defer cancel()
	return a.service.CreateDayEntry(ctx, entry.Date, entry.ContentMD)
}

func (a *App) DeleteDayEntry(id int64) (map[string]bool, error) {
	ctx, cancel := a.defaultCallContext()
	defer cancel()
	return ok(a.service.DeleteDayEntry(ctx, id))
}

func (a *App) ListWeeks() ([]service.WeeklyLog, error) {
	ctx, cancel := a.defaultCallContext()
	defer cancel()
	return a.service.ListWeeklyLogs(ctx)
}

func (a *App) GetWeekBounds() (service.WeekBounds, error) {
	ctx, cancel := a.defaultCallContext()
	defer cancel()
	return a.service.GetWeekBounds(ctx)
}

func (a *App) GetWeek(week string) (service.WeekView, error) {
	ctx, cancel := a.defaultCallContext()
	defer cancel()
	return a.service.GetWeekView(ctx, week)
}

func (a *App) UpdateWeek(week string, log service.WeeklyLog) (service.WeeklyLog, error) {
	ctx, cancel := a.defaultCallContext()
	defer cancel()
	return a.service.UpsertWeeklyLog(ctx, week, log)
}

func (a *App) GenerateWeekDraft(week string) (service.WeeklyLog, error) {
	ctx, cancel := a.defaultCallContext()
	defer cancel()
	return a.service.GenerateWeekDraft(ctx, week)
}

func (a *App) GenerateWeekSummary(week string) (service.WeeklyLog, error) {
	ctx, cancel := a.longCallContext()
	defer cancel()
	return a.service.GenerateWeekSummary(ctx, week)
}

func (a *App) Search(query string, entityType string, limit int, offset int) ([]service.SearchResult, error) {
	ctx, cancel := a.defaultCallContext()
	defer cancel()
	return a.service.Search(ctx, query, entityType, limit, offset)
}

func (a *App) ExportJSON() (SaveResult, error) {
	ctx, cancel := a.longCallContext()
	defer cancel()
	data, err := a.service.ExportJSON(ctx)
	if err != nil {
		return SaveResult{}, err
	}
	return a.saveFile("tracelog-export.json", "JSON Files (*.json)", "*.json", data)
}

func (a *App) ExportMarkdownZip() (SaveResult, error) {
	ctx, cancel := a.longCallContext()
	defer cancel()
	data, err := a.service.ExportMarkdownZip(ctx)
	if err != nil {
		return SaveResult{}, err
	}
	return a.saveFile("tracelog-markdown.zip", "ZIP Archives (*.zip)", "*.zip", data)
}

func (a *App) ExportIssueMarkdown(jiraKey string) (SaveResult, error) {
	ctx, cancel := a.defaultCallContext()
	defer cancel()
	data, filename, err := a.service.ExportIssueMarkdown(ctx, strings.TrimSuffix(jiraKey, ".md"))
	if err != nil {
		return SaveResult{}, err
	}
	return a.saveFile(filename, "Markdown Files (*.md)", "*.md", data)
}

func (a *App) ExportWeekMarkdown(week string) (SaveResult, error) {
	ctx, cancel := a.defaultCallContext()
	defer cancel()
	data, filename, err := a.service.ExportWeekMarkdown(ctx, strings.TrimSuffix(week, ".md"))
	if err != nil {
		return SaveResult{}, err
	}
	return a.saveFile(filename, "Markdown Files (*.md)", "*.md", data)
}

func (a *App) ExportTempTaskMarkdown(id int64) (SaveResult, error) {
	ctx, cancel := a.defaultCallContext()
	defer cancel()
	data, filename, err := a.service.ExportTempTaskMarkdown(ctx, id)
	if err != nil {
		return SaveResult{}, err
	}
	return a.saveFile(filename, "Markdown Files (*.md)", "*.md", data)
}

func (a *App) saveFile(defaultFilename string, filterName string, filterPattern string, data []byte) (SaveResult, error) {
	if a.ctx == nil {
		return SaveResult{}, fmt.Errorf("desktop runtime is not ready")
	}
	path, err := wailsruntime.SaveFileDialog(a.ctx, wailsruntime.SaveDialogOptions{
		Title:                "Save Export",
		DefaultFilename:      defaultFilename,
		CanCreateDirectories: true,
		Filters: []wailsruntime.FileFilter{
			{DisplayName: filterName, Pattern: filterPattern},
		},
	})
	if err != nil {
		return SaveResult{}, err
	}
	if path == "" {
		return SaveResult{Canceled: true}, nil
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return SaveResult{}, err
	}
	return SaveResult{Path: path}, nil
}

func UploadHandler(uploadDir string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || !strings.HasPrefix(r.URL.Path, "/uploads/") {
			http.NotFound(w, r)
			return
		}
		name := strings.TrimPrefix(r.URL.Path, "/uploads/")
		if name == "" || strings.Contains(name, "/") || strings.Contains(name, "\\") {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, filepath.Join(uploadDir, name))
	})
}

func ok(err error) (map[string]bool, error) {
	if err != nil {
		return nil, err
	}
	return map[string]bool{"ok": true}, nil
}

func decodeFileUpload(file FileUpload) ([]byte, error) {
	if file.Data == "" {
		return nil, badRequest("image is required")
	}
	contentType := ""
	encoded := file.Data
	if strings.HasPrefix(encoded, "data:") {
		meta, value, found := strings.Cut(encoded, ",")
		if !found {
			return nil, badRequest("invalid image data")
		}
		if !strings.Contains(meta, ";base64") {
			return nil, badRequest("invalid image data")
		}
		contentType = strings.TrimPrefix(strings.TrimSuffix(meta, ";base64"), "data:")
		encoded = value
	}
	if base64DecodedLen(encoded) > service.MaxImageUploadBytes {
		return nil, badRequest("image must be 8 MB or smaller")
	}
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, badRequest("invalid image data")
	}
	if len(decoded) == 0 {
		return nil, badRequest("image is required")
	}
	if contentType != "" && !strings.HasPrefix(contentType, "image/") {
		return nil, badRequest("file must be an image")
	}
	return decoded, nil
}

func base64DecodedLen(encoded string) int {
	encoded = strings.TrimSpace(encoded)
	if encoded == "" {
		return 0
	}
	padding := 0
	if strings.HasSuffix(encoded, "==") {
		padding = 2
	} else if strings.HasSuffix(encoded, "=") {
		padding = 1
	}
	groups := (len(encoded) + 3) / 4
	return groups*3 - padding
}

func badRequest(message string) error {
	return &service.AppError{Code: http.StatusBadRequest, Message: message, Err: service.ErrBadRequest}
}
