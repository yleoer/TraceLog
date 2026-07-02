package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"tracelog/internal/appsettings"
)

func TestSaveUploadedImageStoresPNG(t *testing.T) {
	dir := t.TempDir()
	svc := New(nil, nil, nil, dir)

	image, err := svc.SaveUploadedImage(context.Background(), UploadFile{
		Filename: "note.png",
		Context:  "issue-GCS-45000-comment",
		Reader:   bytes.NewReader(testPNG),
	})
	if err != nil {
		t.Fatal(err)
	}

	if image.URL == "" || image.Filename == "" {
		t.Fatalf("expected upload response fields, got %#v", image)
	}
	if image.ContentType != "image/png" {
		t.Fatalf("expected image/png, got %q", image.ContentType)
	}
	if !strings.HasPrefix(image.Filename, "issue-gcs-45000-comment-") || !strings.Contains(image.Filename, "-note-") {
		t.Fatalf("expected context prefix in filename, got %q", image.Filename)
	}
	if _, err := os.Stat(filepath.Join(dir, image.Filename)); err != nil {
		t.Fatal(err)
	}
}

func TestWeekRangeRejectsOutOfRangeISOWeek(t *testing.T) {
	if _, _, err := weekRange("2026-W00", time.UTC); err == nil {
		t.Fatal("expected week 00 to be rejected")
	}
	if _, _, err := weekRange("2026-W54", time.UTC); err == nil {
		t.Fatal("expected week 54 to be rejected")
	}
	if _, _, err := weekRange("2027-W53", time.UTC); err == nil {
		t.Fatal("expected 2027-W53 to be rejected because 2027 has 52 ISO weeks")
	}
	start, end, err := weekRange("2026-W52", time.UTC)
	if err != nil {
		t.Fatalf("expected valid week 52, got %v", err)
	}
	if start != "2026-12-21T00:00:00Z" || end != "2026-12-28T00:00:00Z" {
		t.Fatalf("unexpected 2026-W52 range: %s %s", start, end)
	}
}

func TestUploadFilenameSanitizesContext(t *testing.T) {
	name := uploadFilename("../Issue GCS-45000 评论!@#", "..\\截图 01.png", time.Date(2026, 6, 30, 15, 30, 12, 0, time.UTC), ".png")
	if !strings.HasPrefix(name, "issue-gcs-45000-20260630T153012-01-") {
		t.Fatalf("expected sanitized context and timestamp, got %q", name)
	}
	if strings.Contains(name, "..") || strings.Contains(name, "/") || strings.Contains(name, "\\") {
		t.Fatalf("expected safe filename, got %q", name)
	}
}

func TestSaveUploadedImageRejectsNonImage(t *testing.T) {
	svc := New(nil, nil, nil, t.TempDir())

	_, err := svc.SaveUploadedImage(context.Background(), UploadFile{
		Filename: "note.txt",
		Reader:   bytes.NewBufferString("plain text"),
	})
	if err == nil {
		t.Fatal("expected error")
	}
	appErr, ok := err.(*AppError)
	if !ok || appErr.Code != http.StatusBadRequest {
		t.Fatalf("expected bad request AppError, got %#v", err)
	}
}

func TestUploadedImageDataURLReturnsDataURL(t *testing.T) {
	dir := t.TempDir()
	svc := New(nil, nil, nil, dir)

	image, err := svc.SaveUploadedImage(context.Background(), UploadFile{
		Filename: "note.png",
		Reader:   bytes.NewReader(testPNG),
	})
	if err != nil {
		t.Fatal(err)
	}

	data, err := svc.UploadedImageDataURL(context.Background(), image.URL)
	if err != nil {
		t.Fatal(err)
	}
	if data.URL != image.URL {
		t.Fatalf("expected url %q, got %q", image.URL, data.URL)
	}
	if !strings.HasPrefix(data.DataURL, "data:image/png;base64,") {
		t.Fatalf("expected png data url, got %q", data.DataURL)
	}
}

func TestUploadedImageDataURLRejectsInvalidURL(t *testing.T) {
	svc := New(nil, nil, nil, t.TempDir())

	_, err := svc.UploadedImageDataURL(context.Background(), "../note.png")
	if err == nil {
		t.Fatal("expected error")
	}
	appErr, ok := err.(*AppError)
	if !ok || appErr.Code != http.StatusBadRequest {
		t.Fatalf("expected bad request AppError, got %#v", err)
	}
}

