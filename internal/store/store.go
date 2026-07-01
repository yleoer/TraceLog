package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"html"
	"strings"
	"time"

	"tracelog/internal/service"
)

type Store struct {
	db *sql.DB
}

const activityEventsSchema = `
CREATE TABLE IF NOT EXISTS activity_events (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  source TEXT NOT NULL,
  ref_id INTEGER NOT NULL,
  ref_key TEXT NOT NULL DEFAULT '',
  ref_title TEXT NOT NULL,
  event_type TEXT NOT NULL,
  content_md TEXT NOT NULL DEFAULT '',
  happened_at TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_activity_events_happened_at ON activity_events(happened_at);
CREATE INDEX IF NOT EXISTS idx_activity_events_ref ON activity_events(source, ref_id);`

type eventTable struct {
	name     string
	parentID string
}

type eventRow struct {
	ID         int64
	ParentID   int64
	EventType  string
	ContentMD  string
	HappenedAt string
	CreatedAt  string
	UpdatedAt  string
}

var (
	issueEventTable    = eventTable{name: "issue_events", parentID: "issue_id"}
	tempTaskEventTable = eventTable{name: "temp_task_events", parentID: "temp_task_id"}
)

func New(database *sql.DB) *Store {
	return &Store{db: database}
}

func (s *Store) ensureActivityEventsTable(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, activityEventsSchema)
	return err
}

