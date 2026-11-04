use super::{
    local,
    remote::{self, text},
    Service,
};
use crate::{
    error::{invalid, AppError, Result},
    models::*,
};
use chrono::{DateTime, Utc};
use serde_json::Value;

impl Service {
    pub async fn jira_get(
        &self,
        settings: &AppSettings,
        path: &str,
        query: &[(&str, String)],
    ) -> Result<Value> {
        let jira = &settings.jira;
        if jira.base_url.is_empty() || jira.email.is_empty() || jira.api_token.is_empty() {
            return Err(invalid("请先配置 Jira 地址、邮箱和 API Token"));
        }
        remote::json(
            self.http
                .get(remote::endpoint(&jira.base_url, path)?)
                .basic_auth(&jira.email, Some(&jira.api_token))
                .query(query),
            "Jira",
        )
        .await
    }

    pub async fn jira_author(&self, settings: &AppSettings) -> Result<String> {
        let value = self.jira_get(settings, "/rest/api/3/myself", &[]).await?;
        let author = text(&value["accountId"]);
        if author.is_empty() {
            return Err(AppError::Remote("Jira 未返回当前账户 ID".into()));
        }
        Ok(author)
    }

    pub async fn import_jira(&self, key: &str) -> Result<Issue> {
        let key = key.trim().to_uppercase();
        if !local::valid_jira_key(&key) {
            return Err(invalid("Jira 编号格式无效"));
        }
        let settings = self.load_settings()?;
        let fields = self.jira_get(&settings, "/rest/api/3/field", &[]).await?;
        let custom = fields
            .as_array()
            .and_then(|items| {
                items
                    .iter()
                    .find(|v| text(&v["name"]).eq_ignore_ascii_case("Release Requested"))
            })
            .map(|v| text(&v["id"]))
            .unwrap_or_default();
        let mut requested = "summary,description,status,priority,labels,issuetype,created,updated,reporter,assignee,components,fixVersions".to_string();
        if !custom.is_empty() {
            requested.push(',');
            requested.push_str(&custom);
        }
        let raw = self
            .jira_get(
                &settings,
                &format!("/rest/api/3/issue/{key}"),
                &[("fields", requested)],
            )
            .await?;
        let mut histories = Vec::new();
        let mut start = 0;
        for page_number in 0..1000 {
            let page = self
                .jira_get(
                    &settings,
                    &format!("/rest/api/3/issue/{key}/changelog"),
                    &[("startAt", start.to_string()), ("maxResults", "100".into())],
                )
                .await?;
            let values = page["values"]
                .as_array()
                .ok_or_else(|| AppError::Remote("Jira 历史分页格式无效".into()))?;
            histories.extend_from_slice(values);
            let next = start + values.len();
            if page["isLast"] == true
                || values.is_empty()
                || page["total"].as_u64().is_some_and(|n| next >= n as usize)
            {
                break;
            }
            if page_number == 999 {
                return Err(AppError::Remote("Jira 历史分页超过限制".into()));
            }
            start = next;
        }
        let f = &raw["fields"];
        let (periods, started_at, completed_at) = work_periods(&histories);
        let browse = format!("{}/browse/{key}", settings.jira.base_url);
        let mut background = format!("Imported from Jira: [{key}]({browse})\n\n");
        for (label, value) in [
            ("Type", &f["issuetype"]["name"]),
            ("Jira status", &f["status"]["name"]),
            ("Jira priority", &f["priority"]["name"]),
            ("Reporter", &f["reporter"]["displayName"]),
            ("Assignee", &f["assignee"]["displayName"]),
            ("Jira created", &f["created"]),
            ("Jira updated", &f["updated"]),
        ] {
            let value = text(value);
            if !value.is_empty() {
                background.push_str(&format!("- {label}: {value}\n"));
            }
        }
        let release = names(&f[&custom]);
        for (label, value) in [
            ("Components", names(&f["components"])),
            (
                "发布请求",
                if release.is_empty() {
                    names(&f["fixVersions"])
                } else {
                    release
                },
            ),
        ] {
            if !value.is_empty() {
                background.push_str(&format!("- {label}: {value}\n"));
            }
        }
        let description = adf(&f["description"]);
        if !description.trim().is_empty() {
            background.push_str(&format!("\n## Jira Description\n\n{}", description.trim()));
        }
        let tags: Vec<String> = f["labels"]
            .as_array()
            .map(|v| {
                v.iter()
                    .filter_map(Value::as_str)
                    .map(str::to_owned)
                    .collect()
            })
            .unwrap_or_default();
        Ok(Issue {
            jira_key: key,
            title: text(&f["summary"]),
            status: text(&f["status"]["name"]),
            priority: priority(&text(&f["priority"]["name"])).into(),
            tags: tags.clone(),
            jira_tags: tags,
            background_md: background.trim().into(),
            links: vec![Link {
                title: "Jira".into(),
                url: browse,
                r#type: "jira".into(),
            }],
            work_periods: periods,
            started_at,
            completed_at,
            ..Default::default()
        })
    }

    pub async fn refresh_jira(&self, key: &str) -> Result<Issue> {
        let imported = self.import_jira(key).await?;
        self.with_db(|db| {
            let mut current = local::get_issue(db, &imported.jira_key)?;
            current.title = imported.title;
            current.status = imported.status;
            current.priority = imported.priority;
            current.jira_tags = imported.jira_tags;
            current.background_md = imported.background_md;
            current.links = imported.links;
            current.started_at = imported.started_at;
            current.completed_at = imported.completed_at;
            current.work_periods = imported.work_periods;
            local::save_issue(db, Some(&current.jira_key.clone()), current, true)
        })
    }
}

