package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"strconv"
	"strings"
	"testing"

	_ "modernc.org/sqlite"

	"tracelog/internal/service"
)

func TestListIssuesFiltersByExactJSONTag(t *testing.T) {
	store := newTestStore(t)
	insertTestIssue(t, store.db, "GCS-1", "In Progress", `["frontend"]`, "")
	insertTestIssue(t, store.db, "GCS-2", "In Progress", `["front"]`, "")

	issues, err := store.ListIssues(context.Background(), service.IssueFilter{Tag: "front"})
	if err != nil {
		t.Fatal(err)
	}

	if len(issues) != 1 || issues[0].JiraKey != "GCS-2" {
		t.Fatalf("expected only GCS-2 for exact tag front, got %#v", issues)
	}
}

func TestListIssuesSortsByImportTimeByDefault(t *testing.T) {
	store := newTestStore(t)
	firstID := insertTestIssue(t, store.db, "GCS-1", "In Progress", `[]`, "")
	secondID := insertTestIssue(t, store.db, "GCS-2", "In Progress", `[]`, "")
	if _, err := store.db.Exec(`UPDATE issues SET created_at = ?, updated_at = ? WHERE id = ?`, "2026-06-20T00:00:00Z", "2026-06-30T00:00:00Z", firstID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.Exec(`UPDATE issues SET created_at = ?, updated_at = ? WHERE id = ?`, "2026-06-25T00:00:00Z", "2026-06-21T00:00:00Z", secondID); err != nil {
		t.Fatal(err)
	}

	issues, err := store.ListIssues(context.Background(), service.IssueFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) != 2 || issues[0].JiraKey != "GCS-2" || issues[1].JiraKey != "GCS-1" {
		t.Fatalf("expected import time descending order, got %#v", issues)
	}
}

func TestListIssuesWithWorkActivityBetweenExcludesImportOnlyIssues(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	commentedID := insertTestIssue(t, store.db, "GCS-1", "In Progress", `[]`, "")
	startedID := insertTestIssue(t, store.db, "GCS-2", "In Progress", `[]`, "")
	completedID := insertTestIssue(t, store.db, "GCS-3", "Done", `[]`, "")
	insertTestIssue(t, store.db, "GCS-4", "To Do", `[]`, "")

	if _, err := store.db.Exec(`INSERT INTO issue_events (issue_id, event_type, content_md, happened_at, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		commentedID, "note", "weekly comment", "2026-06-24T10:00:00Z", "2026-06-24T10:00:00Z", "2026-06-24T10:00:00Z"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.Exec(`UPDATE issues SET started_at = ? WHERE id = ?`, "2026-06-25T10:00:00Z", startedID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.Exec(`UPDATE issues SET completed_at = ? WHERE id = ?`, "2026-06-26T10:00:00Z", completedID); err != nil {
		t.Fatal(err)
	}

	issues, err := store.ListIssuesWithWorkActivityBetween(ctx, "2026-06-23T00:00:00Z", "2026-06-30T00:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	keys := map[string]bool{}
	for _, issue := range issues {
		keys[issue.JiraKey] = true
	}
	if len(keys) != 3 || !keys["GCS-1"] || !keys["GCS-2"] || !keys["GCS-3"] || keys["GCS-4"] {
		t.Fatalf("expected issues with comments, starts, or completions only, got %#v", issues)
	}
}

func TestListMonthGanttItemsBetweenGroupsOverlappingWorkPeriods(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	legacyIssueID := insertTestIssue(t, store.db, "GCS-1", "Done", `[]`, "")
	periodIssueID := insertTestIssue(t, store.db, "GCS-2", "In Progress", `[]`, "")
	taskID := insertTestTempTask(t, store.db)

	if _, err := store.db.Exec(`UPDATE issues SET started_at = ?, completed_at = ? WHERE id = ?`, "2026-06-30T09:00:00Z", "2026-07-02T17:00:00Z", legacyIssueID); err != nil {
		t.Fatal(err)
	}
	if err := store.ReplaceIssueWorkPeriods(ctx, periodIssueID, []service.IssueWorkPeriod{
		{StartedAt: "2026-06-28T09:00:00Z", CompletedAt: "2026-07-03T17:00:00Z"},
		{StartedAt: "2026-07-20T09:00:00Z"},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.Exec(`UPDATE temp_tasks SET started_at = ?, completed_at = ? WHERE id = ?`, "2026-07-10T09:00:00Z", "2026-07-12T17:00:00Z", taskID); err != nil {
		t.Fatal(err)
	}

	items, err := store.ListMonthGanttItemsBetween(ctx, "2026-06-30T16:00:00Z", "2026-07-31T16:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	byKey := map[string]service.MonthGanttItem{}
	for _, item := range items {
		byKey[item.Source+":"+item.JiraKey+":"+strconv.FormatInt(item.ID, 10)] = item
	}
	legacy := byKey["issue:GCS-1:"+strconv.FormatInt(legacyIssueID, 10)]
	period := byKey["issue:GCS-2:"+strconv.FormatInt(periodIssueID, 10)]
	task := byKey["temp_task::"+strconv.FormatInt(taskID, 10)]
	if len(items) != 3 || len(legacy.Periods) != 1 || len(period.Periods) != 2 || len(task.Periods) != 1 {
		t.Fatalf("expected legacy issue, grouped periods, and temp task, got %#v", items)
	}
	if period.Periods[1].CompletedAt != "" {
		t.Fatalf("expected active work period to remain open, got %#v", period.Periods[1])
	}
}

func TestListCustomIssueTagsReturnsDistinctSortedTags(t *testing.T) {
	store := newTestStore(t)
	issues := []service.Issue{
		{
			JiraKey:    "GCS-1",
			Title:      "First issue",
			Status:     "In Progress",
			Priority:   "medium",
			CustomTags: []string{"shared", "frontend", "shared"},
			CreatedAt:  "2026-06-20T00:00:00Z",
			UpdatedAt:  "2026-06-20T00:00:00Z",
		},
		{
			JiraKey:    "GCS-2",
			Title:      "Second issue",
			Status:     "In Progress",
			Priority:   "medium",
			CustomTags: []string{"Backend", "frontend"},
			CreatedAt:  "2026-06-21T00:00:00Z",
			UpdatedAt:  "2026-06-21T00:00:00Z",
		},
	}
	for _, issue := range issues {
		if _, err := store.CreateIssue(context.Background(), issue); err != nil {
			t.Fatal(err)
		}
	}

	tags, err := store.ListCustomIssueTags(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	assertTags(t, tags, "Backend", "frontend", "shared")
}

func TestIssueTagSourcesPersistAndFilterByAggregateTags(t *testing.T) {
	store := newTestStore(t)
	created, err := store.CreateIssue(context.Background(), service.Issue{
		JiraKey:    "GCS-1",
		Title:      "Test issue",
		Status:     "In Progress",
		Priority:   "medium",
		JiraTags:   []string{"backend", "shared", "backend"},
		CustomTags: []string{"personal", "shared", "  focus  "},
		CreatedAt:  "2026-06-23T00:00:00Z",
		UpdatedAt:  "2026-06-23T00:00:00Z",
	})
	if err != nil {
		t.Fatal(err)
	}

	issue, err := store.GetIssue(context.Background(), created.JiraKey)
	if err != nil {
		t.Fatal(err)
	}
	assertTags(t, issue.JiraTags, "backend", "shared")
	assertTags(t, issue.CustomTags, "personal", "shared", "focus")
	assertTags(t, issue.Tags, "backend", "shared", "personal", "focus")

	for _, tag := range []string{"backend", "personal", "focus"} {
		issues, err := store.ListIssues(context.Background(), service.IssueFilter{Tag: tag})
		if err != nil {
			t.Fatal(err)
		}
		if len(issues) != 1 || issues[0].JiraKey != created.JiraKey {
			t.Fatalf("expected aggregate tag %q to find %s, got %#v", tag, created.JiraKey, issues)
		}
	}

	issue.JiraTags = []string{"platform"}
	issue.CustomTags = []string{"personal", "review"}
	updated, err := store.UpdateIssue(context.Background(), issue)
	if err != nil {
		t.Fatal(err)
	}
	assertTags(t, updated.JiraTags, "platform")
	assertTags(t, updated.CustomTags, "personal", "review")
	assertTags(t, updated.Tags, "platform", "personal", "review")
}

func TestMigrateLegacyIssueTagsToCustomTagSources(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})
	if _, err := db.Exec(`
CREATE TABLE issues (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  tags TEXT NOT NULL DEFAULT '[]'
);
CREATE TABLE app_schema_meta (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL
);
CREATE TABLE issue_tags (
  issue_id INTEGER NOT NULL,
  tag TEXT NOT NULL,
  PRIMARY KEY (issue_id, tag)
);
INSERT INTO issues (tags) VALUES ('["legacy-jira", "personal", "personal"]');`); err != nil {
		t.Fatal(err)
	}

	store := New(db)
	if err := store.ensureIssueTagSourceColumns(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := store.migrateLegacyIssueTags(context.Background()); err != nil {
		t.Fatal(err)
	}

	var tags, jiraTags, customTags string
	if err := db.QueryRow(`SELECT tags, jira_tags, custom_tags FROM issues WHERE id = 1`).Scan(&tags, &jiraTags, &customTags); err != nil {
		t.Fatal(err)
	}
	if tags != `["legacy-jira","personal"]` || jiraTags != `[]` || customTags != `["legacy-jira","personal"]` {
		t.Fatalf("expected legacy tags to become custom tags, got tags=%s jira=%s custom=%s", tags, jiraTags, customTags)
	}
	var indexedTags string
	if err := db.QueryRow(`SELECT group_concat(tag, ',') FROM issue_tags WHERE issue_id = 1 ORDER BY tag`).Scan(&indexedTags); err != nil {
		t.Fatal(err)
	}
	if indexedTags != "legacy-jira,personal" {
		t.Fatalf("expected migrated aggregate tags in filter index, got %q", indexedTags)
	}
	version, err := store.metaValue(context.Background(), "issue_tag_sources_version")
	if err != nil {
		t.Fatal(err)
	}
	if version != "1" {
		t.Fatalf("expected migration version 1, got %q", version)
	}
}

func TestListIssuesSearchUsesFTSIndex(t *testing.T) {
	store := newTestStore(t)
	issueID := insertTestIssue(t, store.db, "GCS-1", "In Progress", `["frontend"]`, "contains indexed keyword")
	if err := store.UpsertSearchIndex(context.Background(), "issue", "GCS-1", "GCS-1 Test issue", "contains indexed keyword", "2026-06-23T00:00:00Z"); err != nil {
		t.Fatal(err)
	}
	insertTestIssue(t, store.db, "GCS-2", "In Progress", `["frontend"]`, "contains indexed keyword")

	issues, err := store.ListIssues(context.Background(), service.IssueFilter{Query: "indexed"})
	if err != nil {
		t.Fatal(err)
	}

	if len(issues) != 1 || issues[0].ID != issueID {
		t.Fatalf("expected only FTS-indexed issue, got %#v", issues)
	}
}

func TestListIssuesStatusMatchesStoredStatusOrJiraBackground(t *testing.T) {
	store := newTestStore(t)
	insertTestIssue(t, store.db, "GCS-1", "In Progress", `[]`, "")
	insertTestIssue(t, store.db, "GCS-2", "processing", `[]`, "- Jira status: In Progress\n")
	insertTestIssue(t, store.db, "GCS-3", "Done", `[]`, "- Jira status: Done\n")

	issues, err := store.ListIssues(context.Background(), service.IssueFilter{Status: "In Progress"})
	if err != nil {
		t.Fatal(err)
	}

	keys := map[string]bool{}
	for _, issue := range issues {
		keys[issue.JiraKey] = true
	}
	if len(keys) != 2 || !keys["GCS-1"] || !keys["GCS-2"] {
		t.Fatalf("expected stored and Jira-background In Progress issues, got %#v", issues)
	}
}

func TestScanIssueReturnsInvalidJSONError(t *testing.T) {
	store := newTestStore(t)
	insertTestIssue(t, store.db, "GCS-1", "In Progress", `not-json`, "")

	_, err := store.GetIssue(context.Background(), "GCS-1")
	if err == nil || !strings.Contains(err.Error(), "decode issue.tags") {
		t.Fatalf("expected invalid issue tags error, got %v", err)
	}
}

func TestScanTempTaskReturnsInvalidJSONError(t *testing.T) {
	store := newTestStore(t)
	id := insertTestTempTaskWithTags(t, store.db, `not-json`, "bad task")

	_, err := store.GetTempTask(context.Background(), id)
	if err == nil || !strings.Contains(err.Error(), "decode temp_tasks.tags") {
		t.Fatalf("expected invalid temp task tags error, got %v", err)
	}
}

func TestReplaceIssueWorkPeriodsCreatesTableForExistingDatabase(t *testing.T) {
	store := newTestStore(t)
	issueID := insertTestIssue(t, store.db, "GCS-1", "Done", `[]`, "")
	periods := []service.IssueWorkPeriod{
		{
			StartedAt:      "2026-07-01T09:00:00Z",
			CompletedAt:    "2026-07-02T17:30:00Z",
			ReopenedAt:     "2026-07-03T09:00:00Z",
			ReopenedStatus: "To Do",
		},
		{StartedAt: "2026-07-05T10:00:00Z"},
	}
	if err := store.ReplaceIssueWorkPeriods(context.Background(), issueID, periods); err != nil {
		t.Fatal(err)
	}

	got, err := store.ListIssueWorkPeriods(context.Background(), issueID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].ReopenedAt != periods[0].ReopenedAt || got[1].StartedAt != periods[1].StartedAt {
		t.Fatalf("unexpected persisted periods %#v", got)
	}

	if err := store.ReplaceIssueWorkPeriods(context.Background(), issueID, nil); err != nil {
		t.Fatal(err)
	}
	got, err = store.ListIssueWorkPeriods(context.Background(), issueID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("expected replacement to remove stale periods, got %#v", got)
	}
}

func TestDeleteIssueRemovesSearchIndexByJiraKey(t *testing.T) {
	store := newTestStore(t)
	insertTestIssue(t, store.db, "GCS-1", "In Progress", `[]`, "")
	if err := store.UpsertSearchIndex(context.Background(), "issue", "GCS-1", "GCS-1 Test", "body", "2026-06-23T00:00:00Z"); err != nil {
		t.Fatal(err)
	}

	if err := store.DeleteIssue(context.Background(), "GCS-1"); err != nil {
		t.Fatal(err)
	}

	var count int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM search_index WHERE entity_type = 'issue' AND entity_id = 'GCS-1'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("expected issue search index to be deleted, got %d rows", count)
	}
}

func TestSearchResultURLsPointToDetailPages(t *testing.T) {
	store := newTestStore(t)
	issueID := insertTestIssue(t, store.db, "GCS-1", "In Progress", `[]`, "")
	issueEventID := insertTestIssueEvent(t, store.db, issueID)
	issueTodoID := insertTestIssueTodo(t, store.db, issueID)
	tempTaskID := insertTestTempTask(t, store.db)
	tempTaskEventID := insertTestTempTaskEvent(t, store.db, tempTaskID)

	index := []struct {
		entityType string
		entityID   string
		title      string
		expected   string
	}{
		{"issue", "GCS-1", "issue hit", "/issues/GCS-1"},
		{"issue_event", intString(issueEventID), "issue event hit", "/issues/GCS-1"},
		{"issue_todo", "GCS-1:" + intString(issueTodoID), "todo hit", "/issues/GCS-1"},
		{"temp_task", intString(tempTaskID), "temp task hit", "/temp-tasks/" + intString(tempTaskID)},
		{"temp_task_event", intString(tempTaskEventID), "temp task event hit", "/temp-tasks/" + intString(tempTaskID)},
		{"weekly_log", "2026-W26", "weekly hit", "/weeks/2026-W26"},
	}
	for _, item := range index {
		if err := store.UpsertSearchIndex(context.Background(), item.entityType, item.entityID, item.title, "jumptarget", "2026-06-23T00:00:00Z"); err != nil {
			t.Fatal(err)
		}
	}

	results, err := store.Search(context.Background(), "jumptarget", "", 50, 0)
	if err != nil {
		t.Fatal(err)
	}

	urls := map[string]string{}
	for _, result := range results {
		urls[result.Type+":"+result.ID] = result.URL
	}
	for _, item := range index {
		key := item.entityType + ":" + item.entityID
		if urls[key] != item.expected {
			t.Fatalf("expected %s to route to %s, got %q from %#v", key, item.expected, urls[key], results)
		}
	}
}

func TestEnsurePerformanceSchemaBackfillsTagsSearchAndRefs(t *testing.T) {
	store := newTestStore(t)
	insertTestIssue(t, store.db, "GCS-1", "In Progress", `["frontend"]`, "legacy searchable issue")
	insertTestTempTaskWithTags(t, store.db, `["ops"]`, "legacy searchable task")
	if _, err := store.db.Exec(`DELETE FROM issue_tags; DELETE FROM temp_task_tags; DELETE FROM search_index; DELETE FROM search_index_refs; DELETE FROM app_schema_meta;`); err != nil {
		t.Fatal(err)
	}

	if err := store.EnsurePerformanceSchema(context.Background()); err != nil {
		t.Fatal(err)
	}

	issues, err := store.ListIssues(context.Background(), service.IssueFilter{Tag: "frontend", Query: "legacy"})
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) != 1 || issues[0].JiraKey != "GCS-1" {
		t.Fatalf("expected backfilled issue tag and search index, got %#v", issues)
	}
	tasks, err := store.ListTempTasks(context.Background(), service.TempTaskFilter{Tag: "ops", Query: "task"})
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 1 || tasks[0].Title != "legacy searchable task" {
		t.Fatalf("expected backfilled task tag and search index, got %#v", tasks)
	}

	var refs int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM search_index_refs`).Scan(&refs); err != nil {
		t.Fatal(err)
	}
	if refs == 0 {
		t.Fatal("expected search index refs to be backfilled")
	}
}

func TestUploadedImageReferencedFindsMarkdownReferences(t *testing.T) {
	store := newTestStore(t)
	uploadURL := "/uploads/20260630T120000-test.png"
	insertTestIssue(t, store.db, "GCS-1", "In Progress", `[]`, "![note]("+uploadURL+")")

	referenced, err := store.UploadedImageReferenced(context.Background(), uploadURL)
	if err != nil {
		t.Fatal(err)
	}
	if !referenced {
		t.Fatal("expected uploaded image to be referenced")
	}

	referenced, err = store.UploadedImageReferenced(context.Background(), "/uploads/missing.png")
	if err != nil {
		t.Fatal(err)
	}
	if referenced {
		t.Fatal("expected missing upload url to be unreferenced")
	}
}

func TestFirstActivityDateUsesEarliestBusinessDate(t *testing.T) {
	store := newTestStore(t)
	issueID := insertTestIssue(t, store.db, "GCS-1", "In Progress", `[]`, "")
	if _, err := store.db.Exec(`UPDATE issues SET started_at = ? WHERE id = ?`, "2026-06-22T00:00:00Z", issueID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.Exec(`INSERT INTO day_entries (date, content_md, created_at, updated_at) VALUES (?, ?, ?, ?)`,
		"2026-06-15", "day note", "2026-06-30T00:00:00Z", "2026-06-30T00:00:00Z"); err != nil {
		t.Fatal(err)
	}

	first, err := store.FirstActivityDate(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if first != "2026-06-15" {
		t.Fatalf("expected earliest activity date 2026-06-15, got %q", first)
	}
}

func TestListActivityEventsBetweenReturnsDeletedRefs(t *testing.T) {
	store := newTestStore(t)
	event := service.DayComment{
		Source:     "temp_task",
		RefID:      42,
		RefTitle:   "Deleted task",
		EventType:  "deleted",
		ContentMD:  "删除临时需求",
		HappenedAt: "2026-06-23T10:00:00Z",
	}
	if err := store.CreateActivityEvent(context.Background(), event); err != nil {
		t.Fatal(err)
	}

	events, err := store.ListActivityEventsBetween(context.Background(), "2026-06-23T00:00:00Z", "2026-06-24T00:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].EventType != "deleted" || events[0].RefTitle != "Deleted task" {
		t.Fatalf("expected deleted activity event, got %#v", events)
	}
}

func TestListCompletedIssueTodoCommentsBetweenReturnsDoneTodos(t *testing.T) {
	store := newTestStore(t)
	issueID := insertTestIssue(t, store.db, "GCS-1", "In Progress", `[]`, "")
	doneTodoID := insertTestIssueTodo(t, store.db, issueID)
	openTodoID := insertTestIssueTodo(t, store.db, issueID)
	if _, err := store.db.Exec(`UPDATE issue_todos SET done = 1, content = ?, updated_at = ? WHERE id = ?`, "finish report", "2026-06-24T10:00:00Z", doneTodoID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.Exec(`UPDATE issue_todos SET done = 0, content = ?, updated_at = ? WHERE id = ?`, "open followup", "2026-06-24T11:00:00Z", openTodoID); err != nil {
		t.Fatal(err)
	}

	comments, err := store.ListCompletedIssueTodoCommentsBetween(context.Background(), "2026-06-24T00:00:00Z", "2026-06-25T00:00:00Z")
	if err != nil {
		t.Fatal(err)
	}

	if len(comments) != 1 {
		t.Fatalf("expected one completed todo comment, got %#v", comments)
	}
	comment := comments[0]
	if comment.EventID != doneTodoID || comment.EventType != "todo_done" || comment.ContentMD != "完成 TODO：finish report" {
		t.Fatalf("unexpected completed todo comment: %#v", comment)
	}
	if comment.Source != "issue" || comment.RefKey != "GCS-1" || comment.RefID != issueID {
		t.Fatalf("expected todo comment to reference issue, got %#v", comment)
	}
}

func TestTempoTimeCacheStoresRangesAndWorklogs(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	worklog := service.TimeWorklog{
		TempoWorklogID:   1001,
		WorkItemKey:      "CORETIME-80",
		Description:      "cached work",
		StartDate:        "2026-07-01",
		StartTime:        "08:00:00",
		EndTime:          "12:00:00",
		TimeSpentSeconds: 4 * 3600,
		Self:             "https://api.tempo.io/4/worklogs/1001",
	}

	if err := store.ReplaceCachedTimeWorklogs(ctx, "account-1", "2026-06-29", "2026-07-05", []service.TimeWorklog{worklog}, "2026-07-02T00:00:00Z"); err != nil {
		t.Fatal(err)
	}
	cacheRange, err := store.GetTimeCacheRange(ctx, "account-1", "2026-07-01", "2026-07-01")
	if err != nil {
		t.Fatal(err)
	}
	if cacheRange.StartDate != "2026-06-29" || cacheRange.EndDate != "2026-07-05" {
		t.Fatalf("expected enclosing week cache range, got %#v", cacheRange)
	}

	worklogs, err := store.ListCachedTimeWorklogs(ctx, "account-1", "2026-07-01", "2026-07-01")
	if err != nil {
		t.Fatal(err)
	}
	if len(worklogs) != 1 || worklogs[0].TempoWorklogID != 1001 || worklogs[0].Hours != 4 {
		t.Fatalf("expected cached worklog, got %#v", worklogs)
	}

	worklog.TempoWorklogID = 1002
	worklog.StartTime = "12:00:00"
	worklog.EndTime = "16:00:00"
	if err := store.UpsertCachedTimeWorklog(ctx, "account-1", worklog, "2026-07-02T01:00:00Z"); err != nil {
		t.Fatal(err)
	}
	worklogs, err = store.ListCachedTimeWorklogs(ctx, "account-1", "2026-07-01", "2026-07-01")
	if err != nil {
		t.Fatal(err)
	}
	if len(worklogs) != 2 {
		t.Fatalf("expected appended cached worklog, got %#v", worklogs)
	}
}

func newTestStore(t *testing.T) *Store {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	schema := `
CREATE TABLE issues (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  jira_key TEXT NOT NULL UNIQUE,
  title TEXT NOT NULL,
  status TEXT NOT NULL,
  priority TEXT NOT NULL,
  tags TEXT NOT NULL DEFAULT '[]',
  jira_tags TEXT NOT NULL DEFAULT '[]',
  custom_tags TEXT NOT NULL DEFAULT '[]',
  summary_md TEXT NOT NULL DEFAULT '',
  background_md TEXT NOT NULL DEFAULT '',
  analysis_md TEXT NOT NULL DEFAULT '',
  solution_md TEXT NOT NULL DEFAULT '',
  actions_md TEXT NOT NULL DEFAULT '',
  result_md TEXT NOT NULL DEFAULT '',
  todo_md TEXT NOT NULL DEFAULT '',
  links_json TEXT NOT NULL DEFAULT '[]',
  started_at TEXT NOT NULL DEFAULT '',
  completed_at TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE TABLE issue_events (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  issue_id INTEGER NOT NULL,
  event_type TEXT NOT NULL,
  content_md TEXT NOT NULL,
  happened_at TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  FOREIGN KEY (issue_id) REFERENCES issues(id) ON DELETE CASCADE
);
CREATE TABLE issue_todos (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  issue_id INTEGER NOT NULL,
  content TEXT NOT NULL,
  due_at TEXT NOT NULL DEFAULT '',
  done INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  FOREIGN KEY (issue_id) REFERENCES issues(id) ON DELETE CASCADE
);
CREATE TABLE issue_tags (
  issue_id INTEGER NOT NULL,
  tag TEXT NOT NULL,
  PRIMARY KEY (issue_id, tag),
  FOREIGN KEY (issue_id) REFERENCES issues(id) ON DELETE CASCADE
);
CREATE TABLE temp_tasks (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  title TEXT NOT NULL,
  source TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL,
  priority TEXT NOT NULL,
  tags TEXT NOT NULL DEFAULT '[]',
  content_md TEXT NOT NULL DEFAULT '',
  started_at TEXT NOT NULL DEFAULT '',
  completed_at TEXT NOT NULL DEFAULT '',
  converted_to_jira INTEGER NOT NULL DEFAULT 0,
  converted_jira_key TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE TABLE temp_task_tags (
  temp_task_id INTEGER NOT NULL,
  tag TEXT NOT NULL,
  PRIMARY KEY (temp_task_id, tag),
  FOREIGN KEY (temp_task_id) REFERENCES temp_tasks(id) ON DELETE CASCADE
);
CREATE TABLE temp_task_events (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  temp_task_id INTEGER NOT NULL,
  event_type TEXT NOT NULL,
  content_md TEXT NOT NULL,
  happened_at TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  FOREIGN KEY (temp_task_id) REFERENCES temp_tasks(id) ON DELETE CASCADE
);
CREATE TABLE weekly_logs (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  week TEXT NOT NULL UNIQUE,
  summary_md TEXT NOT NULL DEFAULT '',
  next_plan_md TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE TABLE day_entries (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  date TEXT NOT NULL,
  content_md TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE TABLE tempo_time_worklogs (
  account_id TEXT NOT NULL,
  tempo_worklog_id INTEGER NOT NULL,
  work_item_key TEXT NOT NULL DEFAULT '',
  description TEXT NOT NULL DEFAULT '',
  start_date TEXT NOT NULL,
  start_time TEXT NOT NULL,
  end_time TEXT NOT NULL,
  time_spent_seconds INTEGER NOT NULL DEFAULT 0,
  self TEXT NOT NULL DEFAULT '',
  cached_at TEXT NOT NULL,
  PRIMARY KEY (account_id, tempo_worklog_id)
);
CREATE TABLE tempo_time_cache_ranges (
  account_id TEXT NOT NULL,
  start_date TEXT NOT NULL,
  end_date TEXT NOT NULL,
  refreshed_at TEXT NOT NULL,
  PRIMARY KEY (account_id, start_date, end_date)
);
CREATE TABLE activity_events (
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
);`
	if _, err := db.Exec(schema); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE VIRTUAL TABLE search_index USING fts5(entity_type UNINDEXED, entity_id UNINDEXED, title, body, updated_at UNINDEXED, tokenize = 'unicode61')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE search_index_refs (entity_type TEXT NOT NULL, entity_id TEXT NOT NULL, search_rowid INTEGER NOT NULL UNIQUE, PRIMARY KEY (entity_type, entity_id)); CREATE TABLE app_schema_meta (key TEXT PRIMARY KEY, value TEXT NOT NULL);`); err != nil {
		t.Fatal(err)
	}
	return New(db)
}

func insertTestIssue(t *testing.T, db *sql.DB, jiraKey string, status string, tags string, background string) int64 {
	t.Helper()
	result, err := db.Exec(
		`INSERT INTO issues (jira_key, title, status, priority, tags, background_md, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		jiraKey,
		"Test issue",
		status,
		"medium",
		tags,
		background,
		"2026-06-23T00:00:00Z",
		"2026-06-23T00:00:00Z",
	)
	if err != nil {
		t.Fatal(err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	var parsedTags []string
	if err := json.Unmarshal([]byte(tags), &parsedTags); err == nil {
		for _, tag := range parsedTags {
			if _, err := db.Exec(`INSERT OR IGNORE INTO issue_tags (issue_id, tag) VALUES (?, ?)`, id, tag); err != nil {
				t.Fatal(err)
			}
		}
	}
	return id
}

func assertTags(t *testing.T, actual []string, expected ...string) {
	t.Helper()
	if strings.Join(actual, ",") != strings.Join(expected, ",") {
		t.Fatalf("expected tags %#v, got %#v", expected, actual)
	}
}

func insertTestIssueEvent(t *testing.T, db *sql.DB, issueID int64) int64 {
	t.Helper()
	result, err := db.Exec(
		`INSERT INTO issue_events (issue_id, event_type, content_md, happened_at, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		issueID,
		"note",
		"event body",
		"2026-06-23T00:00:00Z",
		"2026-06-23T00:00:00Z",
		"2026-06-23T00:00:00Z",
	)
	if err != nil {
		t.Fatal(err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func insertTestIssueTodo(t *testing.T, db *sql.DB, issueID int64) int64 {
	t.Helper()
	result, err := db.Exec(
		`INSERT INTO issue_todos (issue_id, content, due_at, done, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		issueID,
		"todo body",
		"",
		0,
		"2026-06-23T00:00:00Z",
		"2026-06-23T00:00:00Z",
	)
	if err != nil {
		t.Fatal(err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func insertTestTempTask(t *testing.T, db *sql.DB) int64 {
	return insertTestTempTaskWithTags(t, db, `[]`, "Temp task")
}

func insertTestTempTaskWithTags(t *testing.T, db *sql.DB, tags string, title string) int64 {
	t.Helper()
	result, err := db.Exec(
		`INSERT INTO temp_tasks (title, status, priority, tags, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		title,
		"todo",
		"medium",
		tags,
		"2026-06-23T00:00:00Z",
		"2026-06-23T00:00:00Z",
	)
	if err != nil {
		t.Fatal(err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	var parsedTags []string
	if err := json.Unmarshal([]byte(tags), &parsedTags); err == nil {
		for _, tag := range parsedTags {
			if _, err := db.Exec(`INSERT OR IGNORE INTO temp_task_tags (temp_task_id, tag) VALUES (?, ?)`, id, tag); err != nil {
				t.Fatal(err)
			}
		}
	}
	return id
}

func insertTestTempTaskEvent(t *testing.T, db *sql.DB, taskID int64) int64 {
	t.Helper()
	result, err := db.Exec(
		`INSERT INTO temp_task_events (temp_task_id, event_type, content_md, happened_at, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		taskID,
		"note",
		"event body",
		"2026-06-23T00:00:00Z",
		"2026-06-23T00:00:00Z",
		"2026-06-23T00:00:00Z",
	)
	if err != nil {
		t.Fatal(err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func intString(id int64) string {
	return strconv.FormatInt(id, 10)
}