func (s *Store) ListIssues(ctx context.Context, filter service.IssueFilter) ([]service.Issue, error) {
	query := `SELECT id, jira_key, title, status, priority, tags, summary_md, background_md, analysis_md, solution_md, actions_md, result_md, todo_md, links_json, started_at, completed_at, created_at, updated_at FROM issues`
	args := []any{}
	where := []string{}
	if filter.Query != "" {
		like := "%" + filter.Query + "%"
		where = append(where, `(jira_key LIKE ? OR title LIKE ? OR background_md LIKE ? OR analysis_md LIKE ? OR solution_md LIKE ? OR actions_md LIKE ? OR result_md LIKE ? OR todo_md LIKE ?)`)
		args = append(args, like, like, like, like, like, like, like, like)
	}
	if filter.Status != "" {
		where = append(where, "(status = ? OR background_md LIKE ?)")
		args = append(args, filter.Status, "%Jira status: "+filter.Status+"%")
	}
	if filter.Tag != "" {
		tagPattern, err := tagFilterPattern(filter.Tag)
		if err != nil {
			return nil, err
		}
		where = append(where, "tags LIKE ?")
		args = append(args, tagPattern)
	}
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += " ORDER BY updated_at DESC"
	if !filter.All {
		query += " LIMIT ? OFFSET ?"
		args = append(args, limitOrDefault(filter.Limit), max(filter.Offset, 0))
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanIssues(rows)
}

func (s *Store) GetIssue(ctx context.Context, jiraKey string) (service.Issue, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, jira_key, title, status, priority, tags, summary_md, background_md, analysis_md, solution_md, actions_md, result_md, todo_md, links_json, started_at, completed_at, created_at, updated_at FROM issues WHERE jira_key = ?`, jiraKey)
	return scanIssue(row)
}

func (s *Store) CreateIssue(ctx context.Context, issue service.Issue) (service.Issue, error) {
	tags, links, err := encodeIssueJSON(issue)
	if err != nil {
		return service.Issue{}, err
	}
	result, err := s.db.ExecContext(ctx, `INSERT INTO issues (jira_key, title, status, priority, tags, summary_md, background_md, analysis_md, solution_md, actions_md, result_md, todo_md, links_json, started_at, completed_at, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		issue.JiraKey, issue.Title, issue.Status, issue.Priority, tags, issue.SummaryMD, issue.BackgroundMD, issue.AnalysisMD, issue.SolutionMD, issue.ActionsMD, issue.ResultMD, issue.TodoMD, links, issue.StartedAt, issue.CompletedAt, issue.CreatedAt, issue.UpdatedAt)
	if err != nil {
		return service.Issue{}, err
	}
	issue.ID, _ = result.LastInsertId()
	return issue, nil
}

func (s *Store) UpdateIssue(ctx context.Context, issue service.Issue) (service.Issue, error) {
	tags, links, err := encodeIssueJSON(issue)
	if err != nil {
		return service.Issue{}, err
	}
	_, err = s.db.ExecContext(ctx, `UPDATE issues SET title = ?, status = ?, priority = ?, tags = ?, summary_md = ?, background_md = ?, analysis_md = ?, solution_md = ?, actions_md = ?, result_md = ?, todo_md = ?, links_json = ?, started_at = ?, completed_at = ?, updated_at = ? WHERE jira_key = ?`,
		issue.Title, issue.Status, issue.Priority, tags, issue.SummaryMD, issue.BackgroundMD, issue.AnalysisMD, issue.SolutionMD, issue.ActionsMD, issue.ResultMD, issue.TodoMD, links, issue.StartedAt, issue.CompletedAt, issue.UpdatedAt, issue.JiraKey)
	if err != nil {
		return service.Issue{}, err
	}
	return s.GetIssue(ctx, issue.JiraKey)
}

func (s *Store) DeleteIssue(ctx context.Context, jiraKey string) error {
	issue, err := s.GetIssue(ctx, jiraKey)
	if err != nil {
		return err
	}
	if err := s.ensureActivityEventsTable(ctx); err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := tx.ExecContext(ctx, `INSERT INTO activity_events (source, ref_id, ref_key, ref_title, event_type, content_md, happened_at, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"issue", issue.ID, issue.JiraKey, issue.Title, "deleted", "删除 Issue", now, now, now); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM search_index WHERE entity_type = 'issue_event' AND entity_id IN (SELECT CAST(id AS TEXT) FROM issue_events WHERE issue_id = ?)`, issue.ID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM search_index WHERE entity_type = 'issue_todo' AND entity_id LIKE ?`, issue.JiraKey+":%"); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM search_index WHERE entity_type = 'issue' AND entity_id = ?`, issue.JiraKey); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM issues WHERE jira_key = ?`, jiraKey); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) UploadedImageReferenced(ctx context.Context, url string) (bool, error) {
	like := "%" + url + "%"
	queries := []string{
		`SELECT 1 FROM issues WHERE summary_md LIKE ? OR background_md LIKE ? OR analysis_md LIKE ? OR solution_md LIKE ? OR actions_md LIKE ? OR result_md LIKE ? OR todo_md LIKE ? LIMIT 1`,
		`SELECT 1 FROM issue_events WHERE content_md LIKE ? LIMIT 1`,
		`SELECT 1 FROM temp_tasks WHERE content_md LIKE ? LIMIT 1`,
		`SELECT 1 FROM temp_task_events WHERE content_md LIKE ? LIMIT 1`,
		`SELECT 1 FROM weekly_logs WHERE summary_md LIKE ? OR next_plan_md LIKE ? LIMIT 1`,
		`SELECT 1 FROM day_entries WHERE content_md LIKE ? LIMIT 1`,
	}
	for _, query := range queries {
		args := make([]any, strings.Count(query, "?"))
		for index := range args {
			args[index] = like
		}
		var found int
		err := s.db.QueryRowContext(ctx, query, args...).Scan(&found)
		if err == nil {
			return true, nil
		}
		if err != sql.ErrNoRows {
			return false, err
		}
	}
	return false, nil
}

func (s *Store) ListIssueTodos(ctx context.Context, issueID int64, includeDone bool) ([]service.IssueTodo, error) {
	query := `SELECT t.id, t.issue_id, i.jira_key, t.content, t.due_at, t.done, t.created_at, t.updated_at FROM issue_todos t JOIN issues i ON i.id = t.issue_id WHERE t.issue_id = ?`
	args := []any{issueID}
	if !includeDone {
		query += ` AND t.done = 0`
	}
	query += ` ORDER BY t.done ASC, CASE WHEN t.due_at = '' THEN 1 ELSE 0 END, t.due_at ASC, t.updated_at DESC`
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanIssueTodos(rows)
}

func (s *Store) ListOpenIssueTodos(ctx context.Context, limit int) ([]service.IssueTodo, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT t.id, t.issue_id, i.jira_key, t.content, t.due_at, t.done, t.created_at, t.updated_at FROM issue_todos t JOIN issues i ON i.id = t.issue_id WHERE t.done = 0 ORDER BY CASE WHEN t.due_at = '' THEN 1 ELSE 0 END, t.due_at ASC, t.updated_at DESC LIMIT ?`, limitOrDefault(limit))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanIssueTodos(rows)
}

func (s *Store) ListIssueTodosDueBetween(ctx context.Context, start string, end string) ([]service.IssueTodo, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT t.id, t.issue_id, i.jira_key, t.content, t.due_at, t.done, t.created_at, t.updated_at FROM issue_todos t JOIN issues i ON i.id = t.issue_id WHERE t.due_at >= ? AND t.due_at < ? ORDER BY t.done ASC, t.due_at ASC`, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanIssueTodos(rows)
}

func (s *Store) CreateIssueTodo(ctx context.Context, todo service.IssueTodo) (service.IssueTodo, error) {
	result, err := s.db.ExecContext(ctx, `INSERT INTO issue_todos (issue_id, content, due_at, done, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		todo.IssueID, todo.Content, todo.DueAt, boolInt(todo.Done), todo.CreatedAt, todo.UpdatedAt)
	if err != nil {
		return service.IssueTodo{}, err
	}
	todo.ID, _ = result.LastInsertId()
	return s.GetIssueTodo(ctx, todo.ID)
}

