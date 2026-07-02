package service

import (
	"fmt"
	"regexp"
	"strings"

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
