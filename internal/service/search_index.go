package service

import (
	"context"
	"fmt"
	"strings"
)

func (s *IssueService) indexIssue(ctx context.Context, issue Issue) error {
	return indexIssue(ctx, s.repo, issue)
}

func indexIssue(ctx context.Context, repo SearchIndexRepository, issue Issue) error {
	body := strings.Join([]string{issue.JiraKey, issue.SummaryMD, issue.BackgroundMD, issue.AnalysisMD, issue.SolutionMD, issue.ActionsMD, issue.ResultMD, issue.TodoMD, issue.StartedAt, issue.CompletedAt}, "\n")
	return repo.UpsertSearchIndex(ctx, "issue", issue.JiraKey, issue.JiraKey+" "+issue.Title, body, issue.UpdatedAt)
}

func (s *IssueService) indexIssueEvent(ctx context.Context, issue Issue, event IssueEvent) error {
	return indexIssueEvent(ctx, s.repo, issue, event)
}

func indexIssueEvent(ctx context.Context, repo SearchIndexRepository, issue Issue, event IssueEvent) error {
	title := issue.JiraKey + " " + event.EventType
	return repo.UpsertSearchIndex(ctx, "issue_event", fmt.Sprint(event.ID), title, event.ContentMD, event.UpdatedAt)
}

func (s *TempTaskService) indexTempTask(ctx context.Context, task TempTask) error {
	return indexTempTask(ctx, s.repo, task)
}

func indexTempTask(ctx context.Context, repo SearchIndexRepository, task TempTask) error {
	body := strings.Join([]string{task.Source, task.ContentMD, task.StartedAt, task.CompletedAt, task.ConvertedJiraKey}, "\n")
	return repo.UpsertSearchIndex(ctx, "temp_task", fmt.Sprint(task.ID), task.Title, body, task.UpdatedAt)
}

func (s *TempTaskService) indexTempTaskEvent(ctx context.Context, task TempTask, event TempTaskEvent) error {
	return indexTempTaskEvent(ctx, s.repo, task, event)
}

func indexTempTaskEvent(ctx context.Context, repo SearchIndexRepository, task TempTask, event TempTaskEvent) error {
	title := task.Title + " " + event.EventType
	return repo.UpsertSearchIndex(ctx, "temp_task_event", fmt.Sprint(event.ID), title, event.ContentMD, event.UpdatedAt)
}

func (s *IssueService) indexIssueTodo(ctx context.Context, issue Issue, todo IssueTodo) error {
	return indexIssueTodo(ctx, s.repo, issue, todo)
}

func indexIssueTodo(ctx context.Context, repo SearchIndexRepository, issue Issue, todo IssueTodo) error {
	title := issue.JiraKey + " TODO"
	return repo.UpsertSearchIndex(ctx, "issue_todo", issue.JiraKey+":"+fmt.Sprint(todo.ID), title, todo.Content+"\n"+todo.DueAt, todo.UpdatedAt)
}