func (s *Store) UpdateIssueTodo(ctx context.Context, todo service.IssueTodo) (service.IssueTodo, error) {
	_, err := s.db.ExecContext(ctx, `UPDATE issue_todos SET content = ?, due_at = ?, done = ?, updated_at = ? WHERE id = ?`,
		todo.Content, todo.DueAt, boolInt(todo.Done), todo.UpdatedAt, todo.ID)
	if err != nil {
		return service.IssueTodo{}, err
	}
	return s.GetIssueTodo(ctx, todo.ID)
}

func (s *Store) GetIssueTodo(ctx context.Context, id int64) (service.IssueTodo, error) {
	row := s.db.QueryRowContext(ctx, `SELECT t.id, t.issue_id, i.jira_key, t.content, t.due_at, t.done, t.created_at, t.updated_at FROM issue_todos t JOIN issues i ON i.id = t.issue_id WHERE t.id = ?`, id)
	return scanIssueTodo(row)
}

func (s *Store) DeleteIssueTodo(ctx context.Context, id int64) error {
	todo, err := s.GetIssueTodo(ctx, id)
	if err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx, `DELETE FROM issue_todos WHERE id = ?`, id); err != nil {
		return err
	}
	return s.DeleteSearchIndex(ctx, "issue_todo", todo.JiraKey+":"+fmt.Sprint(id))
}

func (s *Store) ListIssueEvents(ctx context.Context, issueID int64) ([]service.IssueEvent, error) {
	rows, err := s.listEventRows(ctx, issueEventTable, issueID)
	if err != nil {
		return nil, err
	}
	return mapEventRows(rows, issueEventFromRow), nil
}

func (s *Store) CreateIssueEvent(ctx context.Context, event service.IssueEvent) (service.IssueEvent, error) {
	row, err := s.createEventRow(ctx, issueEventTable, eventRowFromIssueEvent(event))
	if err != nil {
		return service.IssueEvent{}, err
	}
	return issueEventFromRow(row), nil
}

func (s *Store) UpdateIssueEvent(ctx context.Context, event service.IssueEvent) (service.IssueEvent, error) {
	if err := s.updateEventRow(ctx, issueEventTable, eventRowFromIssueEvent(event)); err != nil {
		return service.IssueEvent{}, err
	}
	return s.GetIssueEvent(ctx, event.ID)
}

func (s *Store) GetIssueEvent(ctx context.Context, id int64) (service.IssueEvent, error) {
	row, err := s.getEventRow(ctx, issueEventTable, id)
	return issueEventFromRow(row), err
}

func (s *Store) DeleteIssueEvent(ctx context.Context, id int64) error {
	if err := s.deleteEventRow(ctx, issueEventTable, id); err != nil {
		return err
	}
	return s.DeleteSearchIndex(ctx, "issue_event", fmt.Sprint(id))
}

func (s *Store) ListTempTaskEvents(ctx context.Context, taskID int64) ([]service.TempTaskEvent, error) {
	rows, err := s.listEventRows(ctx, tempTaskEventTable, taskID)
	if err != nil {
		return nil, err
	}
	return mapEventRows(rows, tempTaskEventFromRow), nil
}

func (s *Store) CreateTempTaskEvent(ctx context.Context, event service.TempTaskEvent) (service.TempTaskEvent, error) {
	row, err := s.createEventRow(ctx, tempTaskEventTable, eventRowFromTempTaskEvent(event))
	if err != nil {
		return service.TempTaskEvent{}, err
	}
	return tempTaskEventFromRow(row), nil
}

func (s *Store) UpdateTempTaskEvent(ctx context.Context, event service.TempTaskEvent) (service.TempTaskEvent, error) {
	if err := s.updateEventRow(ctx, tempTaskEventTable, eventRowFromTempTaskEvent(event)); err != nil {
		return service.TempTaskEvent{}, err
	}
	return s.GetTempTaskEvent(ctx, event.ID)
}

func (s *Store) GetTempTaskEvent(ctx context.Context, id int64) (service.TempTaskEvent, error) {
	row, err := s.getEventRow(ctx, tempTaskEventTable, id)
	return tempTaskEventFromRow(row), err
}

func (s *Store) DeleteTempTaskEvent(ctx context.Context, id int64) error {
	return s.deleteEventRow(ctx, tempTaskEventTable, id)
}

