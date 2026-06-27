-- name: GetIssueByJiraKey :one
SELECT * FROM issues WHERE jira_key = ?;

-- name: DeleteIssueByJiraKey :exec
DELETE FROM issues WHERE jira_key = ?;

-- name: ListIssueEvents :many
SELECT * FROM issue_events WHERE issue_id = ? ORDER BY happened_at DESC, created_at DESC;

-- name: DeleteIssueEvent :exec
DELETE FROM issue_events WHERE id = ?;
