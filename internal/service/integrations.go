package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"tracelog/internal/ai"
	"tracelog/internal/appsettings"
	"tracelog/internal/jira"
	"tracelog/internal/tempo"
)

func (s *IntegrationService) GenerateIssueSummary(ctx context.Context, jiraKey string) (IssueSummaryResponse, error) {
	issue, err := s.issues.GetIssue(ctx, jiraKey)
	if err != nil {
		return IssueSummaryResponse{}, err
	}
	events, err := s.issues.ListIssueEvents(ctx, jiraKey)
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
	updated, err := s.issues.UpdateIssue(ctx, issue.JiraKey, issue)
	if err != nil {
		return IssueSummaryResponse{}, err
	}
	return IssueSummaryResponse{Summary: updated.SummaryMD, Issue: updated}, nil
}

func (s *IntegrationService) ImportJiraIssue(ctx context.Context, jiraKey string) (Issue, error) {
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

func (s *serviceCore) jiraClient() (*jira.Client, error) {
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

func (s *TimeService) tempoSettingsWithAuthor(ctx context.Context) (appsettings.Settings, string, error) {
	settings, err := s.settings.Load()
	if err != nil {
		return appsettings.Settings{}, "", err
	}
	if settings.Tempo.AuthorAccountID != "" {
		return settings, settings.Tempo.AuthorAccountID, nil
	}
	accountID, err := s.fetchTempoAuthorAccountID(ctx, settings)
	if err != nil {
		return appsettings.Settings{}, "", err
	}
	settings.Tempo.AuthorAccountID = accountID
	saved, err := s.settings.Save(settings)
	if err != nil {
		return appsettings.Settings{}, "", err
	}
	return saved, saved.Tempo.AuthorAccountID, nil
}

func (s *serviceCore) fetchTempoAuthorAccountID(ctx context.Context, settings appsettings.Settings) (string, error) {
	client := jira.New(jira.Config{
		BaseURL:  settings.Jira.BaseURL,
		Email:    settings.Jira.Email,
		APIToken: settings.Jira.APIToken,
	})
	user, err := client.Myself(ctx)
	if err != nil {
		return "", mapJiraLookupError("get jira current user", err)
	}
	return user.AccountID, nil
}

func (s *TimeService) cachedTimeWorklogs(ctx context.Context, accountID string, startDate string, endDate string) ([]TimeWorklog, bool, error) {
	_, err := s.repo.GetTimeCacheRange(ctx, accountID, startDate, endDate)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, ErrNotFound) {
			return nil, false, nil
		}
		return nil, false, err
	}
	worklogs, err := s.repo.ListCachedTimeWorklogs(ctx, accountID, startDate, endDate)
	if err != nil {
		return nil, false, err
	}
	return worklogs, true, nil
}

func (s *TimeService) timeWorklogsForRange(ctx context.Context, tempoClient *tempo.Client, accountID string, startDate string, endDate string) ([]TimeWorklog, error) {
	cached, ok, err := s.cachedTimeWorklogs(ctx, accountID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	if ok {
		return cached, nil
	}
	worklogs, err := tempoClient.ListUserWorklogs(ctx, accountID, tempo.WorklogFilter{From: startDate, To: endDate, Limit: 100})
	if err != nil {
		return nil, mapTempoListError(err)
	}
	cachedWorklogs := timeWorklogsFromTempo(worklogs)
	if err := s.repo.ReplaceCachedTimeWorklogs(ctx, accountID, startDate, endDate, cachedWorklogs, nowString()); err != nil {
		return nil, err
	}
	return cachedWorklogs, nil
}

func mapJiraLookupError(action string, err error) error {
	switch {
	case errors.Is(err, jira.ErrNotConfigured):
		return &AppError{Code: 400, Message: "jira integration is not configured", Err: err}
	case errors.Is(err, jira.ErrUnauthorized):
		return &AppError{Code: 401, Message: "jira authentication failed", Err: err}
	case errors.Is(err, jira.ErrNotFound):
		return &AppError{Code: 404, Message: "jira issue not found", Err: err}
	default:
		return fmt.Errorf("%s: %w", action, err)
	}
}

func mapTempoWorklogError(err error) error {
	switch {
	case errors.Is(err, tempo.ErrNotConfigured):
		return &AppError{Code: 400, Message: "tempo integration is not configured", Err: err}
	case errors.Is(err, tempo.ErrUnauthorized):
		return &AppError{Code: 401, Message: "tempo authentication failed", Err: err}
	default:
		return err
	}
}

func mapTempoListError(err error) error {
	switch {
	case errors.Is(err, tempo.ErrNotConfigured):
		return &AppError{Code: 400, Message: "tempo integration is not configured", Err: err}
	case errors.Is(err, tempo.ErrUnauthorized):
		return &AppError{Code: 401, Message: "tempo authentication failed", Err: err}
	default:
		return fmt.Errorf("list tempo worklogs: %w", err)
	}
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
