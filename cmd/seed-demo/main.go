package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"tracelog/internal/app"
	"tracelog/internal/config"
	"tracelog/internal/db"
	"tracelog/internal/service"
)

func main() {
	ctx := context.Background()
	cfg := config.Load()
	database, err := db.Open(cfg)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer database.Close()

	if err := db.Migrate(database, "db/migrations"); err != nil {
		log.Fatalf("run migrations: %v", err)
	}

	runtime, err := app.NewRuntime(cfg, database)
	if err != nil {
		log.Fatalf("create runtime: %v", err)
	}

	if err := seed(ctx, database, runtime.Service, cfg.Location); err != nil {
		log.Fatalf("seed demo data: %v", err)
	}
	fmt.Printf("Seeded demo data into %s\n", cfg.DatabasePath)
}

func seed(ctx context.Context, database *sql.DB, svc *service.Service, loc *time.Location) error {
	if loc == nil {
		loc = time.Local
	}
	now := time.Now().In(loc)
	today := beginningOfDay(now)
	yesterday := today.AddDate(0, 0, -1)
	tomorrow := today.AddDate(0, 0, 1)
	week := service.CurrentWeek(loc)

	issues := []service.Issue{
		{
			JiraKey:   "TL-1001",
			Title:     "优化 Wails 原生绑定后的 Issue 详情页",
			Status:    "processing",
			Priority:  "high",
			Tags:      []string{"wails", "frontend", "demo"},
			SummaryMD: "桌面端已经切换到 Wails 原生绑定，当前重点验证详情页读写、评论时间线和结构化 TODO。",
			BackgroundMD: strings.Join([]string{
				"从前后端 HTTP 架构迁移到根目录 Wails 项目后，需要确认所有页面都通过 `frontend/src/api/client.ts` 调用原生绑定。",
				"",
				"- 不再启动本地 Gin API",
				"- 不依赖 Docker Compose",
				"- 数据写入桌面端 SQLite",
			}, "\n"),
			AnalysisMD: "详情页主要风险在字段名映射、`wailsjs` 绑定类型和 Markdown 编辑内容保存。",
			SolutionMD: "保留前端 API facade，但底层只调用 `DesktopApp.*`。用模拟数据覆盖列表、详情、TODO、评论和搜索。",
			ActionsMD:  "- 打开 TL-1001 详情页\n- 新增一条评论\n- 勾选一个 TODO\n- 搜索 `原生绑定`",
			ResultMD:   "待验证。",
			TodoMD:     "- [ ] 验证详情页保存\n- [ ] 验证 Markdown 渲染\n- [ ] 验证导出",
			Links: []service.Link{
				{Title: "Wails docs", URL: "https://wails.io/docs", Type: "reference"},
				{Title: "Local migration note", URL: "tracelog://demo/wails-migration", Type: "note"},
			},
			StartedAt: formatRFC3339(today.Add(9*time.Hour + 15*time.Minute)),
			UpdatedAt: formatRFC3339(now.Add(-35 * time.Minute)),
		},
		{
			JiraKey:      "TL-1002",
			Title:        "补齐周报视图的日工作聚合",
			Status:       "analysis",
			Priority:     "medium",
			Tags:         []string{"weekly", "today", "demo"},
			SummaryMD:    "验证 Today 和 Weekly 页面是否能聚合 issue 评论、临时任务评论、手动日记录和 TODO。",
			BackgroundMD: "周报视图依赖本周时间范围内的 issue、event、temp task、todo 和 day entry。",
			AnalysisMD:   "如果时间字段不完整，周报可能为空；模拟数据覆盖今天、昨天和明天的边界。",
			SolutionMD:   "用本周内的多种记录填充视图，并预置一份 weekly log。",
			ActionsMD:    "- 打开 Today 页面\n- 打开 Weekly 页面\n- 生成周报草稿",
			ResultMD:     "周报应该能看到本周进行中、已完成和待办项。",
			TodoMD:       "- [ ] 检查周报草稿\n- [ ] 检查每日记录排序",
			StartedAt:    formatRFC3339(yesterday.Add(14 * time.Hour)),
			UpdatedAt:    formatRFC3339(now.Add(-2 * time.Hour)),
		},
		{
			JiraKey:      "TL-1003",
			Title:        "验证 Markdown 导出和搜索索引",
			Status:       "done",
			Priority:     "low",
			Tags:         []string{"export", "search", "demo"},
			SummaryMD:    "该记录用于验证搜索、Markdown 导出和已完成事项展示。",
			BackgroundMD: "包含唯一关键词 `demo-export-check`，方便搜索页快速定位。",
			AnalysisMD:   "导出内容需要包含结构化 TODO 和时间线事件。",
			SolutionMD:   "通过 service 层创建并索引数据。",
			ActionsMD:    "- 搜索 demo-export-check\n- 导出 TL-1003 Markdown",
			ResultMD:     "已完成，可作为导出 smoke 数据。",
			TodoMD:       "- [x] 搜索关键词准备完成",
			StartedAt:    formatRFC3339(yesterday.Add(10 * time.Hour)),
			CompletedAt:  formatRFC3339(yesterday.Add(17*time.Hour + 30*time.Minute)),
			UpdatedAt:    formatRFC3339(yesterday.Add(17*time.Hour + 30*time.Minute)),
		},
	}

	for _, issue := range issues {
		if err := upsertIssue(ctx, svc, issue); err != nil {
			return err
		}
	}

	events := map[string][]service.IssueEvent{
		"TL-1001": {
			{EventType: "analysis", ContentMD: "确认详情页的基础字段已经从 Wails binding 返回，下一步验证保存后列表实时刷新。", HappenedAt: formatRFC3339(today.Add(10 * time.Hour))},
			{EventType: "action", ContentMD: "补充模拟评论：用户可以在时间线里继续追加分析、决策或结果。", HappenedAt: formatRFC3339(now.Add(-25 * time.Minute))},
			{EventType: "blocker", ContentMD: "需要特别观察上传图片和 `/uploads/...` 静态访问是否仍然正常。", HappenedAt: formatRFC3339(now.Add(-15 * time.Minute))},
		},
		"TL-1002": {
			{EventType: "note", ContentMD: "Today 页面应该展示这条 issue 评论，并归到今天的工作面板。", HappenedAt: formatRFC3339(today.Add(11*time.Hour + 30*time.Minute))},
			{EventType: "decision", ContentMD: "周报草稿先覆盖本周数据，不跨周抓取历史事项。", HappenedAt: formatRFC3339(today.Add(16 * time.Hour))},
		},
		"TL-1003": {
			{EventType: "result", ContentMD: "导出 smoke 数据准备完成，包含关键词 demo-export-check。", HappenedAt: formatRFC3339(yesterday.Add(17 * time.Hour))},
		},
	}
	for key, items := range events {
		if err := replaceIssueEvents(ctx, svc, key, items); err != nil {
			return err
		}
	}

	todos := map[string][]service.IssueTodo{
		"TL-1001": {
			{Content: "验证详情页修改后能回写 SQLite", DueAt: formatRFC3339(today.Add(18 * time.Hour))},
			{Content: "上传一张截图并确认 Markdown 中可以渲染", DueAt: formatRFC3339(tomorrow.Add(11 * time.Hour))},
		},
		"TL-1002": {
			{Content: "检查 Today 页面是否展示手动日记录", DueAt: formatRFC3339(today.Add(17 * time.Hour))},
			{Content: "确认 Weekly 页面可以生成草稿", DueAt: formatRFC3339(tomorrow.Add(15 * time.Hour))},
		},
		"TL-1003": {
			{Content: "导出单个 issue Markdown", DueAt: formatRFC3339(yesterday.Add(16 * time.Hour)), Done: true},
		},
	}
	for key, items := range todos {
		if err := replaceIssueTodos(ctx, svc, key, items); err != nil {
			return err
		}
	}

	tasks := []service.TempTask{
		{
			Title:     "整理 Wails 发布包验证清单",
			Source:    "demo seed",
			Status:    "todo",
			Priority:  "medium",
			Tags:      []string{"release", "checklist", "demo"},
			ContentMD: "- 检查 Windows 安装包\n- 检查 GitHub Actions artifact\n- 记录手动验证结果",
			ResultMD:  "",
			StartedAt: formatRFC3339(today.Add(13 * time.Hour)),
			UpdatedAt: formatRFC3339(today.Add(13 * time.Hour)),
		},
		{
			Title:     "临时排查：列表筛选条件不生效",
			Source:    "用户反馈",
			Status:    "processing",
			Priority:  "high",
			Tags:      []string{"filter", "demo"},
			ContentMD: "用 `status=processing` 和 tag 组合筛选，确认列表页查询参数到 store 的链路。",
			ResultMD:  "正在排查。",
			StartedAt: formatRFC3339(today.Add(9 * time.Hour)),
			UpdatedAt: formatRFC3339(now.Add(-55 * time.Minute)),
		},
		{
			Title:       "已完成：删除旧 Docker 文档",
			Source:      "迁移收尾",
			Status:      "done",
			Priority:    "low",
			Tags:        []string{"cleanup", "demo"},
			ContentMD:   "确认 README 不再引导使用 Docker Compose。",
			ResultMD:    "已从 README 中移除旧前后端部署说明。",
			StartedAt:   formatRFC3339(yesterday.Add(9 * time.Hour)),
			CompletedAt: formatRFC3339(yesterday.Add(11 * time.Hour)),
			UpdatedAt:   formatRFC3339(yesterday.Add(11 * time.Hour)),
		},
	}
	taskIDs := map[string]int64{}
	for _, task := range tasks {
		created, err := upsertTempTask(ctx, svc, task)
		if err != nil {
			return err
		}
		taskIDs[task.Title] = created.ID
	}

	taskEvents := map[string][]service.TempTaskEvent{
		"整理 Wails 发布包验证清单": {
			{EventType: "note", ContentMD: "发布检查清单已创建，后续补截图和平台结果。", HappenedAt: formatRFC3339(today.Add(13*time.Hour + 20*time.Minute))},
		},
		"临时排查：列表筛选条件不生效": {
			{EventType: "analysis", ContentMD: "初步怀疑是状态筛选和 tag 筛选组合时参数没有同步。", HappenedAt: formatRFC3339(today.Add(10*time.Hour + 45*time.Minute))},
			{EventType: "action", ContentMD: "用模拟数据覆盖 todo、processing、done 三种状态。", HappenedAt: formatRFC3339(now.Add(-45 * time.Minute))},
		},
		"已完成：删除旧 Docker 文档": {
			{EventType: "result", ContentMD: "旧 Docker Compose 路径已清理，验证数据保留此完成项。", HappenedAt: formatRFC3339(yesterday.Add(11 * time.Hour))},
		},
	}
	for title, items := range taskEvents {
		id := taskIDs[title]
		if id == 0 {
			return fmt.Errorf("missing temp task id for %q", title)
		}
		if err := replaceTempTaskEvents(ctx, svc, id, items); err != nil {
			return err
		}
	}

	if err := replaceDayEntries(ctx, database, svc, today.Format("2006-01-02"), []string{
		"上午验证 Wails 原生绑定页面加载，确认不再访问本地 HTTP API。",
		"下午检查 Today/Weekly 聚合视图，补充 demo 数据用于手工 smoke test。",
	}); err != nil {
		return err
	}
	if err := replaceDayEntries(ctx, database, svc, yesterday.Format("2006-01-02"), []string{
		"完成旧 Docker Compose 文档清理，并确认根目录 Wails 构建通过。",
	}); err != nil {
		return err
	}

	_, err := svc.UpsertWeeklyLog(ctx, week, service.WeeklyLog{
		SummaryMD: strings.Join([]string{
			"# 本周模拟周报",
			"",
			"- 完成 Wails 根目录迁移后的基础验证。",
			"- 补齐 Issue、临时任务、TODO、日记录和搜索数据。",
			"- 继续关注打包产物和图片上传 smoke test。",
		}, "\n"),
		NextPlanMD: "- 验证 Windows 安装包\n- 检查 macOS/Linux Actions artifact\n- 完成一次 Markdown/JSON 导出",
	})
	return err
}

