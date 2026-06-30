# TraceLog Agent 指南

本文件面向在本仓库工作的 AI 助手。开始修改前先阅读本指南，并优先遵守仓库现有代码风格。

## 项目概览

- TraceLog 是根目录 Wails 桌面应用，技术栈为 Go + Wails + Vue 3 + SQLite。
- 不再保留旧的前端 + 后端 + Docker Compose 架构。
- Go 入口、Wails 配置、数据库迁移、服务层、store 层都位于仓库根目录结构下。
- 前端只能通过 `frontend/src/api/client.ts` 调用 Wails 原生绑定。
- 不要重新引入 REST fallback、`fetch('/api/...')` 或本地 HTTP 桥。

## 目录约定

- `main.go`、`assets.go`：Wails 应用入口和嵌入资源。
- `internal/desktop`：Wails 暴露给前端的应用接口。
- `internal/service`：业务逻辑、验证、导出、搜索索引、周报生成等。
- `internal/store`：SQLite 读写层，当前使用手写 SQL。
- `internal/db`：数据库打开和 goose 迁移执行。
- `db/migrations/001_init.sql`：唯一初始化迁移文件。
- `frontend/src/api/client.ts`：前端调用 Wails 原生绑定的唯一入口。
- `frontend/src/wailsjs`：Wails 生成绑定，修改 Go 暴露类型后需要同步生成或谨慎更新。
- `cmd/seed-demo`：模拟数据填充命令。

## 数据库约定

- 当前迁移目录只保留 `db/migrations/001_init.sql`。
- 不要新增多段历史迁移，除非用户明确要求恢复增量迁移模式。
- 修改 schema 时，同步更新：
  - `db/migrations/001_init.sql`
  - `internal/store` 中对应 SQL 和 scan 逻辑
  - `internal/service/models.go`
  - `frontend/src/types/index.ts`
  - `frontend/src/wailsjs/go/models.ts`
  - 相关测试和 seed 数据
- 本项目不再使用 sqlc；不要恢复 `db/query` 或 `sqlc.yaml`。

## 前端约定

- 前端 API 调用统一经过 `frontend/src/api/client.ts`。
- 页面组件应沿用现有 Vue + Naive UI + Tailwind 风格。
- 不要添加落地页式 UI；功能页面应直接呈现可用工作界面。
- 修改 Wails 暴露模型后，确认前端类型和绑定文件一致。
- 构建会生成 `frontend/dist/app`，验证后必须清理；`frontend/dist` 只保留 `.gitkeep`。

## Go 约定

- 业务规则放在 `internal/service`，数据库细节放在 `internal/store`。
- Store 层 SQL 字段顺序必须和 scan 顺序保持一致。
- 修改 Go 文件后运行 `gofmt`。
- 不要把构建产物、缓存或本地数据提交进仓库。

## 验证命令

常规修改后优先运行：

```powershell
go test ./...
```

涉及前端或 Wails 绑定时运行：

```powershell
npm run build
```

命令目录：

```powershell
cd frontend
npm run build
```

前端构建后清理：

```powershell
Remove-Item -Recurse -Force frontend/dist/app
```

如需验证模拟数据：

```powershell
go run ./cmd/seed-demo
```

## Git 和产物

- 不要提交 `frontend/dist/app`。
- `build/appicon.png` 和 `build/windows/icon.ico` 是应用图标资源，可以提交。
- 其他 `build/` 产物继续忽略。
- 在已有未提交改动时，不要回滚用户改动；只修改和当前任务相关的文件。
- 如需创建 git commit，提交说明必须使用中文：第一段先总结本次修改，随后详细罗列每一项修改。
- 如需提交 release 版本，必须先按功能和修改类型完整罗列所有变更，并等待用户确认后再提交。
