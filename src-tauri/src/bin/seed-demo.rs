use tracelog::{
    config::Config,
    models::*,
    service::{local, views, Service},
};

fn main() -> Result<(), Box<dyn std::error::Error>> {
    let args: Vec<String> = std::env::args().collect();
    if args.len() != 3 || args[1] != "--data-dir" {
        return Err("用法：cargo run --bin seed-demo -- --data-dir <独立数据目录>".into());
    }
    let service = Service::new(Config::isolated(std::path::PathBuf::from(&args[2])))?;
    let date = chrono::Utc::now()
        .with_timezone(&service.config.timezone)
        .date_naive();
    service.with_db(|db|{
        for (i,title,status) in [(1,"修复升级与重启","In Progress"),(2,"核对旧版数据兼容","Done"),(3,"完善周报导出","To Do")]{
            let key=format!("TL-100{i}");
            if local::get_issue(db,&key).is_ok(){continue;}
            local::save_issue(db,None,Issue{jira_key:key.clone(),title:title.into(),status:status.into(),priority:"high".into(),custom_tags:vec!["演示".into(),"Rust".into()],summary_md:"用于独立目录中的桌面验收。".into(),background_md:"![demo](/uploads/demo.png)".into(),started_at:date.and_hms_opt(0,0,0).unwrap().and_utc().to_rfc3339(),..Default::default()},false)?;
            local::save_issue_event(db,Some(&key),0,IssueEvent{event_type:"note".into(),content_md:"记录今天的实现和验证结果。".into(),..Default::default()})?;
            local::save_todo(db,Some(&key),0,IssueTodo{content:"核对验收清单".into(),due_at:date.and_hms_opt(9,0,0).unwrap().and_utc().to_rfc3339(),..Default::default()})?;
        }
        for title in ["整理项目笔记","排查临时反馈","复核安装器"]{
            let exists:bool=db.query_row("SELECT EXISTS(SELECT 1 FROM temp_tasks WHERE source='TraceLog Rust Demo' AND title=?1)",[title],|r|r.get(0))?;
            if exists{continue;}
            let task=local::save_task(db,0,TempTask{title:title.into(),source:"TraceLog Rust Demo".into(),content_md:"一项演示临时工作。".into(),status:"processing".into(),..Default::default()})?;
            local::save_task_event(db,Some(task.id),0,TempTaskEvent{event_type:"note".into(),content_md:"演示跟进评论。".into(),..Default::default()})?;
        }
        let exists:bool=db.query_row("SELECT EXISTS(SELECT 1 FROM day_entries WHERE content_md='Rust 迁移演示日记录')",[],|r|r.get(0))?;
        if !exists{local::create_day(db,DayEntry{date:date.to_string(),content_md:"Rust 迁移演示日记录".into(),..Default::default()})?;}
        let week=views::current_week(service.config.timezone);
        let view=views::week(db,&week,service.config.timezone)?;
        if view.log.id==0{views::draft_week(db,&week,service.config.timezone)?;}
        Ok(())
    })?;
    let image = service.config.data_dir.join("uploads/demo.png");
    if !image.exists() {
        use base64::Engine;
        std::fs::write(image,base64::engine::general_purpose::STANDARD.decode("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR4nGNgYGD4DwABBAEAX+XDSwAAAABJRU5ErkJggg==")?)?;
    }
    println!(
        "演示数据已写入 {}（重复执行保留已有记录）",
        service.config.data_dir.display()
    );
    Ok(())
}