func TestDeleteUploadedImageRemovesUnreferencedFile(t *testing.T) {
	dir := t.TempDir()
	svc := New(nil, nil, nil, dir)

	image, err := svc.SaveUploadedImage(context.Background(), UploadFile{
		Filename: "note.png",
		Reader:   bytes.NewReader(testPNG),
	})
	if err != nil {
		t.Fatal(err)
	}

	deleted, err := svc.DeleteUploadedImage(context.Background(), image.URL)
	if err != nil {
		t.Fatal(err)
	}
	if !deleted {
		t.Fatal("expected image to be deleted")
	}
	if _, err := os.Stat(filepath.Join(dir, image.Filename)); !os.IsNotExist(err) {
		t.Fatalf("expected uploaded file to be removed, got %v", err)
	}
}

func TestDeleteUploadedImageKeepsReferencedFile(t *testing.T) {
	dir := t.TempDir()
	svc := New(nil, nil, nil, dir)

	image, err := svc.SaveUploadedImage(context.Background(), UploadFile{
		Filename: "note.png",
		Reader:   bytes.NewReader(testPNG),
	})
	if err != nil {
		t.Fatal(err)
	}
	svc.core.repo = uploadReferenceRepo{url: image.URL}

	deleted, err := svc.DeleteUploadedImage(context.Background(), image.URL)
	if err != nil {
		t.Fatal(err)
	}
	if deleted {
		t.Fatal("expected referenced image to be kept")
	}
	if _, err := os.Stat(filepath.Join(dir, image.Filename)); err != nil {
		t.Fatal(err)
	}
}

func TestDeleteUploadedImageRejectsInvalidURL(t *testing.T) {
	svc := New(nil, nil, nil, t.TempDir())

	_, err := svc.DeleteUploadedImage(context.Background(), "../note.png")
	if err == nil {
		t.Fatal("expected error")
	}
	appErr, ok := err.(*AppError)
	if !ok || appErr.Code != http.StatusBadRequest {
		t.Fatalf("expected bad request AppError, got %#v", err)
	}
}

