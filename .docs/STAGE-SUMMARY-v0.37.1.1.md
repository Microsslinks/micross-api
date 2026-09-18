# 阶段总结：v0.37.1.1 基准

> 截至 2026-09-19，本仓库升级改造以此 tag 为**唯一基准**。
> 此前的所有功能 / 文档 / 测试 均已落在 `v0.37.1.1` 的 commit `825bc416` 上。

## 一、本阶段交付物

### 1.1 版本与发布链路

| 项 | 值 |
|---|---|
| 基准 tag | `v0.37.1.1` |
| 基准 commit | `825bc4162eb9d1558dbc8db600a2acce3b77d6e8` |
| 上一稳定版 | `v0.37.1`（`8b7779a0`），向后跳了一个 patch |
| 上一上游 | `calciumion/new-api`（fork 自 QuantumNous/new-api） |
| GitHub | github.com/dukaworks/micross-api |
| Docker 仓库 | ghcr.io/dukaworks/micross-api |
| Docker tag 范围 | `v0.37.1`, `v0.37.1.1`, `latest`（均含 amd64 + arm64 + multi-arch manifest） |
| Linux/macOS/Windows backend binary | `releases/tag/v0.37.1.1`（含 checksums） |
| Electron Desktop | 310MB Windows installer（GH Actions artifact） |
| CI workflows | `ci.yml`, `docker-build.yml`, `docker-image-branch.yml`, `electron-build.yml`, `pr-check.yml`, `release.yml`, `sync-release-to-gitcode.yml` |

### 1.2 任务全景

| Task | 范围 | commit 基线 |
|---|---|---|
| task-01-preparation | 环境、DB、backend/frontend 结构普查 | 已归档 |
| task-02-business-goals | 业务方案 01-08 (8 个业务方案文档) | 已归档 |
| task-03~06 | 导航 / 折扣 / 角色 / 多方案基础 | 已归档 |
| task-07 | rebrand (calciumion → dukaworks) | 多次 commit |
| task-08-12 | P3-P5 阶段：客户注册 / topup / 佣金 / 订阅 | 已归档 |
| task-13 | 7 语言原生 i18n (en/zh-CN/zh-TW/fr/ja/ru/vi) | 已归档 |
| task-14 | 历史技术债清理 | 已归档 |
| task-15-16 | 用户画像 / 重置邀请人 | 已归档 |
| task-17 | 深度清理（schema / test / 依赖 / 安全） | `46138a4c` 起 |
| task-18 | 快速修复（订阅支付 + 邀请修复） | 已归档 |
| task-19 | shadow ledger 草图 | 已归档 |
| task-20 | **本阶段主线**：ledger 4 类事件 + P4 风控（5+6+7+8）| `46138a4c` ~ `e82cbf4b` |

### 1.3 目录结构（升级后）

```
.
├── .docs/
│   ├── README.md                       # .docs 入口
│   ├── CHANGELOG.md                    # 全版本变更日志
│   ├── master-plan.md                  # 主方案
│   ├── project-task-list-2026-09-14.md # 项目任务清单
│   ├── summary.md                      # 项目摘要
│   ├── CONVENTIONS.md                  # ⭐ 升级改造基本规则（新增）
│   ├── STAGE-SUMMARY-v0.37.1.1.md      # ⭐ 本文档
│   ├── _archive/                       # 历史工作文件（归档）
│   │   └── README.md                   # 归档说明
│   ├── deploy/                         # 部署文档（活跃）
│   └── release/                        # 发布说明（活跃）
│       ├── v0.37.1.md
│       └── v0.37.1.1.md                # ⭐ 最新
├── AGENTS.md / CLAUDE.md               # AI 协作指令（保留）
├── README.md / README.zh_CN.md / ...   # 5 语言 README（task-13）
├── docker-compose.yml                  # 默认 dev 兼容 compose
├── docker-compose.dev.yml              # 开发 compose
├── docker-compose.prod.yml             # ⭐ 生产硬化版（新增）
├── Dockerfile                          # multi-stage（Bun + Go + Debian-slim）
├── Dockerfile.dev                      # dev 后端镜像（with source）
├── VERSION                             # 当前为 v0.37.1.1
├── go.mod / go.sum                     # 依赖
├── main.go                             # 入口（含 6 类 cron task 注册）
├── relaykit/                           # local submodule（go.mod 引用）
├── cmd/, common/, constant/, controller/, deploy/, dto/, electron/, i18n/,
│   logger/, middleware/, model/, oauth/, pkg/, relay/, router/, scripts/,
│   service/, setting/, types/, web/    # 主代码
├── bin/                                # build 产物（gitignored）
├── logs/                               # 日志目录（gitignored）
└── web/                                # 前端（Bun + Rsbuild + React）
```

## 二、本阶段产出的代码亮点

### 2.1 task-20 ledger + 风控（核心增量）