func upsertIssue(ctx context.Context, svc *service.Service, issue service.Issue) error {
	existing, err := svc.GetIssue(ctx, issue.JiraKey)
	if err == nil {
		issue.CreatedAt = existing.CreatedAt
		_, err = svc.UpdateIssue(ctx, issue.JiraKey, issue)
		return err
	}
	if !isNotFound(err) {
		return err
	}
	_, err = svc.CreateIssue(ctx, issue)
	return err
}

func replaceIssueEvents(ctx context.Context, svc *service.Service, jiraKey string, events []service.IssueEvent) error {
	existing, err := svc.ListIssueEvents(ctx, jiraKey)
	if err != nil {
		return err
	}
	for _, event := range existing {
		if err := svc.DeleteIssueEvent(ctx, event.ID); err != nil {
			return err
		}
	}
	for _, event := range events {
		if _, err := svc.CreateIssueEvent(ctx, jiraKey, event); err != nil {
			return err
		}
	}
	return nil
}

func replaceIssueTodos(ctx context.Context, svc *service.Service, jiraKey string, todos []service.IssueTodo) error {
	existing, err := svc.ListIssueTodos(ctx, jiraKey, true)
	if err != nil {
		return err
	}
	for _, todo := range existing {
		if err := svc.DeleteIssueTodo(ctx, todo.ID); err != nil {
			return err
		}
	}
	for _, todo := range todos {
		if _, err := svc.CreateIssueTodo(ctx, jiraKey, todo); err != nil {
			return err
		}
	}
	return nil
}

