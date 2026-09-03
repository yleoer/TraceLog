package service

import "testing"

func TestIsDashboardActiveIssueStatusIncludesTodoAndReview(t *testing.T) {
	for _, status := range []string{"TODO", "To Do", "待办", "In Progress", "In Review"} {
		if !isDashboardActiveIssueStatus(status) {
			t.Fatalf("expected %q to be included on the dashboard", status)
		}
	}
	if isDashboardActiveIssueStatus("Done") {
		t.Fatal("expected completed issue to be excluded from the dashboard")
	}
}
