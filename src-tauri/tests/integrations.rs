use base64::{engine::general_purpose::STANDARD, Engine};
use serde_json::{json, Value};
use std::{
    io::{Read, Write},
    net::TcpListener,
    sync::{
        atomic::{AtomicBool, Ordering},
        Arc,
    },
    thread,
    time::Duration,
};
use tracelog::{
    config::Config,
    models::*,
    service::{local, uploads, Service},
};

struct Server {
    url: String,
    stop: Arc<AtomicBool>,
    worker: Option<thread::JoinHandle<()>>,
}
impl Server {
    fn new(mut response: impl FnMut(&str, &Value) -> (u16, Value) + Send + 'static) -> Self {
        let listener = TcpListener::bind("127.0.0.1:0").unwrap();
        listener.set_nonblocking(true).unwrap();
        let url = format!("http://{}", listener.local_addr().unwrap());
        let stop = Arc::new(AtomicBool::new(false));
        let flag = stop.clone();
        let worker = thread::spawn(move || {
            while !flag.load(Ordering::Relaxed) {
                let Ok((mut socket, _)) = listener.accept() else {
                    thread::sleep(Duration::from_millis(5));
                    continue;
                };
                // Windows accept 会继承监听 socket 的非阻塞模式。
                socket.set_nonblocking(false).unwrap();
                socket
                    .set_read_timeout(Some(Duration::from_secs(3)))
                    .unwrap();
                let mut data = Vec::new();
                let mut buf = [0; 4096];
                let header_end = loop {
                    let n = socket.read(&mut buf).unwrap();
                    assert!(n > 0);
                    data.extend_from_slice(&buf[..n]);
                    if let Some(i) = data.windows(4).position(|w| w == b"\r\n\r\n") {
                        break i + 4;
                    }
                };
                let headers = String::from_utf8_lossy(&data[..header_end]).to_string();
                let length = headers
                    .lines()
                    .find_map(|l| {
                        l.split_once(':')
                            .filter(|(k, _)| k.eq_ignore_ascii_case("content-length"))
                            .map(|(_, v)| v.trim().parse::<usize>().unwrap())
                    })
                    .unwrap_or(0);
                while data.len() < header_end + length {
                    let n = socket.read(&mut buf).unwrap();
                    assert!(n > 0);
                    data.extend_from_slice(&buf[..n]);
                }
                let body = serde_json::from_slice(&data[header_end..header_end + length])
                    .unwrap_or(Value::Null);
                let (status, payload) = response(&headers, &body);
                let payload = payload.to_string();
                write!(socket,"HTTP/1.1 {status} Test\r\nContent-Type: application/json\r\nContent-Length: {}\r\nConnection: close\r\n\r\n{payload}",payload.len()).unwrap();
            }
        });
        Self {
            url,
            stop,
            worker: Some(worker),
        }
    }
}
impl Drop for Server {
    fn drop(&mut self) {
        self.stop.store(true, Ordering::Relaxed);
        let result = self.worker.take().unwrap().join();
        if !thread::panicking() {
            result.unwrap();
        }
    }
}
fn settings(service: &Service, url: &str) {
    let mut s = service.load_settings().unwrap();
    s.jira.base_url = url.into();
    s.jira.email = "test@example.com".into();
    s.jira.api_token = "test-jira-secret".into();
    s.tempo.base_url = url.into();
    s.tempo.api_token = "test-tempo-secret".into();
    s.tempo.author_account_id = "account".into();
    s.openai.base_url = url.into();
    s.openai.api_key = "test-ai-secret".into();
    service.settings.save(&s).unwrap();
}
fn change(at: &str, from: &str, to: &str) -> Value {
    json!({"created":at,"items":[{"field":"status","fromString":from,"toString":to}]})
}

#[tokio::test]
async fn jira_optional_fields_failure_uses_fix_versions() {
    let root = tempfile::tempdir().unwrap();
    let service = Service::new(Config::isolated(root.path().into())).unwrap();
    let server = Server::new(|headers, _| {
        if headers.contains("/field ") {
            return (503, json!({}));
        }
        if headers.contains("/changelog?") {
            return (200, json!({"values":[],"isLast":true}));
        }
        (
            200,
            json!({"fields":{"summary":"可导入的事项","fixVersions":[{"name":"fallback-release"}]}}),
        )
    });
    settings(&service, &server.url);
    let imported = service.import_jira("TL-1001").await.unwrap();
    assert!(imported.background_md.contains("fallback-release"));
}

