# 项目升级改造基本规则

> 本文档是本仓库的**永久治理文件**。升级、新功能、bug 修复、CI 改动、文档维护都按此执行。
> 任何与本文档冲突的临时操作，必须先更新本文档并提交，再实施改动。

## §1 版本与 tag 规则

### 1.1 基准

- **当前基准**：tag `v0.37.1.1`，commit `825bc416`。详见 `STAGE-SUMMARY-v0.37.1.1.md`。
- 升级改造必须从基准 HEAD 拉分支，**不允许**从更早 tag 切回做 hotfix（除非向后兼容且不破坏 main）。

### 1.2 版本号三段制

```
v<MAJOR>.<MINOR>.<PATCH>[-PRERELEASE][.HOTFIX]
```

| 类型 | 适用 | 例子 |
|---|---|---|
| MAJOR bump | 重大架构变更、不兼容 API、license 变化 | v1.0.0 |
| MINOR bump | 新功能 / 业务方案落地 | v0.38.0 |
| PATCH bump | 同一 MINOR 内的 bug 修复 | `v0.37.2` |
| HOTFIX suffix | **非语义版本合规**（4 段数字）；需走 CI workaround | `v0.37.1.1`、`v0.37.1.2` |

**严禁任意发挥**如 `v0.37.2-rc1` 或 `v0.37`（缺 PATCH）。PR 检查时如发现违规 tag 格式，CI 应拒收。

### 1.3 tag 触发的 CI 链路

```
git tag → push origin
   ↓
┌──────────────────┬──────────────────┬──────────────────┐
docker-build.yml   release.yml       electron-build.yml
│ → ghcr.io        │ → GitHub Release │ → Windows .exe
│ → multi-arch     │   + binary tgz   │   artifact
│ → cosign sign    │   + checksums    │
└──────────────────┴──────────────────┴──────────────────┘
                ↓
        shared: tag-triggered workflow, 不需要 secrets
```

`docker-image-branch.yml` 不参与 tag 触发，只手动 `workflow_dispatch` 用（alpha / nightly 分支）。

### 1.4 4-segment tag 的 workaround（必须遵守）

如果用了 `v0.37.X.Y` 形式，必须按 `electron-build.yml` 当前步骤走：

1. Step 8：`npm pkg set version="$STRICT_VERSION"` 其中 `STRICT_VERSION` 是从全 tag sed 出 3-segment（已经在 yml 脚本里）。
2. Step 10：`npx electron-builder --win --config.extraMetadata.version="$ECHO_FULL_VERSION"` 用 extraMetadata 覆盖回去。
3. `ECHO_FULL_VERSION` 在 Step 8 通过 `echo "ECHO_FULL_VERSION=$VERSION" >> $GITHUB_ENV` 传。

**禁止**用：
- `npm version <X>` （npm 7+ 拒绝 4-segment）
- `--config.version=<X>` （schema 不识别 root-level version）
- `--config.npm.version=<X>` （不存在此 schema 路径）

如果以后想抛弃 4-segment tag 风格，方案是：
- 改用 `v0.MINOR.PATCH-rc.N` prerelease 形式（`npm version` 接受），**或**
- 直接 bump MINOR：`v0.37.1.1` → `v0.38.0`（破坏性最小）

## §2 数据库 / ORM 规则

### 2.1 三种 DB 兼容

代码必须能跑 SQLite（dev / CI）+ PostgreSQL（prod）+ MySQL（可选）。**禁用任何只在一种 DB 工作而让其他 fail 的写法**：

| ❌ 写法 | ✅ 兼容写法 | 原因 |
|---|---|---|
| `WHERE bool_col = 1` | `WHERE bool_col` 或 `WHERE bool_col = true` | PG 严格类型拒绝 `boolean = integer` |
| `WHERE bool_col = 0` | `WHERE NOT bool_col` 或 `WHERE bool_col = false` | 同上 |
| `COALESCE(SUM(CASE WHEN breach = 1 THEN 1 ELSE 0 END), 0)` | `COALESCE(SUM(CASE WHEN breach THEN 1 ELSE 0 END), 0)` | 同上 |
| `SELECT * FROM t WHERE id IN (1,2,3)`（手拼）| 用 GORM `IN ?` + slice | 防 SQL injection + 类型化 |
| `WHERE JSON_EXTRACT(col, '$.k')` | 应用层解析 | MySQL / SQLite / PG JSON 函数签名不同 |

