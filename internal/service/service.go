package service

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"tracelog/internal/ai"
	"tracelog/internal/appsettings"
	"tracelog/internal/jira"
)

var (
	ErrNotFound   = errors.New("not found")
	ErrBadRequest = errors.New("bad request")
)

type AppError struct {
	Code    int
	Message string
	Err     error
}

func (e *AppError) Error() string {
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Err
}

type Repository interface {
	ListIssues(context.Context, IssueFilter) ([]Issue, error)
	GetIssue(context.Context, string) (Issue, error)
	CreateIssue(context.Context, Issue) (Issue, error)
	UpdateIssue(context.Context, Issue) (Issue, error)
	DeleteIssue(context.Context, string) error
	ListIssueTodos(context.Context, int64, bool) ([]IssueTodo, error)
	ListOpenIssueTodos(context.Context, int) ([]IssueTodo, error)
	ListIssueTodosDueBetween(context.Context, string, string) ([]IssueTodo, error)
	CreateIssueTodo(context.Context, IssueTodo) (IssueTodo, error)
	UpdateIssueTodo(context.Context, IssueTodo) (IssueTodo, error)
	GetIssueTodo(context.Context, int64) (IssueTodo, error)
	DeleteIssueTodo(context.Context, int64) error
	ListIssueEvents(context.Context, int64) ([]IssueEvent, error)
	CreateIssueEvent(context.Context, IssueEvent) (IssueEvent, error)
	UpdateIssueEvent(context.Context, IssueEvent) (IssueEvent, error)
	GetIssueEvent(context.Context, int64) (IssueEvent, error)
	DeleteIssueEvent(context.Context, int64) error
	ListTempTasks(context.Context, TempTaskFilter) ([]TempTask, error)
	GetTempTask(context.Context, int64) (TempTask, error)
	CreateTempTask(context.Context, TempTask) (TempTask, error)
	UpdateTempTask(context.Context, TempTask) (TempTask, error)
	DeleteTempTask(context.Context, int64) error
	ListTempTaskEvents(context.Context, int64) ([]TempTaskEvent, error)
	CreateTempTaskEvent(context.Context, TempTaskEvent) (TempTaskEvent, error)
	UpdateTempTaskEvent(context.Context, TempTaskEvent) (TempTaskEvent, error)
	GetTempTaskEvent(context.Context, int64) (TempTaskEvent, error)
	DeleteTempTaskEvent(context.Context, int64) error
	ListIssueCommentsBetween(context.Context, string, string) ([]DayComment, error)
	ListTempTaskCommentsBetween(context.Context, string, string) ([]DayComment, error)
	ListDayEntriesBetween(context.Context, string, string) ([]DayEntry, error)
	CreateDayEntry(context.Context, DayEntry) (DayEntry, error)
	DeleteDayEntry(context.Context, int64) error
	GetWeeklyLog(context.Context, string) (WeeklyLog, error)
	UpsertWeeklyLog(context.Context, WeeklyLog) (WeeklyLog, error)
	ListWeeklyLogs(context.Context) ([]WeeklyLog, error)
	ListIssuesUpdatedBetween(context.Context, string, string) ([]Issue, error)
	ListEventsBetween(context.Context, string, string) ([]IssueEvent, error)
	ListTempTasksUpdatedBetween(context.Context, string, string) ([]TempTask, error)
	UpsertSearchIndex(context.Context, string, string, string, string, string) error
	DeleteSearchIndex(context.Context, string, string) error
	Search(context.Context, string, string, int, int) ([]SearchResult, error)
}

type Service struct {
	repo      Repository
	jira      *jira.Client
	settings  *appsettings.Store
	uploadDir string
	loc       *time.Location
}

func New(repo Repository, jiraClient *jira.Client, settingsStore *appsettings.Store, uploadDir ...string) *Service {
	dir := ""
	if len(uploadDir) > 0 {
		dir = uploadDir[0]
	}
	return &Service{repo: repo, jira: jiraClient, settings: settingsStore, uploadDir: dir, loc: time.UTC}
}

// SetLocation overrides the timezone used for day and week boundary calculations.
// Timestamps stay stored in UTC; only bucketing/range math uses this location.
func (s *Service) SetLocation(loc *time.Location) {
	if loc != nil {
		s.loc = loc
	}
}

func (s *Service) ListIssues(ctx context.Context, filter IssueFilter) ([]Issue, error) {
	return s.repo.ListIssues(ctx, filter)
}

func (s *Service) GetIssue(ctx context.Context, jiraKey string) (Issue, error) {
	issue, err := s.repo.GetIssue(ctx, strings.ToUpper(jiraKey))
	return issue, mapError(err)
}

func (s *Service) CreateIssue(ctx context.Context, issue Issue) (Issue, error) {
	now := nowString()
	issue.JiraKey = strings.ToUpper(strings.TrimSpace(issue.JiraKey))
	issue.Title = strings.TrimSpace(issue.Title)
	issue.Status = strings.TrimSpace(issue.Status)
	defaultIssueFields(&issue)
	if err := validateIssue(issue, true); err != nil {
		return Issue{}, err
	}
	issue.CreatedAt = now
	issue.UpdatedAt = now
	created, err := s.repo.CreateIssue(ctx, issue)
	if err != nil {
		return Issue{}, err
	}
	return created, s.indexIssue(ctx, created)
}

func (s *Service) UpdateIssue(ctx context.Context, jiraKey string, issue Issue) (Issue, error) {
	existing, err := s.GetIssue(ctx, jiraKey)
	if err != nil {
		return Issue{}, err
	}
	issue.ID = existing.ID
	issue.JiraKey = existing.JiraKey
	issue.CreatedAt = existing.CreatedAt
	issue.UpdatedAt = nowString()
	issue.Title = strings.TrimSpace(issue.Title)
	issue.Status = strings.TrimSpace(issue.Status)
	defaultIssueFields(&issue)
	if err := validateIssue(issue, false); err != nil {
		return Issue{}, err
	}
	updated, err := s.repo.UpdateIssue(ctx, issue)
	if err != nil {
		return Issue{}, err
	}
	return updated, s.indexIssue(ctx, updated)
}

func (s *Service) DeleteIssue(ctx context.Context, jiraKey string) error {
	return mapError(s.repo.DeleteIssue(ctx, strings.ToUpper(jiraKey)))
}

func (s *Service) ListIssueTodos(ctx context.Context, jiraKey string, includeDone bool) ([]IssueTodo, error) {
	issue, err := s.GetIssue(ctx, jiraKey)
	if err != nil {
		return nil, err
	}
	return s.repo.ListIssueTodos(ctx, issue.ID, includeDone)
}

func (s *Service) CreateIssueTodo(ctx context.Context, jiraKey string, todo IssueTodo) (IssueTodo, error) {
	issue, err := s.GetIssue(ctx, jiraKey)
	if err != nil {
		return IssueTodo{}, err
	}
	now := nowString()
	todo.IssueID = issue.ID
	todo.Content = strings.TrimSpace(todo.Content)
	todo.DueAt = strings.TrimSpace(todo.DueAt)
	todo.CreatedAt = now
	todo.UpdatedAt = now
	if err := validateIssueTodo(todo); err != nil {
		return IssueTodo{}, err
	}
	created, err := s.repo.CreateIssueTodo(ctx, todo)
	if err != nil {
		return IssueTodo{}, err
	}
	return created, s.indexIssueTodo(ctx, issue, created)
}

