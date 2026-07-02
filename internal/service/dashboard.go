package service

import (
	"context"
	"time"
)

func (s *DashboardService) Dashboard(ctx context.Context) (Dashboard, error) {
	issues, err := s.issues.ListIssues(ctx, IssueFilter{Limit: 50})
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
	tasks, err := s.tempTasks.ListTempTasks(ctx, TempTaskFilter{Status: "todo", Limit: 8})
	if err != nil {
		return Dashboard{}, err
	}
	todos, err := s.repo.ListOpenIssueTodos(ctx, 8)
	if err != nil {
		return Dashboard{}, err
	}
	week, err := s.weekly.GetWeekView(ctx, CurrentWeek(s.loc))
	if err != nil {
		return Dashboard{}, err
	}
	return Dashboard{RecentIssues: recent, ActiveIssues: active, TempTasks: tasks, Todos: todos, Week: week}, nil
}

func (s *DashboardService) TodayWorkflow(ctx context.Context) (TodayWorkflow, error) {
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