fn names(value: &Value) -> String {
    if let Some(values) = value.as_array() {
        return values
            .iter()
            .map(names)
            .filter(|s| !s.is_empty())
            .collect::<Vec<_>>()
            .join(", ");
    }
    if value.is_string() {
        text(value)
    } else {
        text(&value["name"])
    }
}

fn priority(value: &str) -> &str {
    let value = value.to_lowercase();
    if ["highest", "critical", "blocker", "urgent"]
        .iter()
        .any(|s| value.contains(s))
    {
        "urgent"
    } else if ["high", "major"].iter().any(|s| value.contains(s)) {
        "high"
    } else if ["low", "minor", "trivial"]
        .iter()
        .any(|s| value.contains(s))
    {
        "low"
    } else {
        "medium"
    }
}
fn active(status: &str) -> bool {
    let s = status.to_lowercase();
    ["processing", "progress", "indeterminate", "处理中"]
        .iter()
        .any(|n| s.contains(n))
}
fn done(status: &str) -> bool {
    let s = status.to_lowercase();
    ["done", "closed", "resolved", "完成", "关闭"]
        .iter()
        .any(|n| s.contains(n))
}
fn todo(status: &str) -> bool {
    matches!(
        status.trim().to_lowercase().as_str(),
        "to do" | "to-do" | "todo"
    ) || status.contains("待处理")
}

pub fn work_periods(histories: &[Value]) -> (Vec<IssueWorkPeriod>, String, String) {
    let mut changes = Vec::new();
    for h in histories {
        let value = text(&h["created"]);
        let Ok(time) = DateTime::parse_from_rfc3339(&value)
            .or_else(|_| DateTime::parse_from_str(&value, "%Y-%m-%dT%H:%M:%S%.f%z"))
        else {
            continue;
        };
        if let Some(items) = h["items"].as_array() {
            for item in items {
                if text(&item["field"]).eq_ignore_ascii_case("status")
                    || text(&item["fieldId"]).eq_ignore_ascii_case("status")
                {
                    changes.push((time, text(&item["fromString"]), text(&item["toString"])));
                }
            }
        }
    }
    changes.sort_by_key(|v| v.0);
    let mut periods: Vec<IssueWorkPeriod> = Vec::new();
    let mut start = String::new();
    let mut end = String::new();
    let mut current = None;
    for (time, from, to) in changes {
        let at = time
            .with_timezone(&Utc)
            .to_rfc3339_opts(chrono::SecondsFormat::Secs, true);
        if start.is_empty() && todo(&from) && active(&to) {
            start = at.clone();
        }
        if active(&to) && !active(&from) {
            if current.is_none() {
                periods.push(IssueWorkPeriod {
                    started_at: at,
                    ..Default::default()
                });
                current = Some(periods.len() - 1);
            }
        } else if current.is_some() && done(&to) && !done(&from) {
            periods[current.take().unwrap()].completed_at = at.clone();
            if end.is_empty() {
                end = at;
            }
        } else if current.is_none() && todo(&to) && !todo(&from) {
            if let Some(last) = periods.last_mut() {
                if !last.completed_at.is_empty() && last.reopened_at.is_empty() {
                    last.reopened_at = at;
                    last.reopened_status = to;
                }
            }
        }
    }
    (periods, start, end)
}

pub fn adf(node: &Value) -> String {
    render_adf(node, 0)
}
fn render_adf(node: &Value, depth: usize) -> String {
    if depth > 64 {
        return String::new();
    }
    if let Some(value) = node.as_str() {
        return value.into();
    }
    let kind = node["type"].as_str().unwrap_or("");
    let children = node["content"].as_array().map(Vec::as_slice).unwrap_or(&[]);
    let content = children
        .iter()
        .map(|v| render_adf(v, depth + 1))
        .collect::<String>();
    match kind {
        "paragraph" => format!("{}\n\n", content.trim()),
        "heading" => format!(
            "{} {}\n\n",
            "#".repeat(node["attrs"]["level"].as_u64().unwrap_or(2).clamp(1, 6) as usize),
            content.trim()
        ),
        "orderedList" | "bulletList" => {
            children
                .iter()
                .enumerate()
                .map(|(i, v)| {
                    format!(
                        "{} {}\n",
                        if kind == "orderedList" {
                            format!("{}.", i + 1)
                        } else {
                            "-".into()
                        },
                        render_adf(v, depth + 1).trim().replace('\n', "\n  ")
                    )
                })
                .collect::<String>()
                + "\n"
        }
        "blockquote" => {
            content
                .trim()
                .lines()
                .map(|l| format!("> {l}\n"))
                .collect::<String>()
                + "\n"
        }
        "codeBlock" => format!(
            "```{}\n{}\n```\n\n",
            text(&node["attrs"]["language"]),
            content.trim()
        ),
        "rule" => "---\n\n".into(),
        "hardBreak" => "\n".into(),
        "text" => {
            let mut result = text(&node["text"]);
            if let Some(marks) = node["marks"].as_array() {
                for mark in marks {
                    result = match mark["type"].as_str().unwrap_or("") {
                        "strong" => format!("**{result}**"),
                        "em" => format!("*{result}*"),
                        "code" => format!("`{result}`"),
                        "strike" => format!("~~{result}~~"),
                        "link" => format!("[{result}]({})", text(&mark["attrs"]["href"])),
                        _ => result,
                    };
                }
            }
            result
        }
        "mention" => text(&node["attrs"]["text"]),
        "inlineCard" | "blockCard" => text(&node["attrs"]["url"]),
        _ => content,
    }
}
