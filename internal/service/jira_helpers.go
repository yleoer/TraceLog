package service

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"tracelog/internal/jira"
)

func extractJiraKey(value string) string {
	match := regexp.MustCompile(`[A-Z][A-Z0-9]+-\d+`).FindString(strings.ToUpper(value))
	return match
}

func renderJiraBackground(issue jira.Issue, browseURL string) string {
	fields := issue.Fields
	var b strings.Builder
	fmt.Fprintf(&b, "Imported from Jira: [%s](%s)\n\n", issue.Key, browseURL)
	if fields.IssueType.Name != "" {
		fmt.Fprintf(&b, "- Type: %s\n", fields.IssueType.Name)
	}
	if fields.Status.Name != "" {
		fmt.Fprintf(&b, "- Jira status: %s\n", fields.Status.Name)
	}
	if fields.Priority.Name != "" {
		fmt.Fprintf(&b, "- Jira priority: %s\n", fields.Priority.Name)
	}
	if fields.Reporter.DisplayName != "" {
		fmt.Fprintf(&b, "- Reporter: %s\n", fields.Reporter.DisplayName)
	}
	if fields.Assignee.DisplayName != "" {
		fmt.Fprintf(&b, "- Assignee: %s\n", fields.Assignee.DisplayName)
	}
	if fields.Created != "" {
		fmt.Fprintf(&b, "- Jira created: %s\n", fields.Created)
	}
	if fields.Updated != "" {
		fmt.Fprintf(&b, "- Jira updated: %s\n", fields.Updated)
	}
	writeNames(&b, "Components", fields.Components)
	if !writeNames(&b, "发布请求", fields.ReleaseRequested) {
		writeNames(&b, "发布请求", fields.FixVersions)
	}

	description := jira.ADFToMarkdown(fields.Description)
	if description != "" {
		b.WriteString("\n## Jira Description\n\n")
		b.WriteString(description)
		b.WriteString("\n")
	}
	return strings.TrimSpace(b.String())
}

func issueWorkPeriodsFromChangelog(histories []jira.ChangeHistory) ([]IssueWorkPeriod, string, string) {
	type statusChange struct {
		from     string
		to       string
		happened time.Time
		sequence int
	}

	changes := []statusChange{}
	sequence := 0
	for _, history := range histories {
		happened, ok := parseJiraHistoryTime(history.Created)
		if !ok {
			continue
		}
		for _, item := range history.Items {
			if !strings.EqualFold(item.Field, "status") && !strings.EqualFold(item.FieldID, "status") {
				continue
			}
			changes = append(changes, statusChange{
				from:     strings.TrimSpace(item.FromString),
				to:       strings.TrimSpace(item.ToString),
				happened: happened,
				sequence: sequence,
			})
			sequence++
		}
	}

	sort.SliceStable(changes, func(i, j int) bool {
		if changes[i].happened.Equal(changes[j].happened) {
			return changes[i].sequence < changes[j].sequence
		}
		return changes[i].happened.Before(changes[j].happened)
	})

	periods := []IssueWorkPeriod{}
	firstStartedAt := ""
	firstCompletedAt := ""
	activePeriod := -1
	for _, change := range changes {
		happenedAt := change.happened.UTC().Format(time.RFC3339)
		if firstStartedAt == "" && isToDoJiraStatus(change.from) && isActiveIssueStatus(change.to) {
			firstStartedAt = happenedAt
		}

		if isActiveIssueStatus(change.to) && !isActiveIssueStatus(change.from) {
			if activePeriod == -1 {
				periods = append(periods, IssueWorkPeriod{StartedAt: happenedAt})
				activePeriod = len(periods) - 1
			}
			continue
		}

		if activePeriod != -1 && isDoneIssueStatus(change.to) && !isDoneIssueStatus(change.from) {
			periods[activePeriod].CompletedAt = happenedAt
			if firstCompletedAt == "" {
				firstCompletedAt = happenedAt
			}
			activePeriod = -1
			continue
		}

		if activePeriod == -1 && isToDoJiraStatus(change.to) && !isToDoJiraStatus(change.from) && len(periods) > 0 {
			last := &periods[len(periods)-1]
			if last.CompletedAt != "" && last.ReopenedAt == "" {
				last.ReopenedAt = happenedAt
				last.ReopenedStatus = change.to
			}
		}
	}
	return periods, firstStartedAt, firstCompletedAt
}

func parseJiraHistoryTime(value string) (time.Time, bool) {
	value = strings.TrimSpace(value)
	for _, layout := range []string{
		time.RFC3339Nano,
		"2006-01-02T15:04:05.999999999-0700",
		"2006-01-02T15:04:05-0700",
	} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed, true
		}
	}
	return time.Time{}, false
}

func isToDoJiraStatus(status string) bool {
	normalized := strings.ToLower(strings.TrimSpace(status))
	return normalized == "to do" || normalized == "to-do" || normalized == "todo" || strings.Contains(status, "待处理")
}

func writeNames(b *strings.Builder, label string, values []jira.Named) bool {
	if len(values) == 0 {
		return false
	}
	names := make([]string, 0, len(values))
	for _, value := range values {
		if value.Name != "" {
			names = append(names, value.Name)
		}
	}
	if len(names) > 0 {
		fmt.Fprintf(b, "- %s: %s\n", label, strings.Join(names, ", "))
		return true
	}
	return false
}

func mapJiraStatus(status jira.Named) string {
	if status.StatusCategory != nil {
		switch strings.ToLower(status.StatusCategory.Key) {
		case "done":
			return "done"
		case "indeterminate":
			return "processing"
		}
	}
	name := strings.ToLower(status.Name)
	switch {
	case strings.Contains(name, "closed"):
		return "closed"
	case strings.Contains(name, "done"), strings.Contains(name, "resolved"):
		return "done"
	case strings.Contains(name, "progress"), strings.Contains(name, "处理中"):
		return "processing"
	default:
		return "analysis"
	}
}

func mapJiraPriority(priority string) string {
	name := strings.ToLower(priority)
	switch {
	case strings.Contains(name, "highest"), strings.Contains(name, "critical"), strings.Contains(name, "blocker"), strings.Contains(name, "urgent"):
		return "urgent"
	case strings.Contains(name, "high"), strings.Contains(name, "major"):
		return "high"
	case strings.Contains(name, "low"), strings.Contains(name, "minor"), strings.Contains(name, "trivial"):
		return "low"
	default:
		return "medium"
	}
}