func (s *Service) UpdateIssueTodo(ctx context.Context, id int64, todo IssueTodo) (IssueTodo, error) {
	existing, err := s.repo.GetIssueTodo(ctx, id)
	if err != nil {
		return IssueTodo{}, mapError(err)
	}
	issue, err := s.GetIssue(ctx, existing.JiraKey)
	if err != nil {
		return IssueTodo{}, err
	}
	todo.ID = existing.ID
	todo.IssueID = existing.IssueID
	todo.CreatedAt = existing.CreatedAt
	todo.UpdatedAt = nowString()
	todo.Content = strings.TrimSpace(todo.Content)
	todo.DueAt = strings.TrimSpace(todo.DueAt)
	if err := validateIssueTodo(todo); err != nil {
		return IssueTodo{}, err
	}
	updated, err := s.repo.UpdateIssueTodo(ctx, todo)
	if err != nil {
		return IssueTodo{}, err
	}
	return updated, s.indexIssueTodo(ctx, issue, updated)
}

func (s *Service) DeleteIssueTodo(ctx context.Context, id int64) error {
	return mapError(s.repo.DeleteIssueTodo(ctx, id))
}

func (s *Service) ListIssueEvents(ctx context.Context, jiraKey string) ([]IssueEvent, error) {
	issue, err := s.GetIssue(ctx, jiraKey)
	if err != nil {
		return nil, err
	}
	return s.repo.ListIssueEvents(ctx, issue.ID)
}

func (s *Service) CreateIssueEvent(ctx context.Context, jiraKey string, event IssueEvent) (IssueEvent, error) {
	issue, err := s.GetIssue(ctx, jiraKey)
	if err != nil {
		return IssueEvent{}, err
	}
	now := nowString()
	event.IssueID = issue.ID
	event.EventType = strings.TrimSpace(event.EventType)
	event.ContentMD = strings.TrimSpace(event.ContentMD)
	if event.HappenedAt == "" {
		event.HappenedAt = now
	}
	event.CreatedAt = now
	event.UpdatedAt = now
	if err := validateEvent(event); err != nil {
		return IssueEvent{}, err
	}
	created, err := s.repo.CreateIssueEvent(ctx, event)
	if err != nil {
		return IssueEvent{}, err
	}
	return created, s.indexIssueEvent(ctx, issue, created)
}

func (s *Service) UpdateIssueEvent(ctx context.Context, id int64, event IssueEvent) (IssueEvent, error) {
	existing, err := s.repo.GetIssueEvent(ctx, id)
	if err != nil {
		return IssueEvent{}, mapError(err)
	}
	event.ID = id
	event.IssueID = existing.IssueID
	event.CreatedAt = existing.CreatedAt
	event.UpdatedAt = nowString()
	if event.HappenedAt == "" {
		event.HappenedAt = existing.HappenedAt
	}
	if err := validateEvent(event); err != nil {
		return IssueEvent{}, err
	}
	updated, err := s.repo.UpdateIssueEvent(ctx, event)
	if err != nil {
		return IssueEvent{}, err
	}
	return updated, s.repo.UpsertSearchIndex(ctx, "issue_event", fmt.Sprint(updated.ID), updated.EventType, updated.ContentMD, updated.UpdatedAt)
}

func (s *Service) DeleteIssueEvent(ctx context.Context, id int64) error {
	return mapError(s.repo.DeleteIssueEvent(ctx, id))
}

func (s *Service) ListTempTaskEvents(ctx context.Context, taskID int64) ([]TempTaskEvent, error) {
	if _, err := s.GetTempTask(ctx, taskID); err != nil {
		return nil, err
	}
	return s.repo.ListTempTaskEvents(ctx, taskID)
}

func (s *Service) CreateTempTaskEvent(ctx context.Context, taskID int64, event TempTaskEvent) (TempTaskEvent, error) {
	task, err := s.GetTempTask(ctx, taskID)
	if err != nil {
		return TempTaskEvent{}, err
	}
	now := nowString()
	event.TempTaskID = taskID
	event.EventType = strings.TrimSpace(event.EventType)
	event.ContentMD = strings.TrimSpace(event.ContentMD)
	if event.HappenedAt == "" {
		event.HappenedAt = now
	}
	event.CreatedAt = now
	event.UpdatedAt = now
	if err := validateTempTaskEvent(event); err != nil {
		return TempTaskEvent{}, err
	}
	created, err := s.repo.CreateTempTaskEvent(ctx, event)
	if err != nil {
		return TempTaskEvent{}, err
	}
	return created, s.indexTempTaskEvent(ctx, task, created)
}

func (s *Service) UpdateTempTaskEvent(ctx context.Context, id int64, event TempTaskEvent) (TempTaskEvent, error) {
	existing, err := s.repo.GetTempTaskEvent(ctx, id)
	if err != nil {
		return TempTaskEvent{}, mapError(err)
	}
	event.ID = id
	event.TempTaskID = existing.TempTaskID
	event.CreatedAt = existing.CreatedAt
	event.EventType = strings.TrimSpace(event.EventType)
	event.ContentMD = strings.TrimSpace(event.ContentMD)
	event.UpdatedAt = nowString()
	if event.HappenedAt == "" {
		event.HappenedAt = existing.HappenedAt
	}
	if err := validateTempTaskEvent(event); err != nil {
		return TempTaskEvent{}, err
	}
	updated, err := s.repo.UpdateTempTaskEvent(ctx, event)
	if err != nil {
		return TempTaskEvent{}, err
	}
	task, err := s.GetTempTask(ctx, updated.TempTaskID)
	if err != nil {
		return TempTaskEvent{}, err
	}
	return updated, s.indexTempTaskEvent(ctx, task, updated)
}

func (s *Service) DeleteTempTaskEvent(ctx context.Context, id int64) error {
	if err := s.repo.DeleteTempTaskEvent(ctx, id); err != nil {
		return mapError(err)
	}
	return s.repo.DeleteSearchIndex(ctx, "temp_task_event", fmt.Sprint(id))
}

func (s *Service) CreateDayEntry(ctx context.Context, date string, content string) (DayEntry, error) {
	date = strings.TrimSpace(date)
	content = strings.TrimSpace(content)
	if !dayDatePattern.MatchString(date) {
		return DayEntry{}, badRequest("date must use YYYY-MM-DD format")
	}
	if content == "" {
		return DayEntry{}, badRequest("content_md is required")
	}
	now := nowString()
	return s.repo.CreateDayEntry(ctx, DayEntry{Date: date, ContentMD: content, CreatedAt: now, UpdatedAt: now})
}

func (s *Service) DeleteDayEntry(ctx context.Context, id int64) error {
	return mapError(s.repo.DeleteDayEntry(ctx, id))
}

func (s *Service) ListTempTasks(ctx context.Context, filter TempTaskFilter) ([]TempTask, error) {
	return s.repo.ListTempTasks(ctx, filter)
}

func (s *Service) GetTempTask(ctx context.Context, id int64) (TempTask, error) {
	task, err := s.repo.GetTempTask(ctx, id)
	return task, mapError(err)
}

func (s *Service) CreateTempTask(ctx context.Context, task TempTask) (TempTask, error) {
	now := nowString()
	task.Title = strings.TrimSpace(task.Title)
	defaultTaskFields(&task)
	if err := validateTempTask(task); err != nil {
		return TempTask{}, err
	}
	task.CreatedAt = now
	task.UpdatedAt = now
	created, err := s.repo.CreateTempTask(ctx, task)
	if err != nil {
		return TempTask{}, err
	}
	return created, s.indexTempTask(ctx, created)
}

