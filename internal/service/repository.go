package service

import (
	"context"
)

type IssueRepository interface {
	ListIssues(context.Context, IssueFilter) ([]Issue, error)
	GetIssue(context.Context, string) (Issue, error)
	CreateIssue(context.Context, Issue) (Issue, error)
	UpdateIssue(context.Context, Issue) (Issue, error)
	DeleteIssue(context.Context, string) error
	ListIssuesUpdatedBetween(context.Context, string, string) ([]Issue, error)
}

type IssueTodoRepository interface {
	ListIssueTodos(context.Context, int64, bool) ([]IssueTodo, error)
	ListOpenIssueTodos(context.Context, int) ([]IssueTodo, error)
	ListIssueTodosDueBetween(context.Context, string, string) ([]IssueTodo, error)
	ListCompletedIssueTodoCommentsBetween(context.Context, string, string) ([]DayComment, error)
	CreateIssueTodo(context.Context, IssueTodo) (IssueTodo, error)
	UpdateIssueTodo(context.Context, IssueTodo) (IssueTodo, error)
	GetIssueTodo(context.Context, int64) (IssueTodo, error)
	DeleteIssueTodo(context.Context, int64) error
}

type IssueEventRepository interface {
	ListIssueEvents(context.Context, int64) ([]IssueEvent, error)
	CreateIssueEvent(context.Context, IssueEvent) (IssueEvent, error)
	UpdateIssueEvent(context.Context, IssueEvent) (IssueEvent, error)
	GetIssueEvent(context.Context, int64) (IssueEvent, error)
	DeleteIssueEvent(context.Context, int64) error
	ListEventsBetween(context.Context, string, string) ([]IssueEvent, error)
}

type TempTaskRepository interface {
	ListTempTasks(context.Context, TempTaskFilter) ([]TempTask, error)
	GetTempTask(context.Context, int64) (TempTask, error)
	CreateTempTask(context.Context, TempTask) (TempTask, error)
	UpdateTempTask(context.Context, TempTask) (TempTask, error)
	DeleteTempTask(context.Context, int64) error
	ListTempTasksUpdatedBetween(context.Context, string, string) ([]TempTask, error)
}

type TempTaskEventRepository interface {
	ListTempTaskEvents(context.Context, int64) ([]TempTaskEvent, error)
	CreateTempTaskEvent(context.Context, TempTaskEvent) (TempTaskEvent, error)
	UpdateTempTaskEvent(context.Context, TempTaskEvent) (TempTaskEvent, error)
	GetTempTaskEvent(context.Context, int64) (TempTaskEvent, error)
	DeleteTempTaskEvent(context.Context, int64) error
}

type WeeklyRepository interface {
	GetWeeklyLog(context.Context, string) (WeeklyLog, error)
	UpsertWeeklyLog(context.Context, WeeklyLog) (WeeklyLog, error)
	ListWeeklyLogs(context.Context) ([]WeeklyLog, error)
	FirstActivityDate(context.Context) (string, error)
}

type DayRepository interface {
	ListIssueCommentsBetween(context.Context, string, string) ([]DayComment, error)
	ListTempTaskCommentsBetween(context.Context, string, string) ([]DayComment, error)
	ListActivityEventsBetween(context.Context, string, string) ([]DayComment, error)
	ListDayEntriesBetween(context.Context, string, string) ([]DayEntry, error)
	CreateDayEntry(context.Context, DayEntry) (DayEntry, error)
	DeleteDayEntry(context.Context, int64) error
}

type TimeRepository interface {
	ListCachedTimeWorklogs(context.Context, string, string, string) ([]TimeWorklog, error)
	ReplaceCachedTimeWorklogs(context.Context, string, string, string, []TimeWorklog, string) error
	UpsertCachedTimeWorklog(context.Context, string, TimeWorklog, string) error
	GetTimeCacheRange(context.Context, string, string, string) (TimeCacheRange, error)
}

type ActivityRepository interface {
	CreateActivityEvent(context.Context, DayComment) error
}

type SearchIndexRepository interface {
	UpsertSearchIndex(context.Context, string, string, string, string, string) error
	DeleteSearchIndex(context.Context, string, string) error
}

type SearchRepository interface {
	Search(context.Context, string, string, int, int) ([]SearchResult, error)
}

type UploadRepository interface {
	UploadedImageReferenced(context.Context, string) (bool, error)
}

type Repository interface {
	IssueRepository
	IssueTodoRepository
	IssueEventRepository
	TempTaskRepository
	TempTaskEventRepository
	WeeklyRepository
	DayRepository
	TimeRepository
	ActivityRepository
	SearchIndexRepository
	SearchRepository
}

type transactionRepository interface {
	WithTransaction(context.Context, func(Repository) error) error
}
