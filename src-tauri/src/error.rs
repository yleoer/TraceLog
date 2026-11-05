use serde::{Serialize, Serializer};

pub type Result<T> = std::result::Result<T, AppError>;

#[derive(Debug, thiserror::Error)]
pub enum AppError {
    #[error("{0}")]
    Validation(String),
    #[error("记录不存在")]
    NotFound,
    #[error("{0}")]
    Remote(String),
    #[error("{0}")]
    Database(#[from] rusqlite::Error),
    #[error("{0}")]
    Io(#[from] std::io::Error),
    #[error("{0}")]
    Json(#[from] serde_json::Error),
    #[error("应用状态不可用")]
    State,
}

impl Serialize for AppError {
    fn serialize<S: Serializer>(&self, serializer: S) -> std::result::Result<S::Ok, S::Error> {
        let (code, message) = match self {
            Self::Validation(message) => (400, message.clone()),
            Self::NotFound => (404, self.to_string()),
            Self::Remote(message) => (502, message.clone()),
            Self::Database(rusqlite::Error::SqliteFailure(error, _))
                if error.code == rusqlite::ErrorCode::ConstraintViolation =>
            {
                (409, "记录冲突或关联数据无效".into())
            }
            _ => (500, "操作失败，请查看应用日志".into()),
        };
        serde_json::json!({ "code": code, "message": message }).serialize(serializer)
    }
}

pub fn invalid(message: impl Into<String>) -> AppError {
    AppError::Validation(message.into())
}
