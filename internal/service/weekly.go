package service

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
)

func (s *WeeklyService) ListWeeklyLogs(ctx context.Context) ([]WeeklyLog, error) {
	return s.repo.ListWeeklyLogs(ctx)
}

func (s *WeeklyService) GetWeekBounds(ctx context.Context) (WeekBounds, error) {
	current := CurrentWeek(s.loc)
	firstWeek := current
	firstDate, err := s.repo.FirstActivityDate(ctx)
	if err != nil {
		return WeekBounds{}, err
	}
	if firstDate != "" {
		parsed, err := time.ParseInLocation("2006-01-02", firstDate, s.loc)
		if err != nil {
			return WeekBounds{}, err
		}
		firstWeek = weekFromTime(parsed.In(s.loc))
	}
	logs, err := s.repo.ListWeeklyLogs(ctx)
	if err != nil {
		return WeekBounds{}, err
	}
	for _, log := range logs {
		if _, _, err := weekRange(log.Week, s.loc); err == nil && log.Week < firstWeek {
			firstWeek = log.Week
		}
	}
	if firstWeek > current {
		firstWeek = current
	}
	return WeekBounds{FirstWeek: firstWeek, CurrentWeek: current}, nil
}

func (s *WeeklyService) GetWeekView(ctx context.Context, week string) (WeekView, error) {
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

func (s *WeeklyService) UpsertWeeklyLog(ctx context.Context, week string, log WeeklyLog) (WeeklyLog, error) {
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
	var updated WeeklyLog
	if err := s.withRepository(ctx, func(repo Repository) error {
		var err error
		updated, err = repo.UpsertWeeklyLog(ctx, log)
		if err != nil {
			return err
		}
		return repo.UpsertSearchIndex(ctx, "weekly_log", updated.Week, "Weekly Log "+updated.Week, updated.SummaryMD+"\n"+updated.NextPlanMD, updated.UpdatedAt)
	}); err != nil {
		return WeeklyLog{}, err
	}
	return updated, nil
}

func (s *WeeklyService) GenerateWeekDraft(ctx context.Context, week string) (WeeklyLog, error) {
	view, err := s.GetWeekView(ctx, week)
	if err != nil {
		return WeeklyLog{}, err
	}
	view.Log.SummaryMD = renderWorkflowDraft(view)
	return s.UpsertWeeklyLog(ctx, view.Log.Week, view.Log)
}

func (s *WeeklyService) GenerateWeekSummary(ctx context.Context, week string) (WeeklyLog, error) {
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