func (s *Service) UpdateTempTask(ctx context.Context, id int64, task TempTask) (TempTask, error) {
	existing, err := s.GetTempTask(ctx, id)
	if err != nil {
		return TempTask{}, err
	}
	task.ID = existing.ID
	task.CreatedAt = existing.CreatedAt
	task.UpdatedAt = nowString()
	task.Title = strings.TrimSpace(task.Title)
	defaultTaskFields(&task)
	if err := validateTempTask(task); err != nil {
		return TempTask{}, err
	}
	updated, err := s.repo.UpdateTempTask(ctx, task)
	if err != nil {
		return TempTask{}, err
	}
	return updated, s.indexTempTask(ctx, updated)
}

func (s *Service) DeleteTempTask(ctx context.Context, id int64) error {
	return mapError(s.repo.DeleteTempTask(ctx, id))
}

func (s *Service) ListWeeklyLogs(ctx context.Context) ([]WeeklyLog, error) {
	return s.repo.ListWeeklyLogs(ctx)
}

func (s *Service) GetWeekView(ctx context.Context, week string) (WeekView, error) {
	if week == "" {
		week = CurrentWeek(s.loc)
	}
	start, end, err := weekRange(week, s.loc)
	if err != nil {
		return WeekView{}, badRequest("week must use YYYY-Www format")
	}

	log, err := s.repo.GetWeeklyLog(ctx, week)
	if errors.Is(err, sql.ErrNoRows) {
		now := nowString()
		log = WeeklyLog{Week: week, CreatedAt: now, UpdatedAt: now}
	} else if err != nil {
		return WeekView{}, err
	}

	issues, err := s.repo.ListIssuesUpdatedBetween(ctx, start, end)
	if err != nil {
		return WeekView{}, err
	}
	events, err := s.repo.ListEventsBetween(ctx, start, end)
	if err != nil {
		return WeekView{}, err
	}
	tasks, err := s.repo.ListTempTasksUpdatedBetween(ctx, start, end)
	if err != nil {
		return WeekView{}, err
	}
	todos, err := s.repo.ListIssueTodosDueBetween(ctx, start, end)
	if err != nil {
		return WeekView{}, err
	}

	view := WeekView{Log: log, Issues: issues, Events: events, TempTasks: tasks, Todos: todos}
	for _, issue := range issues {
		label := issue.JiraKey + " " + issue.Title
		if isDoneIssueStatus(issue.Status) {
			view.Done = append(view.Done, label)
		} else {
			view.Active = append(view.Active, label)
		}
	}
	for _, task := range tasks {
		label := task.Title
		if task.Status == "done" {
			view.Done = append(view.Done, label)
		} else {
			view.Active = append(view.Active, label)
		}
	}
	for _, todo := range todos {
		label := todo.JiraKey + " TODO: " + todo.Content
		if todo.Done {
			view.Done = append(view.Done, label)
		} else {
			view.Active = append(view.Active, label)
		}
	}

	view.Days, err = s.buildDays(ctx, start, end, 7)
	if err != nil {
		return WeekView{}, err
	}
	return view, nil
}

func (s *Service) UpsertWeeklyLog(ctx context.Context, week string, log WeeklyLog) (WeeklyLog, error) {
	if _, _, err := weekRange(week, s.loc); err != nil {
		return WeeklyLog{}, badRequest("week must use YYYY-Www format")
	}
	now := nowString()
	existing, err := s.repo.GetWeeklyLog(ctx, week)
	if err == nil {
		log.ID = existing.ID
		log.CreatedAt = existing.CreatedAt
	} else {
		log.CreatedAt = now
	}
	log.Week = week
	log.UpdatedAt = now
	updated, err := s.repo.UpsertWeeklyLog(ctx, log)
	if err != nil {
		return WeeklyLog{}, err
	}
	return updated, s.repo.UpsertSearchIndex(ctx, "weekly_log", updated.Week, "Weekly Log "+updated.Week, updated.SummaryMD+"\n"+updated.NextPlanMD, updated.UpdatedAt)
}

func (s *Service) Search(ctx context.Context, query string, entityType string, limit int, offset int) ([]SearchResult, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return []SearchResult{}, nil
	}
	results, err := s.repo.Search(ctx, escapeFTSQuery(query), entityType, limit, offset)
	if err != nil {
		// Fallback for punctuation-heavy values like Jira keys if FTS parsing rejects the query.
		return s.repo.Search(ctx, `"`+strings.ReplaceAll(query, `"`, `""`)+`"`, entityType, limit, offset)
	}
	return results, err
}

func (s *Service) Dashboard(ctx context.Context) (Dashboard, error) {
	issues, err := s.ListIssues(ctx, IssueFilter{Limit: 50})
	if err != nil {
		return Dashboard{}, err
	}
	recent := issues
	if len(recent) > 8 {
		recent = recent[:8]
	}
	active := []Issue{}
	for _, issue := range issues {
		if isActiveIssueStatus(issue.Status) {
			active = append(active, issue)
			if len(active) == 8 {
				break
			}
		}
	}
	tasks, err := s.ListTempTasks(ctx, TempTaskFilter{Status: "todo", Limit: 8})
	if err != nil {
		return Dashboard{}, err
	}
	todos, err := s.repo.ListOpenIssueTodos(ctx, 8)
	if err != nil {
		return Dashboard{}, err
	}
	week, err := s.GetWeekView(ctx, CurrentWeek(s.loc))
	if err != nil {
		return Dashboard{}, err
	}
	return Dashboard{RecentIssues: recent, ActiveIssues: active, TempTasks: tasks, Todos: todos, Week: week}, nil
}

func (s *Service) TodayWorkflow(ctx context.Context) (TodayWorkflow, error) {
	now := time.Now().In(s.loc)
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, s.loc)
	end := start.AddDate(0, 0, 1)
	startStr := start.UTC().Format(time.RFC3339)
	endStr := end.UTC().Format(time.RFC3339)
	issues, err := s.repo.ListIssuesUpdatedBetween(ctx, startStr, endStr)
	if err != nil {
		return TodayWorkflow{}, err
	}
	events, err := s.repo.ListEventsBetween(ctx, startStr, endStr)
	if err != nil {
		return TodayWorkflow{}, err
	}
	tasks, err := s.repo.ListTempTasksUpdatedBetween(ctx, startStr, endStr)
	if err != nil {
		return TodayWorkflow{}, err
	}
	todos, err := s.repo.ListIssueTodosDueBetween(ctx, startStr, endStr)
	if err != nil {
		return TodayWorkflow{}, err
	}
	workflow := TodayWorkflow{
		Date:        start.Format("2006-01-02"),
		Issues:      issues,
		TempTasks:   tasks,
		Todos:       todos,
		WeeklyDraft: renderWorkflowDraft(WeekView{Issues: issues, Events: events, TempTasks: tasks, Todos: todos}),
	}
	for _, issue := range issues {
		label := issue.JiraKey + " " + issue.Title
		if isDoneIssueStatus(issue.Status) {
			workflow.Done = append(workflow.Done, label)
		} else {
			workflow.Active = append(workflow.Active, label)
		}
	}
	for _, task := range tasks {
		if task.Status == "done" {
			workflow.Done = append(workflow.Done, task.Title)
		} else {
			workflow.Active = append(workflow.Active, task.Title)
		}
	}
	for _, todo := range todos {
		label := todo.JiraKey + " TODO: " + todo.Content
		if todo.Done {
			workflow.Done = append(workflow.Done, label)
		} else {
			workflow.Active = append(workflow.Active, label)
		}
	}

	dayWorks, err := s.buildDays(ctx, startStr, endStr, 1)
	if err != nil {
		return TodayWorkflow{}, err
	}
	if len(dayWorks) > 0 {
		workflow.Day = dayWorks[0]
	}
	return workflow, nil
}

