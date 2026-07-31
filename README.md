# TraceLog

TraceLog 是一个基于 Wails 的个人桌面工作日志应用，用于集中跟进 Jira Issue、临时需求、每日工作和周报。它会在刷新 Jira 时同步状态历史，沉淀工作周期，并提供月度甘特图、带图片的 Markdown 记录、全局搜索和长期 Markdown/JSON 导出。

## 技术栈

- 桌面壳：Wails v2
- 应用逻辑：Go、SQLite、goose migrations
- 前端界面：Vue 3、Vite、TypeScript、Naive UI、markdown-it
- 运行时 API：Vue 通过 Wails 原生绑定直接调用 Go

TraceLog 不再运行本地 HTTP API、Docker Compose 或独立后端服务。`frontend/src/api/client.ts` 是前端唯一的 API facade，底层只委托给 Wails 生成的绑定。

## 核心功能

- **Jira 同步与 Issue 管理**：通过 Issue Key 导入或刷新 Jira 信息，维护简述、评论、TODO、分析与解决方案等 Markdown 记录；支持筛选、搜索、复制 Jira 编号和标题，以及单项 Markdown 导出。
- **状态历史与工作周期**：刷新 Jira 时分页读取完整的 Activity History。首次从 `To Do` 进入 `In Progress` 会记录开始时间，首次进入 `Done`、`Resolved`、`Closed` 等完成状态会记录完成时间；重新打开后再次开始和完成会保留为独立周期。
- **标签管理**：Jira 原始标签与自定义标签分别保存。Issue 列表优先显示自定义标签，没有自定义标签时显示 Jira 标签；输入自定义标签时可匹配已有标签。
- **月度甘特图**：按月展示 Issue 与临时需求的工作周期，支持月份切换、多段周期、进行中事项和跳转详情。
- **每日与周视图**：按当天或指定周汇总工作记录、Issue、临时需求和待跟进 TODO，支持生成 Markdown 草稿、AI 总结、编辑下周计划和导出周报。
- **其他工作流**：支持临时需求跟踪、Tempo Time 查看和提交、全局搜索、AI 简述生成，以及完整 JSON 与 Markdown ZIP 导出。

## 开发

安装前端依赖：

```powershell
cd frontend
npm.cmd ci
cd ..
```

从仓库根目录启动桌面开发模式：

```powershell
wails dev
```

开发模式下 Wails 会启动 Vite，用于前端热更新。启动日志里显示的 Vite URL 只用于加载本地 UI 资源；应用数据调用仍然走 Wails 原生绑定，不走本地 HTTP。

## 模拟数据

可以向当前桌面数据库写入一批可重复执行的演示数据：

```powershell
go run ./cmd/seed-demo
```

默认写入位置是系统用户配置目录下的 `TraceLog/tracelog.db`。数据包含 `TL-1001`、`TL-1002`、`TL-1003`、临时任务、评论、TODO、日记录、周报和搜索索引，方便验证 Dashboard、Issues、Today、Weekly、甘特图、Search 和导出流程。

## 构建

从仓库根目录执行：

```powershell
npm.cmd --prefix frontend run build
wails build -clean -debug -nopackage
Remove-Item -Recurse -Force frontend/dist/app
```

生产构建会把 `frontend/dist/app` 嵌入 Go 二进制，不需要 Docker、浏览器服务器或本地 HTTP API。

## 发布构建

GitHub Actions 会通过 [.github/workflows/release.yml](.github/workflows/release.yml) 为不同平台生成正式桌面产物。

可以在 GitHub Actions 页面手动运行 `Release Builds` workflow 生成可下载 artifact。要发布到 GitHub Release，先提交发布说明，再创建并推送注释 tag：

```powershell
$tag = 'v0.1.10'
git push origin main
git tag -a $tag -m "发布 $tag"
git push origin $tag
```