### 2.2 bool vs int 字段约定

仓库里两类约定并存：

| 模式 | 适用 | 例子 |
|---|---|---|
| `bool` | 需要在 Go 里做布尔运算 / 可读性优先 | `commission_records.breach`, `commission_records.reversed` |
| `int` (0/1) | 需要跨 DB 字面兼容 / 在 SQL 里 `= 1` 频繁 | `users.deleted`, `discount_routing_policy.allow_cost_breach`, `channels.status` |

**新加字段默认用 `int`**（除非有强理由用 bool）。理由见 `model/discount_routing_policy.go:32-35` 注释："三种数据库对布尔列行为不一致"。

### 2.3 GORM schema 变更

- AutoMigrate 加列前要 review GORM 字段顺序，避免被 GORM 重建整表。
- 加 enum / new index 必须用 `migration_notes` 文档化（v0.37.x 已有 master-plan.md §schema）。
- 大表加列（>10M 行）必须在低峰期跑 + 加 timeout。

## §3 CI / GH Actions 规则

### 3.1 Secrets 最小化

- 默认**不**配 Docker Hub / GHCR secrets——所有 publish workflow 用 `${{ secrets.GITHUB_TOKEN }}`（GH 自动给）。
- 必须配 secrets 时，写在 `Settings → Secrets and variables → Actions`，**只允许**：user / org 范围；不能用 environment secrets 跨 repo 共享。
- 任何 secret 名加 `*_TOKEN` / `*_KEY` 后缀，**不**用裸名。

### 3.2 Workflow file 命名

- `*.yml` in `.github/workflows/`，文件命名 `<scope>-<action>.yml`
- 触发器必须是 `on: push: tags:` / `on: push: branches:` / `on: workflow_dispatch:`，不混搭
- 每个 workflow 顶头注释：name + on + 调用关系 + 是否需要 secrets

### 3.3 镜像命名空间

- **一律用 `ghcr.io/dukaworks/micross-api`**。禁止 `docker.io/calciumion/new-api` 或 `ghcr.io/<user>/<repo>` 别名。
- 多架构 tag 完整：`v0.37.1.1-amd64` / `-arm64` / `v0.37.1.1`（manifest）。
- cosign 双签必须（oci image + oci index）。`sign-pro` 不需要。

## §4 测试规则

### 4.1 单元测试

- 改动必须附 `go test ./<package>`。
- DB 相关测试用 SQLite in-memory。**禁止** mock DB（mock 出来的查询跟实际 GORM 行为有 drift）。
- 时间戳 / UUID 等不可预测值必须用 interface 抽象，测试可注入 fake。

### 4.2 集成测试

- 涉及多包协作的功能必须有跨包测试（`model + service + controller`）。
- 关键流程必须 end-to-end：注册 → topup → 消费 → commission 写 ledger → admin 列表。
- TestData 用 `scripts/testdata/` 隔离，不在生产 DB 跑。

### 4.3 全包测试

- 上游 CI `ci.yml` 跑 `go test ./... -count=1 -timeout 180s`。
- 新提交导致任何包 FAIL 必须修到 green。

## §5 文档规则

### 5.1 治理文档（活跃）

| 文件 | 角色 |
|---|---|
| `.docs/README.md` | .docs 入口索引 |
| `.docs/CHANGELOG.md` | 全版本变更日志 |
| `.docs/master-plan.md` | 主方案 / 业务方案对齐 |
| `.docs/project-task-list-2026-09-14.md` | 任务清单 |
| `.docs/summary.md` | 项目摘要 |
| `.docs/STAGE-SUMMARY-v0.37.1.1.md` | 当前阶段总结 |
| `.docs/CONVENTIONS.md` | **本文档**（治理规则）|

任何修改这些文档，必须**先 PR 讨论 → 合入 main**，不留"私有草稿"在仓库里。

### 5.2 工作文件（任务周期内）

- 每个 task 一个 `task-XX-<name>/` 子目录（活跃任务）
- 完成后整个目录移到 `_archive/task-XX-<name>/`
- 不在工作目录里留临时调试 log / 浏览器 profile（已在 .gitignore）

### 5.3 发布说明（每版本）

- `.docs/release/v<X.Y.Z>.md`（必须）
- `.docs/release/v<X.Y.Z>.md` + `.docs/release/v<X.Y.Z.W>.md`（patch 复用前一版本加 hotfix 段即可）
- 内容：发布日期 / commit / 业务方案对齐表 / 新功能 / 升级指南 / 回滚预案 / 监控指标 / 测试覆盖 / commit 表

