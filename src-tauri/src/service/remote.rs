use crate::error::{invalid, AppError, Result};
use serde_json::Value;

pub fn endpoint(base: &str, path: &str) -> Result<url::Url> {
    let url = url::Url::parse(&format!("{}{path}", base.trim_end_matches('/')))
        .map_err(|_| invalid("服务地址无效"))?;
    if !matches!(url.scheme(), "https" | "http")
        || url.host_str().is_none()
        || !url.username().is_empty()
        || url.password().is_some()
    {
        return Err(invalid("服务地址必须使用 HTTP 或 HTTPS，且不能包含凭据"));
    }
    Ok(url)
}

// 不返回远端响应体或包含凭据的请求 URL，避免密钥出现在界面及日志中。
pub async fn json(request: reqwest::RequestBuilder, service: &str) -> Result<Value> {
    receive(request, service, false).await
}

// 可选元数据在非认证 HTTP 错误时缺省；连接、认证和响应格式错误仍需报告。
pub async fn optional_json(request: reqwest::RequestBuilder, service: &str) -> Result<Value> {
    receive(request, service, true).await
}

async fn receive(request: reqwest::RequestBuilder, service: &str, optional: bool) -> Result<Value> {
    let response = request.send().await.map_err(|e| {
        AppError::Remote(format!(
            "{service}{}",
            if e.is_timeout() {
                "请求超时"
            } else {
                "连接失败"
            }
        ))
    })?;
    let status = response.status();
    if !status.is_success() {
        if optional && !matches!(status.as_u16(), 401 | 403) {
            return Ok(Value::Null);
        }
        return Err(AppError::Remote(match status.as_u16() {
            401 | 403 => format!("{service}身份验证失败，请检查凭据与权限"),
            404 => format!("{service}记录不存在"),
            429 => format!("{service}请求过多，请稍后重试"),
            _ => format!("{service}返回 HTTP {}", status.as_u16()),
        }));
    }
    response
        .json()
        .await
        .map_err(|_| AppError::Remote(format!("{service}响应不是有效 JSON")))
}

pub fn text(value: &Value) -> String {
    value.as_str().map(str::to_owned).unwrap_or_default()
}

pub fn integer(value: &Value) -> i64 {
    value
        .as_i64()
        .or_else(|| value.as_str().and_then(|s| s.parse().ok()))
        .unwrap_or_default()
}
