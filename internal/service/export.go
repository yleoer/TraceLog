package service

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

func (s *ExportService) ExportJSON(ctx context.Context) ([]byte, error) {
	issues, err := s.issues.ListIssues(ctx, IssueFilter{All: true})
	if err != nil {
		return nil, err
	}
	eventsByIssue := map[string][]IssueEvent{}
	todosByIssue := map[string][]IssueTodo{}
	for _, issue := range issues {
		events, err := s.issues.ListIssueEvents(ctx, issue.JiraKey)
		if err != nil {
			return nil, err
		}
		todos, err := s.issues.ListIssueTodos(ctx, issue.JiraKey, true)
		if err != nil {
			return nil, err
		}
		eventsByIssue[issue.JiraKey] = events
		todosByIssue[issue.JiraKey] = todos
	}
	tasks, err := s.tempTasks.ListTempTasks(ctx, TempTaskFilter{All: true})
	if err != nil {
		return nil, err
	}
	weeks, err := s.weekly.ListWeeklyLogs(ctx)
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

func (s *ExportService) ExportIssueMarkdown(ctx context.Context, jiraKey string) ([]byte, string, error) {
	issue, err := s.issues.GetIssue(ctx, jiraKey)
	if err != nil {
		return nil, "", err
	}
	events, err := s.issues.ListIssueEvents(ctx, jiraKey)
	if err != nil {
		return nil, "", err
	}
	todos, err := s.issues.ListIssueTodos(ctx, jiraKey, true)
	if err != nil {
		return nil, "", err
	}
	return []byte(renderIssueMarkdown(issue, events, todos)), issue.JiraKey + ".md", nil
}

func (s *ExportService) ExportTempTaskMarkdown(ctx context.Context, id int64) ([]byte, string, error) {
	task, err := s.tempTasks.GetTempTask(ctx, id)
	if err != nil {
		return nil, "", err
	}
	return []byte(renderTempTaskMarkdown(task)), fmt.Sprintf("temp-task-%d.md", task.ID), nil
}

func (s *ExportService) ExportWeekMarkdown(ctx context.Context, week string) ([]byte, string, error) {
	view, err := s.weekly.GetWeekView(ctx, week)
	if err != nil {
		return nil, "", err
	}
	return []byte(renderWeekMarkdown(view)), week + ".md", nil
}

func (s *ExportService) ExportMarkdownZip(ctx context.Context) ([]byte, error) {
	buffer := bytes.NewBuffer(nil)
	writer := zip.NewWriter(buffer)

	jsonData, err := s.ExportJSON(ctx)
	if err != nil {
		return nil, err
	}
	if err := writeZipFile(writer, "data.json", bytes.NewReader(jsonData)); err != nil {
		return nil, err
	}

	issues, err := s.issues.ListIssues(ctx, IssueFilter{All: true})
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

	tasks, err := s.tempTasks.ListTempTasks(ctx, TempTaskFilter{All: true})
	if err != nil {
		return nil, err
	}
	for _, task := range tasks {
		if err := writeZipFile(writer, fmt.Sprintf("temp-tasks/%d.md", task.ID), strings.NewReader(renderTempTaskMarkdown(task))); err != nil {
			return nil, err
		}
	}

	weeks, err := s.weekly.ListWeeklyLogs(ctx)
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

func writeZipFile(writer *zip.Writer, name string, reader io.Reader) error {
	file, err := writer.Create(name)
	if err != nil {
		return err
	}
	_, err = io.Copy(file, reader)
	return err
}
