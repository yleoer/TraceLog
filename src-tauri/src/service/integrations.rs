use super::{
    local, markdown,
    remote::{self, text},
    views, Service,
};
use crate::{
    error::{invalid, AppError, Result},
    models::*,
    settings,
};

impl Service {
    pub async fn update_settings(&self, input: AppSettings) -> Result<AppSettings> {
        let _write = self.settings_write.lock().await;
        let existing = self.load_settings()?;
        let mut next = self
            .settings
            .lock()
            .map_err(|_| AppError::State)?
            .merge_input(input)?;
        if !matches!(next.ai.provider.as_str(), "openai" | "deepseek") {
            return Err(invalid("AI 服务必须是 openai 或 deepseek"));
        }
        if !next.tempo.api_token.is_empty()
            && (next.tempo.api_token != existing.tempo.api_token
                || next.jira.email != existing.jira.email
                || next.jira.api_token != existing.jira.api_token
                || next.jira.base_url != existing.jira.base_url)
        {
            next.tempo.author_account_id = self.jira_author(&next).await?;
        }
        self.settings
            .lock()
            .map_err(|_| AppError::State)?
            .save(&next)?;
        Ok(settings::public(next))
    }

    pub async fn tempo_settings(&self) -> Result<AppSettings> {
        let _write = self.settings_write.lock().await;
        let mut settings = self.load_settings()?;
        if settings.tempo.author_account_id.is_empty() {
            settings.tempo.author_account_id = self.jira_author(&settings).await?;
            self.settings
                .lock()
                .map_err(|_| AppError::State)?
                .save(&settings)?;
        }
        Ok(settings)
    }

    async fn chat(&self, settings: &AppSettings, prompt: &str, content: String) -> Result<String> {
        let provider = match settings.ai.provider.as_str() {
            "openai" => &settings.openai,
            "deepseek" => &settings.deepseek,
            _ => return Err(invalid("AI 服务必须是 openai 或 deepseek")),
        };
        if provider.api_key.is_empty() {
            return Err(invalid("请先配置 AI API Key"));
        }
        let value=remote::json(self.http.post(remote::endpoint(&provider.base_url,"/chat/completions")?)
            .bearer_auth(&provider.api_key).json(&serde_json::json!({"model":provider.model,"messages":[{"role":"system","content":prompt.trim()},{"role":"user","content":content.chars().take(60000).collect::<String>()}],"temperature":0.3})),"AI").await?;
        let summary = text(&value["choices"][0]["message"]["content"])
            .trim()
            .to_string();
        if summary.is_empty() {
            return Err(AppError::Remote("AI 返回了空内容".into()));
        }
        Ok(summary)
    }

    pub async fn issue_summary(&self, key: &str) -> Result<IssueSummaryResponse> {
        let settings = self.load_settings()?;
        let context = self.with_db(|db| {
            Ok(markdown::issue(
                &local::get_issue(db, key)?,
                &local::list_issue_events(db, key)?,
                &local::list_todos(db, key, true)?,
            ))
        })?;
        let summary = self
            .chat(&settings, &settings.prompts.issue_summary, context)
            .await?;
        let issue = self.with_db(|db| {
            let mut current = local::get_issue(db, key)?;
            current.summary_md = summary.clone();
            local::save_issue(db, Some(key), current, false)
        })?;
        Ok(IssueSummaryResponse { summary, issue })
    }

    pub async fn week_summary(&self, week: &str) -> Result<WeeklyLog> {
        let settings = self.load_settings()?;
        let view = self.with_db(|db| views::week(db, week, self.config.timezone))?;
        let summary = self
            .chat(
                &settings,
                &settings.prompts.weekly_summary,
                serde_json::to_string(&view)?,
            )
            .await?;
        self.with_db(|db| {
            let mut current = views::week(db, &view.log.week, self.config.timezone)?.log;
            current.summary_md = summary;
            views::save_week(db, &view.log.week, current, self.config.timezone)
        })
    }
}
