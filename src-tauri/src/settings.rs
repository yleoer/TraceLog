use crate::{config::env, error::Result, models::*};
use std::{
    io::Write,
    path::{Path, PathBuf},
};

pub const ISSUE_PROMPT: &str = "你是一个资深工程师助手。请根据 Jira 信息和阶段评论，总结该 issue 当前的工作简述。输出中文，控制在 2-4 句话，直接给结论，不要写标题、列表或寒暄。";
pub const WEEK_PROMPT: &str = "你是一个资深工程师助手。请根据本周 Jira、评论、临时需求和 TODO，生成中文周总结。结构清晰，包含本周完成、关键进展、风险阻塞、下周建议。直接输出 Markdown，不要寒暄。";

pub struct SettingsStore {
    path: PathBuf,
    defaults: AppSettings,
}

impl SettingsStore {
    pub fn new(data_dir: &Path) -> Self {
        let mut defaults = AppSettings::default();
        defaults.jira.base_url = env("JIRA_BASE_URL", "");
        defaults.jira.email = env("JIRA_EMAIL", "");
        defaults.jira.api_token = env("JIRA_API_TOKEN", "");
        defaults.openai.base_url = env("OPENAI_BASE_URL", "https://api.openai.com/v1");
        defaults.openai.model = env("OPENAI_MODEL", "gpt-4.1-mini");
        defaults.openai.api_key = env("OPENAI_API_KEY", "");
        defaults.deepseek.base_url = env("DEEPSEEK_BASE_URL", "https://api.deepseek.com");
        defaults.deepseek.model = env("DEEPSEEK_MODEL", "deepseek-v4-flash");
        defaults.deepseek.api_key = env("DEEPSEEK_API_KEY", "");
        Self {
            path: data_dir.join("tracelog-settings.json"),
            defaults: normalize(defaults),
        }
    }

    pub fn load(&self) -> Result<AppSettings> {
        match std::fs::read(&self.path) {
            Ok(bytes) => {
                let mut value = serde_json::to_value(&self.defaults)?;
                merge_json(&mut value, serde_json::from_slice(&bytes)?);
                Ok(normalize(serde_json::from_value(value)?))
            }
            Err(e) if e.kind() == std::io::ErrorKind::NotFound => Ok(self.defaults.clone()),
            Err(e) => Err(e.into()),
        }
    }

    pub fn merge_input(&self, input: AppSettings) -> Result<AppSettings> {
        let existing = self.load()?;
        let mut input = normalize(input);
        if input.jira.api_token.is_empty() {
            input.jira.api_token = existing.jira.api_token;
        }
        if input.tempo.api_token.is_empty() {
            input.tempo.api_token = existing.tempo.api_token;
        }
        if input.tempo.author_account_id.is_empty() {
            input.tempo.author_account_id = existing.tempo.author_account_id;
        }
        if input.openai.api_key.is_empty() {
            input.openai.api_key = existing.openai.api_key;
        }
        if input.deepseek.api_key.is_empty() {
            input.deepseek.api_key = existing.deepseek.api_key;
        }
        Ok(input)
    }

    pub fn save(&self, input: &AppSettings) -> Result<()> {
        let parent = self.path.parent().expect("设置文件目录");
        std::fs::create_dir_all(parent)?;
        let mut file = tempfile::NamedTempFile::new_in(parent)?;
        file.write_all(&serde_json::to_vec_pretty(input)?)?;
        file.as_file().sync_all()?;
        file.persist(&self.path).map_err(|e| e.error)?;
        Ok(())
    }

    pub fn merge_changes(&self, before: &AppSettings, next: &AppSettings) -> Result<AppSettings> {
        let mut latest = serde_json::to_value(self.load()?)?;
        apply_changes(
            &mut latest,
            &serde_json::to_value(before)?,
            &serde_json::to_value(next)?,
        );
        Ok(serde_json::from_value(latest)?)
    }
}

fn apply_changes(
    latest: &mut serde_json::Value,
    before: &serde_json::Value,
    next: &serde_json::Value,
) {
    if let (Some(latest), Some(before), Some(next)) =
        (latest.as_object_mut(), before.as_object(), next.as_object())
    {
        for (key, value) in next {
            if let (Some(latest), Some(before)) = (latest.get_mut(key), before.get(key)) {
                apply_changes(latest, before, value);
            }
        }
    } else if before != next {
        *latest = next.clone();
    }
}

pub fn public(mut settings: AppSettings) -> AppSettings {
    settings.jira.has_api_token = !settings.jira.api_token.is_empty();
    settings.jira.api_token.clear();
    settings.tempo.has_api_token = !settings.tempo.api_token.is_empty();
    settings.tempo.api_token.clear();
    settings.openai.has_api_key = !settings.openai.api_key.is_empty();
    settings.openai.api_key.clear();
    settings.deepseek.has_api_key = !settings.deepseek.api_key.is_empty();
    settings.deepseek.api_key.clear();
    settings
}

fn merge_json(base: &mut serde_json::Value, next: serde_json::Value) {
    if let (Some(base), Some(next)) = (base.as_object_mut(), next.as_object()) {
        for (key, value) in next {
            if let Some(existing) = base.get_mut(key) {
                merge_json(existing, value.clone());
            } else {
                base.insert(key.clone(), value.clone());
            }
        }
    } else {
        *base = next;
    }
}

fn normalize(mut settings: AppSettings) -> AppSettings {
    settings.jira.base_url = settings.jira.base_url.trim().trim_end_matches('/').into();
    settings.jira.email = settings.jira.email.trim().into();
    settings.jira.api_token = settings.jira.api_token.trim().into();
    settings.tempo.base_url = settings
        .tempo
        .base_url
        .trim()
        .trim_end_matches('/')
        .trim_end_matches("/4")
        .into();
    if settings.tempo.base_url.is_empty() {
        settings.tempo.base_url = "https://api.tempo.io".into();
    }
    settings.tempo.api_token = settings.tempo.api_token.trim().into();
    settings.tempo.author_account_id = settings.tempo.author_account_id.trim().into();
    settings.ai.provider = settings.ai.provider.trim().to_lowercase();
    if settings.ai.provider.is_empty() {
        settings.ai.provider = "openai".into();
    }
    for (provider, base, model) in [
        (
            &mut settings.openai,
            "https://api.openai.com/v1",
            "gpt-4.1-mini",
        ),
        (
            &mut settings.deepseek,
            "https://api.deepseek.com",
            "deepseek-v4-flash",
        ),
    ] {
        provider.base_url = provider.base_url.trim().trim_end_matches('/').into();
        provider.api_key = provider.api_key.trim().into();
        provider.model = provider.model.trim().into();
        if provider.base_url.is_empty() {
            provider.base_url = base.into();
        }
        if provider.model.is_empty() {
            provider.model = model.into();
        }
    }
    if settings.prompts.issue_summary.trim().is_empty() {
        settings.prompts.issue_summary = ISSUE_PROMPT.into();
    }
    if settings.prompts.weekly_summary.trim().is_empty() {
        settings.prompts.weekly_summary = WEEK_PROMPT.into();
    }
    settings
}
