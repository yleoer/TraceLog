use super::{
    now,
    remote::{self, integer, text},
    views, Service,
};
use crate::{
    error::{invalid, AppError, Result},
    models::*,
    store,
};
use chrono::{Datelike, Duration, NaiveDate, NaiveTime, Timelike};
use rusqlite::{params, Connection};
use serde_json::Value;

pub fn work_items() -> Vec<TimeWorkItem> {
    [
        ("CORETIME-47", "法定节假日"),
        ("CORETIME-11", "个人请假"),
        ("CORETIME-80", "内部SMF需求"),
        ("CORETIME-36", "SE的SMF需求"),
        ("CORETIME-16", "面对客户的文档工作"),
    ]
    .into_iter()
    .map(|(k, l)| TimeWorkItem {
        key: k.into(),
        label: l.into(),
    })
    .collect()
}
fn label(key: &str) -> String {
    work_items()
        .into_iter()
        .find(|v| v.key == key)
        .map(|v| v.label)
        .unwrap_or_default()
}
fn clock(value: &str, seconds: i64) -> String {
    let time = NaiveTime::parse_from_str(value, "%H:%M:%S")
        .or_else(|_| NaiveTime::parse_from_str(value, "%H:%M"))
        .unwrap_or(NaiveTime::from_hms_opt(8, 0, 0).unwrap());
    let normalized =
        (i64::from(time.num_seconds_from_midnight()) + seconds).rem_euclid(86400) as u32;
    NaiveTime::from_num_seconds_from_midnight_opt(normalized, 0)
        .unwrap()
        .format("%H:%M:%S")
        .to_string()
}
fn from_remote(v: &Value) -> TimeWorklog {
    let key = text(&v["issue"]["key"]).to_uppercase();
    let seconds = integer(&v["timeSpentSeconds"]).max(0);
    let start = clock(&text(&v["startTime"]), 0);
    TimeWorklog {
        tempo_worklog_id: integer(&v["tempoWorklogId"]),
        work_item_label: label(&key),
        work_item_key: key,
        description: text(&v["description"]),
        start_date: text(&v["startDate"]),
        end_time: clock(&start, seconds),
        start_time: start,
        time_spent_seconds: seconds,
        hours: seconds as f64 / 3600.0,
        self_url: text(&v["self"]),
    }
}
pub fn cache_worklog(db: &Connection, account: &str, w: &TimeWorklog) -> Result<()> {
    db.execute("INSERT INTO tempo_time_worklogs(account_id,tempo_worklog_id,work_item_key,description,start_date,start_time,end_time,time_spent_seconds,self,cached_at) VALUES (?1,?2,?3,?4,?5,?6,?7,?8,?9,?10) ON CONFLICT(account_id,tempo_worklog_id) DO UPDATE SET work_item_key=excluded.work_item_key,description=excluded.description,start_date=excluded.start_date,start_time=excluded.start_time,end_time=excluded.end_time,time_spent_seconds=excluded.time_spent_seconds,self=excluded.self,cached_at=excluded.cached_at",params![account,w.tempo_worklog_id,w.work_item_key,w.description,w.start_date,w.start_time,w.end_time,w.time_spent_seconds,w.self_url,now()])?;
    Ok(())
}
fn cached(
    db: &Connection,
    account: &str,
    start: &str,
    end: &str,
) -> Result<Option<Vec<TimeWorklog>>> {
    let exists:bool=db.query_row("SELECT EXISTS(SELECT 1 FROM tempo_time_cache_ranges WHERE account_id=?1 AND start_date<=?2 AND end_date>=?3)",params![account,start,end],|r|r.get(0))?;
    if !exists {
        return Ok(None);
    }
    let mut result:Vec<TimeWorklog>=store::query(db,"SELECT tempo_worklog_id,work_item_key,description,start_date,start_time,end_time,time_spent_seconds,self FROM tempo_time_worklogs WHERE account_id=?1 AND start_date BETWEEN ?2 AND ?3 ORDER BY start_date,start_time",params![account,start,end])?;
    for w in &mut result {
        w.hours = w.time_spent_seconds as f64 / 3600.0;
        w.work_item_label = label(&w.work_item_key);
    }
    Ok(Some(result))
}

