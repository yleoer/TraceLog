use crate::models::*;

fn section(title: &str, body: &str) -> String {
    format!("## {title}\n\n{body}\n\n")
}

pub fn todos(todos: &[IssueTodo]) -> String {
    todos
        .iter()
        .map(|todo| {
            format!(
                "- [{}] {}{}\n",
                if todo.done { "x" } else { " " },
                todo.content,
                if todo.due_at.is_empty() {
                    String::new()
                } else {
                    format!(" @due {}", todo.due_at)
                }
            )
        })
        .collect()
}

pub fn issue(issue: &Issue, events: &[IssueEvent], items: &[IssueTodo]) -> String {
    let mut text=format!("# {} {}\n\n- Status: {}\n- Priority: {}\n- Tags: {}\n- Started: {}\n- Completed: {}\n- Created: {}\n- Updated: {}\n\n",issue.jira_key,issue.title,issue.status,issue.priority,issue.tags.join(", "),issue.started_at,issue.completed_at,issue.created_at,issue.updated_at);
    for (title, body) in [
        ("Summary", &issue.summary_md),
        ("Background", &issue.background_md),
        ("Analysis", &issue.analysis_md),
        ("Solution", &issue.solution_md),
        ("Actions", &issue.actions_md),
        ("Result", &issue.result_md),
        ("TODO", &issue.todo_md),
    ] {
        text.push_str(&section(title, body));
    }
    if !items.is_empty() {
        text.push_str(&section("Structured TODOs", &todos(items)));
    }
    if !issue.links.is_empty() {
        text.push_str("## Links\n\n");
        for link in &issue.links {
            text.push_str(&format!(
                "- [{}]({}) ({})\n",
                link.title, link.url, link.r#type
            ));
        }
        text.push('\n');
    }
    text.push_str("## Timeline\n\n");
    for event in events {
        text.push_str(&format!(
            "### {} {}\n\n{}\n\n",
            event.happened_at, event.event_type, event.content_md
        ));
    }
    text
}

pub fn task(task: &TempTask) -> String {
    format!("# {}\n\n- Source: {}\n- Status: {}\n- Priority: {}\n- Tags: {}\n- Started: {}\n- Completed: {}\n- Converted to Jira: {}\n- Jira Key: {}\n\n{}",task.title,task.source,task.status,task.priority,task.tags.join(", "),task.started_at,task.completed_at,task.converted_to_jira,task.converted_jira_key,section("Content",&task.content_md))
}

pub fn week(view: &WeekView) -> String {
    let mut text = format!("# Weekly Log {}\n\n## Jira Issues\n\n", view.log.week);
    for issue in &view.issues {
        text.push_str(&format!(
            "- {} {} ({})\n",
            issue.jira_key, issue.title, issue.status
        ));
    }
    text.push_str("\n## Temp Tasks\n\n");
    for task in &view.temp_tasks {
        text.push_str(&format!("- {} ({})\n", task.title, task.status));
    }
    text.push_str(&format!(
        "\n## TODOs\n\n{}\n## Done\n\n",
        todos(&view.todos)
    ));
    for line in &view.done {
        text.push_str(&format!("- {line}\n"));
    }
    text.push_str("\n## Active\n\n");
    for line in &view.active {
        text.push_str(&format!("- {line}\n"));
    }
    text.push_str(&section("Summary", &view.log.summary_md));
    text.push_str(&section("Next Plan", &view.log.next_plan_md));
    text
}

pub fn workflow(view: &WeekView) -> String {
    let mut text = "## 本周处理\n\n".to_string();
    if view.issues.is_empty() && view.temp_tasks.is_empty() {
        text.push_str("- 暂无记录\n");
    }
    for issue in &view.issues {
        text.push_str(&format!(
            "- {}：{}（{}）\n",
            issue.jira_key,
            if issue.summary_md.trim().is_empty() {
                &issue.title
            } else {
                issue.summary_md.trim()
            },
            issue.status
        ));
    }
    for task in &view.temp_tasks {
        text.push_str(&format!("- 临时需求：{}（{}）\n", task.title, task.status));
    }
    if !view.events.is_empty() {
        text.push_str("\n## 关键过程\n\n");
        for event in &view.events {
            let line = event
                .content_md
                .lines()
                .map(|l| l.trim_matches(['#', '-', '*', '[', ']', ' ']))
                .find(|l| !l.is_empty())
                .unwrap_or("Untitled")
                .chars()
                .take(80)
                .collect::<String>();
            text.push_str(&format!("- {}：{}\n", event.happened_at, line));
        }
    }
    if !view.todos.is_empty() {
        text.push_str(&format!("\n## 后续 TODO\n\n{}", todos(&view.todos)));
    }
    text.push_str("\n## 下周计划\n\n- \n");
    text
}
