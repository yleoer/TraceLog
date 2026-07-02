package service

import (
	"database/sql"
	"errors"
	"regexp"
	"strings"
	"time"
)

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