func TestCleanupUnusedUploadedImagesRemovesOnlyUnreferencedImages(t *testing.T) {
	dir := t.TempDir()
	svc := New(nil, nil, nil, dir)

	unused, err := svc.SaveUploadedImage(context.Background(), UploadFile{
		Filename: "unused.png",
		Reader:   bytes.NewReader(testPNG),
	})
	if err != nil {
		t.Fatal(err)
	}
	referenced, err := svc.SaveUploadedImage(context.Background(), UploadFile{
		Filename: "referenced.png",
		Reader:   bytes.NewReader(testPNG),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("keep me"), 0o644); err != nil {
		t.Fatal(err)
	}
	svc.core.repo = uploadReferenceRepo{url: referenced.URL}

	result, err := svc.CleanupUnusedUploadedImages(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result.Scanned != 2 || result.Deleted != 1 || result.Kept != 1 || result.Failed != 0 {
		t.Fatalf("unexpected cleanup result: %#v", result)
	}
	if result.FreedBytes <= 0 {
		t.Fatalf("expected freed bytes, got %#v", result)
	}
	if _, err := os.Stat(filepath.Join(dir, unused.Filename)); !os.IsNotExist(err) {
		t.Fatalf("expected unused image to be removed, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, referenced.Filename)); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "notes.txt")); err != nil {
		t.Fatal(err)
	}
}

func TestCleanupUnusedUploadedImagesMissingDirectory(t *testing.T) {
	svc := New(nil, nil, nil, filepath.Join(t.TempDir(), "missing"))

	result, err := svc.CleanupUnusedUploadedImages(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result != (UploadedImageCleanup{}) {
		t.Fatalf("expected empty cleanup result, got %#v", result)
	}
}

func TestBucketByDayGroupsCommentsByReference(t *testing.T) {
	days := []time.Time{time.Date(2026, 6, 23, 0, 0, 0, 0, time.UTC)}
	comments := []DayComment{
		{Source: "issue", RefID: 1, RefKey: "GCS-1", RefTitle: "Issue one", EventType: "created", ContentMD: "", HappenedAt: "2026-06-23T09:00:00Z"},
		{Source: "issue", RefID: 1, RefKey: "GCS-1", RefTitle: "Issue one", EventType: "note", ContentMD: "comment", HappenedAt: "2026-06-23T10:00:00Z"},
		{Source: "temp_task", RefID: 2, RefTitle: "Task one", EventType: "deleted", ContentMD: "删除临时需求", HappenedAt: "2026-06-23T11:00:00Z"},
	}

	result := bucketByDay(time.UTC, days, comments, nil)

	if len(result) != 1 || len(result[0].Activities) != 2 {
		t.Fatalf("expected two grouped activities, got %#v", result)
	}
	if len(result[0].Activities[0].Comments) != 2 {
		t.Fatalf("expected issue comments grouped together, got %#v", result[0].Activities[0])
	}
	if result[0].Activities[1].Comments[0].EventType != "deleted" {
		t.Fatalf("expected deleted temp task activity, got %#v", result[0].Activities[1])
	}
}

func TestNormalizeLogTimeRequestBuildsDateRange(t *testing.T) {
	input := LogTimeRequest{
		WorkItemKey: " coretime-80 ",
		Description: " SMF work ",
		Hours:       8,
		StartDate:   "2026-07-01",
		EndDate:     "2026-07-03",
	}

	request, dates, err := normalizeLogTimeRequest(input, time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	if request.WorkItemKey != "CORETIME-80" {
		t.Fatalf("expected normalized work item key, got %q", request.WorkItemKey)
	}
	if request.Description != "SMF work" {
		t.Fatalf("expected trimmed description, got %q", request.Description)
	}
	expected := []string{"2026-07-01", "2026-07-02", "2026-07-03"}
	if strings.Join(dates, ",") != strings.Join(expected, ",") {
		t.Fatalf("expected dates %#v, got %#v", expected, dates)
	}
}

func TestNormalizeLogTimeRequestRejectsInvalidHours(t *testing.T) {
	_, _, err := normalizeLogTimeRequest(LogTimeRequest{
		WorkItemKey: "CORETIME-80",
		Description: "SMF work",
		Hours:       9,
		StartDate:   "2026-07-01",
		EndDate:     "2026-07-01",
	}, time.UTC)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAllowedTimeWorkItem(t *testing.T) {
	if !allowedTimeWorkItem("CORETIME-47") {
		t.Fatal("expected CORETIME-47 to be allowed")
	}
	if allowedTimeWorkItem("CORETIME-999") {
		t.Fatal("expected unknown work item to be rejected")
	}
}

func TestCreateIssueRollsBackWhenIndexingFails(t *testing.T) {
	repo := &rollbackRepo{indexErr: errors.New("index failed")}
	svc := New(repo, nil, nil)

	_, err := svc.CreateIssue(context.Background(), Issue{
		JiraKey:  "GCS-1",
		Title:    "Test issue",
		Status:   "analysis",
		Priority: "medium",
	})
	if err == nil {
		t.Fatal("expected create issue to fail")
	}
	if len(repo.issues) != 0 {
		t.Fatalf("expected transaction rollback to remove created issue, got %#v", repo.issues)
	}
	if repo.createdActivities != 0 {
		t.Fatalf("expected transaction rollback to remove activity, got %d", repo.createdActivities)
	}
}

func TestUpdateSettingsFetchesTempoAuthorWhenTokenChanges(t *testing.T) {
	var calls int
	var authErr string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/3/myself" {
			http.NotFound(w, r)
			return
		}
		calls++
		email, token, ok := r.BasicAuth()
		if !ok || email != "dev@example.com" || token != "jira-token" {
			authErr = "unexpected jira credentials"
			http.Error(w, authErr, http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"accountId": "account-123"})
	}))
	defer server.Close()

	store := appsettings.New(filepath.Join(t.TempDir(), "settings.json"), appsettings.Settings{
		Jira: appsettings.JiraSettings{
			BaseURL:  server.URL,
			Email:    "dev@example.com",
			APIToken: "jira-token",
		},
		Tempo: appsettings.TempoSettings{
			BaseURL:  "https://api.tempo.io",
			APIToken: "old-tempo-token",
		},
	})
	svc := New(nil, nil, store)

	settings, err := svc.UpdateSettings(context.Background(), AppSettings{
		Jira: JiraSettings{
			BaseURL: server.URL,
			Email:   "dev@example.com",
		},
		Tempo: TempoSettings{
			BaseURL:  "https://api.tempo.io",
			APIToken: "new-tempo-token",
		},
		AI:       AISettings{Provider: "openai"},
		OpenAI:   ProviderSettings{BaseURL: "https://api.openai.com/v1", Model: "gpt-4.1-mini"},
		DeepSeek: ProviderSettings{BaseURL: "https://api.deepseek.com", Model: "deepseek-v4-flash"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if authErr != "" {
		t.Fatal(authErr)
	}
	if settings.Tempo.AuthorAccountID != "account-123" {
		t.Fatalf("expected fetched account id, got %q", settings.Tempo.AuthorAccountID)
	}
	if calls != 1 {
		t.Fatalf("expected one jira /myself call, got %d", calls)
	}

	saved, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if saved.Jira.APIToken != "jira-token" {
		t.Fatalf("expected saved jira token to be preserved, got %q", saved.Jira.APIToken)
	}
	if saved.Tempo.APIToken != "new-tempo-token" {
		t.Fatalf("expected new tempo token to be saved, got %q", saved.Tempo.APIToken)
	}
	if saved.Tempo.AuthorAccountID != "account-123" {
		t.Fatalf("expected account id to be saved, got %q", saved.Tempo.AuthorAccountID)
	}
}

func TestUpdateSettingsDoesNotRefetchTempoAuthorWhenTokenUnchanged(t *testing.T) {
	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		http.Error(w, "unexpected call", http.StatusInternalServerError)
	}))
	defer server.Close()

	store := appsettings.New(filepath.Join(t.TempDir(), "settings.json"), appsettings.Settings{
		Jira: appsettings.JiraSettings{
			BaseURL:  server.URL,
			Email:    "dev@example.com",
			APIToken: "jira-token",
		},
		Tempo: appsettings.TempoSettings{
			BaseURL:         "https://api.tempo.io",
			APIToken:        "tempo-token",
			AuthorAccountID: "account-123",
		},
	})
	svc := New(nil, nil, store)

	settings, err := svc.UpdateSettings(context.Background(), AppSettings{
		Jira: JiraSettings{
			BaseURL: server.URL,
			Email:   "dev@example.com",
		},
		Tempo: TempoSettings{
			BaseURL:         "https://api.tempo.io",
			AuthorAccountID: "account-123",
		},
		AI:       AISettings{Provider: "openai"},
		OpenAI:   ProviderSettings{BaseURL: "https://api.openai.com/v1", Model: "gpt-4.1-mini"},
		DeepSeek: ProviderSettings{BaseURL: "https://api.deepseek.com", Model: "deepseek-v4-flash"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if settings.Tempo.AuthorAccountID != "account-123" {
		t.Fatalf("expected existing account id, got %q", settings.Tempo.AuthorAccountID)
	}
	if calls != 0 {
		t.Fatalf("did not expect jira call for unchanged tempo token, got %d", calls)
	}
}

func TestTimeStartCalculationsAppendAfterLoggedSeconds(t *testing.T) {
	if got := startTimeAfterLoggedSeconds(0); got != "08:00:00" {
		t.Fatalf("expected first entry to start at 08:00:00, got %q", got)
	}
	if got := startTimeAfterLoggedSeconds(4 * 3600); got != "12:00:00" {
		t.Fatalf("expected second 4h entry to start at 12:00:00, got %q", got)
	}
	if got := endTimeFromStart("12:00:00", 4); got != "16:00:00" {
		t.Fatalf("expected 4h entry from noon to end at 16:00:00, got %q", got)
	}
}

type uploadReferenceRepo struct {
	Repository
	url string
}

func (repo uploadReferenceRepo) UploadedImageReferenced(_ context.Context, url string) (bool, error) {
	return url == repo.url, nil
}

type rollbackRepo struct {
	Repository
	issues            []Issue
	createdActivities int
	indexErr          error
	nextID            int64
}

func (repo *rollbackRepo) WithTransaction(ctx context.Context, fn func(Repository) error) error {
	snapshotIssues := append([]Issue(nil), repo.issues...)
	snapshotActivities := repo.createdActivities
	if err := fn(repo); err != nil {
		repo.issues = snapshotIssues
		repo.createdActivities = snapshotActivities
		return err
	}
	_ = ctx
	return nil
}

func (repo *rollbackRepo) CreateIssue(_ context.Context, issue Issue) (Issue, error) {
	repo.nextID++
	issue.ID = repo.nextID
	repo.issues = append(repo.issues, issue)
	return issue, nil
}

func (repo *rollbackRepo) CreateActivityEvent(_ context.Context, event DayComment) error {
	repo.createdActivities++
	return nil
}

func (repo *rollbackRepo) UpsertSearchIndex(_ context.Context, entityType string, entityID string, title string, body string, updatedAt string) error {
	return repo.indexErr
}

var testPNG = []byte{
	0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
	0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
	0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
	0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4,
	0x89,
}
