use crate::error::{invalid, Result};
use rusqlite::{params_from_iter, types::Value as SqlValue, Connection};
use serde::Serialize;

#[derive(Clone, Copy)]
pub enum Table {
    Issue,
    Task,
    IssueEvent,
    TaskEvent,
    Todo,
    Day,
}

impl Table {
    pub fn name(self) -> &'static str {
        match self {
            Self::Issue => "issues",
            Self::Task => "temp_tasks",
            Self::IssueEvent => "issue_events",
            Self::TaskEvent => "temp_task_events",
            Self::Todo => "issue_todos",
            Self::Day => "day_entries",
        }
    }

    fn columns(self) -> &'static [&'static str] {
        match self {
            Self::Issue => &[
                "jira_key",
                "title",
                "status",
                "priority",
                "tags",
                "jira_tags",
                "custom_tags",
                "summary_md",
                "background_md",
                "analysis_md",
                "solution_md",
                "actions_md",
                "result_md",
                "todo_md",
                "links_json",
                "started_at",
                "completed_at",
                "created_at",
                "updated_at",
            ],
            Self::Task => &[
                "title",
                "source",
                "status",
                "priority",
                "tags",
                "content_md",
                "started_at",
                "completed_at",
                "converted_to_jira",
                "converted_jira_key",
                "created_at",
                "updated_at",
            ],
            Self::IssueEvent => &[
                "issue_id",
                "event_type",
                "content_md",
                "happened_at",
                "created_at",
                "updated_at",
            ],
            Self::TaskEvent => &[
                "temp_task_id",
                "event_type",
                "content_md",
                "happened_at",
                "created_at",
                "updated_at",
            ],
            Self::Todo => &[
                "issue_id",
                "content",
                "due_at",
                "done",
                "created_at",
                "updated_at",
            ],
            Self::Day => &["date", "content_md", "created_at", "updated_at"],
        }
    }
}

pub fn save(db: &Connection, table: Table, id: i64, item: &impl Serialize) -> Result<i64> {
    let object = serde_json::to_value(item)?;
    let mut values = Vec::new();
    for column in table.columns() {
        let key = if *column == "links_json" {
            "links"
        } else {
            column
        };
        let value = object
            .get(key)
            .ok_or_else(|| invalid(format!("记录缺少字段 {key}")))?;
        values.push(match value {
            serde_json::Value::String(v) => SqlValue::Text(v.clone()),
            serde_json::Value::Bool(v) => SqlValue::Integer(i64::from(*v)),
            serde_json::Value::Number(v) => {
                SqlValue::Integer(v.as_i64().ok_or_else(|| invalid("记录数值无效"))?)
            }
            serde_json::Value::Array(_) | serde_json::Value::Object(_) => {
                SqlValue::Text(serde_json::to_string(value)?)
            }
            serde_json::Value::Null => SqlValue::Null,
        });
    }
    if id == 0 {
        let slots = vec!["?"; values.len()].join(",");
        db.execute(
            &format!(
                "INSERT INTO {} ({}) VALUES ({slots})",
                table.name(),
                table.columns().join(",")
            ),
            params_from_iter(values),
        )?;
        Ok(db.last_insert_rowid())
    } else {
        let assignments = table
            .columns()
            .iter()
            .map(|v| format!("{v} = ?"))
            .collect::<Vec<_>>()
            .join(",");
        values.push(SqlValue::Integer(id));
        db.execute(
            &format!("UPDATE {} SET {assignments} WHERE id = ?", table.name()),
            params_from_iter(values),
        )?;
        Ok(id)
    }
}
