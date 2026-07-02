package service

import (
	"context"
	"fmt"
	"strings"
)

func (s *IssueService) ListIssues(ctx context.Context, filter IssueFilter) ([]Issue, error) {
	return s.repo.ListIssues(ctx, filter)
}

func (s *IssueService) GetIssue(ctx context.Context, jiraKey string) (Issue, error) {
	issue, err := s.repo.GetIssue(ctx, strings.ToUpper(jiraKey))
	return issue, mapError(err)
}

func (s *IssueService) CreateIssue(ctx context.Context, issue Issue) (Issue, error) {
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
	var created Issue
	if err := s.withRepository(ctx, func(repo Repository) error {
		var err error
		created, err = repo.CreateIssue(ctx, issue)
		if err != nil {
			return err
		}
		if err := recordIssueActivity(ctx, repo, created, "created", ""); err != nil {
			return err
		}
		return indexIssue(ctx, repo, created)
	}); err != nil {
		return Issue{}, err
	}
	return created, nil
}

func (s *IssueService) UpdateIssue(ctx context.Context, jiraKey string, issue Issue) (Issue, error) {
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
	var updated Issue
	if err := s.withRepository(ctx, func(repo Repository) error {
		var err error
		updated, err = repo.UpdateIssue(ctx, issue)
		if err != nil {
			return err
		}
		return indexIssue(ctx, repo, updated)
	}); err != nil {
		return Issue{}, err
	}
	return updated, nil
}

func (s *IssueService) DeleteIssue(ctx context.Context, jiraKey string) error {
	return mapError(s.repo.DeleteIssue(ctx, strings.ToUpper(jiraKey)))
}

func (s *IssueService) ListIssueTodos(ctx context.Context, jiraKey string, includeDone bool) ([]IssueTodo, error) {
	issue, err := s.GetIssue(ctx, jiraKey)
	if err != nil {
		return nil, err
	}
	return s.repo.ListIssueTodos(ctx, issue.ID, includeDone)
}

func (s *IssueService) CreateIssueTodo(ctx context.Context, jiraKey string, todo IssueTodo) (IssueTodo, error) {
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
	var created IssueTodo
	if err := s.withRepository(ctx, func(repo Repository) error {
		var err error
		created, err = repo.CreateIssueTodo(ctx, todo)
		if err != nil {
			return err
		}
		return indexIssueTodo(ctx, repo, issue, created)
	}); err != nil {
		return IssueTodo{}, err
	}
	return created, nil
}

func (s *IssueService) UpdateIssueTodo(ctx context.Context, id int64, todo IssueTodo) (IssueTodo, error) {
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
	var updated IssueTodo
	if err := s.withRepository(ctx, func(repo Repository) error {
		var err error
		updated, err = repo.UpdateIssueTodo(ctx, todo)
		if err != nil {
			return err
		}
		return indexIssueTodo(ctx, repo, issue, updated)
	}); err != nil {
		return IssueTodo{}, err
	}
	return updated, nil
}

func (s *IssueService) DeleteIssueTodo(ctx context.Context, id int64) error {
	return mapError(s.repo.DeleteIssueTodo(ctx, id))
}

func (s *IssueService) ListIssueEvents(ctx context.Context, jiraKey string) ([]IssueEvent, error) {
	issue, err := s.GetIssue(ctx, jiraKey)
	if err != nil {
		return nil, err
	}
	return s.repo.ListIssueEvents(ctx, issue.ID)
}

func (s *IssueService) CreateIssueEvent(ctx context.Context, jiraKey string, event IssueEvent) (IssueEvent, error) {
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
	var created IssueEvent
	if err := s.withRepository(ctx, func(repo Repository) error {
		var err error
		created, err = repo.CreateIssueEvent(ctx, event)
		if err != nil {
			return err
		}
		return indexIssueEvent(ctx, repo, issue, created)
	}); err != nil {
		return IssueEvent{}, err
	}
	return created, nil
}

func (s *IssueService) UpdateIssueEvent(ctx context.Context, id int64, event IssueEvent) (IssueEvent, error) {
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
	var updated IssueEvent
	if err := s.withRepository(ctx, func(repo Repository) error {
		var err error
		updated, err = repo.UpdateIssueEvent(ctx, event)
		if err != nil {
			return err
		}
		return repo.UpsertSearchIndex(ctx, "issue_event", fmt.Sprint(updated.ID), updated.EventType, updated.ContentMD, updated.UpdatedAt)
	}); err != nil {
		return IssueEvent{}, err
	}
	return updated, nil
}

func (s *IssueService) DeleteIssueEvent(ctx context.Context, id int64) error {
	return mapError(s.repo.DeleteIssueEvent(ctx, id))
}