### 5.4 部署文档（活跃）

- `.docs/deploy/docker-vps-quickstart.md`（必须）
- 新增部署方式（如 k8s / helm chart）单开 `.docs/deploy/<mode>.md`

## §6 Docker / 发布规则

### 6.1 镜像生产路径

```
main HEAD
  ↓ (git tag → push)
GH Actions: docker-build.yml
  ↓
amd64 + arm64 → ghcr.io/dukaworks/micross-api:<TAG>-{amd64,arm64}
  ↓
create_manifests job
  ↓
ghcr.io/dukaworks/micross-api:<TAG>     # multi-arch manifest
ghcr.io/dukaworks/micross-api:latest   # 同上指向最近 release
```

### 6.2 Dockerfile 守则

- 多阶段 build（builder + runtime）保持
- **ARG VERSION** 必须由 workflow `--build-arg VERSION=<TAG>` 注入（不用 `local` 默认）
- alpine / debian-slim 优先；超过 500MB 的 layer 要审查
- 基础镜像 SHA pin（`oven/bun:1@sha256:...`）

### 6.3 docker-compose 守则

- `docker-compose.yml`：dev 友好（默认密码 + build 块），本地一键起
- `docker-compose.prod.yml`：生产硬化（env 强制 + no build + loopback port）
- 容器**不**直暴露公网；前端 nginx/caddy 反代
- 数据库容器必须有 healthcheck + depends_on condition: service_healthy

## §7 安全性

### 7.1 Secrets

- 代码里**禁止**硬编码 token / password / key
- `docker-compose.prod.yml` 的所有密码从 `.env` 读，`.env` 设 `chmod 600` 并加入 .gitignore
- AGENTS.md 已有的："never write passwords to docker-compose.yml 默认值"原则继续生效

### 7.2 认证

- `SESSION_COOKIE_SECURE=true` 必须在 HTTPS 上线后启用
- `SESSION_COOKIE_TRUSTED_URL` 列精确 Origin（不 wildcard）
- `TRUSTED_PROXIES` 列精确 CIDR（不能用 `0.0.0.0/0`）
- `USER_SESSION_ACTIVE_LIMIT=50` 等限流配置按生产实情调

### 7.3 输入校验

- Controller 层必须 `ShouldBindJSON` + `Validator` 校验后再到 service
- GORM 写入前在 model 层做字段长度 / 范围 / 业务规则校验
- 日志记录敏感字段时（password / token）必须 redact

## §8 提交 / 分支 / PR 规则

### 8.1 Commit message

```
<type>(<scope>): <short summary>

<optional body — what & why, no how>

<optional footer — refs, breaking changes>
```

| type | 用途 |
|---|---|
| `feat` | 新功能 |
| `fix` | bug 修复 |
| `refactor` | 代码重构（不改变行为）|
| `docs` | 文档 |
| `test` | 测试 |
| `ci` | CI / workflow |
| `chore` | 杂项（依赖 / 注释 / 格式）|
| `perf` | 性能 |
| `revert` | 回滚 |

### 8.2 分支

- `main`：稳定分支，每版本一个 commit
- `task-XX-<desc>`：任务开发（生命周期：合入 main 后删除）
- `fix/<short>`：hotfix 用，从 main HEAD 拉
- **不允许**长期分支（>2 周未合 main）

### 8.3 PR 检查

- 必须通过 `ci.yml` + `pr-check.yml`
- Reviewer 至少 1 人（项目目前是单人维护，需外部 review 时再调）
- Squash merge，不保留中间 commits

## §9 文档 / 规则变更

任何对本文档的修改：
1. 必须提 PR 写清楚改了什么 + 为什么
2. 改动要 commit 到 `v0.37.1.1` 之后的一次 task
3. 重大变更要在 `.docs/CHANGELOG.md` 加一条"治理规则更新"项

## §10 例外条款

以下场景**必须**显式讨论后才能做：
- 改 AGPLv3 license header（涉及上游）
- 升级 Go major version（可能 break 大量代码）
- 加新数据库方言（MySQL / ClickHouse 等）
- 引入新外部服务依赖（Sentry / Stripe / Slack）

规则文档结束。下次改动请直接编辑本文档并 PR。