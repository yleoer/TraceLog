# TraceLog Agent 指南

## 项目约定

- 这是单人、单机使用的个人项目，优先直接、易维护的实现，不为多用户、多机或假设的并发场景添加通用框架。
- 仅面向 Windows amd64，技术栈为 Rust + Tauri 2 + Vue 3 + Naive UI + SQLite。
- 根目录 Cargo workspace；`src-tauri/src/main.rs` 为入口，`commands.rs` 为 IPC，`service` 为业务层，`store` 为 SQL 层。
- 前端只能通过 `frontend/src/api/client.ts` 调用原生能力，不引入 REST fallback、本地 HTTP 业务桥或独立服务。
- 沿用现有工作界面，不添加落地页式 UI。
- 不恢复 Go/Wails、手工更新助手或旧的独立后端；历史实现可从 Git 查询。

## 数据与类型

- 只保留 `db/migrations/001_init.sql`，Rust 仅读取 Up 段；不恢复多段历史迁移或 sqlc。
- 修改 schema 同步 SQL/scan、Rust models、前端 types、测试与 seed。
- 修改 IPC 参数和模型后同步 `client.ts` 与 `frontend/src/types/index.ts`。
- 网络请求期间不得持有 SQLite 锁；异步结果只写回需要更新的业务字段，保留用户内容。不添加通用并发冲突检测或字段合并框架。
- 不修改正式用户数据做验证；旧库迁移必须先一致性备份、事务执行并支持重复启动。

## 文档与验证

- 用户可见功能、配置、构建或发布流程有变化时必须更新 README。
- Rust 修改后运行 `cargo fmt --all`、`cargo test --workspace` 和 `cargo clippy --workspace --all-targets --all-features -- -D warnings`。
- 前端或 IPC 修改后运行 `npm.cmd --prefix frontend run build`。
- 验证后清理 `frontend/dist/app`；删除前确认绝对路径位于当前工作区。`frontend/dist` 只保留 `.gitkeep`。
- 保留核心业务和已修复问题的快速测试，不扩展故障注入、并发竞争或攻击输入矩阵。常规改动不运行真实安装、连续升降级或桌面端到端验收；只有直接修改安装或升级机制且需要确认时，才做有针对性的手工检查。
- 不为了重复验收构建多个版本的安装包；普通修改完成必要检查后即可提交，不默认重新生成正式安装器。

## Git 与产物

- 每完成一个迁移阶段进行一次 commit，提交说明用中文：首段总结，随后逐项列出修改。
- 不回滚用户已有改动，不提交构建产物、缓存、密钥或本地数据。
- `build/appicon.png` 和 `build/windows/icon.ico` 是可提交的图标资源；其他 build 产物忽略。
- 创建发布版本前完整列出功能与变更类型并等待用户确认；创建标签前新增 `release-notes/<tag>.md`，以该文件作为 GitHub Release 文案。
