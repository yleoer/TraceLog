package service

import (
	"testing"

	"tracelog/internal/jira"
)

func TestIssueWorkPeriodsFromChangelogTracksReopenedPeriods(t *testing.T) {
	histories := []jira.ChangeHistory{
		statusHistory("5", "2026-07-06T16:00:00Z", "In Progress", "Resolved"),
		statusHistory("1", "2026-07-01T09:00:00.000+0000", "To Do", "In Progress"),
		statusHistory("3", "2026-07-03T09:00:00Z", "Done", "To Do"),
		statusHistory("2", "2026-07-02T17:30:00Z", "In Progress", "Done"),
		statusHistory("4", "2026-07-05T10:00:00Z", "To Do", "In Progress"),
	}

	periods, startedAt, completedAt := issueWorkPeriodsFromChangelog(histories)
	if startedAt != "2026-07-01T09:00:00Z" || completedAt != "2026-07-02T17:30:00Z" {
		t.Fatalf("unexpected first timestamps: started=%q completed=%q", startedAt, completedAt)
	}
	if len(periods) != 2 {
		t.Fatalf("expected two work periods, got %#v", periods)
	}
	if periods[0].StartedAt != "2026-07-01T09:00:00Z" || periods[0].CompletedAt != "2026-07-02T17:30:00Z" {
		t.Fatalf("unexpected first period %#v", periods[0])
	}
	if periods[0].ReopenedAt != "2026-07-03T09:00:00Z" || periods[0].ReopenedStatus != "To Do" {
		t.Fatalf("expected pending interval after reopen, got %#v", periods[0])
	}
	if periods[1].StartedAt != "2026-07-05T10:00:00Z" || periods[1].CompletedAt != "2026-07-06T16:00:00Z" {
		t.Fatalf("unexpected second period %#v", periods[1])
	}
}

func TestIssueWorkPeriodsFromChangelogStartsDirectlyAfterReopen(t *testing.T) {
	histories := []jira.ChangeHistory{
		statusHistory("1", "2026-07-01T09:00:00Z", "To Do", "In Progress"),
		statusHistory("2", "2026-07-02T17:30:00Z", "In Progress", "Done"),
		statusHistory("3", "2026-07-03T09:00:00Z", "Done", "In Progress"),
	}

	periods, _, _ := issueWorkPeriodsFromChangelog(histories)
	if len(periods) != 2 || periods[1].StartedAt != "2026-07-03T09:00:00Z" || periods[1].CompletedAt != "" {
		t.Fatalf("expected an active reopened period, got %#v", periods)
	}
}

func statusHistory(id string, created string, from string, to string) jira.ChangeHistory {
	return jira.ChangeHistory{
		ID:      id,
		Created: created,
		Items:   []jira.ChangeItem{{Field: "status", FromString: from, ToString: to}},
	}
}