impl Service {
    async fn tempo_request(
        &self,
        settings: &AppSettings,
        path: &str,
        body: Option<&Value>,
        query: &[(&str, String)],
    ) -> Result<Value> {
        if settings.tempo.api_token.is_empty() {
            return Err(invalid("请先配置 Tempo API Token"));
        }
        let endpoint = remote::endpoint(&settings.tempo.base_url, path)?;
        let request = if let Some(body) = body {
            self.http.post(endpoint).json(body)
        } else {
            self.http.get(endpoint)
        };
        remote::json(
            request.bearer_auth(&settings.tempo.api_token).query(query),
            "Tempo",
        )
        .await
    }
    async fn time_range(
        &self,
        settings: &AppSettings,
        start: &str,
        end: &str,
        force: bool,
    ) -> Result<Vec<TimeWorklog>> {
        let account = &settings.tempo.author_account_id;
        if !force {
            if let Some(worklogs) = self.with_db(|db| cached(db, account, start, end))? {
                return Ok(worklogs);
            }
        }
        let encoded: String = url::form_urlencoded::byte_serialize(account.as_bytes()).collect();
        let mut worklogs = Vec::new();
        let mut offset = 0;
        loop {
            let page = self
                .tempo_request(
                    settings,
                    &format!("/4/worklogs/user/{encoded}"),
                    None,
                    &[
                        ("from", start.into()),
                        ("to", end.into()),
                        ("limit", "100".into()),
                        ("offset", offset.to_string()),
                    ],
                )
                .await?;
            let values = page["results"]
                .as_array()
                .ok_or_else(|| AppError::Remote("Tempo 分页格式无效".into()))?;
            worklogs.extend(
                values
                    .iter()
                    .map(from_remote)
                    .filter(|w| w.start_date.as_str() >= start && w.start_date.as_str() <= end),
            );
            if text(&page["metadata"]["next"]).is_empty() || values.is_empty() {
                break;
            }
            // 根据已返回条数推进，不访问远端给出的 URL，以防转发凭据。
            offset += values.len();
        }
        self.with_db(|db|{
            let tx=db.unchecked_transaction()?;
            tx.execute("DELETE FROM tempo_time_worklogs WHERE account_id=?1 AND start_date BETWEEN ?2 AND ?3",params![account,start,end])?;
            for w in &worklogs{cache_worklog(&tx,account,w)?;}
            tx.execute("INSERT OR REPLACE INTO tempo_time_cache_ranges(account_id,start_date,end_date,refreshed_at) VALUES (?1,?2,?3,?4)",params![account,start,end,now()])?;
            tx.commit()?;Ok(())
        })?;
        Ok(worklogs)
    }
    pub async fn time_week(&self, week: &str, force: bool) -> Result<TimeWeekView> {
        let start = views::week_start(week, self.config.timezone)?;
        let week = start.format("%G-W%V").to_string();
        let end = start + Duration::days(6);
        let settings = self.tempo_settings().await?;
        let worklogs = self
            .time_range(&settings, &start.to_string(), &end.to_string(), force)
            .await?;
        let mut view = TimeWeekView {
            week,
            start_date: start.to_string(),
            end_date: end.to_string(),
            work_items: work_items(),
            ..Default::default()
        };
        for i in 0..7 {
            let date = start + Duration::days(i);
            let logs: Vec<_> = worklogs
                .iter()
                .filter(|w| w.start_date == date.to_string())
                .cloned()
                .collect();
            let hours = logs.iter().map(|w| w.hours).sum();
            view.days.push(TimeDay {
                date: date.to_string(),
                weekday: ["周一", "周二", "周三", "周四", "周五", "周六", "周日"]
                    [date.weekday().num_days_from_monday() as usize]
                    .into(),
                worklogs: logs,
                total_hours: hours,
            });
        }
        view.total_hours = worklogs.iter().map(|w| w.hours).sum();
        view.worklogs = worklogs;
        view.worklogs
            .sort_by(|a, b| (&a.start_date, &a.start_time).cmp(&(&b.start_date, &b.start_time)));
        Ok(view)
    }
    pub async fn log_time(&self, mut request: LogTimeRequest) -> Result<LogTimeResult> {
        request.work_item_key = request.work_item_key.trim().to_uppercase();
        request.description = request.description.trim().into();
        if request.end_date.trim().is_empty() {
            request.end_date = request.start_date.clone();
        }
        if !work_items().iter().any(|w| w.key == request.work_item_key) {
            return Err(invalid("工时事项不在允许列表中"));
        }
        if request.description.is_empty()
            || !request.hours.is_finite()
            || request.hours.fract() != 0.0
            || !(1.0..=8.0).contains(&request.hours)
        {
            return Err(invalid("请填写工作内容，小时数必须是 1 到 8 的整数"));
        }
        let parse = |s: &str| {
            NaiveDate::parse_from_str(s, "%Y-%m-%d")
                .ok()
                .filter(|d| d.to_string() == s)
                .ok_or_else(|| invalid("日期必须为 YYYY-MM-DD"))
        };
        let start = parse(&request.start_date)?;
        let end = parse(&request.end_date)?;
        let count = (end - start).num_days() + 1;
        if !(1..=31).contains(&count) {
            return Err(invalid("工时日期范围必须为 1 到 31 天"));
        }
        let settings = self.tempo_settings().await?;
        let issue = self
            .jira_get(
                &settings,
                &format!("/rest/api/3/issue/{}", request.work_item_key),
                &[("fields", "id".into())],
            )
            .await?;
        let issue_id = integer(&issue["id"]);
        if issue_id <= 0 {
            return Err(AppError::Remote("Jira 未返回有效 Issue ID".into()));
        }
        let existing = self
            .time_range(&settings, &request.start_date, &request.end_date, false)
            .await?;
        let mut result = LogTimeResult {
            work_item_key: request.work_item_key.clone(),
            description: request.description.clone(),
            hours: request.hours,
            start_date: request.start_date.clone(),
            end_date: request.end_date.clone(),
            total: count,
            ..Default::default()
        };
        for i in 0..count {
            let date = (start + Duration::days(i)).to_string();
            let occupied: i64 = existing
                .iter()
                .filter(|w| w.start_date == date)
                .map(|w| w.time_spent_seconds.max(0))
                .sum();
            let start_time = clock("08:00:00", occupied);
            let seconds = (request.hours * 3600.0) as i64;
            let mut entry = LogTimeEntry {
                date: date.clone(),
                start_time: start_time.clone(),
                end_time: clock(&start_time, seconds),
                ..Default::default()
            };
            let body = serde_json::json!({"authorAccountId":settings.tempo.author_account_id,"issueId":issue_id,"timeSpentSeconds":seconds,"billableSeconds":seconds,"startDate":date,"startTime":start_time,"description":request.description});
            match self
                .tempo_request(&settings, "/4/worklogs", Some(&body), &[])
                .await
            {
                Err(e) => {
                    entry.error = e.to_string();
                    result.failed += 1;
                }
                Ok(value) => {
                    entry.tempo_worklog_id = integer(&value["tempoWorklogId"]);
                    entry.self_url = text(&value["self"]);
                    // 远端已成功时不能报告为可重试失败，避免用户再次提交重复工时。
                    let cache = self.with_db(|db| {
                        if entry.tempo_worklog_id <= 0 {
                            return Err(invalid("Tempo 返回了无效工时 ID"));
                        }
                        cache_worklog(
                            db,
                            &settings.tempo.author_account_id,
                            &TimeWorklog {
                                tempo_worklog_id: entry.tempo_worklog_id,
                                work_item_key: request.work_item_key.clone(),
                                description: request.description.clone(),
                                start_date: date,
                                start_time: start_time.clone(),
                                end_time: entry.end_time.clone(),
                                time_spent_seconds: seconds,
                                self_url: entry.self_url.clone(),
                                ..Default::default()
                            },
                        )
                    });
                    if let Err(e) = cache {
                        crate::logging::record(
                            &self.config.data_dir,
                            &format!("工时已提交但缓存失败：{e}"),
                        );
                        let _ = self.with_db(|db| {
                            db.execute(
                                "DELETE FROM tempo_time_cache_ranges WHERE account_id=?1",
                                [&settings.tempo.author_account_id],
                            )?;
                            Ok(())
                        });
                    }
                    result.successful += 1;
                }
            }
            result.entries.push(entry);
        }
        Ok(result)
    }
}