func (s *Store) ListTempTasks(ctx context.Context, filter service.TempTaskFilter) ([]service.TempTask, error) {
	query := `SELECT id, title, source, status, priority, tags, content_md, started_at, completed_at, converted_to_jira, converted_jira_key, created_at, updated_at FROM temp_tasks`
	args := []any{}
	where := []string{}
	if filter.Query != "" {
		like := "%" + filter.Query + "%"
		where = append(where, `(title LIKE ? OR source LIKE ? OR content_md LIKE ?)`)
		args = append(args, like, like, like)
	}
	if filter.Status != "" {
		where = append(where, "status = ?")
		args = append(args, filter.Status)
	}
	if filter.Tag != "" {
		tagPattern, err := tagFilterPattern(filter.Tag)
		if err != nil {
			return nil, err
		}
		where = append(where, "tags LIKE ?")
		args = append(args, tagPattern)
	}
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += " ORDER BY updated_at DESC"
	if !filter.All {
		query += " LIMIT ? OFFSET ?"
		args = append(args, limitOrDefault(filter.Limit), max(filter.Offset, 0))
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanTempTasks(rows)
}

func (s *Store) GetTempTask(ctx context.Context, id int64) (service.TempTask, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, title, source, status, priority, tags, content_md, started_at, completed_at, converted_to_jira, converted_jira_key, created_at, updated_at FROM temp_tasks WHERE id = ?`, id)
	return scanTempTask(row)
}

func (s *Store) CreateTempTask(ctx context.Context, task service.TempTask) (service.TempTask, error) {
	tags, err := json.Marshal(task.Tags)
	if err != nil {
		return service.TempTask{}, err
	}
	result, err := s.db.ExecContext(ctx, `INSERT INTO temp_tasks (title, source, status, priority, tags, content_md, started_at, completed_at, converted_to_jira, converted_jira_key, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		task.Title, task.Source, task.Status, task.Priority, string(tags), task.ContentMD, task.StartedAt, task.CompletedAt, task.ConvertedToJira, task.ConvertedJiraKey, task.CreatedAt, task.UpdatedAt)
	if err != nil {
		return service.TempTask{}, err
	}
	task.ID, _ = result.LastInsertId()
	return task, nil
}

func (s *Store) UpdateTempTask(ctx context.Context, task service.TempTask) (service.TempTask, error) {
	tags, err := json.Marshal(task.Tags)
	if err != nil {
		return service.TempTask{}, err
	}
	_, err = s.db.ExecContext(ctx, `UPDATE temp_tasks SET title = ?, source = ?, status = ?, priority = ?, tags = ?, content_md = ?, started_at = ?, completed_at = ?, converted_to_jira = ?, converted_jira_key = ?, updated_at = ? WHERE id = ?`,
		task.Title, task.Source, task.Status, task.Priority, string(tags), task.ContentMD, task.StartedAt, task.CompletedAt, task.ConvertedToJira, task.ConvertedJiraKey, task.UpdatedAt, task.ID)
	if err != nil {
		return service.TempTask{}, err
	}
	return s.GetTempTask(ctx, task.ID)
}

func (s *Store) DeleteTempTask(ctx context.Context, id int64) error {
	task, err := s.GetTempTask(ctx, id)
	if err != nil {
		return err
	}
	if err := s.ensureActivityEventsTable(ctx); err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := tx.ExecContext(ctx, `INSERT INTO activity_events (source, ref_id, ref_key, ref_title, event_type, content_md, happened_at, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"temp_task", task.ID, "", task.Title, "deleted", "删除临时需求", now, now, now); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM temp_tasks WHERE id = ?`, id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM search_index WHERE entity_type = 'temp_task' AND entity_id = ?`, fmt.Sprint(id)); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) GetWeeklyLog(ctx context.Context, week string) (service.WeeklyLog, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, week, summary_md, next_plan_md, created_at, updated_at FROM weekly_logs WHERE week = ?`, week)
	return scanWeeklyLog(row)
}

func (s *Store) UpsertWeeklyLog(ctx context.Context, log service.WeeklyLog) (service.WeeklyLog, error) {
	_, err := s.db.ExecContext(ctx, `INSERT INTO weekly_logs (week, summary_md, next_plan_md, created_at, updated_at) VALUES (?, ?, ?, ?, ?) ON CONFLICT(week) DO UPDATE SET summary_md = excluded.summary_md, next_plan_md = excluded.next_plan_md, updated_at = excluded.updated_at`,
		log.Week, log.SummaryMD, log.NextPlanMD, log.CreatedAt, log.UpdatedAt)
	if err != nil {
		return service.WeeklyLog{}, err
	}
	return s.GetWeeklyLog(ctx, log.Week)
}

func (s *Store) ListWeeklyLogs(ctx context.Context) ([]service.WeeklyLog, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, week, summary_md, next_plan_md, created_at, updated_at FROM weekly_logs ORDER BY week DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	logs := []service.WeeklyLog{}
	for rows.Next() {
		log, err := scanWeeklyLogRow(rows)
		if err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}
	return logs, rows.Err()
}

func (s *Store) FirstActivityDate(ctx context.Context) (string, error) {
	if err := s.ensureActivityEventsTable(ctx); err != nil {
		return "", err
	}
	var first sql.NullString
	err := s.db.QueryRowContext(ctx, `
SELECT MIN(date_value) FROM (
  SELECT substr(created_at, 1, 10) AS date_value FROM issues WHERE created_at <> ''
  UNION ALL SELECT substr(updated_at, 1, 10) FROM issues WHERE updated_at <> ''
  UNION ALL SELECT substr(started_at, 1, 10) FROM issues WHERE started_at <> ''
  UNION ALL SELECT substr(completed_at, 1, 10) FROM issues WHERE completed_at <> ''
  UNION ALL SELECT substr(happened_at, 1, 10) FROM issue_events WHERE happened_at <> ''
  UNION ALL SELECT substr(due_at, 1, 10) FROM issue_todos WHERE due_at <> ''
  UNION ALL SELECT substr(created_at, 1, 10) FROM temp_tasks WHERE created_at <> ''
  UNION ALL SELECT substr(updated_at, 1, 10) FROM temp_tasks WHERE updated_at <> ''
  UNION ALL SELECT substr(started_at, 1, 10) FROM temp_tasks WHERE started_at <> ''
  UNION ALL SELECT substr(completed_at, 1, 10) FROM temp_tasks WHERE completed_at <> ''
  UNION ALL SELECT substr(happened_at, 1, 10) FROM temp_task_events WHERE happened_at <> ''
  UNION ALL SELECT substr(happened_at, 1, 10) FROM activity_events WHERE happened_at <> ''
  UNION ALL SELECT date FROM day_entries WHERE date <> ''
)`).Scan(&first)
	if err != nil {
		return "", err
	}
	if !first.Valid {
		return "", nil
	}
	return first.String, nil
}

func (s *Store) ListIssuesUpdatedBetween(ctx context.Context, start string, end string) ([]service.Issue, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT DISTINCT i.id, i.jira_key, i.title, i.status, i.priority, i.tags, i.summary_md, i.background_md, i.analysis_md, i.solution_md, i.actions_md, i.result_md, i.todo_md, i.links_json, i.started_at, i.completed_at, i.created_at, i.updated_at FROM issues i LEFT JOIN issue_events e ON e.issue_id = i.id WHERE (i.updated_at >= ? AND i.updated_at < ?) OR (e.happened_at >= ? AND e.happened_at < ?) OR (i.started_at >= ? AND i.started_at < ?) OR (i.completed_at >= ? AND i.completed_at < ?) ORDER BY i.updated_at DESC`, start, end, start, end, start, end, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanIssues(rows)
}

func (s *Store) ListEventsBetween(ctx context.Context, start string, end string) ([]service.IssueEvent, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, issue_id, event_type, content_md, happened_at, created_at, updated_at FROM issue_events WHERE happened_at >= ? AND happened_at < ? ORDER BY happened_at DESC`, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	events := []service.IssueEvent{}
	for rows.Next() {
		var event service.IssueEvent
		if err := rows.Scan(&event.ID, &event.IssueID, &event.EventType, &event.ContentMD, &event.HappenedAt, &event.CreatedAt, &event.UpdatedAt); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}

func (s *Store) ListTempTasksUpdatedBetween(ctx context.Context, start string, end string) ([]service.TempTask, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, title, source, status, priority, tags, content_md, started_at, completed_at, converted_to_jira, converted_jira_key, created_at, updated_at FROM temp_tasks WHERE (created_at >= ? AND created_at < ?) OR (updated_at >= ? AND updated_at < ?) OR (started_at >= ? AND started_at < ?) OR (completed_at >= ? AND completed_at < ?) ORDER BY updated_at DESC`, start, end, start, end, start, end, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTempTasks(rows)
}

func (s *Store) ListIssueCommentsBetween(ctx context.Context, start string, end string) ([]service.DayComment, error) {
	if err := s.ensureActivityEventsTable(ctx); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT e.id, e.event_type, e.content_md, e.happened_at, i.id, i.jira_key, i.title
FROM issue_events e
JOIN issues i ON e.issue_id = i.id
WHERE e.happened_at >= ? AND e.happened_at < ?
UNION ALL
SELECT 0, 'created', '添加 Issue', i.created_at, i.id, i.jira_key, i.title
FROM issues i
WHERE i.created_at >= ? AND i.created_at < ?
  AND NOT EXISTS (SELECT 1 FROM activity_events a WHERE a.source = 'issue' AND a.ref_id = i.id AND a.event_type = 'created')
ORDER BY 4 ASC`, start, end, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	comments := []service.DayComment{}
	for rows.Next() {
		c := service.DayComment{Source: "issue"}
		if err := rows.Scan(&c.EventID, &c.EventType, &c.ContentMD, &c.HappenedAt, &c.RefID, &c.RefKey, &c.RefTitle); err != nil {
			return nil, err
		}
		comments = append(comments, c)
	}
	return comments, rows.Err()
}

func (s *Store) ListTempTaskCommentsBetween(ctx context.Context, start string, end string) ([]service.DayComment, error) {
	if err := s.ensureActivityEventsTable(ctx); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT e.id, e.event_type, e.content_md, e.happened_at, t.id, t.title
FROM temp_task_events e
JOIN temp_tasks t ON e.temp_task_id = t.id
WHERE e.happened_at >= ? AND e.happened_at < ?
UNION ALL
SELECT 0, 'created', '添加临时需求', t.created_at, t.id, t.title
FROM temp_tasks t
WHERE t.created_at >= ? AND t.created_at < ?
  AND NOT EXISTS (SELECT 1 FROM activity_events a WHERE a.source = 'temp_task' AND a.ref_id = t.id AND a.event_type = 'created')
ORDER BY 4 ASC`, start, end, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	comments := []service.DayComment{}
	for rows.Next() {
		c := service.DayComment{Source: "temp_task"}
		if err := rows.Scan(&c.EventID, &c.EventType, &c.ContentMD, &c.HappenedAt, &c.RefID, &c.RefTitle); err != nil {
			return nil, err
		}
		comments = append(comments, c)
	}
	return comments, rows.Err()
}

func (s *Store) ListActivityEventsBetween(ctx context.Context, start string, end string) ([]service.DayComment, error) {
	if err := s.ensureActivityEventsTable(ctx); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id, source, ref_id, ref_key, ref_title, event_type, content_md, happened_at FROM activity_events WHERE happened_at >= ? AND happened_at < ? ORDER BY happened_at ASC`, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	events := []service.DayComment{}
	for rows.Next() {
		var event service.DayComment
		if err := rows.Scan(&event.EventID, &event.Source, &event.RefID, &event.RefKey, &event.RefTitle, &event.EventType, &event.ContentMD, &event.HappenedAt); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}

func (s *Store) CreateActivityEvent(ctx context.Context, event service.DayComment) error {
	if err := s.ensureActivityEventsTable(ctx); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO activity_events (source, ref_id, ref_key, ref_title, event_type, content_md, happened_at, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		event.Source, event.RefID, event.RefKey, event.RefTitle, event.EventType, event.ContentMD, event.HappenedAt, event.HappenedAt, event.HappenedAt)
	return err
}

func (s *Store) ListDayEntriesBetween(ctx context.Context, startDate string, endDate string) ([]service.DayEntry, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, date, content_md, created_at, updated_at FROM day_entries WHERE date >= ? AND date <= ? ORDER BY created_at ASC`, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	entries := []service.DayEntry{}
	for rows.Next() {
		var e service.DayEntry
		if err := rows.Scan(&e.ID, &e.Date, &e.ContentMD, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

func (s *Store) CreateDayEntry(ctx context.Context, entry service.DayEntry) (service.DayEntry, error) {
	result, err := s.db.ExecContext(ctx, `INSERT INTO day_entries (date, content_md, created_at, updated_at) VALUES (?, ?, ?, ?)`,
		entry.Date, entry.ContentMD, entry.CreatedAt, entry.UpdatedAt)
	if err != nil {
		return service.DayEntry{}, err
	}
	entry.ID, _ = result.LastInsertId()
	return entry, nil
}

func (s *Store) DeleteDayEntry(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM day_entries WHERE id = ?`, id)
	return err
}

func (s *Store) UpsertSearchIndex(ctx context.Context, entityType string, entityID string, title string, body string, updatedAt string) error {
	if err := s.DeleteSearchIndex(ctx, entityType, entityID); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO search_index (entity_type, entity_id, title, body, updated_at) VALUES (?, ?, ?, ?, ?)`, entityType, entityID, title, body, updatedAt)
	return err
}

func (s *Store) DeleteSearchIndex(ctx context.Context, entityType string, entityID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM search_index WHERE entity_type = ? AND entity_id = ?`, entityType, entityID)
	return err
}

func (s *Store) Search(ctx context.Context, query string, entityType string, limit int, offset int) ([]service.SearchResult, error) {
	sqlQuery := `SELECT entity_type, entity_id, title, snippet(search_index, 3, '[[[HL]]]', '[[[/HL]]]', '...', 12), updated_at,
		CASE entity_type
			WHEN 'issue' THEN '/issues/' || entity_id
			WHEN 'issue_event' THEN COALESCE((SELECT '/issues/' || i.jira_key FROM issue_events e JOIN issues i ON i.id = e.issue_id WHERE CAST(e.id AS TEXT) = search_index.entity_id), '/issues')
			WHEN 'issue_todo' THEN CASE WHEN instr(entity_id, ':') > 0 THEN '/issues/' || substr(entity_id, 1, instr(entity_id, ':') - 1) ELSE '/issues' END
			WHEN 'temp_task' THEN '/temp-tasks/' || entity_id
			WHEN 'temp_task_event' THEN COALESCE((SELECT '/temp-tasks/' || e.temp_task_id FROM temp_task_events e WHERE CAST(e.id AS TEXT) = search_index.entity_id), '/temp-tasks')
			WHEN 'weekly_log' THEN '/weeks/' || entity_id
			ELSE '/search'
		END AS url
		FROM search_index WHERE search_index MATCH ?`
	args := []any{query}
	if entityType != "" {
		sqlQuery += ` AND entity_type = ?`
		args = append(args, entityType)
	}
	sqlQuery += ` ORDER BY rank LIMIT ? OFFSET ?`
	args = append(args, limitOrDefault(limit), max(offset, 0))

	rows, err := s.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := []service.SearchResult{}
	for rows.Next() {
		var result service.SearchResult
		if err := rows.Scan(&result.Type, &result.ID, &result.Title, &result.Snippet, &result.UpdatedAt, &result.URL); err != nil {
			return nil, err
		}
		result.Snippet = safeSnippetHTML(result.Snippet)
		results = append(results, result)
	}
	return results, rows.Err()
}

func encodeIssueJSON(issue service.Issue) (string, string, error) {
	tags, err := json.Marshal(issue.Tags)
	if err != nil {
		return "", "", err
	}
	links, err := json.Marshal(issue.Links)
	if err != nil {
		return "", "", err
	}
	return string(tags), string(links), nil
}

func tagFilterPattern(tag string) (string, error) {
	encoded, err := json.Marshal(tag)
	if err != nil {
		return "", err
	}
	return "%" + string(encoded) + "%", nil
}

type scanner interface {
	Scan(dest ...any) error
}

func (s *Store) listEventRows(ctx context.Context, table eventTable, parentID int64) ([]eventRow, error) {
	query := fmt.Sprintf(`SELECT id, %s, event_type, content_md, happened_at, created_at, updated_at FROM %s WHERE %s = ? ORDER BY happened_at DESC, created_at DESC`, table.parentID, table.name, table.parentID)
	rows, err := s.db.QueryContext(ctx, query, parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanEventRows(rows)
}

func (s *Store) getEventRow(ctx context.Context, table eventTable, id int64) (eventRow, error) {
	query := fmt.Sprintf(`SELECT id, %s, event_type, content_md, happened_at, created_at, updated_at FROM %s WHERE id = ?`, table.parentID, table.name)
	return scanEventRow(s.db.QueryRowContext(ctx, query, id))
}

func (s *Store) createEventRow(ctx context.Context, table eventTable, event eventRow) (eventRow, error) {
	query := fmt.Sprintf(`INSERT INTO %s (%s, event_type, content_md, happened_at, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`, table.name, table.parentID)
	result, err := s.db.ExecContext(ctx, query, event.ParentID, event.EventType, event.ContentMD, event.HappenedAt, event.CreatedAt, event.UpdatedAt)
	if err != nil {
		return eventRow{}, err
	}
	event.ID, _ = result.LastInsertId()
	return event, nil
}

func (s *Store) updateEventRow(ctx context.Context, table eventTable, event eventRow) error {
	query := fmt.Sprintf(`UPDATE %s SET event_type = ?, content_md = ?, happened_at = ?, updated_at = ? WHERE id = ?`, table.name)
	_, err := s.db.ExecContext(ctx, query, event.EventType, event.ContentMD, event.HappenedAt, event.UpdatedAt, event.ID)
	return err
}

func (s *Store) deleteEventRow(ctx context.Context, table eventTable, id int64) error {
	query := fmt.Sprintf(`DELETE FROM %s WHERE id = ?`, table.name)
	_, err := s.db.ExecContext(ctx, query, id)
	return err
}

func scanEventRow(row scanner) (eventRow, error) {
	var event eventRow
	err := row.Scan(&event.ID, &event.ParentID, &event.EventType, &event.ContentMD, &event.HappenedAt, &event.CreatedAt, &event.UpdatedAt)
	return event, err
}

func scanEventRows(rows *sql.Rows) ([]eventRow, error) {
	events := []eventRow{}
	for rows.Next() {
		event, err := scanEventRow(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}

func mapEventRows[T any](rows []eventRow, convert func(eventRow) T) []T {
	events := make([]T, 0, len(rows))
	for _, row := range rows {
		events = append(events, convert(row))
	}
	return events
}

func eventRowFromIssueEvent(event service.IssueEvent) eventRow {
	return eventRow{
		ID:         event.ID,
		ParentID:   event.IssueID,
		EventType:  event.EventType,
		ContentMD:  event.ContentMD,
		HappenedAt: event.HappenedAt,
		CreatedAt:  event.CreatedAt,
		UpdatedAt:  event.UpdatedAt,
	}
}

func issueEventFromRow(row eventRow) service.IssueEvent {
	return service.IssueEvent{
		ID:         row.ID,
		IssueID:    row.ParentID,
		EventType:  row.EventType,
		ContentMD:  row.ContentMD,
		HappenedAt: row.HappenedAt,
		CreatedAt:  row.CreatedAt,
		UpdatedAt:  row.UpdatedAt,
	}
}

func eventRowFromTempTaskEvent(event service.TempTaskEvent) eventRow {
	return eventRow{
		ID:         event.ID,
		ParentID:   event.TempTaskID,
		EventType:  event.EventType,
		ContentMD:  event.ContentMD,
		HappenedAt: event.HappenedAt,
		CreatedAt:  event.CreatedAt,
		UpdatedAt:  event.UpdatedAt,
	}
}

func tempTaskEventFromRow(row eventRow) service.TempTaskEvent {
	return service.TempTaskEvent{
		ID:         row.ID,
		TempTaskID: row.ParentID,
		EventType:  row.EventType,
		ContentMD:  row.ContentMD,
		HappenedAt: row.HappenedAt,
		CreatedAt:  row.CreatedAt,
		UpdatedAt:  row.UpdatedAt,
	}
}

func scanIssue(row scanner) (service.Issue, error) {
	var issue service.Issue
	var tags string
	var links string
	err := row.Scan(&issue.ID, &issue.JiraKey, &issue.Title, &issue.Status, &issue.Priority, &tags, &issue.SummaryMD, &issue.BackgroundMD, &issue.AnalysisMD, &issue.SolutionMD, &issue.ActionsMD, &issue.ResultMD, &issue.TodoMD, &links, &issue.StartedAt, &issue.CompletedAt, &issue.CreatedAt, &issue.UpdatedAt)
	if err != nil {
		return service.Issue{}, err
	}
	_ = json.Unmarshal([]byte(tags), &issue.Tags)
	_ = json.Unmarshal([]byte(links), &issue.Links)
	return issue, nil
}

func scanIssues(rows *sql.Rows) ([]service.Issue, error) {
	issues := []service.Issue{}
	for rows.Next() {
		issue, err := scanIssue(rows)
		if err != nil {
			return nil, err
		}
		issues = append(issues, issue)
	}
	return issues, rows.Err()
}

func scanIssueTodo(row scanner) (service.IssueTodo, error) {
	var todo service.IssueTodo
	var done int
	err := row.Scan(&todo.ID, &todo.IssueID, &todo.JiraKey, &todo.Content, &todo.DueAt, &done, &todo.CreatedAt, &todo.UpdatedAt)
	if err != nil {
		return service.IssueTodo{}, err
	}
	todo.Done = done == 1
	return todo, nil
}

func scanIssueTodos(rows *sql.Rows) ([]service.IssueTodo, error) {
	todos := []service.IssueTodo{}
	for rows.Next() {
		todo, err := scanIssueTodo(rows)
		if err != nil {
			return nil, err
		}
		todos = append(todos, todo)
	}
	return todos, rows.Err()
}

func scanTempTask(row scanner) (service.TempTask, error) {
	var task service.TempTask
	var tags string
	var converted int
	err := row.Scan(&task.ID, &task.Title, &task.Source, &task.Status, &task.Priority, &tags, &task.ContentMD, &task.StartedAt, &task.CompletedAt, &converted, &task.ConvertedJiraKey, &task.CreatedAt, &task.UpdatedAt)
	if err != nil {
		return service.TempTask{}, err
	}
	task.ConvertedToJira = converted == 1
	_ = json.Unmarshal([]byte(tags), &task.Tags)
	return task, nil
}

func scanTempTasks(rows *sql.Rows) ([]service.TempTask, error) {
	tasks := []service.TempTask{}
	for rows.Next() {
		task, err := scanTempTask(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	return tasks, rows.Err()
}

func scanWeeklyLog(row scanner) (service.WeeklyLog, error) {
	return scanWeeklyLogRow(row)
}

func scanWeeklyLogRow(row scanner) (service.WeeklyLog, error) {
	var log service.WeeklyLog
	err := row.Scan(&log.ID, &log.Week, &log.SummaryMD, &log.NextPlanMD, &log.CreatedAt, &log.UpdatedAt)
	return log, err
}

func limitOrDefault(limit int) int {
	if limit <= 0 || limit > 200 {
		return 50
	}
	return limit
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func safeSnippetHTML(snippet string) string {
	escaped := html.EscapeString(snippet)
	escaped = strings.ReplaceAll(escaped, "[[[HL]]]", "<mark>")
	escaped = strings.ReplaceAll(escaped, "[[[/HL]]]", "</mark>")
	return escaped
}

func urlForResult(entityType string, id string) string {
	switch entityType {
	case "issue":
		return "/issues/" + id
	case "issue_event":
		return "/search"
	case "issue_todo":
		if jiraKey, _, ok := strings.Cut(id, ":"); ok && jiraKey != "" {
			return "/issues/" + jiraKey
		}
		return "/issues"
	case "temp_task":
		return "/temp-tasks/" + id
	case "weekly_log":
		return "/weeks/" + id
	default:
		return "/search"
	}
}