func upsertTempTask(ctx context.Context, svc *service.Service, task service.TempTask) (service.TempTask, error) {
	tasks, err := svc.ListTempTasks(ctx, service.TempTaskFilter{All: true})
	if err != nil {
		return service.TempTask{}, err
	}
	for _, existing := range tasks {
		if existing.Title == task.Title {
			return svc.UpdateTempTask(ctx, existing.ID, task)
		}
	}
	return svc.CreateTempTask(ctx, task)
}

func replaceTempTaskEvents(ctx context.Context, svc *service.Service, taskID int64, events []service.TempTaskEvent) error {
	existing, err := svc.ListTempTaskEvents(ctx, taskID)
	if err != nil {
		return err
	}
	for _, event := range existing {
		if err := svc.DeleteTempTaskEvent(ctx, event.ID); err != nil {
			return err
		}
	}
	for _, event := range events {
		if _, err := svc.CreateTempTaskEvent(ctx, taskID, event); err != nil {
			return err
		}
	}
	return nil
}

func replaceDayEntries(ctx context.Context, database *sql.DB, svc *service.Service, date string, contents []string) error {
	for _, content := range contents {
		if _, err := database.ExecContext(ctx, `DELETE FROM day_entries WHERE date = ? AND content_md = ?`, date, content); err != nil {
			return err
		}
	}
	for _, content := range contents {
		if _, err := svc.CreateDayEntry(ctx, date, content); err != nil {
			return err
		}
	}
	return nil
}

func isNotFound(err error) bool {
	var appErr *service.AppError
	return errors.As(err, &appErr) && appErr.Code == 404
}

func beginningOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func formatRFC3339(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}