| 子项 | 落点 | commit |
|---|---|---|
| §20.1 topup 写 ledger | `service/topup.go` `CompleteTopUp` | `46138a4c` |
| §20.2 refund 写 ledger | `service/funding_source.go` | `783b12c7` |
| §20.3 agent_quota_grant 写 ledger | `service/agent_customers.go` `IssueQuotaToCustomer` | `ac858617` |
| §20.4 自邀拦截 | `model/user.go` `ValidateInviterForRegistration`（年龄/域名/频率 3 重门槛）| `2577434b` |
| §20.5 ring + first-topup 拦截 | `model/user.go` `DetectInviteRing` + `GetUserTotalConsumeQuota` | `1b32c960` |
| §20.6 ReverseCommission | `model/commission.go` 新增 4 字段 + `service.ReverseCommission` 事务封装 | `457f0489` |
| §20.7 admin HTTP + cron | controller + router + service + main.go | `008c9a40` |
| §20.8 admin UI | `web/src/features/admin-commission/`（12 列 + 撤销对话框） | `e82cbf4b` |

测试增量：**+32 个新测试 + 6 个测试改动**，全包 0 FAIL。

### 2.2 task-17 深度清理

| 项 | 内容 |
|---|---|
| schema 收紧 | 索引重排 + 字段类型收敛（schema 文件 unified） |
| 测试夹具 | 移除老旧 DB 残留引用，统一用 task-17 fixtures |
| 安全 | SESSION_COOKIE_SECURE / TRUSTED_PROXIES / USER_SESSION_ACTIVE_LIMIT 等硬约束 |
| 文档同步 | master-plan / CHANGELOG / task-17 README 全部对齐到 main |

### 2.3 task-13 7 语言 i18n

| 语种 | 翻译完成度 | 备注 |
|---|---|---|
| en / zh-CN | 100% | 与 release 同步 |
| zh-TW | 100% | 单独维护繁体 |
| fr / ja / ru / vi | ~100% | task-13 后 3 周持续迭代；未翻译项走 i18n fallback |

前端改造：`web/src/i18n/` 7 个 locale 文件 + oxlint + tsgo -b 0 errors。

### 2.4 task-07 rebrand + 后续 ghcr.io 迁移

- `dukaworks/micross-api` 仓库命名生效（GitHub）
- CI 镜像从 Docker Hub 迁到 **ghcr.io**（commit `e4d14047` + `f92c7a05`）
- 不再需要任何 `DOCKERHUB_*` secrets，CI 用 GitHub 自动 token

### 2.5 docker-compose.prod.yml（新增）

`docker-compose.yml` 仍是 dev-friendly（含 build 块 / 123456 默认密码便于本地一键起）；
`docker-compose.prod.yml` 是生产硬化版（无 build 块 / 强制 .env / 端口 loopback / Redis AOF）。

### 2.6 patch v0.37.1.1 修复

4 个 commit 在 v0.37.1 上打的 patch：

| commit | 内容 |
|---|---|
| `27f87ea7` | 修 wallet 500（PG `boolean = integer`）|
| `083300ed` | 修 npm version strict semver |
| `4f70f689` | 修 electron-builder strict semver（trim package.json）|
| `825bc416` | 修 `--config.version` → `--config.extraMetadata.version` |

## 三、生产环境关键决策（已生效）

| 决策 | 理由 |
|---|---|
| **ghcr.io 而非 Docker Hub** | fork 仓库不配 secrets，用 GH 自动 token |
| **public package visibility** | VPS `docker pull` 不需要 PAT |
| **v0.37.1.1 镜像VERSION hardcode** | Dockerfile `ARG VERSION` 由 workflow 注入 |
| **Multi-arch manifest via buildx imagetools** | GHCR 自动 + cosign 双签 |
| **数据库三选一** (SQLite / Postgres / MySQL) | docker-compose.yml 默认 PG；SQLite 用于本地 / demo |
| **commission_records bool 字段** | 用了 bool + 用 `WHEN breach THEN` 而非 `breach = 1` 兼容 PG |

## 四、当前已知短板（v0.37.2+ 候选）

| 项 | 风险 | 建议行动 |
|---|---|---|
| bool 字段仍可能未全面审计 | 漏过一处 `= 1` 写 hard query 就在 PG 上 fail | 升级前跑 `git grep -nE '(breach\|reversed\|enabled)\s*=\s*[01]' -- '*.go'` |
| 4-segment tag 仍需 workaround | 任何 v0.37.X.Y 都得走 `npm pkg set` + `--config.extraMetadata.version` | 文档化在 `CONVENTIONS.md` §tag 规则 |
| Redis 未启用 AOF 的默认配置 | 容器重启会丢缓存（业务可承受，但 session / 风控查询会变慢）| 加 `--appendonly yes` |
| Electron 仅 Windows | macOS / Linux 没生成 | 加 `os: [macos-latest, windows-latest]` 到 matrix |
| `pg_data` 备份无自动化 | 靠 pg_dump 手动 | 加 cron（v0.37.2 候选） |
| 风控白名单无机制 | 误判需要 admin 手动 undo | v0.37.2 候选 |

## 五、基线声明

**今后任何升级改造 / bug 修复 / 新功能开发，必须以 `v0.37.1.1` 为起点。** 不允许基于任何更早 tag 做工作（除非明确做 hotfix 向后兼容）。新工作完成后通过以下流程进入下一阶段：

1. 在 main 上开 feature 分支（命名 `task-XX-<short-desc>` 或 `fix/<short-desc>`）
2. 完成后 PR + squash commit + 走 `.github/workflows/ci.yml` + `pr-check.yml`
3. 决定版本号（minor `v0.38.0` / patch `v0.37.2` / hotfix `v0.37.1.2`）
4. 打 tag → 自动触发 publish 链路

详细规则见 `CONVENTIONS.md`。