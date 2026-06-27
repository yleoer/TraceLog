<!-- TRELLIS:START -->
# Trellis 使用说明

这些说明面向在本项目中工作的 AI 助手。

本项目由 Trellis 管理。开发前需要读取的项目知识位于 `.trellis/`：

- `.trellis/workflow.md`：开发阶段、何时创建任务、技能路由规则
- `.trellis/spec/`：按包和层划分的编码规范；在修改对应层代码前必须阅读
- `.trellis/workspace/`：开发者日志和会话记录
- `.trellis/tasks/`：进行中和已归档任务，包括 PRD、调研和 jsonl 上下文

如果当前平台提供 Trellis 命令，例如 `/trellis:finish-work` 或 `/trellis:continue`，优先使用命令，不要手动重复流程。不同平台暴露的命令可能不完全一致。

如果使用 Codex 或其他支持 agent 的工具，项目内还可能提供额外助手：

- `.agents/skills/`：可复用的 Trellis skills
- `.codex/agents/`：可选的自定义 subagents

在本项目中开始编码前，先读取相关 Trellis spec；涉及 Wails、Go、Vue、SQLite、搜索、导出或发布构建时，按对应层的规范执行验证。

当前项目约定：

- TraceLog 是根目录 Wails 桌面应用，不再保留旧的前端 + 后端 + Docker Compose 架构。
- 前端只能通过 `frontend/src/api/client.ts` 调用 Wails 原生绑定，不要重新引入 REST fallback、`fetch('/api/...')` 或本地 HTTP 桥。
- Go 入口、Wails 配置、数据库迁移、服务层、store 层都位于仓库根目录结构下。
- 构建产物、缓存和生成的前端静态文件不要提交；`frontend/dist` 只保留 `.gitkeep`，构建后生成的 `frontend/dist/app` 需要清理。
- `build/appicon.png` 和 `build/windows/icon.ico` 是应用图标资源，可以提交；其他 `build/` 产物继续忽略。
- 如需验证模拟数据，可运行 `go run ./cmd/seed-demo`。

由 Trellis 管理。本区块外的编辑会被保留；本区块内的内容未来可能被 `trellis update` 覆盖。

<!-- TRELLIS:END -->