如需自定义 GitHub Release 文案，先在 `release-notes/` 下创建与 tag 同名的 Markdown 文件，例如 `release-notes/v0.1.9.md`。workflow 会优先使用该文件；如果文件不存在，则回退到 GitHub 自动生成的 release notes。

如需用仓库中的文案更新历史 GitHub Release，确认 `gh` 已登录后执行：

```powershell
Get-ChildItem release-notes/*.md | ForEach-Object { gh release edit $_.BaseName --notes-file $_.FullName }
```

workflow 会从根目录构建 Wails 应用并上传：

- Windows amd64 NSIS 安装器：`.exe`
- Linux amd64 Debian 安装包：`.deb`
- macOS universal 磁盘镜像：`.dmg`

## 数据和配置

桌面数据默认存放在系统用户配置目录下的 `TraceLog` 目录，包含：

- `tracelog.db`
- `tracelog-settings.json`
- `uploads/` 下的用户上传图片

可选的开发覆盖环境变量：

- `APP_DATA_DIR`
- `DATABASE_PATH`
- `APP_TIMEZONE`
- `JIRA_BASE_URL`
- `JIRA_EMAIL`
- `JIRA_API_TOKEN`
- `OPENAI_BASE_URL`
- `OPENAI_API_KEY`
- `OPENAI_MODEL`
- `DEEPSEEK_BASE_URL`
- `DEEPSEEK_API_KEY`
- `DEEPSEEK_MODEL`

`TRACELOG_UPDATE_URL` 仅用于本地升级验收：它会覆盖 GitHub Release 查询地址，生产环境不应设置此变量。

Jira 和 AI 凭据也可以在 Settings 页面保存。

## 界面说明

- **Dashboard**：按导入时间展示最近导入的 Issues，并集中查看进行中 Issues、临时需求、待跟进 TODO 和本周概览。
- **Issues**：列表默认按导入时间倒序排列；支持按状态、标签和关键词筛选。详情页展示 Jira 元数据、开始和完成时间；存在多个工作周期时会显示每段周期与重新打开记录。
- **临时需求**：记录计划外工作，可维护来源、状态、优先级、起止时间和关联 Jira。
- **今日工作流**：汇总当天更新的 Issues、每日记录、临时需求和 TODO。
- **周视图**：Issues 仅纳入当周有评论、开始时间或完成时间的项目，不因导入时间而纳入；临时需求和后续 TODO 只在有数据时展示。
- **甘特图**：显示所选月份内与该月重叠的工作周期，每条横线表示一个开始至完成的周期。
- **Time**：查看 Tempo 周工时，并按日期范围提交工作日志。
- **全局搜索与设置**：搜索 Issue、评论、TODO、临时需求和周报；配置 Jira、AI、提示词和导出。

## 验证

```powershell
go test ./...
npm.cmd --prefix frontend run build
wails build -clean -debug -nopackage
```

如果 `wails build` 在 Windows 上提示 `TraceLog-dev.exe` 被占用，请先关闭正在运行的 TraceLog 窗口后再执行。

### Windows 在线升级验收

每个 Windows 发布版本都应在发布前执行一次真实的 `rc.1 -> rc.2` NSIS 升级。脚本会构建两个安装包，在临时目录安装旧版本、启动本地更新源、自动触发升级，并验证进程退出、安装器结果、重启、二进制哈希、更新助手和升级日志。

前提是本机已安装 Wails、NSIS 和 WebView2 开发依赖；建议在 Windows Sandbox 或虚拟机执行。脚本使用用户级测试安装器和临时数据目录，不会覆盖正式安装。

```powershell
.\scripts\test-upgrade.ps1 -KeepArtifacts
```

通过 `-MakensisPath` 可以指定便携版或非标准位置的 `makensis.exe`，通过 `-RunRoot` 可以指定测试证据目录。测试失败时临时目录会保留，脚本会输出证据路径；成功后加 `-KeepArtifacts` 也可保留日志和两个测试安装包。
