use crate::{
    error::{invalid, AppError, Result},
    models::*,
};
use base64::{engine::general_purpose::STANDARD, Engine};
use rusqlite::Connection;
use std::{
    io::Write,
    path::{Path, PathBuf},
};

const MAX_SIZE: usize = 8 * 1024 * 1024;
#[derive(serde::Deserialize)]
pub struct UploadFile {
    pub name: String,
    pub data: String,
    #[serde(default)]
    pub context: String,
}

fn kind(data: &[u8]) -> Result<(&'static str, &'static str)> {
    if data.starts_with(b"\x89PNG\r\n\x1a\n") {
        Ok(("image/png", "png"))
    } else if data.starts_with(&[0xff, 0xd8, 0xff]) {
        Ok(("image/jpeg", "jpg"))
    } else if data.starts_with(b"GIF87a") || data.starts_with(b"GIF89a") {
        Ok(("image/gif", "gif"))
    } else if data.len() >= 12 && data.starts_with(b"RIFF") && &data[8..12] == b"WEBP" {
        Ok(("image/webp", "webp"))
    } else {
        Err(invalid("仅支持 PNG、JPEG、GIF 和 WebP 图片"))
    }
}
fn slug(value: &str) -> String {
    value
        .to_ascii_lowercase()
        .chars()
        .map(|c| if c.is_ascii_alphanumeric() { c } else { '-' })
        .collect::<String>()
        .split('-')
        .filter(|s| !s.is_empty())
        .collect::<Vec<_>>()
        .join("-")
        .chars()
        .take(60)
        .collect()
}
fn safe_path(root: &Path, url: &str) -> Result<PathBuf> {
    let name = url
        .strip_prefix("/uploads/")
        .ok_or_else(|| invalid("图片地址无效"))?;
    if name.is_empty()
        || name.contains("..")
        || !name
            .bytes()
            .all(|b| b.is_ascii_alphanumeric() || b"._-".contains(&b))
        || !matches!(
            Path::new(name)
                .extension()
                .and_then(|s| s.to_str())
                .unwrap_or("")
                .to_ascii_lowercase()
                .as_str(),
            "png" | "jpg" | "jpeg" | "gif" | "webp"
        )
    {
        return Err(invalid("图片地址无效"));
    }
    let path = root.join(name);
    if path.exists()
        && (path.symlink_metadata()?.file_type().is_symlink()
            || path.canonicalize()?.parent() != Some(root.canonicalize()?.as_path()))
    {
        return Err(invalid("图片路径无效"));
    }
    Ok(path)
}
pub fn save(root: &Path, file: UploadFile) -> Result<UploadedImage> {
    let encoded = file
        .data
        .split_once(',')
        .filter(|(header, _)| header.starts_with("data:") && header.ends_with(";base64"))
        .map(|(_, v)| v)
        .ok_or_else(|| invalid("图片编码无效"))?;
    if encoded.len() > MAX_SIZE.div_ceil(3) * 4 {
        return Err(invalid("图片不能超过 8 MiB"));
    }
    let data = STANDARD
        .decode(encoded)
        .map_err(|_| invalid("图片编码无效"))?;
    if data.len() > MAX_SIZE {
        return Err(invalid("图片不能超过 8 MiB"));
    }
    let (content_type, ext) = kind(&data)?;
    std::fs::create_dir_all(root)?;
    let context = slug(&file.context);
    let context = if context.is_empty() {
        "upload"
    } else {
        &context
    };
    let base = slug(&file.name);
    let prefix = format!(
        "{context}-{}-{base}-",
        chrono::Utc::now().format("%Y%m%dT%H%M%S")
    );
    let mut output = tempfile::Builder::new()
        .prefix(&prefix)
        .suffix(&format!(".{ext}"))
        .tempfile_in(root)?;
    output.write_all(&data)?;
    output.as_file().sync_all()?;
    let (_, path) = output.keep().map_err(|e| e.error)?;
    let filename = path.file_name().unwrap().to_string_lossy().into_owned();
    Ok(UploadedImage {
        url: format!("/uploads/{filename}"),
        filename,
        content_type: content_type.into(),
        size: data.len() as i64,
    })
}
pub fn data_url(root: &Path, url: &str) -> Result<UploadedImageData> {
    let path = safe_path(root, url)?;
    let metadata = path.metadata().map_err(|e| {
        if e.kind() == std::io::ErrorKind::NotFound {
            AppError::NotFound
        } else {
            e.into()
        }
    })?;
    if !metadata.is_file() || metadata.len() > MAX_SIZE as u64 {
        return Err(invalid("图片文件无效或超过 8 MiB"));
    }
    let data = std::fs::read(path)?;
    let (kind, _) = kind(&data)?;
    Ok(UploadedImageData {
        url: url.into(),
        data_url: format!("data:{kind};base64,{}", STANDARD.encode(data)),
    })
}
fn referenced(db: &Connection, url: &str) -> Result<bool> {
    Ok(db.query_row("SELECT EXISTS(
        SELECT 1 FROM issues WHERE instr(summary_md,?1) OR instr(background_md,?1) OR instr(analysis_md,?1) OR instr(solution_md,?1) OR instr(actions_md,?1) OR instr(result_md,?1) OR instr(todo_md,?1) OR instr(links_json,?1)
        UNION ALL SELECT 1 FROM issue_events WHERE instr(content_md,?1)
        UNION ALL SELECT 1 FROM issue_todos WHERE instr(content,?1)
        UNION ALL SELECT 1 FROM temp_tasks WHERE instr(content_md,?1)
        UNION ALL SELECT 1 FROM temp_task_events WHERE instr(content_md,?1)
        UNION ALL SELECT 1 FROM weekly_logs WHERE instr(summary_md,?1) OR instr(next_plan_md,?1)
        UNION ALL SELECT 1 FROM day_entries WHERE instr(content_md,?1))",[url],|r|r.get(0))?)
}
pub fn delete(db: &Connection, root: &Path, url: &str) -> Result<bool> {
    let path = safe_path(root, url)?;
    if referenced(db, url)? {
        return Ok(false);
    }
    match std::fs::remove_file(path) {
        Ok(()) => Ok(true),
        Err(e) if e.kind() == std::io::ErrorKind::NotFound => Ok(true),
        Err(e) => Err(e.into()),
    }
}
pub fn cleanup(db: &Connection, root: &Path) -> Result<UploadedImageCleanup> {
    let mut result = UploadedImageCleanup::default();
    if !root.exists() {
        return Ok(result);
    }
    for entry in std::fs::read_dir(root)? {
        let entry = entry?;
        if !entry.file_type()?.is_file() {
            continue;
        }
        let url = format!("/uploads/{}", entry.file_name().to_string_lossy());
        if safe_path(root, &url).is_err() {
            continue;
        }
        result.scanned += 1;
        let bytes = entry.metadata()?.len() as i64;
        match delete(db, root, &url) {
            Ok(true) => {
                result.deleted += 1;
                result.freed_bytes += bytes;
            }
            Ok(false) => result.kept += 1,
            Err(_) => result.failed += 1,
        }
    }
    Ok(result)
}
