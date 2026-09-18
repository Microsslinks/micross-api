# Archive (历史工作文档)

> 存放从 v0.37.x 升级前的所有 task 工作文件、临时调研文档、调试日志。
> 与活跃文档的区分：active docs 在 `../release/`、`../deploy/` 和 `.docs/` 顶层；
> 这一目录下的内容**不再维护**，只在需要追溯历史时查阅。

## 目录结构

| 子目录 | 含义 |
|---|---|
| `task-01-preparation/` ~ `task-20-ledger-and-risk/` | 任务工作文件（含 README.md / summary.md / 子任务拆分 / 测试记录）。每个 task 是一次独立开发周期的产物。 |
| `backup/` | 备份快照（已合并/废弃分支的快照文件）。 |
| `env/` | 环境配置笔记（.env 模板、密钥生成记录）。 |
| `frontend/` | 前端相关调研与一致性检查产出（含 `ui-consistency.md`）。 |
| `governance/` | 治理规范（如 `ui-entry-rule.md`）。 |
| `rebrand/` | rebrand (calciumion/new-api → dukaworks/micross-api) 期间的脚本与资产。 |
| `run/` | 调试运行日志与浏览器 profile 残留（**绝大多数是临时调试产物，不需要查看**）。 |

## 顶层散落的 md

| 文件 | 含义 |
|---|---|
| `todo-next-major-customer-code-plan.md` | v0.38+ 候选：客户号方案深度改造 brainstorm。 |
| `todo-next-major-model-list-scope.md` | v0.38+ 候选：模型列表 scope 重整 brainstorm。 |
| `todo-readme-final-polish.md` | README 终稿打磨 checklist（task-13 衍生）。 |

## 何时到这里找东西

- 查某次 task 的设计意图：进入对应 `task-XX-name/` 看 README.md。
- 查历史 commit 上下文：`git log -- .docs/_archive/<path>`。
- 查早期调试日志：**不推荐**——`.docs/_archive/run/` 是混乱的临时输出，新问题用 `docker compose logs micross-api` 取实时日志。

## 何时不读这一目录

- 想了解当前架构：看 `../master-plan.md` + `../summary.md` + `../release/v0.37.1.1.md`。
- 想部署：看 `../deploy/docker-vps-quickstart.md`。
- 想知道下一次升级怎么做：看 `../CONVENTIONS.md`。

## 归档规则

新内容不要直接放到 `_archive/` 下。流程：

1. 在 `task-XX-name/` 下建立工作目录（task-XX 是当前活跃 task 编号）
2. 工作完成后按当前 .docs 治理规范决定：移到 `_archive/task-XX-name/` 或保留在 `.docs/` 顶层作为治理文档。
3. 任何 debug log / browser profile / 临时 JSON 输出**不入 git**——它们原本就在 .gitignore 里（`plans`, `.codebuddy/`, `dev/` 等），通过 `.docs/_archive/run/` 的存在仅用于追溯，不进版本控制。