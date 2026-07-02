# TraceLog

TraceLog 是一个基于 Wails 的桌面应用，用于记录 Jira 问题分析、处理过程笔记、临时需求、每日工作、周报、全局搜索、带图片的 Markdown 记录，以及长期 Markdown/JSON 导出。

## 技术栈

- 桌面壳：Wails v2
- 应用逻辑：Go、SQLite、goose migrations
- 前端界面：Vue 3、Vite、TypeScript、Naive UI、markdown-it
- 运行时 API：Vue 通过 Wails 原生绑定直接调用 Go

TraceLog 不再运行本地 HTTP API、Docker Compose 或独立后端服务。`frontend/src/api/client.ts` 是前端唯一的 API facade，底层只委托给 Wails 生成的绑定。

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

默认写入位置是系统用户配置目录下的 `TraceLog/tracelog.db`。数据包含 `TL-1001`、`TL-1002`、`TL-1003`、临时任务、评论、TODO、日记录、周报和搜索索引，方便验证 Dashboard、Issues、Today、Weekly、Search 和导出流程。

## 构建

从仓库根目录执行：

```powershell
npm.cmd --prefix frontend run build
wails build -clean -debug -nopackage
```

生产构建会把 `frontend/dist/app` 嵌入 Go 二进制，不需要 Docker、浏览器服务器或本地 HTTP API。

## 发布构建

GitHub Actions 会通过 [.github/workflows/release.yml](.github/workflows/release.yml) 为不同平台生成正式桌面产物。

可以在 GitHub Actions 页面手动运行 `Release Builds` workflow 生成可下载 artifact。要发布到 GitHub Release，推送版本 tag：

```powershell
git tag v0.1.0
git push origin v0.1.0
```

如需自定义 GitHub Release 文案，先在 `release-notes/` 下创建同名 Markdown 文件，例如 `release-notes/v0.1.0.md`。workflow 会优先使用该文件；如果文件不存在，则回退到 GitHub 自动生成的 release notes。

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

Jira 和 AI 凭据也可以在 Settings 页面保存。

## 主要功能

- Dashboard：查看最近 Issue、进行中的 Issue、临时任务和待跟进 TODO。
- Issue 详情：维护 Jira 元数据、摘要、开始/完成时间、时间线评论、带图片的 Markdown 笔记和结构化 TODO。
- Today：查看当天 Issue 更新、评论、临时任务、到期 TODO、手动日记录和周报草稿片段。
- Weekly：聚合本周 Issue、评论、临时任务、TODO、已完成项、进行中事项，并生成 Markdown 草稿或 AI 周总结。
- Search：全局搜索 Issue、评论、TODO、临时任务和周报；搜索结果可直接跳转到对应详情页。
- Settings：配置 Jira/AI、编辑提示词、导出完整 JSON 和 Markdown ZIP。

## 验证

```powershell
go test ./...
npm.cmd --prefix frontend run build
wails build -clean -debug -nopackage
```

如果 `wails build` 在 Windows 上提示 `TraceLog-dev.exe` 被占用，请先关闭正在运行的 TraceLog 窗口后再执行。