func (s *Service) GenerateWeekDraft(ctx context.Context, week string) (WeeklyLog, error) {
	view, err := s.GetWeekView(ctx, week)
	if err != nil {
		return WeeklyLog{}, err
	}
	view.Log.SummaryMD = renderWorkflowDraft(view)
	return s.UpsertWeeklyLog(ctx, view.Log.Week, view.Log)
}

func (s *Service) GenerateWeekSummary(ctx context.Context, week string) (WeeklyLog, error) {
	view, err := s.GetWeekView(ctx, week)
	if err != nil {
		return WeeklyLog{}, err
	}
	settings, err := s.settings.Load()
	if err != nil {
		return WeeklyLog{}, err
	}
	aiClient, err := aiClientFromSettings(settings)
	if err != nil {
		return WeeklyLog{}, err
	}
	summary, err := aiClient.Chat(ctx, buildWeekSummaryMessages(view, settings.Prompts.WeeklySummary))
	if err != nil {
		return WeeklyLog{}, mapAIChatError("generate week summary", err)
	}
	view.Log.SummaryMD = strings.TrimSpace(summary)
	return s.UpsertWeeklyLog(ctx, view.Log.Week, view.Log)
}

func (s *Service) GetSettings(ctx context.Context) (AppSettings, error) {
	settings, err := s.settings.Load()
	if err != nil {
		return AppSettings{}, err
	}
	return toServiceSettings(appsettings.Public(settings)), nil
}

func (s *Service) UpdateSettings(ctx context.Context, input AppSettings) (AppSettings, error) {
	settings, err := s.settings.Save(fromServiceSettings(input))
	if err != nil {
		return AppSettings{}, err
	}
	return toServiceSettings(appsettings.Public(settings)), nil
}

type UploadFile struct {
	Filename string
	Context  string
	Reader   io.Reader
}

const maxImageUploadBytes = 8 << 20

