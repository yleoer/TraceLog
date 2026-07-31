package service

import (
	"context"
	"testing"
	"time"
)

func TestGetMonthGanttUsesConfiguredMonthBoundaries(t *testing.T) {
	repo := &ganttRepo{items: []MonthGanttItem{{ID: 1, Source: "issue", JiraKey: "GCS-1", Title: "Issue"}}}
	svc := New(repo, nil, nil)
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatal(err)
	}
	svc.SetLocation(location)

	view, err := svc.GetMonthGantt(context.Background(), "2026-07")
	if err != nil {
		t.Fatal(err)
	}
	if view.Month != "2026-07" || len(view.Items) != 1 {
		t.Fatalf("unexpected month gantt view %#v", view)
	}
	if repo.start != "2026-06-30T16:00:00Z" || repo.end != "2026-07-31T16:00:00Z" {
		t.Fatalf("unexpected month bounds %q to %q", repo.start, repo.end)
	}
}

type ganttRepo struct {
	Repository
	items []MonthGanttItem
	start string
	end   string
}

func (repo *ganttRepo) ListMonthGanttItemsBetween(_ context.Context, start string, end string) ([]MonthGanttItem, error) {
	repo.start = start
	repo.end = end
	return repo.items, nil
}
