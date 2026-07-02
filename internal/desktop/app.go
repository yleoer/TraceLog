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

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	runtimeapp "tracelog/internal/app"
	"tracelog/internal/config"
	"tracelog/internal/db"
	"tracelog/internal/service"
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

func (a *App) Dashboard() (service.Dashboard, error) {
	return a.service.Dashboard(context.Background())
}

func (a *App) Today() (service.TodayWorkflow, error) {
	return a.service.TodayWorkflow(context.Background())
}

func (a *App) GetSettings() (service.AppSettings, error) {
	return a.service.GetSettings(context.Background())
}

func (a *App) UpdateSettings(settings service.AppSettings) (service.AppSettings, error) {
	return a.service.UpdateSettings(context.Background(), settings)
}

func (a *App) ListTimeWorkItems() ([]service.TimeWorkItem, error) {
	return a.service.ListTimeWorkItems(context.Background())
}

func (a *App) GetTimeWeek(week string) (service.TimeWeekView, error) {
	return a.service.GetTimeWeek(context.Background(), week)
}

func (a *App) RefreshTimeWeek(week string) (service.TimeWeekView, error) {
	return a.service.RefreshTimeWeek(context.Background(), week)
}

func (a *App) LogTempoTime(request service.LogTimeRequest) (service.LogTimeResult, error) {
	return a.service.LogTempoTime(context.Background(), request)
}

func (a *App) ListIssues(filter service.IssueFilter) ([]service.Issue, error) {
	return a.service.ListIssues(context.Background(), filter)
}

func (a *App) ImportJiraIssue(jiraKey string) (service.Issue, error) {
	return a.service.ImportJiraIssue(context.Background(), jiraKey)
}

func (a *App) CreateIssue(issue service.Issue) (service.Issue, error) {
	return a.service.CreateIssue(context.Background(), issue)
}

func (a *App) GetIssue(jiraKey string) (service.Issue, error) {
	return a.service.GetIssue(context.Background(), jiraKey)
}

func (a *App) UpdateIssue(jiraKey string, issue service.Issue) (service.Issue, error) {
	return a.service.UpdateIssue(context.Background(), jiraKey, issue)
}

func (a *App) DeleteIssue(jiraKey string) (map[string]bool, error) {
	return ok(a.service.DeleteIssue(context.Background(), jiraKey))
}

func (a *App) GenerateIssueSummary(jiraKey string) (service.IssueSummaryResponse, error) {
	return a.service.GenerateIssueSummary(context.Background(), jiraKey)
}

func (a *App) ListIssueEvents(jiraKey string) ([]service.IssueEvent, error) {
	return a.service.ListIssueEvents(context.Background(), jiraKey)
}

func (a *App) CreateIssueEvent(jiraKey string, event service.IssueEvent) (service.IssueEvent, error) {
	return a.service.CreateIssueEvent(context.Background(), jiraKey, event)
}

func (a *App) UpdateIssueEvent(id int64, event service.IssueEvent) (service.IssueEvent, error) {
	return a.service.UpdateIssueEvent(context.Background(), id, event)
}

func (a *App) DeleteIssueEvent(id int64) (map[string]bool, error) {
	return ok(a.service.DeleteIssueEvent(context.Background(), id))
}

func (a *App) ListIssueTodos(jiraKey string, includeDone bool) ([]service.IssueTodo, error) {
	return a.service.ListIssueTodos(context.Background(), jiraKey, includeDone)
}

func (a *App) CreateIssueTodo(jiraKey string, todo service.IssueTodo) (service.IssueTodo, error) {
	return a.service.CreateIssueTodo(context.Background(), jiraKey, todo)
}

func (a *App) UpdateIssueTodo(id int64, todo service.IssueTodo) (service.IssueTodo, error) {
	return a.service.UpdateIssueTodo(context.Background(), id, todo)
}

func (a *App) DeleteIssueTodo(id int64) (map[string]bool, error) {
	return ok(a.service.DeleteIssueTodo(context.Background(), id))
}

func (a *App) UploadImage(file FileUpload) (service.UploadedImage, error) {
	data, err := decodeFileUpload(file)
	if err != nil {
		return service.UploadedImage{}, err
	}
	return a.service.SaveUploadedImage(context.Background(), service.UploadFile{
		Filename: file.Name,
		Context:  file.Context,
		Reader:   bytes.NewReader(data),
	})
}

func (a *App) GetUploadedImageDataURL(url string) (service.UploadedImageData, error) {
	return a.service.UploadedImageDataURL(context.Background(), url)
}

func (a *App) DeleteUploadedImage(url string) (map[string]bool, error) {
	deleted, err := a.service.DeleteUploadedImage(context.Background(), url)
	if err != nil {
		return nil, err
	}
	return map[string]bool{"ok": true, "deleted": deleted}, nil
}