func (s *Service) SaveUploadedImage(ctx context.Context, file UploadFile) (UploadedImage, error) {
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

func (s *Service) UploadedImageDataURL(ctx context.Context, url string) (UploadedImageData, error) {
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

func (s *Service) DeleteUploadedImage(ctx context.Context, url string) (bool, error) {
	filename, err := uploadFilenameFromURL(url)
	if err != nil {
		return false, err
	}
	if s.uploadDir == "" {
		return false, badRequest("upload storage is not configured")
	}
	return s.deleteUploadedImageFile(ctx, filename)
}

func (s *Service) deleteUploadedImageFile(ctx context.Context, filename string) (bool, error) {
	normalizedURL := "/uploads/" + filename
	if referenceRepo, ok := s.repo.(interface {
		UploadedImageReferenced(context.Context, string) (bool, error)
	}); ok {
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

func (s *Service) CleanupUnusedUploadedImages(ctx context.Context) (UploadedImageCleanup, error) {
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

func (s *Service) GenerateIssueSummary(ctx context.Context, jiraKey string) (IssueSummaryResponse, error) {
	issue, err := s.GetIssue(ctx, jiraKey)
	if err != nil {
		return IssueSummaryResponse{}, err
	}
	events, err := s.ListIssueEvents(ctx, jiraKey)
	if err != nil {
		return IssueSummaryResponse{}, err
	}
	settings, err := s.settings.Load()
	if err != nil {
		return IssueSummaryResponse{}, err
	}
	aiClient, err := aiClientFromSettings(settings)
	if err != nil {
		return IssueSummaryResponse{}, err
	}
	summary, err := aiClient.Chat(ctx, buildIssueSummaryMessages(issue, events, settings.Prompts.IssueSummary))
	if err != nil {
		return IssueSummaryResponse{}, mapAIChatError("generate issue summary", err)
	}
	issue.SummaryMD = strings.TrimSpace(summary)
	updated, err := s.UpdateIssue(ctx, issue.JiraKey, issue)
	if err != nil {
		return IssueSummaryResponse{}, err
	}
	return IssueSummaryResponse{Summary: updated.SummaryMD, Issue: updated}, nil
}

func (s *Service) ImportJiraIssue(ctx context.Context, jiraKey string) (Issue, error) {
	jiraKey = strings.ToUpper(strings.TrimSpace(jiraKey))
	if !regexp.MustCompile(`^[A-Z][A-Z0-9]+-\d+$`).MatchString(jiraKey) {
		return Issue{}, badRequest("jira_key must look like GCS-45000")
	}
	jiraClient, err := s.jiraClient()
	if err != nil {
		return Issue{}, err
	}
	remoteIssue, err := jiraClient.GetIssue(ctx, jiraKey)
	if err != nil {
		switch {
		case errors.Is(err, jira.ErrNotConfigured):
			return Issue{}, &AppError{Code: 400, Message: "jira integration is not configured", Err: err}
		case errors.Is(err, jira.ErrUnauthorized):
			return Issue{}, &AppError{Code: 401, Message: "jira authentication failed", Err: err}
		case errors.Is(err, jira.ErrNotFound):
			return Issue{}, &AppError{Code: 404, Message: "jira issue not found", Err: err}
		default:
			return Issue{}, fmt.Errorf("import jira issue: %w", err)
		}
	}

	fields := remoteIssue.Fields
	browseURL := jiraClient.BrowseURL(remoteIssue.Key)
	issue := Issue{
		JiraKey:      remoteIssue.Key,
		Title:        fields.Summary,
		Status:       strings.TrimSpace(fields.Status.Name),
		Priority:     mapJiraPriority(fields.Priority.Name),
		Tags:         fields.Labels,
		BackgroundMD: renderJiraBackground(remoteIssue, browseURL),
		Links: []Link{
			{Title: "Jira", URL: browseURL, Type: "jira"},
		},
	}
	defaultIssueFields(&issue)
	return issue, nil
}

func (s *Service) jiraClient() (*jira.Client, error) {
	if s.settings == nil {
		return s.jira, nil
	}
	settings, err := s.settings.Load()
	if err != nil {
		return nil, err
	}
	client := jira.New(jira.Config{
		BaseURL:  settings.Jira.BaseURL,
		Email:    settings.Jira.Email,
		APIToken: settings.Jira.APIToken,
	})
	return client, nil
}

func aiClientFromSettings(settings appsettings.Settings) (*ai.Client, error) {
	switch strings.ToLower(settings.AI.Provider) {
	case "deepseek":
		return ai.New(ai.Config{
			BaseURL: settings.DeepSeek.BaseURL,
			APIKey:  settings.DeepSeek.APIKey,
			Model:   settings.DeepSeek.Model,
		}), nil
	case "openai", "":
		return ai.New(ai.Config{
			BaseURL: settings.OpenAI.BaseURL,
			APIKey:  settings.OpenAI.APIKey,
			Model:   settings.OpenAI.Model,
		}), nil
	default:
		return nil, badRequest("ai provider must be openai or deepseek")
	}
}

func mapAIChatError(action string, err error) error {
	switch {
	case errors.Is(err, ai.ErrNotConfigured):
		return &AppError{Code: http.StatusBadRequest, Message: "ai provider is not configured", Err: err}
	case errors.Is(err, ai.ErrUnauthorized):
		return &AppError{Code: http.StatusUnauthorized, Message: "ai authentication failed", Err: err}
	default:
		return fmt.Errorf("%s: %w", action, err)
	}
}

func buildIssueSummaryMessages(issue Issue, events []IssueEvent, prompt string) []ai.Message {
	var b strings.Builder
	fmt.Fprintf(&b, "Issue: %s\nTitle: %s\nStatus: %s\nPriority: %s\nTags: %s\n\n", issue.JiraKey, issue.Title, issue.Status, issue.Priority, strings.Join(issue.Tags, ", "))
	if len(issue.Links) > 0 {
		b.WriteString("Links:\n")
		for _, link := range issue.Links {
			fmt.Fprintf(&b, "- %s: %s\n", link.Title, link.URL)
		}
		b.WriteString("\n")
	}
	if issue.BackgroundMD != "" {
		fmt.Fprintf(&b, "Jira information and description:\n%s\n\n", truncate(issue.BackgroundMD, 6000))
	}
	if len(events) > 0 {
		b.WriteString("Comments:\n")
		for _, event := range events {
			fmt.Fprintf(&b, "- %s: %s\n", event.HappenedAt, truncate(event.ContentMD, 1200))
		}
	}
	return []ai.Message{
		{
			Role:    "system",
			Content: strings.TrimSpace(prompt),
		},
		{
			Role:    "user",
			Content: b.String(),
		},
	}
}

func buildWeekSummaryMessages(view WeekView, prompt string) []ai.Message {
	var b strings.Builder
	fmt.Fprintf(&b, "Week: %s\n\n", view.Log.Week)
	hasRecords := false
	if len(view.Issues) > 0 {
		hasRecords = true
		b.WriteString("Issues:\n")
		for _, issue := range view.Issues {
			fmt.Fprintf(&b, "- %s %s | status: %s | priority: %s | summary: %s\n", issue.JiraKey, issue.Title, issue.Status, issue.Priority, truncate(issue.SummaryMD, 800))
		}
		b.WriteString("\n")
	}
	if len(view.Events) > 0 {
		hasRecords = true
		b.WriteString("Issue events:\n")
		for _, event := range view.Events {
			fmt.Fprintf(&b, "- %s | %s | %s\n", event.HappenedAt, event.EventType, truncate(event.ContentMD, 1000))
		}
		b.WriteString("\n")
	}
	if len(view.TempTasks) > 0 {
		hasRecords = true
		b.WriteString("Temp tasks:\n")
		for _, task := range view.TempTasks {
			fmt.Fprintf(&b, "- %s | status: %s | priority: %s | content: %s\n", task.Title, task.Status, task.Priority, truncate(task.ContentMD, 800))
		}
		b.WriteString("\n")
	}
	if len(view.Todos) > 0 {
		hasRecords = true
		b.WriteString("TODOs:\n")
		for _, todo := range view.Todos {
			status := "open"
			if todo.Done {
				status = "done"
			}
			fmt.Fprintf(&b, "- %s | %s | due: %s | %s\n", todo.JiraKey, status, todo.DueAt, todo.Content)
		}
		b.WriteString("\n")
	}
	if len(view.Done) > 0 {
		hasRecords = true
		b.WriteString("Done items:\n")
		for _, item := range view.Done {
			fmt.Fprintf(&b, "- %s\n", item)
		}
		b.WriteString("\n")
	}
	if len(view.Active) > 0 {
		hasRecords = true
		b.WriteString("Active items:\n")
		for _, item := range view.Active {
			fmt.Fprintf(&b, "- %s\n", item)
		}
		b.WriteString("\n")
	}
	if view.Log.SummaryMD != "" {
		fmt.Fprintf(&b, "Existing weekly summary draft:\n%s\n\n", truncate(view.Log.SummaryMD, 2000))
	}
	if view.Log.NextPlanMD != "" {
		fmt.Fprintf(&b, "Existing next plan:\n%s\n\n", truncate(view.Log.NextPlanMD, 1200))
	}
	if !hasRecords {
		b.WriteString("No records for this week.\n")
	}
	return []ai.Message{
		{
			Role:    "system",
			Content: strings.TrimSpace(prompt),
		},
		{
			Role:    "user",
			Content: b.String(),
		},
	}
}

func truncate(value string, max int) string {
	value = strings.TrimSpace(value)
	if len(value) <= max {
		return value
	}
	return value[:max] + "\n...(truncated)"
}

func toServiceSettings(settings appsettings.Settings) AppSettings {
	return AppSettings{
		Jira: JiraSettings{
			BaseURL:     settings.Jira.BaseURL,
			Email:       settings.Jira.Email,
			APIToken:    settings.Jira.APIToken,
			HasAPIToken: settings.Jira.HasAPIToken,
		},
		AI: AISettings{Provider: settings.AI.Provider},
		OpenAI: ProviderSettings{
			BaseURL:   settings.OpenAI.BaseURL,
			Model:     settings.OpenAI.Model,
			APIKey:    settings.OpenAI.APIKey,
			HasAPIKey: settings.OpenAI.HasAPIKey,
		},
		DeepSeek: ProviderSettings{
			BaseURL:   settings.DeepSeek.BaseURL,
			Model:     settings.DeepSeek.Model,
			APIKey:    settings.DeepSeek.APIKey,
			HasAPIKey: settings.DeepSeek.HasAPIKey,
		},
		Prompts: PromptSettings{
			IssueSummary:  settings.Prompts.IssueSummary,
			WeeklySummary: settings.Prompts.WeeklySummary,
		},
	}
}

func fromServiceSettings(settings AppSettings) appsettings.Settings {
	return appsettings.Settings{
		Jira: appsettings.JiraSettings{
			BaseURL:  settings.Jira.BaseURL,
			Email:    settings.Jira.Email,
			APIToken: settings.Jira.APIToken,
		},
		AI: appsettings.AISettings{Provider: settings.AI.Provider},
		OpenAI: appsettings.ProviderSettings{
			BaseURL: settings.OpenAI.BaseURL,
			Model:   settings.OpenAI.Model,
			APIKey:  settings.OpenAI.APIKey,
		},
		DeepSeek: appsettings.ProviderSettings{
			BaseURL: settings.DeepSeek.BaseURL,
			Model:   settings.DeepSeek.Model,
			APIKey:  settings.DeepSeek.APIKey,
		},
		Prompts: appsettings.PromptSettings{
			IssueSummary:  settings.Prompts.IssueSummary,
			WeeklySummary: settings.Prompts.WeeklySummary,
		},
	}
}

func (s *Service) ExportJSON(ctx context.Context) ([]byte, error) {
	issues, err := s.ListIssues(ctx, IssueFilter{All: true})
	if err != nil {
		return nil, err
	}
	eventsByIssue := map[string][]IssueEvent{}
	todosByIssue := map[string][]IssueTodo{}
	for _, issue := range issues {
		events, err := s.ListIssueEvents(ctx, issue.JiraKey)
		if err != nil {
			return nil, err
		}
		todos, err := s.ListIssueTodos(ctx, issue.JiraKey, true)
		if err != nil {
			return nil, err
		}
		eventsByIssue[issue.JiraKey] = events
		todosByIssue[issue.JiraKey] = todos
	}
	tasks, err := s.ListTempTasks(ctx, TempTaskFilter{All: true})
	if err != nil {
		return nil, err
	}
	weeks, err := s.ListWeeklyLogs(ctx)
	if err != nil {
		return nil, err
	}
	payload := map[string]any{
		"schema_version": 1,
		"exported_at":    nowString(),
		"issues":         issues,
		"issue_events":   eventsByIssue,
		"issue_todos":    todosByIssue,
		"temp_tasks":     tasks,
		"weekly_logs":    weeks,
	}
	return json.MarshalIndent(payload, "", "  ")
}

func (s *Service) ExportIssueMarkdown(ctx context.Context, jiraKey string) ([]byte, string, error) {
	issue, err := s.GetIssue(ctx, jiraKey)
	if err != nil {
		return nil, "", err
	}
	events, err := s.ListIssueEvents(ctx, jiraKey)
	if err != nil {
		return nil, "", err
	}
	todos, err := s.ListIssueTodos(ctx, jiraKey, true)
	if err != nil {
		return nil, "", err
	}
	return []byte(renderIssueMarkdown(issue, events, todos)), issue.JiraKey + ".md", nil
}

func (s *Service) ExportTempTaskMarkdown(ctx context.Context, id int64) ([]byte, string, error) {
	task, err := s.GetTempTask(ctx, id)
	if err != nil {
		return nil, "", err
	}
	return []byte(renderTempTaskMarkdown(task)), fmt.Sprintf("temp-task-%d.md", task.ID), nil
}

func (s *Service) ExportWeekMarkdown(ctx context.Context, week string) ([]byte, string, error) {
	view, err := s.GetWeekView(ctx, week)
	if err != nil {
		return nil, "", err
	}
	return []byte(renderWeekMarkdown(view)), week + ".md", nil
}

func (s *Service) ExportMarkdownZip(ctx context.Context) ([]byte, error) {
	buffer := bytes.NewBuffer(nil)
	writer := zip.NewWriter(buffer)

	jsonData, err := s.ExportJSON(ctx)
	if err != nil {
		return nil, err
	}
	if err := writeZipFile(writer, "data.json", bytes.NewReader(jsonData)); err != nil {
		return nil, err
	}

	issues, err := s.ListIssues(ctx, IssueFilter{All: true})
	if err != nil {
		return nil, err
	}
	for _, issue := range issues {
		content, filename, err := s.ExportIssueMarkdown(ctx, issue.JiraKey)
		if err != nil {
			return nil, err
		}
		if err := writeZipFile(writer, "issues/"+filename, bytes.NewReader(content)); err != nil {
			return nil, err
		}
	}

	tasks, err := s.ListTempTasks(ctx, TempTaskFilter{All: true})
	if err != nil {
		return nil, err
	}
	for _, task := range tasks {
		if err := writeZipFile(writer, fmt.Sprintf("temp-tasks/%d.md", task.ID), strings.NewReader(renderTempTaskMarkdown(task))); err != nil {
			return nil, err
		}
	}

	weeks, err := s.ListWeeklyLogs(ctx)
	if err != nil {
		return nil, err
	}
	for _, log := range weeks {
		content, _, err := s.ExportWeekMarkdown(ctx, log.Week)
		if err != nil {
			return nil, err
		}
		if err := writeZipFile(writer, "weekly-logs/"+log.Week+".md", bytes.NewReader(content)); err != nil {
			return nil, err
		}
	}

	index := fmt.Sprintf("# TraceLog Export\n\nExported at: %s\n\n- Issues: %d\n- Temp tasks: %d\n- Weekly logs: %d\n", nowString(), len(issues), len(tasks), len(weeks))
	if err := writeZipFile(writer, "index.md", strings.NewReader(index)); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func (s *Service) indexIssue(ctx context.Context, issue Issue) error {
	body := strings.Join([]string{issue.JiraKey, issue.SummaryMD, issue.BackgroundMD, issue.AnalysisMD, issue.SolutionMD, issue.ActionsMD, issue.ResultMD, issue.TodoMD, issue.StartedAt, issue.CompletedAt}, "\n")
	return s.repo.UpsertSearchIndex(ctx, "issue", issue.JiraKey, issue.JiraKey+" "+issue.Title, body, issue.UpdatedAt)
}

func (s *Service) indexIssueEvent(ctx context.Context, issue Issue, event IssueEvent) error {
	title := issue.JiraKey + " " + event.EventType
	return s.repo.UpsertSearchIndex(ctx, "issue_event", fmt.Sprint(event.ID), title, event.ContentMD, event.UpdatedAt)
}

func (s *Service) indexTempTask(ctx context.Context, task TempTask) error {
	body := strings.Join([]string{task.Source, task.ContentMD, task.StartedAt, task.CompletedAt, task.ConvertedJiraKey}, "\n")
	return s.repo.UpsertSearchIndex(ctx, "temp_task", fmt.Sprint(task.ID), task.Title, body, task.UpdatedAt)
}

func (s *Service) indexTempTaskEvent(ctx context.Context, task TempTask, event TempTaskEvent) error {
	title := task.Title + " " + event.EventType
	return s.repo.UpsertSearchIndex(ctx, "temp_task_event", fmt.Sprint(event.ID), title, event.ContentMD, event.UpdatedAt)
}

func (s *Service) indexIssueTodo(ctx context.Context, issue Issue, todo IssueTodo) error {
	title := issue.JiraKey + " TODO"
	return s.repo.UpsertSearchIndex(ctx, "issue_todo", issue.JiraKey+":"+fmt.Sprint(todo.ID), title, todo.Content+"\n"+todo.DueAt, todo.UpdatedAt)
}

func validateIssue(issue Issue, requireKey bool) error {
	if requireKey && !regexp.MustCompile(`^[A-Z][A-Z0-9]+-\d+$`).MatchString(issue.JiraKey) {
		return badRequest("jira_key must look like GCS-45000")
	}
	if issue.Title == "" {
		return badRequest("title is required")
	}
	if !validIssueStatus(issue.Status) {
		return badRequest("invalid issue status")
	}
	if !validPriority(issue.Priority) {
		return badRequest("invalid priority")
	}
	return nil
}

func validateEvent(event IssueEvent) error {
	return validateEventFields(event.EventType, event.ContentMD)
}

func validateTempTaskEvent(event TempTaskEvent) error {
	return validateEventFields(event.EventType, event.ContentMD)
}

func validateEventFields(eventType string, contentMD string) error {
	if contentMD == "" {
		return badRequest("content_md is required")
	}
	switch eventType {
	case "note", "analysis", "action", "decision", "blocker", "result":
		return nil
	default:
		return badRequest("invalid event_type")
	}
}

func validateIssueTodo(todo IssueTodo) error {
	if strings.TrimSpace(todo.Content) == "" {
		return badRequest("content is required")
	}
	if todo.DueAt != "" {
		if _, err := time.Parse(time.RFC3339, todo.DueAt); err != nil {
			return badRequest("due_at must be RFC3339")
		}
	}
	return nil
}

func validateTempTask(task TempTask) error {
	if task.Title == "" {
		return badRequest("title is required")
	}
	switch task.Status {
	case "todo", "processing", "done", "suspended":
	default:
		return badRequest("invalid temp task status")
	}
	if !validPriority(task.Priority) {
		return badRequest("invalid priority")
	}
	return nil
}

func defaultIssueFields(issue *Issue) {
	if issue.Status == "" {
		issue.Status = "analysis"
	}
	if issue.Priority == "" {
		issue.Priority = "medium"
	}
	if issue.Tags == nil {
		issue.Tags = []string{}
	}
	if issue.Links == nil {
		issue.Links = []Link{}
	}
}

func defaultTaskFields(task *TempTask) {
	if task.Status == "" {
		task.Status = "todo"
	}
	if task.Priority == "" {
		task.Priority = "medium"
	}
	if task.Tags == nil {
		task.Tags = []string{}
	}
}

func validIssueStatus(status string) bool {
	return strings.TrimSpace(status) != ""
}

func validPriority(priority string) bool {
	switch priority {
	case "low", "medium", "high", "urgent":
		return true
	default:
		return false
	}
}

func mapError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return &AppError{Code: 404, Message: "not found", Err: ErrNotFound}
	}
	return err
}

func isActiveIssueStatus(status string) bool {
	normalized := strings.ToLower(strings.TrimSpace(status))
	return normalized == "processing" ||
		strings.Contains(normalized, "progress") ||
		strings.Contains(normalized, "indeterminate") ||
		strings.Contains(status, "处理中")
}

func isDoneIssueStatus(status string) bool {
	normalized := strings.ToLower(strings.TrimSpace(status))
	return normalized == "done" ||
		normalized == "closed" ||
		strings.Contains(normalized, "resolved") ||
		strings.Contains(normalized, "完成") ||
		strings.Contains(normalized, "关闭")
}

func badRequest(message string) error {
	return &AppError{Code: 400, Message: message, Err: ErrBadRequest}
}

func imageExtension(contentType string) (string, bool) {
	switch contentType {
	case "image/png":
		return ".png", true
	case "image/jpeg":
		return ".jpg", true
	case "image/gif":
		return ".gif", true
	case "image/webp":
		return ".webp", true
	default:
		return "", false
	}
}

func uploadFilenameFromURL(url string) (string, error) {
	name := strings.TrimPrefix(strings.TrimSpace(url), "/uploads/")
	if name == "" || name == url || strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return "", badRequest("invalid upload url")
	}
	return name, nil
}

func isUploadedImageFilename(filename string) bool {
	if filename != filepath.Base(filename) {
		return false
	}
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp":
		return true
	default:
		return false
	}
}

func uploadFilename(context string, originalFilename string, uploadedAt time.Time, ext string) string {
	prefix := uploadContextSlug(context)
	if prefix == "" {
		prefix = "upload"
	}
	base := uploadBaseName(originalFilename)
	original := uploadContextSlug(strings.TrimSuffix(base, filepath.Ext(base)))
	if original == "" {
		return fmt.Sprintf("%s-%s-%s%s", prefix, uploadedAt.Format("20060102T150405"), randomHex(4), ext)
	}
	return fmt.Sprintf("%s-%s-%s-%s%s", prefix, uploadedAt.Format("20060102T150405"), original, randomHex(4), ext)
}

func uploadBaseName(filename string) string {
	filename = strings.TrimSpace(strings.ReplaceAll(filename, "\\", "/"))
	return filepath.Base(filename)
}

func uploadContextSlug(context string) string {
	context = strings.ToLower(strings.TrimSpace(context))
	var b strings.Builder
	lastDash := false
	for _, r := range context {
		isToken := r >= 'a' && r <= 'z' || r >= '0' && r <= '9'
		if isToken {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if b.Len() > 0 && !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

func mapUploadReadError(err error) error {
	if errors.Is(err, os.ErrNotExist) {
		return &AppError{Code: http.StatusNotFound, Message: "uploaded image not found", Err: ErrNotFound}
	}
	return fmt.Errorf("read uploaded image: %w", err)
}

func randomHex(bytesCount int) string {
	buffer := make([]byte, bytesCount)
	if _, err := rand.Read(buffer); err != nil {
		return fmt.Sprint(time.Now().UnixNano())
	}
	return hex.EncodeToString(buffer)
}

func nowString() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func CurrentWeek(loc *time.Location) string {
	year, week := time.Now().In(loc).ISOWeek()
	return fmt.Sprintf("%04d-W%02d", year, week)
}

func weekRange(week string, loc *time.Location) (string, string, error) {
	match := regexp.MustCompile(`^(\d{4})-W(\d{2})$`).FindStringSubmatch(week)
	if match == nil {
		return "", "", fmt.Errorf("invalid week")
	}
	var year, weekNumber int
	if _, err := fmt.Sscanf(week, "%04d-W%02d", &year, &weekNumber); err != nil {
		return "", "", err
	}
	jan4 := time.Date(year, 1, 4, 0, 0, 0, 0, loc)
	weekday := int(jan4.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	week1Monday := jan4.AddDate(0, 0, 1-weekday)
	start := week1Monday.AddDate(0, 0, (weekNumber-1)*7)
	end := start.AddDate(0, 0, 7)
	return start.UTC().Format(time.RFC3339), end.UTC().Format(time.RFC3339), nil
}

var weekdayLabels = []string{"周日", "周一", "周二", "周三", "周四", "周五", "周六"}

var dayDatePattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

func (s *Service) buildDays(ctx context.Context, startStr string, endStr string, dayCount int) ([]DayWork, error) {
	first, err := time.Parse(time.RFC3339, startStr)
	if err != nil {
		return nil, err
	}
	firstLoc := first.In(s.loc)
	days := make([]time.Time, dayCount)
	for i := range days {
		days[i] = firstLoc.AddDate(0, 0, i)
	}
	issueComments, err := s.repo.ListIssueCommentsBetween(ctx, startStr, endStr)
	if err != nil {
		return nil, err
	}
	tempComments, err := s.repo.ListTempTaskCommentsBetween(ctx, startStr, endStr)
	if err != nil {
		return nil, err
	}
	dayEntries, err := s.repo.ListDayEntriesBetween(ctx, days[0].Format("2006-01-02"), days[len(days)-1].Format("2006-01-02"))
	if err != nil {
		return nil, err
	}
	return bucketByDay(s.loc, days, append(issueComments, tempComments...), dayEntries), nil
}

func bucketByDay(loc *time.Location, days []time.Time, comments []DayComment, entries []DayEntry) []DayWork {
	result := make([]DayWork, len(days))
	index := map[string]int{}
	for i, d := range days {
		dateStr := d.Format("2006-01-02")
		result[i] = DayWork{Date: dateStr, Weekday: weekdayLabels[int(d.Weekday())], Comments: []DayComment{}, Entries: []DayEntry{}}
		index[dateStr] = i
	}
	for _, c := range comments {
		if i, ok := index[commentDayString(c.HappenedAt, loc)]; ok {
			c.URL = commentURL(c)
			result[i].Comments = append(result[i].Comments, c)
		}
	}
	for _, e := range entries {
		if i, ok := index[e.Date]; ok {
			result[i].Entries = append(result[i].Entries, e)
		}
	}
	return result
}

func commentDayString(happenedAt string, loc *time.Location) string {
	t, err := time.Parse(time.RFC3339, happenedAt)
	if err != nil {
		return ""
	}
	return t.In(loc).Format("2006-01-02")
}

func commentURL(c DayComment) string {
	if c.Source == "temp_task" {
		return fmt.Sprintf("/temp-tasks/%d", c.RefID)
	}
	return "/issues/" + c.RefKey
}

func escapeFTSQuery(query string) string {
	parts := strings.Fields(query)
	for index, part := range parts {
		parts[index] = `"` + strings.ReplaceAll(part, `"`, `""`) + `"`
	}
	return strings.Join(parts, " ")
}

func writeZipFile(writer *zip.Writer, name string, reader io.Reader) error {
	file, err := writer.Create(name)
	if err != nil {
		return err
	}
	_, err = io.Copy(file, reader)
	return err
}

func extractJiraKey(value string) string {
	match := regexp.MustCompile(`[A-Z][A-Z0-9]+-\d+`).FindString(strings.ToUpper(value))
	return match
}

func firstLine(value string) string {
	for _, line := range strings.Split(strings.TrimSpace(value), "\n") {
		line = strings.TrimSpace(strings.Trim(line, "#-*[] "))
		if line != "" {
			if len(line) > 80 {
				return line[:80]
			}
			return line
		}
	}
	return "Untitled"
}

func renderIssueMarkdown(issue Issue, events []IssueEvent, todos []IssueTodo) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s %s\n\n", issue.JiraKey, issue.Title)
	fmt.Fprintf(&b, "- Status: %s\n- Priority: %s\n- Tags: %s\n- Started: %s\n- Completed: %s\n- Created: %s\n- Updated: %s\n\n", issue.Status, issue.Priority, strings.Join(issue.Tags, ", "), issue.StartedAt, issue.CompletedAt, issue.CreatedAt, issue.UpdatedAt)
	writeSection(&b, "Summary", issue.SummaryMD)
	writeSection(&b, "Background", issue.BackgroundMD)
	writeSection(&b, "Analysis", issue.AnalysisMD)
	writeSection(&b, "Solution", issue.SolutionMD)
	writeSection(&b, "Actions", issue.ActionsMD)
	writeSection(&b, "Result", issue.ResultMD)
	writeSection(&b, "TODO", issue.TodoMD)
	if len(todos) > 0 {
		b.WriteString("## Structured TODOs\n\n")
		b.WriteString(renderIssueTodoMarkdown(todos))
		b.WriteString("\n")
	}
	if len(issue.Links) > 0 {
		b.WriteString("## Links\n\n")
		for _, link := range issue.Links {
			fmt.Fprintf(&b, "- [%s](%s) (%s)\n", link.Title, link.URL, link.Type)
		}
		b.WriteString("\n")
	}
	b.WriteString("## Timeline\n\n")
	for _, event := range events {
		fmt.Fprintf(&b, "### %s %s\n\n%s\n\n", event.HappenedAt, event.EventType, event.ContentMD)
	}
	return b.String()
}

func renderIssueTodoMarkdown(todos []IssueTodo) string {
	var b strings.Builder
	for _, todo := range todos {
		marker := " "
		if todo.Done {
			marker = "x"
		}
		due := ""
		if todo.DueAt != "" {
			due = " @due " + todo.DueAt
		}
		fmt.Fprintf(&b, "- [%s] %s%s\n", marker, todo.Content, due)
	}
	return b.String()
}

func renderTempTaskMarkdown(task TempTask) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", task.Title)
	fmt.Fprintf(&b, "- Source: %s\n- Status: %s\n- Priority: %s\n- Tags: %s\n- Started: %s\n- Completed: %s\n- Converted to Jira: %t\n- Jira Key: %s\n\n", task.Source, task.Status, task.Priority, strings.Join(task.Tags, ", "), task.StartedAt, task.CompletedAt, task.ConvertedToJira, task.ConvertedJiraKey)
	writeSection(&b, "Content", task.ContentMD)
	return b.String()
}

func renderWeekMarkdown(view WeekView) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Weekly Log %s\n\n", view.Log.Week)
	b.WriteString("## Jira Issues\n\n")
	for _, issue := range view.Issues {
		fmt.Fprintf(&b, "- %s %s (%s)\n", issue.JiraKey, issue.Title, issue.Status)
	}
	b.WriteString("\n## Temp Tasks\n\n")
	for _, task := range view.TempTasks {
		fmt.Fprintf(&b, "- %s (%s)\n", task.Title, task.Status)
	}
	b.WriteString("\n## TODOs\n\n")
	b.WriteString(renderIssueTodoMarkdown(view.Todos))
	b.WriteString("\n## Done\n\n")
	for _, item := range view.Done {
		fmt.Fprintf(&b, "- %s\n", item)
	}
	b.WriteString("\n## Active\n\n")
	for _, item := range view.Active {
		fmt.Fprintf(&b, "- %s\n", item)
	}
	writeSection(&b, "Summary", view.Log.SummaryMD)
	writeSection(&b, "Next Plan", view.Log.NextPlanMD)
	return b.String()
}

func renderWorkflowDraft(view WeekView) string {
	var b strings.Builder
	b.WriteString("## 本周处理\n\n")
	if len(view.Issues) == 0 && len(view.TempTasks) == 0 {
		b.WriteString("- 暂无记录\n")
	}
	for _, issue := range view.Issues {
		summary := strings.TrimSpace(issue.SummaryMD)
		if summary == "" {
			summary = issue.Title
		}
		fmt.Fprintf(&b, "- %s：%s（%s）\n", issue.JiraKey, summary, issue.Status)
	}
	for _, task := range view.TempTasks {
		fmt.Fprintf(&b, "- 临时需求：%s（%s）\n", task.Title, task.Status)
	}
	if len(view.Events) > 0 {
		b.WriteString("\n## 关键过程\n\n")
		for _, event := range view.Events {
			fmt.Fprintf(&b, "- %s：%s\n", event.HappenedAt, firstLine(event.ContentMD))
		}
	}
	if len(view.Todos) > 0 {
		b.WriteString("\n## 后续 TODO\n\n")
		b.WriteString(renderIssueTodoMarkdown(view.Todos))
	}
	b.WriteString("\n## 下周计划\n\n- \n")
	return b.String()
}

func writeSection(b *strings.Builder, title string, body string) {
	fmt.Fprintf(b, "## %s\n\n%s\n\n", title, body)
}

func renderJiraBackground(issue jira.Issue, browseURL string) string {
	fields := issue.Fields
	var b strings.Builder
	fmt.Fprintf(&b, "Imported from Jira: [%s](%s)\n\n", issue.Key, browseURL)
	if fields.IssueType.Name != "" {
		fmt.Fprintf(&b, "- Type: %s\n", fields.IssueType.Name)
	}
	if fields.Status.Name != "" {
		fmt.Fprintf(&b, "- Jira status: %s\n", fields.Status.Name)
	}
	if fields.Priority.Name != "" {
		fmt.Fprintf(&b, "- Jira priority: %s\n", fields.Priority.Name)
	}
	if fields.Reporter.DisplayName != "" {
		fmt.Fprintf(&b, "- Reporter: %s\n", fields.Reporter.DisplayName)
	}
	if fields.Assignee.DisplayName != "" {
		fmt.Fprintf(&b, "- Assignee: %s\n", fields.Assignee.DisplayName)
	}
	if fields.Created != "" {
		fmt.Fprintf(&b, "- Jira created: %s\n", fields.Created)
	}
	if fields.Updated != "" {
		fmt.Fprintf(&b, "- Jira updated: %s\n", fields.Updated)
	}
	writeNames(&b, "Components", fields.Components)
	if !writeNames(&b, "发布请求", fields.ReleaseRequested) {
		writeNames(&b, "发布请求", fields.FixVersions)
	}

	description := jira.ADFToMarkdown(fields.Description)
	if description != "" {
		b.WriteString("\n## Jira Description\n\n")
		b.WriteString(description)
		b.WriteString("\n")
	}
	return strings.TrimSpace(b.String())
}

func writeNames(b *strings.Builder, label string, values []jira.Named) bool {
	if len(values) == 0 {
		return false
	}
	names := make([]string, 0, len(values))
	for _, value := range values {
		if value.Name != "" {
			names = append(names, value.Name)
		}
	}
	if len(names) > 0 {
		fmt.Fprintf(b, "- %s: %s\n", label, strings.Join(names, ", "))
		return true
	}
	return false
}

func mapJiraStatus(status jira.Named) string {
	if status.StatusCategory != nil {
		switch strings.ToLower(status.StatusCategory.Key) {
		case "done":
			return "done"
		case "indeterminate":
			return "processing"
		}
	}
	name := strings.ToLower(status.Name)
	switch {
	case strings.Contains(name, "closed"):
		return "closed"
	case strings.Contains(name, "done"), strings.Contains(name, "resolved"):
		return "done"
	case strings.Contains(name, "progress"), strings.Contains(name, "处理中"):
		return "processing"
	default:
		return "analysis"
	}
}

func mapJiraPriority(priority string) string {
	name := strings.ToLower(priority)
	switch {
	case strings.Contains(name, "highest"), strings.Contains(name, "critical"), strings.Contains(name, "blocker"), strings.Contains(name, "urgent"):
		return "urgent"
	case strings.Contains(name, "high"), strings.Contains(name, "major"):
		return "high"
	case strings.Contains(name, "low"), strings.Contains(name, "minor"), strings.Contains(name, "trivial"):
		return "low"
	default:
		return "medium"
	}
}
