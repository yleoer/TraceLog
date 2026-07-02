package service

import (
	"fmt"
	"strings"
)

func firstLine(value string) string {
	for _, line := range strings.Split(strings.TrimSpace(value), "\n") {
		line = strings.TrimSpace(strings.Trim(line, "#-*[] "))
		if line != "" {
			if len(line) > 80 {
				return line[:80]
			}
			return line
		}
	}
	return "Untitled"
}

func renderIssueMarkdown(issue Issue, events []IssueEvent, todos []IssueTodo) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s %s\n\n", issue.JiraKey, issue.Title)
	fmt.Fprintf(&b, "- Status: %s\n- Priority: %s\n- Tags: %s\n- Started: %s\n- Completed: %s\n- Created: %s\n- Updated: %s\n\n", issue.Status, issue.Priority, strings.Join(issue.Tags, ", "), issue.StartedAt, issue.CompletedAt, issue.CreatedAt, issue.UpdatedAt)
	writeSection(&b, "Summary", issue.SummaryMD)
	writeSection(&b, "Background", issue.BackgroundMD)
	writeSection(&b, "Analysis", issue.AnalysisMD)
	writeSection(&b, "Solution", issue.SolutionMD)
	writeSection(&b, "Actions", issue.ActionsMD)
	writeSection(&b, "Result", issue.ResultMD)
	writeSection(&b, "TODO", issue.TodoMD)
	if len(todos) > 0 {
		b.WriteString("## Structured TODOs\n\n")
		b.WriteString(renderIssueTodoMarkdown(todos))
		b.WriteString("\n")
	}
	if len(issue.Links) > 0 {
		b.WriteString("## Links\n\n")
		for _, link := range issue.Links {
			fmt.Fprintf(&b, "- [%s](%s) (%s)\n", link.Title, link.URL, link.Type)
		}
		b.WriteString("\n")
	}
	b.WriteString("## Timeline\n\n")
	for _, event := range events {
		fmt.Fprintf(&b, "### %s %s\n\n%s\n\n", event.HappenedAt, event.EventType, event.ContentMD)
	}
	return b.String()
}

func renderIssueTodoMarkdown(todos []IssueTodo) string {
	var b strings.Builder
	for _, todo := range todos {
		marker := " "
		if todo.Done {
			marker = "x"
		}
		due := ""
		if todo.DueAt != "" {
			due = " @due " + todo.DueAt
		}
		fmt.Fprintf(&b, "- [%s] %s%s\n", marker, todo.Content, due)
	}
	return b.String()
}

func renderTempTaskMarkdown(task TempTask) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", task.Title)
	fmt.Fprintf(&b, "- Source: %s\n- Status: %s\n- Priority: %s\n- Tags: %s\n- Started: %s\n- Completed: %s\n- Converted to Jira: %t\n- Jira Key: %s\n\n", task.Source, task.Status, task.Priority, strings.Join(task.Tags, ", "), task.StartedAt, task.CompletedAt, task.ConvertedToJira, task.ConvertedJiraKey)
	writeSection(&b, "Content", task.ContentMD)
	return b.String()
}

func renderWeekMarkdown(view WeekView) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Weekly Log %s\n\n", view.Log.Week)
	b.WriteString("## Jira Issues\n\n")
	for _, issue := range view.Issues {
		fmt.Fprintf(&b, "- %s %s (%s)\n", issue.JiraKey, issue.Title, issue.Status)
	}
	b.WriteString("\n## Temp Tasks\n\n")
	for _, task := range view.TempTasks {
		fmt.Fprintf(&b, "- %s (%s)\n", task.Title, task.Status)
	}
	b.WriteString("\n## TODOs\n\n")
	b.WriteString(renderIssueTodoMarkdown(view.Todos))
	b.WriteString("\n## Done\n\n")
	for _, item := range view.Done {
		fmt.Fprintf(&b, "- %s\n", item)
	}
	b.WriteString("\n## Active\n\n")
	for _, item := range view.Active {
		fmt.Fprintf(&b, "- %s\n", item)
	}
	writeSection(&b, "Summary", view.Log.SummaryMD)
	writeSection(&b, "Next Plan", view.Log.NextPlanMD)
	return b.String()
}

func renderWorkflowDraft(view WeekView) string {
	var b strings.Builder
	b.WriteString("## 本周处理\n\n")
	if len(view.Issues) == 0 && len(view.TempTasks) == 0 {
		b.WriteString("- 暂无记录\n")
	}
	for _, issue := range view.Issues {
		summary := strings.TrimSpace(issue.SummaryMD)
		if summary == "" {
			summary = issue.Title
		}
		fmt.Fprintf(&b, "- %s：%s（%s）\n", issue.JiraKey, summary, issue.Status)
	}
	for _, task := range view.TempTasks {
		fmt.Fprintf(&b, "- 临时需求：%s（%s）\n", task.Title, task.Status)
	}
	if len(view.Events) > 0 {
		b.WriteString("\n## 关键过程\n\n")
		for _, event := range view.Events {
			fmt.Fprintf(&b, "- %s：%s\n", event.HappenedAt, firstLine(event.ContentMD))
		}
	}
	if len(view.Todos) > 0 {
		b.WriteString("\n## 后续 TODO\n\n")
		b.WriteString(renderIssueTodoMarkdown(view.Todos))
	}
	b.WriteString("\n## 下周计划\n\n- \n")
	return b.String()
}

func writeSection(b *strings.Builder, title string, body string) {
	fmt.Fprintf(b, "## %s\n\n%s\n\n", title, body)
}