func (a *App) CleanupUnusedUploadedImages() (service.UploadedImageCleanup, error) {
	return a.service.CleanupUnusedUploadedImages(context.Background())
}

func (a *App) ListTempTasks(filter service.TempTaskFilter) ([]service.TempTask, error) {
	return a.service.ListTempTasks(context.Background(), filter)
}

func (a *App) CreateTempTask(task service.TempTask) (service.TempTask, error) {
	return a.service.CreateTempTask(context.Background(), task)
}

func (a *App) GetTempTask(id int64) (service.TempTask, error) {
	return a.service.GetTempTask(context.Background(), id)
}

func (a *App) UpdateTempTask(id int64, task service.TempTask) (service.TempTask, error) {
	return a.service.UpdateTempTask(context.Background(), id, task)
}

func (a *App) DeleteTempTask(id int64) (map[string]bool, error) {
	return ok(a.service.DeleteTempTask(context.Background(), id))
}

func (a *App) ListTempTaskEvents(id int64) ([]service.TempTaskEvent, error) {
	return a.service.ListTempTaskEvents(context.Background(), id)
}

func (a *App) CreateTempTaskEvent(id int64, event service.TempTaskEvent) (service.TempTaskEvent, error) {
	return a.service.CreateTempTaskEvent(context.Background(), id, event)
}

func (a *App) UpdateTempTaskEvent(id int64, event service.TempTaskEvent) (service.TempTaskEvent, error) {
	return a.service.UpdateTempTaskEvent(context.Background(), id, event)
}

func (a *App) DeleteTempTaskEvent(id int64) (map[string]bool, error) {
	return ok(a.service.DeleteTempTaskEvent(context.Background(), id))
}

func (a *App) CreateDayEntry(entry service.DayEntry) (service.DayEntry, error) {
	return a.service.CreateDayEntry(context.Background(), entry.Date, entry.ContentMD)
}

func (a *App) DeleteDayEntry(id int64) (map[string]bool, error) {
	return ok(a.service.DeleteDayEntry(context.Background(), id))
}

func (a *App) ListWeeks() ([]service.WeeklyLog, error) {
	return a.service.ListWeeklyLogs(context.Background())
}

func (a *App) GetWeekBounds() (service.WeekBounds, error) {
	return a.service.GetWeekBounds(context.Background())
}

func (a *App) GetWeek(week string) (service.WeekView, error) {
	return a.service.GetWeekView(context.Background(), week)
}

func (a *App) UpdateWeek(week string, log service.WeeklyLog) (service.WeeklyLog, error) {
	return a.service.UpsertWeeklyLog(context.Background(), week, log)
}

func (a *App) GenerateWeekDraft(week string) (service.WeeklyLog, error) {
	return a.service.GenerateWeekDraft(context.Background(), week)
}

func (a *App) GenerateWeekSummary(week string) (service.WeeklyLog, error) {
	return a.service.GenerateWeekSummary(context.Background(), week)
}

func (a *App) Search(query string, entityType string, limit int, offset int) ([]service.SearchResult, error) {
	return a.service.Search(context.Background(), query, entityType, limit, offset)
}

func (a *App) ExportJSON() (SaveResult, error) {
	data, err := a.service.ExportJSON(context.Background())
	if err != nil {
		return SaveResult{}, err
	}
	return a.saveFile("tracelog-export.json", "JSON Files (*.json)", "*.json", data)
}

func (a *App) ExportMarkdownZip() (SaveResult, error) {
	data, err := a.service.ExportMarkdownZip(context.Background())
	if err != nil {
		return SaveResult{}, err
	}
	return a.saveFile("tracelog-markdown.zip", "ZIP Archives (*.zip)", "*.zip", data)
}

func (a *App) ExportIssueMarkdown(jiraKey string) (SaveResult, error) {
	data, filename, err := a.service.ExportIssueMarkdown(context.Background(), strings.TrimSuffix(jiraKey, ".md"))
	if err != nil {
		return SaveResult{}, err
	}
	return a.saveFile(filename, "Markdown Files (*.md)", "*.md", data)
}

func (a *App) ExportWeekMarkdown(week string) (SaveResult, error) {
	data, filename, err := a.service.ExportWeekMarkdown(context.Background(), strings.TrimSuffix(week, ".md"))
	if err != nil {
		return SaveResult{}, err
	}
	return a.saveFile(filename, "Markdown Files (*.md)", "*.md", data)
}

func (a *App) ExportTempTaskMarkdown(id int64) (SaveResult, error) {
	data, filename, err := a.service.ExportTempTaskMarkdown(context.Background(), id)
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

func badRequest(message string) error {
	return &service.AppError{Code: http.StatusBadRequest, Message: message, Err: service.ErrBadRequest}
}
