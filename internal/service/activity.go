package service

import (
	"context"
)

func (s *IssueService) recordIssueActivity(ctx context.Context, issue Issue, eventType string, content string) error {
	return recordIssueActivity(ctx, s.repo, issue, eventType, content)
}

func recordIssueActivity(ctx context.Context, repo ActivityRepository, issue Issue, eventType string, content string) error {
	now := nowString()
	return repo.CreateActivityEvent(ctx, DayComment{
		Source:     "issue",
		RefID:      issue.ID,
		RefKey:     issue.JiraKey,
		RefTitle:   issue.Title,
		EventType:  eventType,
		ContentMD:  content,
		HappenedAt: now,
	})
}

func (s *TempTaskService) recordTempTaskActivity(ctx context.Context, task TempTask, eventType string, content string) error {
	return recordTempTaskActivity(ctx, s.repo, task, eventType, content)
}

func recordTempTaskActivity(ctx context.Context, repo ActivityRepository, task TempTask, eventType string, content string) error {
	now := nowString()
	return repo.CreateActivityEvent(ctx, DayComment{
		Source:     "temp_task",
		RefID:      task.ID,
		RefTitle:   task.Title,
		EventType:  eventType,
		ContentMD:  content,
		HappenedAt: now,
	})
}
