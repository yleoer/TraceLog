# TraceLog

TraceLog 是仅面向 Windows amd64 的个人桌面工作日志应用，采用 Rust + Tauri 2 + Vue 3 + SQLite。保留现有 Naive UI 界面和旧版数据格式，业务调用统一经过 `frontend/src/api/client.ts` 的原生 IPC。

当前 Rust 迁移分支已完成签名升级、旧库兼容、本地业务和外部集成，发布流程正在最后阶段迁移。完整记录见 [迁移记录](docs/rust-migration.md)。

## 功能

- Issue、标签、工作周期、评论、TODO 与临时需求管理。
- 每日记录、今日工作流、ISO 周视图、周报草稿和多周期月度甘特图。
- SQLite FTS 全局搜索，Markdown、JSON 与包含图片的 Markdown ZIP 导出。
- 设置原子保存，空凭据保留原值，返回界面时隐藏密钥。
- Tauri 官方签名更新：下载并校验成功后由 NSIS 接管安装与重启。
- Jira 导入预览与刷新，完整状态历史分页、重开周期、Release Requested 字段及 ADF 转 Markdown；刷新保留用户备注和自定义标签。
- Tempo 周工时缓存与主动刷新，允许事项的日期范围提交（最多 31 天、每天 1–8 整数小时），按已有工时顺延开始时间，逐日报告成功和失败。
- OpenAI 兼容服务和 DeepSeek 的 Issue 简述、周总结及自定义提示词；网络请求不占用数据库锁，写回保留期间新增的其他编辑。
- Markdown 图片上传、原生读取、未引用图片清理，以及原生剪贴板和外部链接。图片限制为 8 MiB 的 PNG、JPEG、GIF 或 WebP。

## 开发与构建

需要 Windows amd64、Rust stable MSVC、Windows C++ Build Tools、Node.js 和 WebView2。

```powershell
npm.cmd ci
npm.cmd --prefix frontend ci
npm.cmd run tauri -- dev
```

开发时 Vite 仅提供界面资源，业务访问仍走原生 IPC。生产构建：

```powershell
npm.cmd run tauri -- build
```

安装器输出在 `target/release/bundle/nsis/`。前端构建输出 `frontend/dist/app`，验证后应删除该目录，只保留 `frontend/dist/.gitkeep`；不要提交构建产物。

## 数据和兼容

默认目录 `%APPDATA%/TraceLog` 包含 `tracelog.db`、`tracelog-settings.json`、`uploads/`、`logs/` 和 `backups/`。支持 `APP_DATA_DIR`、`DATABASE_PATH`、`APP_TIMEZONE`（默认 Asia/Shanghai）。Jira 和 AI 支持原有 `JIRA_*`、`OPENAI_*`、`DEEPSEEK_*` 环境变量，也可在设置页保存。

首次打开旧库前通过 SQLite Online Backup 备份数据库（含 WAL 中已提交数据），并备份设置和图片到 `backups/pre-rust-*`。结构调整和搜索回填在事务内完成，保留旧 goose 元信息，重复启动不会再次转换。仅保留 `db/migrations/001_init.sql`，Rust 仅读取 Up 部分。

需要回退旧版时，关闭应用后从同一个备份目录恢复数据库、设置和图片。不要同时运行两个版本写同一个数据库。

## 验证

```powershell
cargo fmt --all -- --check
cargo test --workspace
cargo clippy --workspace --all-targets --all-features -- -D warnings
npm.cmd --prefix frontend run build
cargo run --example check-legacy -- <旧版数据库文件>
```

兼容检查在临时副本上执行，不修改输入数据库。本地业务测试覆盖事务回滚、标签和周期保留、搜索删除、时区、ISO 周、月边界及导出内容。

## 升级与发布

签名升级公钥位于 `src-tauri/tauri.conf.json`。本机私钥位于 `%USERPROFILE%/.config/tracelog/keys/updater.key`，必须单独备份，不能提交。CI 使用 `TAURI_SIGNING_PRIVATE_KEY` 与 `TAURI_SIGNING_PRIVATE_KEY_PASSWORD`；升级签名和 Windows Authenticode 代码签名分别配置。

```powershell
./scripts/test-tauri-upgrade.ps1 -KeepArtifacts
```

脚本使用临时用户级安装和独立数据目录，构建真实签名 NSIS 安装器，验证进程退出、新版重启、安装后二进制哈希及数据保留。`TRACELOG_UPDATE_URL` 仅在 `update-e2e` 构建中允许回环测试源。

旧 Wails 系统级安装需要手动迁入新的用户级安装，不能直接接管原安装登记。正式发布前必须更新 README、列全变更并获得用户确认；创建标签前新增 `release-notes/<tag>.md`，将其作为 Release 文案。此分支尚未发布版本。
