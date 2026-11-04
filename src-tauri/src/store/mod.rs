use crate::error::{AppError, Result};
use rusqlite::{types::ValueRef, Connection, Params};
use serde::de::DeserializeOwned;
use serde_json::{Map, Value};

pub mod index;

pub fn query<T: DeserializeOwned>(
    db: &Connection,
    sql: &str,
    params: impl Params,
) -> Result<Vec<T>> {
    let mut statement = db.prepare(sql)?;
    let names: Vec<String> = statement
        .column_names()
        .iter()
        .map(|v| v.to_string())
        .collect();
    let mut rows = statement.query(params)?;
    let mut result = Vec::new();
    while let Some(row) = rows.next()? {
        let mut object = Map::new();
        for (index, name) in names.iter().enumerate() {
            let value = match row.get_ref(index)? {
                ValueRef::Null => Value::Null,
                ValueRef::Integer(n) if matches!(name.as_str(), "done" | "converted_to_jira") => {
                    Value::Bool(n != 0)
                }
                ValueRef::Integer(n) => Value::from(n),
                ValueRef::Real(n) => Value::from(n),
                ValueRef::Text(text) => {
                    let text = String::from_utf8_lossy(text).into_owned();
                    if matches!(
                        name.as_str(),
                        "tags" | "jira_tags" | "custom_tags" | "links_json"
                    ) {
                        let value: Value = serde_json::from_str(&text)?;
                        if value.is_null() {
                            Value::Array(Vec::new())
                        } else {
                            value
                        }
                    } else {
                        Value::String(text)
                    }
                }
                ValueRef::Blob(_) => return Err(crate::error::invalid("不支持的数据库二进制字段")),
            };
            object.insert(
                if name == "links_json" {
                    "links".into()
                } else {
                    name.clone()
                },
                value,
            );
        }
        result.push(serde_json::from_value(Value::Object(object))?);
    }
    Ok(result)
}

pub fn one<T: DeserializeOwned>(db: &Connection, sql: &str, params: impl Params) -> Result<T> {
    query(db, sql, params)?
        .into_iter()
        .next()
        .ok_or(AppError::NotFound)
}

pub fn clean_tags(tags: &[String]) -> Vec<String> {
    let mut result = Vec::new();
    for value in tags {
        let value = value.trim().to_string();
        if !value.is_empty() && !result.contains(&value) {
            result.push(value);
        }
    }
    result
}

pub fn fts_query(query: &str) -> String {
    query
        .split_whitespace()
        .map(|part| format!("\"{}\"", part.replace('"', "\"\"")))
        .collect::<Vec<_>>()
        .join(" ")
}
