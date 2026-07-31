package jira

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetIssueFetchesPaginatedChangelog(t *testing.T) {
	changelogCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/rest/api/3/field":
			_, _ = w.Write([]byte(`[]`))
		case "/rest/api/3/issue/GCS-1":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":  "10001",
				"key": "GCS-1",
				"fields": map[string]any{
					"summary": "Issue summary",
					"status":  map[string]string{"name": "In Progress"},
				},
			})
		case "/rest/api/3/issue/GCS-1/changelog":
			changelogCalls++
			if r.URL.Query().Get("maxResults") != "100" {
				t.Fatalf("expected changelog page size 100, got %q", r.URL.Query().Get("maxResults"))
			}
			switch r.URL.Query().Get("startAt") {
			case "0":
				_ = json.NewEncoder(w).Encode(map[string]any{
					"startAt": 0,
					"total":   2,
					"isLast":  false,
					"values": []any{map[string]any{
						"id":      "1",
						"created": "2026-07-01T09:00:00.000+0000",
						"items":   []any{map[string]string{"field": "status", "fromString": "To Do", "toString": "In Progress"}},
					}},
				})
			case "1":
				_ = json.NewEncoder(w).Encode(map[string]any{
					"startAt": 1,
					"total":   2,
					"isLast":  true,
					"values": []any{map[string]any{
						"id":      "2",
						"created": "2026-07-02T17:30:00.000+0000",
						"items":   []any{map[string]string{"field": "status", "fromString": "In Progress", "toString": "Done"}},
					}},
				})
			default:
				t.Fatalf("unexpected changelog offset %q", r.URL.Query().Get("startAt"))
			}
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := New(Config{BaseURL: server.URL, Email: "dev@example.com", APIToken: "token"})
	issue, err := client.GetIssue(context.Background(), "gcs-1")
	if err != nil {
		t.Fatal(err)
	}
	if issue.Key != "GCS-1" || issue.Fields.Summary != "Issue summary" {
		t.Fatalf("unexpected issue %#v", issue)
	}
	if changelogCalls != 2 || len(issue.Changelog) != 2 {
		t.Fatalf("expected two changelog pages, got calls=%d histories=%#v", changelogCalls, issue.Changelog)
	}
}