#[tokio::test]
async fn jira_pages_adf_periods_preview_and_refresh_preserve_local_edits() {
    let root = tempfile::tempdir().unwrap();
    let service = Service::new(Config::isolated(root.path().into())).unwrap();
    let server = Server::new(move |headers, _| {
        assert!(headers.to_lowercase().contains("authorization: basic "));
        if headers.contains("/field ") {
            return (
                200,
                json!([{"id":"customfield_44","name":"Release Requested"}]),
            );
        }
        if headers.contains("/changelog?") {
            return if headers.contains("startAt=0") {
                (
                    200,
                    json!({"startAt":0,"total":4,"values":[change("2026-06-01T08:00:00.000+0800","To Do","In Progress"),change("2026-06-02T08:00:00+0800","In Progress","Done")]}),
                )
            } else {
                assert!(headers.contains("startAt=2"));
                (
                    200,
                    json!({"startAt":2,"total":4,"isLast":true,"values":[change("2026-06-03T00:00:00Z","Done","To Do"),change("2026-06-04T00:00:00Z","To Do","In Progress")]}),
                )
            };
        }
        assert!(headers.contains("customfield_44"));
        (
            200,
            json!({"key":"TL-1001","fields":{"summary":"远端标题","status":{"name":"In Progress"},"priority":{"name":"Highest"},"labels":["remote"],"customfield_44":[{"name":"R-2026"}],"description":{"type":"doc","content":[{"type":"heading","attrs":{"level":2},"content":[{"type":"text","text":"需求说明"}]},{"type":"orderedList","content":[{"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"第一步"}]}]}]}]}}}),
        )
    });
    settings(&service, &server.url);
    let preview = service.import_jira("tl-1001").await.unwrap();
    assert!(service
        .with_db(|db| local::get_issue(db, "TL-1001"))
        .is_err());
    assert_eq!(preview.work_periods.len(), 2);
    assert_eq!(preview.started_at, "2026-06-01T00:00:00Z");
    assert_eq!(preview.completed_at, "2026-06-02T00:00:00Z");
    assert_eq!(preview.work_periods[0].reopened_at, "2026-06-03T00:00:00Z");
    assert!(preview.background_md.contains("R-2026"));
    assert!(preview.background_md.contains("1. 第一步"));
    assert_eq!(preview.priority, "urgent");
    service
        .with_db(|db| {
            local::save_issue(
                db,
                None,
                Issue {
                    custom_tags: vec!["personal".into()],
                    summary_md: "用户简述".into(),
                    analysis_md: "用户分析".into(),
                    ..preview
                },
                false,
            )?;
            Ok(())
        })
        .unwrap();
    let refreshed = service.refresh_jira("TL-1001").await.unwrap();
    assert_eq!(refreshed.analysis_md, "用户分析");
    assert_eq!(refreshed.summary_md, "用户简述");
    assert_eq!(refreshed.custom_tags, ["personal"]);
    assert_eq!(refreshed.jira_tags, ["remote"]);
}

#[tokio::test]
async fn ai_summaries_preserve_saved_content_and_invalid_settings_are_not_saved() {
    let root = tempfile::tempdir().unwrap();
    let service = Service::new(Config::isolated(root.path().into())).unwrap();
    service
        .with_db(|db| {
            local::save_issue(
                db,
                None,
                Issue {
                    jira_key: "TL-1001".into(),
                    title: "AI 上下文".into(),
                    solution_md: "用户方案".into(),
                    ..Default::default()
                },
                false,
            )
        })
        .unwrap();
    let server = Server::new(move |headers, body| {
        if headers.contains("/myself ") {
            return (401, json!({}));
        }
        assert!(headers
            .to_lowercase()
            .contains("authorization: bearer test-ai-secret"));
        assert_eq!(body["messages"][0]["role"], "system");
        let content = body["messages"][1]["content"].as_str().unwrap();
        assert!(content.contains("AI 上下文") || content.contains("2026-W23"));
        (
            200,
            json!({"choices":[{"message":{"content":"  新总结  "}}]}),
        )
    });
    settings(&service, &server.url);
    let result = service.issue_summary("TL-1001").await.unwrap();
    assert_eq!(result.issue.solution_md, "用户方案");
    assert_eq!(result.summary, "新总结");
    let week = service.week_summary("2026-W23").await.unwrap();
    assert_eq!(week.summary_md, "新总结");
    let mut input = service.load_settings().unwrap();
    input.tempo.api_token = "changed-token".into();
    let error = service
        .update_settings(input)
        .await
        .unwrap_err()
        .to_string();
    assert!(error.contains("身份验证"));
    assert_eq!(
        service.load_settings().unwrap().tempo.api_token,
        "test-tempo-secret"
    );
}

#[tokio::test]
async fn tempo_pagination_cache_offsets_and_partial_submission() {
    let root = tempfile::tempdir().unwrap();
    let service = Service::new(Config::isolated(root.path().into())).unwrap();
    let reads = Arc::new(std::sync::atomic::AtomicUsize::new(0));
    let count = reads.clone();
    let server = Server::new(move |headers, body| {
        if headers.contains("/rest/api/3/issue/") {
            return (200, json!({"id":"123"}));
        }
        assert!(headers
            .to_lowercase()
            .contains("authorization: bearer test-tempo-secret"));
        if headers.starts_with("POST") {
            assert_eq!(body["issueId"], 123);
            assert_eq!(body["timeSpentSeconds"], 3600);
            if body["startDate"] == "2026-06-02" {
                return (500, json!({}));
            }
            assert_eq!(body["startTime"], "10:00:00");
            return (
                200,
                json!({"tempoWorklogId":99,"self":"https://api.tempo.io/4/worklogs/99"}),
            );
        }
        count.fetch_add(1, Ordering::Relaxed);
        if headers.contains("offset=0") {
            (
                200,
                json!({"results":[{"tempoWorklogId":1,"timeSpentSeconds":7200,"startDate":"2026-06-01","startTime":"8:00","issue":{"key":"CORETIME-80"}}],"metadata":{"next":"/4/worklogs/user/account?offset=1"}}),
            )
        } else {
            assert!(headers.contains("offset=1"));
            (
                200,
                json!({"results":[{"tempoWorklogId":2,"timeSpentSeconds":3600,"startDate":"2026-06-02","startTime":"08:00:00","issue":{"key":"CORETIME-11"}}],"metadata":{}}),
            )
        }
    });
    settings(&service, &server.url);
    let view = service.time_week("2026-W23", false).await.unwrap();
    assert_eq!(view.total_hours, 3.0);
    assert_eq!(view.days.len(), 7);
    assert_eq!(reads.load(Ordering::Relaxed), 2);
    service.time_week("2026-W23", false).await.unwrap();
    assert_eq!(reads.load(Ordering::Relaxed), 2);
    let input = LogTimeRequest {
        work_item_key: "coretime-80".into(),
        description: "验证部分失败".into(),
        hours: 1.0,
        start_date: "2026-06-01".into(),
        end_date: "2026-06-02".into(),
    };
    assert!(service
        .log_time(LogTimeRequest {
            hours: 1.5,
            ..input.clone()
        })
        .await
        .is_err());
    let result = service.log_time(input).await.unwrap();
    assert_eq!((result.successful, result.failed), (1, 1));
    assert!(!result.entries[1].error.contains("test-tempo-secret"));
    assert_eq!(
        service
            .time_week("2026-W23", false)
            .await
            .unwrap()
            .total_hours,
        4.0
    );
    service.time_week("2026-W23", true).await.unwrap();
    assert_eq!(reads.load(Ordering::Relaxed), 4);
}

#[test]
fn images_can_be_read_and_cleanup_keeps_referenced_files() {
    let root = tempfile::tempdir().unwrap();
    let service = Service::new(Config::isolated(root.path().into())).unwrap();
    let dir = root.path().join("uploads");
    let data = format!(
        "data:image/png;base64,{}",
        STANDARD.encode(b"\x89PNG\r\n\x1a\npayload")
    );
    let image = uploads::save(
        &dir,
        uploads::UploadFile {
            name: "image.png".into(),
            context: "Issue TL-1001".into(),
            data,
        },
    )
    .unwrap();
    assert_eq!(image.content_type, "image/png");
    assert!(image.url.ends_with(".png"));
    assert!(uploads::data_url(&dir, &image.url)
        .unwrap()
        .data_url
        .starts_with("data:image/png;"));
    service
        .with_db(|db| {
            let entry = local::create_day(
                db,
                DayEntry {
                    date: "2026-06-01".into(),
                    content_md: format!("![image]({})", image.url),
                    ..Default::default()
                },
            )?;
            assert!(!uploads::delete(db, &dir, &image.url)?);
            assert_eq!(uploads::cleanup(db, &dir)?.kept, 1);
            local::delete_day(db, entry.id)?;
            assert_eq!(uploads::cleanup(db, &dir)?.deleted, 1);
            Ok(())
        })
        .unwrap();
}
