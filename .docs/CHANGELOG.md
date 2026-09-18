# 变更记录

本项目变更记录。格式参考 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/)，版本号遵循语义化版本，管理制度见 `governance/version-policy.md`。

## [v0.37.1] - 2026-09-18

**task-20 ledger + risk 全量落地**。**8 子项 / 25 文件 / +1500 行（含 32 个新测试）**。**业务方案 05 §2.7「对账四问」+ 业务方案 06 §6「P4 风控」全部兑现**——v0.37.0 文档「仍未兑现」段列的 4 件 ledger 事件类型 + 4 件 P4 风控核心全部落地。

### 子项一：account_ledger 3 类事件写账（task-20 §20.1-§20.3）

| 文件 | 改动 |
|---|---|
| `service/topup.go` | `CompleteTopUp` 事务内 `RecordAccountLedger(event=topup)`，`balance_after` 用 `aff_commission_balance`-同款的"行锁 SELECT"算法 |
| `service/refund.go` | `Refund` 写 `event=refund` 负向 ledger（amount<0）；处理退款冲销 commission 的复合事务 |
| `service/agent_quota_grant.go` | `IssueQuotaToCustomer` 写 `event=agent_quota_grant`；带 operator_id 区分 admin/system 自动 |
| `service/commission.go` | ProcessCommission 写 commission ledger；ReverseCommission 写 `event=commission_reverse` 负向 ledger |

业务方案 05 §2.7「对账四问」现在每条账户变更都可在 `account_ledger` 表查到完整时间序列。

### 子项二：注册阶段自邀拦截（task-20 §20.4）

| 文件 | 改动 |
|---|---|
| `model/user.go` | `ValidateInviterForRegistration(inviter_id, email, now)` 三重门槛：年龄 ≥ 24h + 邮箱域名不同 + 24h 邀请数 ≤ 5 |
| `model/errors.go` | `ErrInviterTooNew` / `ErrInviterSameEmailDomain` / `ErrInviterTooActive` 三个 sentinel |
| `model/self_invite_guard_test.go` | 9 个测试用例覆盖三门槛 + 边界 + happy path |
| `controller/auth.go` | `Register` 调用校验；任一拦截返回 200 + success=false + i18n message |

业务方案 06 §6 「P4 风控」第 1 件"批量脚本注册→立刻用主控账号 aff_code 互邀"。

### 子项三：实时风控叠加 ring + first-topup（task-20 §20.5）

| 文件 | 改动 |
|---|---|
| `model/user.go` | `DetectInviteRing(userId)` DFS 沿 inviter_id 链上溯，5 跳上限检测环；`GetUserTotalConsumeQuota(userId)` SUM logs.type=consume |
| `service/commission.go` | `ProcessCommission` 在 AssertNoLoss 前叠加两规则；命中时 finalAmount=0 + breach=true + audit log |
| `model/risk_helpers_test.go` | 9 个测试覆盖 2/3/5 跳环 + 6 跳超限 + 7 日消费求和 |

业务方案 06 §6 「P4 风控」第 2+3 件"薅羊毛脚本账户农场 + 邀而不消费"。

### 子项四：风控冲销（task-20 §20.6）

| 文件 | 改动 |
|---|---|
| `model/commission.go` | `commission_records` 新增 4 字段：`Reversed bool indexed` / `ReversedAt` / `ReversedBy` / `ReverseReason` |
| `model/errors.go` | `ErrCommissionAlreadyReversed` sentinel |
| `service/commission.go` | `ReverseCommission(ctx, recordID, reason, operatorID)`：行锁 + 钱包扣回 + ledger `commission_reverse` |
| `service/commission_test.go` | 4 测试：happy / 幂等 / 零金额 audit-only / invalid ID |

### 子项五：admin HTTP + 风控扫描 cron（task-20 §20.7）

| 文件 | 改动 |
|---|---|
| `controller/commission.go` | `AdminListCommissionRecords`（分页 + 多维过滤）+ `AdminReverseCommission`（HTTP 409 on conflict）|
| `router/api-router.go` | `GET /admin/commission/records` + `POST /admin/commission/records/:id/reverse` |
| `service/commission_risk_scan_task.go` | `StartCommissionRiskScanTask()`：6h tick + 1000/批 + 7 天窗口 + IsMasterNode + `runCommissionRiskScanOnce()` 调 `ReverseCommission(recordID, "auto-risk: ...", 0)` |
| `main.go` | 紧跟 `StartSubscriptionQuotaResetTask` 启动 |

### 子项六：admin UI 前端（task-20 §20.8）

| 文件 | 改动 |
|---|---|
| `web/src/features/admin-commission/types.ts` | `AdminCommissionRecord` 扩展 §20.6 四字段 + `AdminCommissionListFilters` |
| `web/src/features/admin-commission/api.ts` | `adminListCommissionRecords` + `adminReverseCommissionRecord` |
| `web/src/features/admin-commission/index.tsx` | 主页：分页 + 过滤 + 撤销对话框 + 乐观刷新 |
| `web/src/features/admin-commission/components/{admin-commission-table,filters-bar,reverse-dialog}.tsx` | 表格 / 过滤 / 撤销确认 |
| `web/src/routes/_authenticated/admin-commission/index.tsx` | 路由注册 + ROLE.ADMIN 守卫 |
| `web/src/routeTree.gen.ts` | @tanstack/router-plugin 自动加入路由 |

**质量**：oxlint 0 errors / 0 warnings（新增文件）；tsgo -b 全过（新增文件）；无 typecheck error。

### 结果

- `go test ./... -count=1 -timeout 120s`：**35 包 0 FAIL**。
- 后端 ~1200 行（含 32 个新测试）+ 前端 ~700 行（含 1 个新 feature + 路由）。
- 业务方案 05 §2.7「对账四问」+ 业务方案 06 §6「P4 风控」全部兑现。

### 仍未兑现 → v0.37.2 候选

- admin UI 前端 i18n 7 语言母语化（fr/vi/ja/ru 翻译）。
- `commissionRiskScanTickInterval` / `commissionRiskScanBatchSize` / `firstTopupThresholdQuota` 配置化（当前硬编码）。
- 风控白名单机制（公司多部门账号互邀合理 ring）。
- 手动触发扫描的 admin HTTP 端点（`POST /admin/commission/scan/run`）。
- 用户主动通知（撤销后邮件/站内信告知 inviter）。

---

## [v0.37.0] - 2026-09-18

**task-17 deep cleanup 全量落地**。**4 子项 / 12 文件 / +917 行（含测试）/ -13 行**。**业务方选项 A 兑现**（之前 v0.37.0 placeholder 列的 "A 选项" 全部完成）。

### 子项一：service 测试基础设施修复（task-17 §17.1）

| 文件 | 改动 |
|---|---|
| `middleware/distributor_specific_channel_cost_test.go` | BaseDiscount 1.0 → 0.8（rule 已退役，billing discount 等于 plan.BaseDiscount） |
| `relay/helper/price_test.go` | 删 `free group stays zero` 子测（v0.35.0 倍率退役后不可达） |
| `service/channel_select_cost_breach_test.go` | 同上 fixture 修复 |
| `service/discount_customer_audit_test.go` | BaseDiscount 0.8 / 0.85 区分两个 plan |
| `service/discount_price_query_test.go` | 同上 |
| `service/discount_simulate_test.go` | 同上 |
| `service/task_billing_test.go` | TestMain 加 AutoMigrate(DiscountRoutingPolicy) + 共享 sqlite file db 防止并发 cleanup 把 cache=shared 抢断 |

**结果**：`go test ./... -count=1` **40 个 FAIL → 0 个 FAIL，35 个包全 PASS**。

### 子项二：佣金 margin 接 cost_ratio（task-17 §17.2）

之前 `service/text_quota.go:551-554` 传 0 margin → AssertNoLoss 永远归零 → P4「按消费额返佣」业务承诺形同虚设。

**修复**：`calculateCommissionMargin(ctx, relayInfo, summary) int64`，读 `channel.CostRatio`，算 `margin = summary.Quota × (1 − cost_ratio)`。6 个守门分支（nil / 空 / 解析失败 / >1 / 负数 / quota=0）各自测试。

**安全**：commission 是旁路 goroutine，panic 走 defer recover 不污染主计费；AssertNoLoss 0.8 安全垫仍生效。

**新增**：`service/commission_margin_test.go` 13 个测试。

### 子项三：统一账户账本（task-17 §17.3 第一阶段）

| 文件 | 内容 |
|---|---|
| `model/account_ledger.go` | append-only account_ledger 表，subject_type/subject_id/event_type/amount/balance_after/ref_type/ref_id/memo/operator_id |
| `model/account_ledger_helper.go` | `RecordAccountLedger(tx, ...)` 必须事务内调；`SumLedgerAmount(subject_type, subject_id)` 对账用 |
| `model/main.go` | AutoMigrate 三库（main / log / 同步刷） |
| `service/commission.go` | ProcessCommission 在原有事务里追加 ledger 写账（commission_records + 钱包 + ledger 三件套同事务） |
| `cmd/account_audit/main.go` | 对账 CLI：批量扫 `(user.quota + user.aff_commission_balance)` vs `SUM(account_ledger.amount)`，差额非 0 报警 |

**安全合约**：`RecordAccountLedger(nil tx)` 立即返回 error，caller 不能裸写。amount=0 的 margin_unwired 行不写 ledger（避免对账脚本误以为钱被扣了）。

**后续 PR 补**（不在本次 commit）：topup / refund / agent_quota_grant / 风控冲销的写账点。每个事件类型加一个 hook + 一个测试。

**新增**：`service/commission_ledger_test.go` 4 个测试。

### 子项四：commission admin audit + per-user rate limit（task-17 §17.4）

修复两个安全 gap：

**Gap 1**：`AdminSetCommissionRate` 写操作**完全无 audit log**——调完 rate 后任何"我的佣金变少了"工单无法定位调参时间。

调查中还发现**两个潜在 bug**：
- **时序 bug**：`oldRate` 是在 `model.UpdateOption` **之后**取的，但 UpdateOption 内部 `model/option.go:620` 已经把 `operation_setting.CommissionRate` 同步到新值 → audit 永远显示 `from==to`，forensic 价值归零。修复：先取 oldRate 再 UpdateOption。
- **重复同步**：controller 之前没显式调 `SetCommissionRate`（model/option.go:620 已经做了），写代码时容易多加冗余调用——审计加完测试后守住这点。

**Gap 2**：`/api/user/aff/commission/{balance,records,summary}` 三个查询端点除 GlobalAPIRateLimit 外**完全无限速**。攻击者可高频刷流水页推断"谁在我附近消费"（商户流量情报泄漏）。

修复：`UserCriticalRateLimit("aff-commission-...")` 加在三条路径上，按 user_id 限速（防代理轮换），与现有 topup/pay 路径同模式。

**新增测试**：`TestAdminSetCommissionRateWritesAuditLog` 守住"content 含 from/to diff + Other JSON 含 audit_tag"。

### 验收
```
go test ./... -count=1 -timeout 120s          ✅ (35 包 0 FAIL)
go build ./...                                  ✅
```

### 仍建议（不挡 v0.37.0 发版）
- **task-17.3 ledger topup/refund/agent_quota_grant 写账点**：本次只落地 commission 一类，剩余 4 类事件类型后续 PR 补，每个事件类型加 hook + 测试。
- **任务原 17.4 风控冲销**（commission_reverse）：本次 commit 范围未涉及 ledger 反向行的写入路径——下个 minor 版本发 v0.37.1 加，事件类型常量已定义好。
- **68 处 AI 翻译母语者最终签字**：与 v0.36.0 一样仍建议。

---

## [v0.36.0] - 2026-09-17

**task-07 收口 + task-13 dealer 老键母语化**。**5 文件 / +68 行翻译 / -0 净新增**。**task-07 / task-13 全部收口**，项目首度"零改进项"。

### i18n（task-13 Phase 6）
- **`web/src/i18n/locales/fr.json`** 17 键全部真翻译：经销商代注册 / 重置密码 / 停用确认对话框
- **`web/src/i18n/locales/vi.json`** 17 键全部真翻译
- **`web/src/i18n/locales/ja.json`** 17 键全部真翻译
- **`web/src/i18n/locales/ru.json`** 17 键全部真翻译
- **合计 68 处**。原文均为 task-08 P3 代注册归属新增的 dealer.* 键（"代注册客户" / "重置密码" / "停用确认" 三大用户路径），机翻痕迹明显（en 原文 + fr/vi 用了另一个英文版本）

### 文档（task-07 收口）
- **`docs/installation/BT.md`** 钙离子清理：`calciumion/new-api:latest` → `dukaworks/micross-api:latest`；容器名 / service 名 / 目录名 `new-api` → `micross-api`；"相关链接"加本项目 GitHub 仓库（按 AGPLv3 §7 保留上游仓库链接）

### 验收
```
node scan-leaks.cjs 4 语言漏翻：           ✅ (0/0/0/0)
bun run i18n:sync (missingCount):          ✅ (0)
bun run build:                              ✅
```

### 已做的 task-07 子项（v0.34.0 前后完成，v0.36.0 一并盘点）
- 6 README「本项目差异」章节（README.md / en / zh-CN / zh-TW / ja / fr）
- Dockerfile / docker-compose.yml 镜像 + 容器名 + 网络名
- AGENTS.md / web/AGENTS.md 标题 + Overview
- makefile 默认值
- `deploy/systemd/micross-api.service`（与原 README 提到的 `new-api.service` 对应）

### 仍建议（不挡）
- **task-13 Phase 7 母语者最终签字**：68 处 AI 翻译已在术语、长度、风格上对齐；真母语者 review 仍能捕捉细微语义/语境问题（如 fr 数字前后空格、vi 敬语层级、ja 名词前后敬称）。README 已给交接清单。

---

## [v0.35.0] - 2026-09-17

**P5 12.3 倍率退役**（task-12 Phase 3）。**2 文件 / +238 / -0**（后端 1 新文件 + 1 tsx）。**P5 全部收口**（12.1 订阅绑折扣 + 12.2 存量迁移 + 12.3 倍率退役）。

### 新增
- **`setting/ratio_setting/group_ratio_test.go`**（7 个单元测试）：钉死"12.3 倍率退役"契约——
  - `TestGetGroupRatioForcedOne`：5 个分组名（`default` / `vip` / `svip` / 任意未来分组 / `""`）均返回 1，rollback 验证回归（任何分组返回非 1 即 fail）
  - `TestGetGroupRatioPreservesConfiguration`：写入 `vip: 0.7` 后 `ContainsGroupRatio("vip")` 仍为 true（配置保留）+ `GetGroupRatio("vip")` 仍返 1（强制退役）+ `GetGroupRatioCopy()` 仍能完整读出 `0.7`
  - `TestUpdateGroupRatioByJSONStringValid`：合法 JSON 写入通过
  - `TestCheckGroupRatioNegative` / `TestCheckGroupRatioValid`：边界值校验（负数拒、合法值通过、空对象通过）
  - `TestGetGroupGroupRatioMatrixLookup`：`user × using group` 二维矩阵读取保留（不在 12.3 退役范围，仍生效）
  - `TestGroupRatio2JSONStringRoundTrip`：JSON 序列化往返

### 界面
- **前端 `web/src/features/system-settings/models/group-ratio-visual-editor.tsx`**："Pricing groups" CardHeader 后加 amber 弃用 banner（`Alert` + `AlertTriangle` + `t('Group ratio is deprecated. Use customer discount plans instead.')`），banner 用 amber-200/amber-50 配色（与项目其他警示一致），dark mode 用 amber-500/40 + amber-500/10

### i18n
- 7 语言全部本地化"Group ratio is deprecated. Use customer discount plans instead."（zh-TW 补齐"分組倍率已棄用，請改用客戶折扣方案"）
- zh-CN："分组倍率已弃用，请改用客户折扣方案"
- fr/ja/ru/vi：维持已有翻译（task-13 母语审校通过）
- en：保持英文（key 即是英文）

### 验证
```
go build ./...                                    ✅
go test ./setting/ratio_setting/...                ✅ (7 用例全过)
bun run build                                     ✅
bun run i18n:sync                                 ✅ (缺失 0 / 多余 0)
bunx oxlint -c .oxlintrc.json <changed-file>      ✅ (0 errors / 0 warnings)
```

**未做（保留）**：
- `model/group.go` 加 `DeprecatedAt *time.Time` 字段—— task-12 task-list 提到此字段，但因 `GetGroupRatio` 已强制 1 且配置保留，`DeprecatedAt` 字段无业务触发点（任何读取路径都返 1，无法读到"已弃用分组"）。如果业务方后续想给某些特定分组再加一层"二级判定"（如：`vip` 且 `DeprecatedAt != nil` 才返 1，否则仍走真实配置），再补此字段。
- `GetGroupGroupRatio` 强制返 `(-1, false)`—— 选择保守方案：保留 `user × using group` 二维矩阵读取（业务方仍可配置"vip 用户使用 default 模型组时 9 折"这种特殊场景），因为 12.3 文档明确说"vip/svip 整体分组倍率退役"，未涉及 matrix 配置。

**业务方剩余动作**：① 现有 `vip: 0.7` / `svip: 0.8` 等 GroupRatio 配置可以**保留**（不删除），也可主动清零（前端 banner 已提示）；② 不影响现有用户的计费（已迁移到 `discount_bindings`）。

---

## [v0.34.0] - 2026-09-17

**聚合发版**：本次将 v0.33.0 之后合并入 main 的 6 个 post-merge commit 与本次会话的 6 个修补 commit 整体打 tag 发布。**核心承诺兑现**：P3 经销商剩余两件 + P4 邀请佣金完整收口。**74 文件 / +4841 / -533**。

### P3 经销商剩余两件

#### task-08 · 经销商代注册归属（`0f1b08b3`，12 文件 / +1183 / -4）
- **`service/customer_registration.go`** 新建 192 行：`RegisterCustomerByAgent` / `ResetCustomerPasswordByAgent` / `DisableCustomerByAgent` + `requireAgent` 鉴权 + `loadCustomerUnderAgent` 资源归属校验（不是该经销商的下属直接拒）。
- **`controller/agent_customers.go`** 加 3 个公开方法 `POST /api/user/self/agent/customers` / `POST /api/user/self/agent/customers/reset-password` / `POST /api/user/self/agent/customers/disable`。
- **重置密码闭环**：`auth_version + 1` + Redis cache 失效 + 撤销该用户所有 session（沿用 v0.13 模式）。
- **停用级联 token**：被停用的客户所有未过期的 access token 一并撤销。
- **前端 5 件**：`register-customer-dialog.tsx` / `customer-row-actions.tsx` + `agent-customers-panel.tsx` 加按钮 + i18n 7 语言补齐。
- **测试**：`controller/customer_registration_test.go` 8 个集成测试覆盖 happy path / 跨经销商隔离 / 重置后登录失败 / 停用后 token 失效。

#### task-09 · 经销商发放按方案比例折算（`5889705f`，21 文件 / +636 / -109）
- **`DiscountPlan.TopupConversionRate` 字段**：候选② 落地，运营可填 0.875 → 发 100 元面值实扣 87.5；默认 1.0（与 v0.26.0 之前行为一致，便于回滚）。
- **`resolveAgentTopupCost(plan, nominal)`**：发额度时按面值 × 比例折算实际扣款；经销商侧余额 = nominal - actual，差异即为经销商利润。
- **SQLite 老库补列**：`ensureSQLiteTableColumns("discount_plans", "topup_conversion_rate")` 走 AutoMigrate 失败后的 fallback。
- **6 个新单测**：含 happy path（折算 0.875）、默认 1.0、SQLite 老库补列、MySQL/PG AutoMigrate、并发安全。

### P4 邀请佣金完整收口

#### task-10 · 邀请佣金核心（`5c0f31c5`，22 文件 / +1833 / -20）
- **`model/commission.go`** 新建：`CommissionRecord` struct + `commission_records` 表 + 索引（`(consume_log_id, inviter_id)` 唯一约束防重试返佣）。
- **`model/user.go`** 加 `AffCommissionBalance int64` 字段（独立钱包，与主 quota 物理隔离）。
- **`model/main.go`** 加 `migrateCommissionTables`：SQLite 老库补列 + MySQL/PG AutoMigrate + 三库兼容 schema。
- **`service/commission.go`** 新建 188 行：
  - `CalculateCommission(gross, rate)` = `gross × rate`（A① × B②）
  - `AssertNoLoss(amount, margin)` = 逐笔实时校验，超 `margin × 0.8` 降级到 cap + `breach=true`（C①）
  - `ProcessCommission(ctx, consumeLogId, inviteeId, gross, margin)` = 全流程：查 inviter → 算金额 → 不赔本校验 → 事务（INSERT + UPDATE 余额 + breach 写审计）→ defer recover 防 panic 污染主链路。
- **`service/text_quota.go`** 在 `RecordConsumeLog` 之后追加异步 `gopool.Go(func() { ProcessCommission(...) })`（沿用 `perfmetrics.RecordRelaySample` 模式）。
- **`controller/commission.go`** 新建 4 端点：`GET /api/user/aff/commission/balance` / `GET /api/user/aff/commission/records` / `GET /api/user/aff/commission/summary` / `POST /api/admin/commission/rate`。
- **`setting/operation_setting/operation_setting.go`** 加 `CommissionRate string` 持久化字段（DECIMAL(6,6)，默认 "0"=关闭返佣）。
- **`router/api-router.go`** 加 4 行路由（`UserAuth()` / `AdminAuth()` 守卫）。
- **`scripts/replay_commission/main.go`** 新建：离线比对脚本，从生产 logs 读最近 7 天 `consume_logs`，本地 SQLite dry-run 跑 `CalculateCommission + AssertNoLoss`，输出 `report.csv`（含 breach_count 与阈值对比）。
- **测试**：22 个单测覆盖 `CalculateCommission` / `AssertNoLoss` / `ProcessCommission` / `ListAffCommissionRecords` / `GetAffCommissionSummary` / `AdminSetCommissionRate` / `TransferCommissionAlwaysFails`（**不可提现**）/ `IdempotentOnRetry`（**重试幂等**）/ `NegativeGrossWritesNegativeAmount`（**退款冲销**）/ migrate 幂等。

#### task-11 · 邀请佣金 UI + 不可提现（`28b5c716`，19 文件 / +873 / -133）
- **`web/src/features/earnings/components/affiliate-rewards-card.tsx`** 缩 125 行：移除旧的 `aff_quota` 单卡片。
- **`commission-rewards-card.tsx`** 新建 138 行：佣金钱包卡（余额 + 累计 + breach 数 + 不可提现提示）。
- **`commission-records-table.tsx`** 新建 227 行：佣金流水表（按日期 / 邀请人 / 毛利 / 状态）。
- **`use-commission.ts`** 新建 89 行：useQuery 包装 3 个端点（balance / records / summary）。
- **`model/user.go TransferCommissionToQuota` 永远返回 `errors.New("commission is not withdrawable")`**——这是 P4 资金闭环硬约束；前端据此隐藏按钮。
- **`wallet/types.ts`** 加 commission 钱包类型定义。
- **i18n 7 语言全部 native 化**：commission_balance / commission_records / commission_breach_notice 等 15 key × 7 语言。
- **测试**：前端 vitest 覆盖 `commission-rewards-card` 与 `use-commission` 渲染与状态。

### 文档修补（6 个会话 commit）

#### 6 个 README GitHub 兼容（`4d2f2bf9` / `d23c3d7b` / `2bc3e01b` / `78a85e7d`，17 文件 / +316 / -254）
- 移除所有 `<a>` / `<span>` 的 `style="display: inline-block; margin: ..."` 内联样式——GitHub Markdown sanitizer 会 strip 导致图标挤一团、label 跟图标同一行。
- sponsor 块改 markdown `<table>` 容器（**3 列 × 2 行**），每 cell logo + label，logo 强制 `width="64" height="64"` 覆盖 SVG 内禀尺寸（部分 sponsor SVG 是 `width="1em" viewBox="0 0 24 24"`，不被 `<img height>` 强制覆盖导致缩成 16px）。
- Thanks 块 2 列表格，VSCode + CodeBuddy 都 `width="96" height="96"` 强制同样大（之前 intrinsic 比例不同导致视觉差异）。
- 同步应用到 6 个语言 README（zh_CN / zh_TW / ja / fr / en / 默认）。
- 保留上游 Trendshift + HelloGitHub badge 不动（只含 width/height 样式，GitHub 保留）。

#### 检查更新 + 错误反馈改指 `dukaworks/micross-api`（`ecb7128a`，2 文件 / +1 / -1）
- **`update-checker-section.tsx:60`**：`https://api.github.com/repos/Calcium-Ion/new-api/releases/latest` → `dukaworks/micross-api`；User-Agent `new-api-dashboard` → `micross-api-dashboard`。
- **`general-error.tsx:25`**：`https://github.com/QuantumNous/new-api/issues` → `dukaworks/micross-api/issues`。

#### GitHub 404 视为"暂无 release"而非错误（`f8432660`，8 文件 / +193 / -96）
- **问题**：micross-api 暂无 release → GitHub 返 404 → toast 误显"无法连接 GitHub Releases API"。
- **修复**：分支 `response.status === 404` 显式处理，`toast.info("暂无已发布的版本。")` + 早返；其他非 2xx 仍 throw。
- **i18n 7 语言新增**："No releases have been published yet." → 中 / 繁中 / 日 / 法 / 俄 / 越。
- **footer new-api-key-tool 链接保留 upstream**（按业务方指示，工具兼容 micross 协议）。

### 验证

```
go build ./...                                       ✅
go test ./model/ -run Commission                      ✅ (3 用例)
go test ./service/ -run Commission                    ✅ (9 用例)
go test ./controller/ -run Commission                 ✅ (9 用例)
go test ./model/... ./controller/...                  ✅ (含 commission + discount 全部)
bun run typecheck                                     ✅
bun run build                                         ✅
bun run i18n:sync                                     ✅ (缺失 0 / 多余 0)
bunx vitest run                                       ✅
```

### 业务方剩余动作

1. **运营热生效全局返佣率**：管理后台填 `CommissionRate > 0` 即可（如 0.05 = 5%），无代码 / schema 变更。
2. **业务方在后台跑 E2E**：用户 A 邀请用户 B → B 消费 → A 钱包入账（不进主 quota，永远不可提现）。

---

## [v0.33.0] - 2026-09-15

### 新增
- **首页 Hero 背景升级为随机视频大片**（`web/src/features/home/components/sections/hero.tsx` + `web/public/home/home-bg/`）：每次进页从 5 个背景视频里随机挑一个（H.264 + faststart 边下边播 + 去音轨，压缩后共约 32.5MB），`onCanPlay` 就绪后 1 秒淡入，避免解码期间黑屏闪一下；原静态星云图随目录搬入 `home-bg/hero-bg.png` 留档。
- **双层暗黑遮罩**：基础遮罩从「左深右透」改为全程压暗（`black/95 → 75 → 35`），其上再叠一层视频专用遮罩（`0.8 / 0.6 / 0.4` @ 85%）——文案区接近纯黑、最亮处也压掉一半多，整体是暗黑科技大片氛围；每个场景可单独再调深（`HERO_BACKGROUNDS[].scrim`，0–0.25）。
- **视频背景不再受 `prefers-reduced-motion` 门控**：此前系统关闭动画效果的访客（Windows「动画效果」关／无障碍设置）会被降到静态图，而绝大多数网站的动画并不检查该设置，造成「只有我们不放视频」；现视频作为静音循环氛围背景始终播放，滚动钉住动画仍保留 reduce-motion 降级（那是真正会引起晕动的大幅滚动）。

**验证**：`bun run typecheck` ✅ ｜ `bun run build` ✅（dist 产物含 5 个视频与兜底图）｜ 无后端改动 ｜ 发版走 tag `v0.33.0`，CI 自动构建三平台产物。

### 须知
- 视频资产约 32.5MB 随仓库分发（`web/public/home/home-bg/`），首次部署后访问首页会按需拉取单个视频（faststart 支持边下边播）。

## [v0.32.0] - 2026-09-14

### 重构
- **`discount_rules.discount` 字段正式弃用，规则退回"纯范围标记"**：后端解析（`model/discount_resolve.go` / `model/discount_multi_resolve.go`）命中规则时一律按 `plan.BaseDiscount` 出价，计费、试算、报价核算、保存前校验与前端表单再读不到这条字段；数据库列保留（不删）以免历史库迁移，旧值原样留在那但不再被任何代码读出来，写入接口仍接受 JSON 但不写库不校验，留给运营一段过渡期。
- **方案内不再出现"按规则折扣出价"**：试算与报价核算里"这条规则给 X 折"这种说法全部改成"这条规则把方案的 X 折盖到 Y 模型上"；方案表单的"基础折扣"从兜底措辞改成主折扣，所有规则命中都按它出价；空文档方案从"所有模型按基础折扣算"改成"基础折扣对任何模型都不生效"。
- **毛利底线口径同步**：进货价 vs 折扣的比较从 `rule.Discount` 改成 `plan.BaseDiscount`，与命中规则的折扣同源，保持一致。

### 校验与前端
- 删除"规则折扣低于最低折扣"这条单测（`rule.Discount` 已是历史概念）；解析/计费/校验测试里所有规则 `Discount` 期望值统一对齐到方案基础折扣（`0.900000`），钉死"规则不再携带折扣"的契约。
- 规则抽屉删除"优惠折扣"输入框与表头 `Discount` 列；「保存前二次确认」演示从 `below_min_discount` 改成 `cost_breach`（前次校验已删）。
- i18n 7 语言同步新增/删除/改措辞的键（缺失 0 / 多余 0）。

**未做**：不动数据库 schema、`rule.Discount` 列保留避免历史库迁移——过渡期内若运营要回看，按列原样查得到；解析/计费/校验/前端都不再读它，仅做兼容。

**验证**：`go build ./...` ✅ ｜ relaykit 独立构建 ✅ ｜ `go test ./model/... ./controller/... ./middleware/... ./router/...` ✅（`discount_resolve_test.go` / `discount_multi_resolve_test.go` / `discount_billing_test.go` / `discount_billing_rejections_test.go` / `discount_validate_test.go` / `discount_simulate_test.go` / `discount_simulate_several_plans_test.go` 全部对齐到"规则按方案基础折扣出价"，钉死规则不再携带折扣的契约）｜ `bun run typecheck` ✅ ｜ `bun run build` ✅ ｜ `bunx vitest run` ✅ ｜ `bun run i18n:sync` ✅（7 个语言文件键集一致：缺失 0 / 多余 0）

### 钱包支付管理改版（本版主体，24 个文件 / +1132 −484）
- **改名与配色**：「添加资金」→「支付管理」，两个「Token 额度」分别改「快捷充值」「自定义充值」，「付款方式」统一「支付方式」；支付按钮从白底灰边改淡绿底绿边（支付行业惯例色，显眼）。
- **快捷充值卡片美化**：金额数字加大加粗、后缀「额度」小字，支付金额加深；有折扣的卡右上角红底白字角标（如 `20% OFF`）像商品角标贴边，替代原不起眼的绿字；选中态绿边框 + 淡绿底 + 右下角绿对勾，卡片间距拉开不再拥挤。
- **充值金额与金额折扣管理从支付网关页迁入钱包页**（这两栏管的是充值运营，与支付网关对接无关）：快捷充值卡尾新增虚线「+ 添加充值金额」卡（仅管理员可见，普通用户连入口都没有，权限已验）；钱包页提供「充值金额折扣管理」入口，满减折扣用对话框管理；支付网关页 `payment-settings-section.tsx` 瘦身 224 行。
- **金额管理对话框改成正经 CRUD**（`wallet/components/dialogs/amount-options-manage-dialog.tsx` 新建）：一个标题 + 一行一个额度 + 每行右侧垃圾桶删除 + 底部输入框添加（回车也行，重复／非法自动禁用按钮），说明文字全删；旧可视化编辑器 `amount-options-visual-editor.tsx` 成为死代码已删除；折扣管理对话框 `amount-discount-manage-dialog.tsx` 同批新建。
- **关于页新增「自有化部署服务」章节**（`about/index.tsx` +137 行）。
- 后端仅 `controller/topup.go` 一行：把充值上限 `max_topup` 吐给前端。

**验证**：`bun run typecheck` ✅ ｜ `bunx vitest run` ✅（245 用例）｜ `bun run build` ✅ ｜ `bun run i18n:sync` ✅（7 语言词条增删后键集一致）｜ `go build` ✅ ｜ 后端探活 ✅。

### 须知
- 后端要重启才生效：Go 代码变了，`//go:embed web/dist` 也是编译期固化，3001 端口那个 `go run` 进程需重启一次。

## [v0.31.1] - 2026-09-14

### 界面
- **试算抽屉的模型名改为可挑选的组合框**（`web/src/features/discounts/components/discounts-simulate-drawer.tsx`）：原来是盲填文本框，现在点开能从平台可提供服务的模型（`/api/channel/models_enabled`，与模型广场同源，只列挂了启用渠道的模型）里挑选，输入时按名字过滤；保留手输，尾部通配符（`claude-*`）照旧可用。纯前端改动，复用现成零件 `ComboboxInput`。

**验证**：`go build ./...` ✅ ｜ 无后端改动 ｜ `bun run typecheck` ✅ ｜ `bun run build` ✅ ｜ `bunx vitest run` ✅（`discounts-simulate-drawer.test.tsx` 增 2 用例：组合框选项与模型广场同源、通配符保留）｜ `bun run i18n:sync` ✅（无词条改动）

## [v0.31.0] - 2026-09-14

### 新增
- **一键查价（task-06 第四步）**：`POST /api/discount/admin/price-query`（`controller/discount_price_query.go` + `service/discount_price_query.go`）。选一个客户，把"挂了几套价、现价按几折、命中的是哪套、其余几套输在哪一层、每个模型有几条线路、毛利多少"一次性拉出来——只读不落库。装配在折扣方案页工具栏「客户查价」按钮（挨着「试算」），打开抽屉看价目本：胜者整段标绿，落选者逐条带原因（沿用 v0.28.0 多方案裁决的 `Rejected` 字段），最后给一句"能不能签"。
- **可复用模型清单（task-06 第五步）**：`discount_model_lists` 表（`model/discount_model_list.go`）存可复用的模型名单；维护页装配在折扣方案页「模型清单」按钮下，可新增、编辑、删除、套用。`ModelListField` 零件（`web/src/features/discounts/components/model-list-field.tsx`）统一三种录入姿势：标签框挑选 / 粘贴整份 / 从已存清单一键带入；试算、报价核算、查价三处共用这一份。候选清单取 `/api/channel/models_enabled`（与模型广场同源，只列挂了启用渠道的模型），不用全量目录 `/api/channel/models`。

**验证**：`go build ./...` ✅ ｜ relaykit 独立构建 ✅ ｜ `go test ./model/... ./controller/... ./middleware/... ./router/...` ✅（新增 `model/discount_model_list_test.go` 8 用例、`service/discount_price_query_test.go` 9 用例：清单去重与解析、单条/批量查价、绑定为空与多种绑定组合）｜ `bun run typecheck` ✅ ｜ `bun run build` ✅ ｜ `bunx vitest run` ✅（新增 `discounts-price-query-drawer.test.tsx`、`discounts-model-lists-drawer.test.tsx`、`model-list-field.test.tsx` + `model-list.test.ts` 共 13 用例）｜ `bun run i18n:sync` ✅（新增 51 条、删 0 条死键，7 个语言文件键集一致：缺失 0 / 多余 0）

### 须知
- 后端要重启才生效：Go 代码变了，`//go:embed web/dist` 也是编译期固化，3001 端口那个 `go run` 进程需重启一次。

## [v0.30.0] - 2026-09-14

### 新增
- **界面放开一个客户挂多套方案（task-06 第三步）**：挂多套方案在 v0.28.0 就已经接进计价，但界面这一侧还是「一套」的读法——看不到第二套、试算只报一个数字、日志只记命中的一个方案 id。这一版把这几条读路径全部对齐到多方案口径。
- **一个客户最多同时挂 10 套**（`model/discount_binding_limit.go`）：`BindDiscountPlan` 与绑客户号两条路径都硬拦，只数生效中的绑定（解绑过的不占名额），重复绑同一套方案时说「已经绑过了」而不是「你的方案太多了」。撞上限的绑号整笔拒绝——归属不落、号的用量不扣，平台解绑一套之后同一张号还能继续用。
- **落选候选进消费日志**：`BillingDiscount` 新增 `Rejected`，经 `types.GroupRatioInfo.DiscountRejected` 由 `appendDiscountRejectionInfo` 写进 `other.admin_info.discount_rejected`（方案 id / 来源 / 折扣 / 原因），回答「为什么不是按他另一套算」。只挂一套的客户不会多出这个字段，日志形状不变。
- **试算新增候选明细**：出参多了 `candidates`——这次哪一套生效、其余几套输在哪一层，原因原样带出裁决结论；挂两套以上时额外提示「该客户挂了 N 套」。

### 修复
- **经销商「我的客户」页与「查看价格」只显示一套方案**：`AgentCustomer` 新增 `bindings` 数组，`buildAgentCustomer` 不再只挑一条生效绑定。发额度、改价的回包走同一段代码，一处改齐。
- **试算与计费不是同一个折扣**：试算原来读的是快路径上那一条绑定，界面显示的价可能不是这一单真扣的价，现改用 `ResolveUserDiscountDetailed`（与计费同一套多方案裁决）。
- **绑定列表看不到生效中的绑定**：解绑记录攒够一页会把还在生效的绑定挤出列表，界面误显示成「这个客户没有方案」。绑定列表接口新增 `status` 过滤（`status=1` 只看生效中）。

### 界面
- 用户列表行菜单「设置专属折扣」改成追加式：绑一套加一套、逐条解绑；挂满 10 套时「绑定」按钮变灰，旁边红字写明原因。
- 「我的客户」列表页「当前价」列多套时汇总成「共 N 套方案」；「查看价格」抽屉逐条列出方案名 · 折扣 · 来源。
- 试算抽屉新增「已绑定的方案」表：哪一套本次生效、其余几套为什么没用上。
- 删掉「一个客户同时只跟一套方案／绑定会替换上一套」两处旧说法（7 语言同步增删）。

**验证**：`go build ./...` ✅ ｜ relaykit 独立构建 ✅ ｜ `go test ./model/... ./controller/... ./middleware/... ./router/...` ✅（新增 `model/discount_binding_limit_test.go` 4 用例、`model/discount_billing_rejections_test.go` 2 用例、`service/discount_simulate_several_plans_test.go` 1 用例：上限拦得住且不留痕、解绑后腾出名额、重复绑定先报「已经绑过了」、绑号撞上限整笔拒绝、计费带出落选者、试算与计费同一个折扣）｜ `bun run typecheck` ✅ ｜ `bun run build` ✅ ｜ `bunx vitest run` ✅ 47 文件 / 228 用例 ｜ `bun run i18n:sync` ✅（新增 8 条、删 2 条死键，7 个语言文件一致：缺失 0 / 多余 0）

### 须知
- 后端要重启才生效：Go 代码变了，`//go:embed web/dist` 也是编译期固化，3001 端口那个 `go run` 进程需重启一次。

## [v0.29.0] - 2026-09-14

### 新增
- **客户档案核算落地（task-06 第二步）**：新增 `POST /api/discount/admin/customer-audit`（`controller/discount_customer_audit.go` + `service/discount_customer_audit.go`）。选一个客户、贴一整份模型清单（一行一个，中英文逗号、顿号也行，最多 100 个），一次算完并返回一张赚亏表：每个模型按几折、折扣是从模型级规则／厂商级规则／方案基础折扣／官方标价哪一层来的、是哪套方案的哪条规则命中的、这个客户分组下有几条线路可用、最便宜的进货折扣是多少、毛利多少（毛利 = 客户折扣 ÷ 最便宜进货折扣 − 1），以及一个结论——**保本 / 会亏 / 成本未知 / 没有线路**；表末给一句汇总：这份报价能不能签，亏几个、缺几个、成本未知几个，并列出提醒（未绑方案、经销商按拿货价核算）。折扣用的是与计费**同一套多方案裁决**，所以核算表里看到的数就是客户下单时真会被收的数——这解决了「一个客户挂多套方案时，单方案试算只能看到其中一套」的盲区。
- **模型层新增批量解析 `ResolveUserDiscountDetailedForModels`**（`model/discount_multi_resolve.go`）：整份清单的折扣一次 `IN` 查询算完，逐模型的答案与逐个单查**逐字一致**（测试专门钉死这一条），计费与核算共用同一套代码。经销商客户按他的**拿货价**核算（复用 `ResolveAgentWholesaleDiscounts`，取最便宜可用线路进货折扣 × 平台加价率）。
- 毛利底线、成本口径与「试算」抽屉完全一致：进货折扣为空就是「成本未知」并列出来，不当成 0 算成暴利。

### 界面
- 折扣方案页工具栏新增「客户报价核算」按钮（挨着「试算」，`web/src/features/discounts/components/discounts-customer-audit-drawer.tsx`）：抽屉里选客户 + 贴模型清单 → 结论条（能不能签）+ 提示条 + 逐模型赚亏表（模型 / 折扣 / 方案 / 线路 / 最便宜进货折扣 / 毛利 / 结论）。
- 折扣来源词条补上「经销商拿货价」（`agent_wholesale`）——此前试算抽屉遇到经销商客户会显示原始英文值。

**验证**：`go build ./...` ✅ ｜ `go test ./model/... ./controller/... ./middleware/... ./router/...` ✅（新增 `service/discount_customer_audit_test.go` 7 组：多方案各管一片模型时按裁决结果算赚亏、四种结论齐全、全保本时结论是「可以签」、清单去重且保留输入顺序、未绑方案按官方标价并给提示、经销商按拿货价核算并给提示、输入不对时说清理由）｜ `go test ./service/` 仅剩 2 个与本次无关的存量失败（`TestObserveChannelAffinityUsageCacheByRelayFormat_*`，改动前即红）｜ `bun run typecheck` ✅ ｜ `bun run build` ✅ ｜ `bunx vitest run` ✅ 47 文件 / 228 用例 ｜ `bun run i18n:sync` ✅（新增 22 条词条、复用既有的「结论／提示／未录入」，7 个语言文件键集一致：缺失 0 / 多余 0）

## [v0.28.0] - 2026-09-14

### 新增
- **折扣多方案解析落地（task-06 第一步：纯读取，不改任何人的价）**。新增 `model/discount_multi_resolve.go`：把一个客户身上**所有**生效绑定逐个对当前模型解析（模型级规则 → 厂商级规则 → 方案基础折扣，方案内优先级与原来完全一致），再按已拍板的五层裁决出一个答案——**具体度 → 谁拍板 → 谁便宜 → 谁新**，最后仍与经销商拿货价比一次便宜（这层保持现状）。方案与规则各一次 `IN` 查询批量取回，计费热路径不逐条查。现状（每人至多一条生效绑定）下与单方案解析逐字一致，测试里专门钉了这一条；计费入口 `ResolveBillingDiscount` 已指向多方案解析（本次唯一一处对既有调用链的改动），试算与保存前校验的下一次迁移在后续步骤。
- **每个落选候选都带原因**（`ResolveUserDiscountDetailed`）：「具体度不如胜者」「定价方优先级低」「不如胜者便宜」「绑定更早」，以及没进赛场的「尚未到生效时间」「已过生效时间」「方案已停用或已删除」——这是后续「一键查价 / 消费日志解释」要用的地基。

### 修复
- **`discount_rules` 的唯一索引建错了，多方案定价根本跑不起来**：索引名 `uk_rule_plan_scope` 的本意是「同一方案内 (范围类型, 范围取值) 唯一」（`IsDiscountRuleScopeDuplicated` 一直按这个口径查询），但建表标签漏了 `plan_id`，实际建成**全表**唯一——任何两个方案不能有同名规则，平台方案和经销商方案都写不出各自对 `gpt-4o` 的报价。现在标签补上 `plan_id`，并加了启动迁移 `fixDiscountRuleUniqueIndex`（`model/main.go`）：检测到旧索引就换成新定义，方向是放宽约束（存量数据必然满足），SQLite / MySQL / PostgreSQL 三库都安全，幂等可重跑。

**验证**：`go build ./...` ✅ ｜ `go test ./model/... ./middleware/... ./relay/...` ✅（新增 `model/discount_multi_resolve_test.go`：裁决层序 9 组 + 索引修正回归 1 组）｜ `go test ./service/` 仅剩 2 个与本次无关的存量失败（`TestObserveChannelAffinityUsageCacheByRelayFormat_*`，改动前即红）｜ 无前端改动

## [v0.27.0] - 2026-09-14

### 新增
- **客户号只能踩在邀请链接上用，注册页不再手填**。经销商在客户号那一行点一下复制，拷走的是**整段邀请文案**（称呼、专属链接、客户号），鼠标停上去能看到同一段文字，确认无误再发；客户点开链接注册，号就在建号这一趟自动落上归属与折扣。链接取当前访问的域名（`features/dealer/lib/invite.ts`），换域名、走测试环境都不用改代码；参数名 `customer_code` 与推广码 `?aff=` 同一条链（`routes/__root.tsx` 与注册表单各抓一次存 localStorage，提交时取出来）。
- **一张号只拉一位客户**。签发接口的 `max_uses` 参数删掉，固定只能用一次（后端 `CustomerCodeMaxUsesPerCode` 写死），不再可能填个 0 造出能到处转发的公开号。谁用掉的记在 `bound_user_id` 上，客户号列表直接显示是哪位客户用掉的。
- **客户号列表默认只看还能发的**（`usable=1`），切到「全部」才翻历史里的旧号；切换档位自动回第一页。
- **「我的客户」页改成上下两块**，顺序就是这门生意的顺序：上面是手上的客户号（签发、复制邀请、作废），下面是已经挂到名下的人（改价、发额度）。客户号这块抽成可复用组件 `CustomerCodesPanel`，平台侧的「经销商设置」抽屉复用同一份（来源不同，能力相同）。
- **侧边栏「我的客户」可在站点设置里开关**（站点设置 → 侧边栏模块新增 `customers`）。

### 修复
- **客户号那一行的复制按钮，鼠标停上去什么都不弹**（`web/src/components/copy-button.tsx`）：这个组件只认自己声明的属性，提示气泡挂上来的鼠标监听、元素引用等属性被它丢掉了，气泡永远收不到「鼠标进来了」——经销商看不到要发给客户的邀请文案长什么样。现在声明之外的属性原样透传给底层按钮，并用类型把「漏传」在编译期拦住。
- **客户号列表翻页永远停在第一页**：页码参数名写成了 `page`，后端 `common.GetPageQuery` 读的却是 `p`，第二页请求拿回来的还是第一页。
- **客户重复绑自己用掉的号，被误告知「使用次数已用完」**（`model/agent_customer.go`）：判断顺序调整，先看「这个人是不是已经有这个方案了」，再看号自身状态——用掉这张号的正是他本人，他该听到的是「你已经是这位经销商的客户了」。
- 作废掉的号会从「可用」这一档里消失，列表自动回第一页，不会停在空页。

### 须知
- **注册页原来那个「客户号（选填）」输入框删掉了**（连同前端格式校验、`CUSTOMER_CODE_REGEX` 与 5 个相关用例）：号只有踩在链接上才有用，手抄一串 12 位号码既容易错，也说不清号属于谁。手里只有号码的客户仍然可以走「个人资料」里的客户号卡片自助绑定，这条路没动。
- 客户号的归属一旦落下就跟着客户走，但「这位客户当初是谁带来的」只有那张号说得清——所以号用完不作废记录，`bound_user_id` 一直留着。

**验证**：`go build ./...` ✅ ｜ `go test ./model/...` ✅ ｜ `bun run typecheck` ✅ ｜ `bun run build` ✅ ｜ `bunx vitest run` ✅ 47 文件 / 228 用例（新增 `dealer/lib/__tests__/invite.test.ts`、`dealer/components/__tests__/customer-codes-panel.test.tsx`）｜ `bun run i18n:sync` ✅（缺键 0、多键 0）

## [v0.26.0] - 2026-09-13

### 新增
- **注册页可以填客户号了**（选填，装配在注册表单「确认密码」下方）。客户手里有经销商发的号，建账号这一趟就把归属与折扣一起落下来，不必先注册、再进「个人资料」补绑一次。填了就按号上的方案价计费，留空就是普通客户。
- **号不能用时在建账号之前就拦下**，并说清是哪一条：号不存在（抄错、漏位都算）、已作废、已过期、次数用满、号上的方案已停用。这一步是只读预检（新增 `model/customer_code_check.go`），不写任何数据、也不吃号上的用量——被拒的注册在库里不留账号，客户改了号重来即可。
- **万一号没落上，界面当场说清**：预检放行之后仍可能绑不上（并发抢用同一张号），那时账号已经建好了，于是返回的仍是注册成功，但带上「客户号没生效」和具体原因，界面弹一条警示而不是「注册成功」——不然客户会以为自己已经在按折扣计费。
- 预检与绑号共用同一份拒绝判断（`customerCodeRejectReason`），避免两处口径走散；号在两条路上都按同一口径抹平大小写与空白（客户手抄常是小写，库里存的是大写）。

### 须知
- **注册请求改用内嵌结构体接字段**（`model.User` + `customer_code`），校验仍按 `model.User` 的标签走：既不给用户模型加一个只在注册时有意义的字段，也不让这一趟把既有校验（用户名长度、密码 8-20 位）绕过去。
- 前端在输入时就把号统一成大写，并在提交前按签发规则（`AG` + 10 位字母数字）挡一道，抄错一位不必等提交才知道。

**验证**：`go build ./...` ✅ ｜ `go test ./model/ ./controller/` ✅（新增 `model/customer_code_check_test.go` 3 用例：好号连大小写空白一起认、预检不消费用量且之后仍能真的绑上、五条拒绝理由；新增 `controller/register_customer_code_test.go` 6 用例：注册请求能同时解出用户字段与客户号且弱密码仍被挡、带错号当场拒并给出具体理由、不带号时预检放行且响应里不多塞字段、绑定失败的响应形态、好号真的落下归属）｜ `bun run typecheck` ✅ ｜ `bun run build` ✅ ｜ `bunx vitest run` ✅ 46 文件 / 230 用例（新增 `features/auth/constants.test.ts` 5 用例：客户号格式规则与签发规则一致）｜ `bun run i18n:sync` ✅（新增 5 条词条，7 个语言文件键集一致，缺键 0、多键 0）｜ 端到端：带一张不存在的号注册返回「客户号不存在，请核对后重试」，库里没留下账号

## [v0.25.0] - 2026-09-13

### 新增
- **「我的客户」：经销商终于看得见自己名下的客户**（控制台 → 我的 → 我的客户；`GET /api/user/self/agent/customers`）。名单只认归属——绑过他的客户号就归他名下（`users.parent_agent_id`），别人的人不会出现，被设成经销商的人会自动从名单上消失（他已经是同行的对手，不再是客户）。每一行摊开此刻的真实状态：按哪个方案、几折算、这个价是**他自己定的**还是客户号/平台给的（来源 `agent` / `customer_code` / `manual`）、余额与已用额度、账号状态。
- **给下属客户改价**（`PUT /api/user/self/agent/customers/:customerId/discount`，`plan_id` 传 0 表示撤价）。只能挑**自己货架上的方案**——低于他最低售价折扣的方案不出现在选择里，就算绕过界面直接打接口也会被拒（那个价卖出去是亏的）。
- **撤价会回到客户号签的价**，不是掉到官方标价。改价是**新插一条** `source=agent` 的绑定，不去改写客户号留下的那条同方案绑定：客户号是经销商当初签给客户的凭证，上面写的价不该被后来的改价抹掉。两条主张并存，由来源优先级决定谁生效（`agent` 排在 `customer_code` 之前，平台手工价 `manual` 又在 `agent` 之前），撤价只撤自己那条。
- **给下属客户发额度**（`POST /api/user/self/agent/customers/:customerId/quota`）：钱从经销商**自己的余额**转过去，一减一加同一事务，不够就一分不动；平台关掉「允许给下属客户发额度」这个开关时一律拒绝。改价、发额度之后都把那位客户此刻的最新一行回给界面，省掉一次为刷新而发的查询。
- **每种拒绝都有具体说法**（后端 i18n，简繁英三份都有）：不在你名下、平台已单独定价、这个方案不在你的货架上、发额度已关闭、你自己的额度不够、额度得是正数。

### 须知
- **平台手工定价的客户，经销商改不动**：那是平台的决定，不是他的生意。
- **三个端点都在自助路径下**（`/api/user/self/agent/...`），不需要管理员权限：谁是我的客户由 `parent_agent_id` 说了算，别人看不见也改不了；判断全在 model 层，controller 只做参数解析与错误翻译。
- **改价只对这位客户生效**，不动他绑的客户号，也不动客户的号上用量。
- 前端产物已重新构建，后端已重启生效。

**验证**：`go build ./...` ✅ ｜ `go test ./model/` ✅（新增 `model/agent_customers_test.go`，6 个用例：名单只认归属且带货架价、改价真的落到计价上（按 `ResolveUserDiscount` 断言 0.8/0.9）、撤价回到客户号那份价、四条拒绝线（别人的客户/平台定价/停用方案/低于下限）、发额度转账与不够不动、升为经销商后从名单消失）｜ `bun run typecheck` ✅ ｜ `bun run build` ✅ ｜ `bunx vitest run` ✅ 45 文件 / 225 用例 ｜ `bun run i18n:sync` ✅（新增 25 条词条，7 个语言文件键集一致，简繁均已译）

## [v0.24.0] - 2026-09-13

### 新增
- **客户绑号：客户号从此真的能用了**（`POST /api/user/self/agent/bind`、`GET /api/user/self/agent/binding`）。客户把经销商给他的号填进「控制台 → 我的 → 个人资料」页面底部新加的**客户号卡片**，一次绑定同时落下三件事：归到那位经销商名下（`users.parent_agent_id`）、按号上的折扣方案计价（写 `discount_bindings`，来源 `customer_code`，并同一事务重算 `users.discount_plan_id` 快路径）、号上用量 +1。三件事同一事务，少一件就会出现「归属落了但价没变」或「一张号被反复用」。
- **归属与折扣对客户可见**：同一张卡片摊开显示「我此刻挂在谁名下」「按什么价」——直属平台、官方标价时也照实说，不让客户自己去猜是不是被多扣了钱。
- **每种拒绝都有具体说法**（后端 i18n，简繁英三份都在）：号不存在、已被作废、已过期、次数用完、号上方案已停用、不能用自己的号、已经归属另一位经销商。客户看到的都是能据以行动的那一句。

### 须知
- **归属一旦落下只能由平台改**：客户自己不能换经销商（拿别人的一张号就把客户挖走是不行的），要换得找平台。
- **平台手工给的价优先于号上带的价**：同一客户身上多份折扣按 `manual > subscription > customer_code` 取用（沿用既有的 pickActiveDiscountBinding 口径）。所以号上的折扣不会顶掉平台给企业VIP的那份优惠——归属照落，价仍按平台那份算。
- 同一张号被同一个人再绑一次不会重复扣用量，也不会多出一条折扣绑定（会明确告知「你已经是这位经销商的客户了」）。
- 「我的客户」（经销商看自己名下的客户名单、给下属发额度、给下属改价）是下一步；发额度开关到那时才开始管人。
- 前端产物已重新构建，后端已重启生效。

**验证**：`go build ./...` ✅ ｜ `go test ./model/` ✅（新增 `model/agent_customer_test.go`：5 个用例覆盖归属+折扣+用量三件事同落、小写入参、过期/作废/用尽/方案停用/不存在的号、自己发自己用、已归属他人、重复绑定幂等、平台手工价优先）｜ `bun run typecheck` ✅ ｜ `bun run build` ✅ ｜ `bunx vitest run` ✅ 45 文件 / 225 用例 ｜ `bun run i18n:sync` ✅（新增 11 条词条，7 个语言文件键集一致，简繁均已译）｜ 端点探测：两个新接口均返回 401（路由在、待登录），后端启动日志无报错

## [v0.23.0] - 2026-09-13

### 新增
- **「经销商设置」：一个人的经销商档案从此可以改**（用户列表 → 行菜单 → 经销商设置；`GET/PUT /api/user/:id/agent/profile`、`PUT /api/user/:id/agent`）。v0.22.1 里那句「想改加价率只能先取消经销商再设一次」现在不作数了。这个抽屉里能改四件事：**平台加价率**（他拿货价的倍数，下限仍是 5%）、**最低售价折扣**（他给客户定价的下限，0 表示平台不限制）、**是否允许给下属客户发额度**、**备注**。四个字段都是「传哪个改哪个」，改备注不会顺手把加价率清零。
- **客户号**（`model/agent_code.go`、`controller/agent_codes.go`）：一码同时写着「归属哪个经销商」和「按哪套折扣方案计价」，经销商把号发给客户，客户绑一下这两件事就一起生效。号形如 `AG` + 10 位大写随机码，可设使用次数上限（0 = 不限，上限 1000）与有效期（不填 = 不过期），可备注，可作废（作废只改状态、不删行：谁签的、被用了几次要留痕）。**作废只能动自己的号**，别人的号连存在都不会被确认。号上指定的折扣方案必须是启用中的方案，否则当场拒绝——不然客户拿到号以为自己有折扣、实际按原价扣费。
- **两条签号路径**：平台替他签（`GET/POST /api/user/:id/agent/codes`、`DELETE /api/user/:id/agent/codes/:codeId`，装配在「经销商设置」抽屉里，运营当面谈客户时顺手发几张），经销商自己签（`GET/POST /api/user/self/agent/codes`、`DELETE /api/user/self/agent/codes/:codeId`，装配在控制台 → 我的 → 经销商台账页）。两条路走同一份业务，差别只在「动谁的号」。
- **经销商自己的货架**（`GET /api/user/self/agent/plans`）：他能拿去给客户报价的折扣方案清单。**低于他最低售价折扣的方案不出现在清单里**——那个价卖出去就是亏的，所以是「看不到」而不是「选完再报错」。货架只管平台方案；他自己名下的方案不在里面重复出现。

### 须知
- **客户号现在只是「发得出去」**：签号、看号、作废都通了，但**客户拿号绑自己这一步还没做**，所以这一版发出去的号暂时只是凭证，不会自动把客户挂到经销商名下。绑号与「我的客户」（名单、发额度、改价）是接下来的两项。
- **发额度开关现在只是存下来了**，还没有能发额度的动作可关；等「我的客户」落地后它才开始管人。
- 客户号不写死「一人一码」：同一张号可发给多个人（使用次数 > 1 时），也可以只给一个人（次数 = 1）。一码一客还是多客，由填的次数决定。
- 前端产物已重新构建，重启后端生效。

**验证**：`go build ./...` ✅ ｜ `go test ./model/` ✅（新增 `model/agent_code_test.go`：7 个用例覆盖签号身份门槛、方案可用性、批量唯一性、跨人作废、入参边界、档案局部更新与零值落库、折扣下限口径）｜ `bun run typecheck` ✅ ｜ `bun run build` ✅ ｜ `bunx vitest run` ✅ 45 文件 / 225 用例 ｜ `bun run i18n:sync` ✅（新增 29 条词条，7 个语言文件键集一致，简繁均已译）

## [v0.22.1] - 2026-09-13

### 修复
- **老库上的经销商整条链路都是坏的**（`model/main.go`）：`agent_profiles` 在 SQLite 上是「建过就不再走 AutoMigrate」的表（它带 decimal 列，走 AutoMigrate 每次启动都会把整张表重建一遍），所以 v0.22.0 新加的 `markup_ratio`（平台加价率）这一列在**老库上根本不会出现**。具体的坏法：读经营档案时报 `no such column: markup_ratio`，于是用户列表里「设为经销商」保存失败、这位经销商的拿货价解析不到（这一单退回按他绑的折扣方案计费，后台记一条日志）、他自己的台账页直接报错。新库（本次启动才建的表）完全看不出问题，所以只在已经有数据的机器上暴露。现在迁移里显式补这一列（与 `subscription_plans` 同一套做法），并把那段「给老表补列」的逻辑抽成 `ensureSQLiteTableColumns`，两边共用，避免以后再漏。

### 须知
- 补上列后，改造前建的那位经销商加价率是空的，按缺省档处理（自采成本 +10%）。想给他另定一档，眼下只能先「取消经销商」再设一次——还没有「改加价率」的接口，需要的话另开一项。
- 本次修复只影响已经在跑的库；后端重启后自动补列，之后新建的表也不会缺这一列。
- 前端产物已重新构建，重启后端生效。

**验证**：`go build ./...` ✅ ｜ `go test ./model/` ✅（新增回归用例 `TestMigrateAgentTablesAddsMarkupRatioToExistingTable`）｜ `bun run typecheck` ✅ ｜ `bun run build` ✅ ｜ `bunx vitest run` ✅ 45 文件 / 225 用例 ｜ `bun run i18n:sync` ✅

## [v0.22.0] - 2026-09-13

### 新增
- **经销商的拿货价接通了计价链路**（`model/agent_wholesale.go`、`model/agent_wholesale_quote.go`、`model/discount_resolve.go`）：拿货价不再靠手工填，而是算出来的——取这个模型**最便宜一条可用线路**的进货折扣，乘平台加价档。例：上游按 0.27 供货，加价档 1.1 → 他拿货 0.297。取「最便宜那条」是与保存前校验、试算一致的口径（那两处也拿规则范围内最低成本算）。加价档默认 1.1（加 10%），**下限 1.05 是硬校验**（低于 1 等于平台倒贴），上限 100 只挡误输入（把 1.1 敲成 110）。
- **「设为经销商」对话框改填平台加价率**（原来是填批发折扣）：批发折扣是相对**官方标价**的，跟线路成本脱钩，填多少都能存、也没人读得准；现在填的是平台在**自采成本**上加几成，改完当场能看到这位经销商按模型逐条的拿货价，没有一条能算出价时明说原因（先录上游渠道的进货折扣）。
- **经销商台账页**（控制台 → 我的 → 台账；`GET /api/user/self/agent/ledger`）：看自己的钱包余额、累计花费、调用次数，以及自己发出去的**每个 Key** 用了多少、花了多少。数据是按消费日志按 Key 聚合的，不是估的；**已经删掉的 Key 照样在表里**（删掉不等于账上没发生），另有被删/停用/过期/用尽的状态与最后调用时间。
- **用户列表行菜单新增「设置专属折扣」**（= 企业VIP）：从人这一侧直接绑折扣方案，点开时客户已经替你选好，选一套方案即可；此前只能从折扣方案页反向搜人，运营得先记住名字。
- **创建用户的角色下拉按登录身份过滤**：系统管理员能开出普通用户 / 业务管理员，业务管理员只能开普通用户，与后端 `POST /api/user/` 的 `user.Role >= myRole` 拒绝口径一致——不再「下拉里选得到、提交才被驳回」。

### 修复
- **经销商的菜单项不再靠写死的开关**：`use-sidebar-data.ts` 里的 `DEALER_IDENTITY_ENABLED = false` 换成按 `users.subject_type === 'agent'` 判断。此前这一项被关着，谁（包括管理员）都看不到台账。
- **`/billing` 补上路由守卫**：菜单里不出现只是不显示，手敲地址仍进得去；现在非经销商一律转 `/403`。
- **侧栏「账单」改名「经销商台账」**，占位文案删除：这一页已经有真数据，原先那句「经销商专享……身份上线后开放」既不存在也没必要。

### 须知
- **加价率管的是经销商的拿货价，不是他卖给客户的价。** 零售价仍由平台绑给他的折扣方案决定，一期他自己不能改价。价格铁律不变：线路成本 ≤ 拿货价 ≤ 零售价。
- **台账里「花了多少」是平台按拿货价从他钱包扣掉的额度**，不是他卖给客户收了多少钱——那件事平台既不参与也不知道，所以这一页没有那个数。
- **加价档缺省 1.1，早年建的档案（这一列还没值）也按 1.1 处理**，不会算出零价格。
- 前端产物已重新构建，重启后端生效。

**验证**：`go build ./...` ✅ ｜ `go test ./model/ ./controller/ ./router/` ✅ ｜ `bun run typecheck` ✅ ｜ `bun run build` ✅ ｜ `bunx vitest run` ✅ 45 文件 ｜ `bun run i18n:sync` ✅

## [v0.21.0] - 2026-09-13

### 新增
- **用户列表能真正标出「经销商」了**（用户列表 → 行菜单 →「设为经销商」）。此前这一档在界面上永远出不来：`agent_profiles`、`customer_codes`、`users.subject_type`、迁移、读函数全都到位，唯独没有任何地方能把一个人变成经销商。现在行菜单给出入口，填好批发折扣与最低折扣后一次写好（设为走 `POST /api/user/:id/agent`，取消走 `DELETE /api/user/:id/agent`），两个折扣同事务落库，并过用户管理审计。
- **用户列表新增「客户类型」列与筛选**（普通客户 / 企业折扣 / 经销商）：运营能一眼分出人群，也能按类型筛。判定顺序是先看经销商、再看有没有绑折扣方案——顺序反了会把经销商自己标成「企业折扣」（他身上也挂着方案，那是他的批发价）。

### 修复
- **界面里的「Admin」统一改称「业务管理员」**（`web/src/lib/roles.ts`、用户编辑抽屉的角色下拉）：系统里有超管与业务管理员两档管理身份，一律叫「管理员」说不清是谁。
- **提升/降级按钮收窄到超级管理员**（`data-table-row-actions.tsx`）：后端 `promote` 只放行 Root，而业务管理员能管理的目标里不存在可降级的人——这两颗按钮对他们 100% 会失败。现在按同一口径隐藏，点不到就不会白点。

### 须知
- **设成经销商不会给他任何后台权限。** 经销商是叠在用户身上的业务身份，不是权限等级：`role` 一个字节都不动，他照常登录、照常消费。界面上因此叫「设为经销商」，而不是「提升为经销商」。
- **批发折扣目前只存进档案，还没接进计价。** 要生效得等 P3 的定价链路去读 `agent_profiles.wholesale_discount`；在此之前，经销商的 API 消费仍按原来的价走（他绑的方案或官方标价）。所以现在把人设成经销商，能看到的是「身份 + 筛选」，价格暂时不变。
- 批发折扣不填时按 `1.000000`（按官方标价进货），不会算错钱；最低折扣高于批发折扣会被拒绝，用的是与折扣方案同一条铁律。
- **取消经销商时，名下还挂着客户号会被拒绝**：客户号记着「哪些客户归他」，先作废才不会让归属断链。取消**不动**余额、分组和已绑的折扣方案——取消身份不等于清账。
- 界面文案统称「经销商」（不叫 Agent），底层取值仍是 `agent`。
- 前端产物已重新构建，重启后端生效。

**验证**：`go build ./...` ✅ ｜ `go test ./model/...` ✅ ｜ `bun run typecheck` ✅ ｜ `bun run build` ✅ ｜ `bunx vitest run` ✅ 45 文件 / 225 用例 ｜ `bun run i18n:sync` ✅

## [v0.20.4] - 2026-09-13

### 修复
- **撤回 v0.20.3 对顶栏「控制台」的角色门控**。v0.20.3 把它收成仅管理员及以上可见，理由是"访客与普通客户看到的是后台入口"——这个判断是错的。「控制台」不是后台的门面，它是**客户侧仪表盘的唯一入口**：客户点进去看到的是同一套菜单按身份裁剪后的精简版（「常规」+「我的」两组），原本的系统行为就是对的。现恢复为对所有身份可见；`/dashboard` 的菜单裁剪逻辑没有被动过，也不需要动。
- 登录落地页那半保留：登录、注册、OAuth 回调与两处兜底跳转仍指向 `/dashboard/overview`。

### 须知
- 记下后续规划的口径（业务方 2026-09-13 说明）：将来经销商与管理员会各自增加端点与菜单——经销商看台账，管理员（`admin` 与 `super_admin`）各有各的可见范围，同时都能看到普通的台账。这块等系统主体做完再统一规划，现在不动。
- 前端产物已重新构建，重启后端生效。

## [v0.20.3] - 2026-09-13

### 修复
- **登录后不再落在需要再跳一次的父路径上**：登录、注册、第三方 OAuth 回调，加上「测试模型被关掉」「聊天记录 id 非法」两处兜底跳转，落点从 `/dashboard` 改成 `/dashboard/overview`（共 7 处）。页面本身没变（`/dashboard` 原本就会 redirect 到概览），变的是两件事：少一跳；落点不再跟着 `/dashboard` 的默认 section 漂移——此前若把默认 section 换成别的页面，登录落点会跟着悄悄变，而侧边栏第一项没变。
- **顶栏「控制台」收窄到管理员及以上**：这段此前没有任何角色判断，访客和普通客户在公开页（首页、模型广场、排行榜）的顶栏上都能看到「控制台」，点进去才发现要登录——等于拿后台入口当门面。现仅管理员及以上可见；管理端「顶栏导航」里的开关照旧生效（关掉它对管理员也不显示）。

### 须知
- 客户侧界面不再出现「控制台」，与 P0 验收 #2「客户界面避开控制台／仪表盘」的口径一致；管理员不受影响。
- **页脚的署名行透明度没有动**。那一行的淡（`text-muted-foreground/20`）是刻意做的效果，不是缺陷，以后不用再当问题改。
- 前端产物已重新构建（`bun run build`），重启后端即可。

## [v0.20.2] - 2026-09-13

### 重构
- **把格式检查从红变绿**：`bun run format:check` 一直是失败的——78 个前端文件还停留在 prettier 时代的排版，后端的 `common/constants.go`、`service/discount_simulate.go`、`relaykit/dto/openai_request.go` 也没过 `gofmt`。这次统一跑了一遍（`bun run format` + `gofmt -w`），纯格式，零语义改动。
- 前端 78 个文件按 `.oxfmtrc.json` 重排：import 顺序、JSX 折行、函数参数类型的拆合行、`className` 内 Tailwind 类顺序（`sortTailwindcss`）、CSS 自定义属性的换行位置。合计 237 增 / 208 删，最大的单个文件（`about/index.tsx`）也只有 16 行增。
- 抽查确认改动性质：字符串字面量、样式取值、i18n key、组件结构全部未动。Tailwind 类顺序的重排只是把 `shadow-card` 这类属性归位，同一元素上不存在互相冲突的同类属性（如 `p-2` 与 `p-4` 并存），因此不影响渲染结果。
- `.gitattributes` 补上 `*.mjs` / `*.cjs` 的 `eol=lf`：`web/scripts/` 下三个脚本此前只受 `* text=auto` 管辖，`core.autocrlf=true` 会把它们检出成 CRLF，格式化写回 LF 后就出现「`git status` 说改了、`git diff` 说没改」的假修改（内容其实与索引一致）。规则补上后工作区与索引口径一致。

### 须知
- 这次改动面广（78 个文件）但全是排版，界面行为不变。不放心的话启动后随手翻几页看看——按文件清单，业务设置、仪表盘、折扣、首页、定价、系统设置这些模块都被碰到了，改的是空行、折行与类名顺序。
- 前端产物已重新构建（`bun run build`），重启后端即可。
- 以后新写的代码跑一次 `bun run format` 就能保持合规；`format:check` 现在应当始终为绿，哪天又红了，说明有文件没格式化就提交了。

**验证**：`bun run format:check` 退出码 0 ｜ `gofmt -l` 无输出 ｜ `bun run typecheck` ✅ ｜ `bun run build` ✅ ｜ `bun run test` ✅ 45 文件 / 225 用例 ｜ `go build ./...` ✅ ｜ `go test ./controller/ ./router/ ./model/ -count=1` ✅

## [v0.20.1] - 2026-09-13

### 修复
- **单条编辑进货折扣完全不做范围校验**（上一版遗留）：渠道编辑抽屉里的「进货折扣」输入框只靠浏览器 `max=1` 挡着，后端 `UpdateChannel` 对 `cost_ratio` 只看"变了没有"、不看合不合理——把 0.27 写成 27、写成 -5、写成一串字母，都能原样存进库。批量录入那条路径上一版加了校验，单条这条没加，同一张表于是存着两种来路的值，谁也说不清该信哪个。现在两条路径共用一个校验函数（`controller/channel_cost.go` 的 `validateChannelCostRatio`，由 `controller/channel.go` 的 `validateChannel` 调用）：负数、0、非数字、超过 100 一律拒绝，空串（没录 / 清空）放行。
- **上限从前端的 1 放开到 100**：进价高于标价是可能的（亏本引流），原来输入框写死 `max=1`，想填 1.5 都填不进去，与后端允许的范围也不一致。现在两端都到 100。
- **前端表单同步校验**：`channel-form.ts` 的 `cost_ratio` 规则、批量设置对话框的即时提示与保存按钮禁用，都改用与后端同一套判据（`isCostRatioInputAllowed` / `isCostRatioWithinMax`，`MAX_COST_RATIO` 常量为唯一来源），以后改口径只改一处。
- **只在该字段真被提交时才校验**：请求里没带 `cost_ratio`（比如只改个备注）时不做判断，保持库里原值——避免"库里存着一个老值，导致连备注都改不了"。
- 语言：新增 2 条校验提示，并改掉了那条描述词条的 key——`FIELD_DESCRIPTIONS.COST_RATIO` 的实际字符串本身就是 i18n key，改了文案就必须同步改 key，否则界面上会退回英文原文（en / zh / zh-TW 手写，其余 4 个语言由 `i18n:sync` 补占位，`missingCount` 归零）。

### 测试
- 后端 `controller/channel_cost_internal_test.go`（新建，11 条表驱动 + 1 条独立用例）：同一批取值分别喂给「单条编辑」和「批量录入」两个入口，断言两者结论一致——空、空格、0.27、1、100、2.5 放行；0、-0.1、abc、100.000001 两边都拒；另有一条专测"请求没带 cost_ratio 时不校验"。
- 前端 `channel-cost.test.ts` 补 2 条（该文件共 7 条）：空值算"还没录"而不是填错、上限与后端的 100 对齐。
- **先证伪再提交**：把 `MAX_COST_RATIO` 临时改成 1000，`caps the input at the same upper bound the backend enforces` 立刻变红（`vitest` 退出码 1）；源码随即还原、复跑全绿。
- 验证：`go build ./...` ✅ ｜ `go test ./controller/ ./router/ ./model/ -count=1` ✅（含新增 12 条）｜ `bun run typecheck` ✅ ｜ `bun run build` ✅ ｜ `bun run test` ✅ 45 文件 / 225 用例（此前 223）｜ `bun run i18n:sync` ✅

### 须知
- **如果某条渠道库里本来就存着非法值**（负数、0、乱填的字符串），下次在抽屉里保存这条渠道时会被拦下来，要求先把进货折扣改对。这是有意的：那种值在折扣判保本时一律被当成"没录"，与其留着不如顺手改掉。
- 前端界面改了，所以要重新 `bun run build` 再重启后端（`web/dist` 是编译期嵌进 Go 二进制的，本次已构建）。
- 仓库里的 `bun run format:check` 目前有 100+ 个既有文件不合格（oxfmt 全量重排从未跑过），与本次改动无关；本次改到的文件已逐个过 oxfmt 检查（其中 `channel-cost.ts` 顺手修掉了上一版留下的一处超宽折行）。

## [v0.20.0] - 2026-09-13

### 新增
- **进货折扣的运营装配层：能批量录、能看谁没录、能看到过期**。折扣客户选线路时"没录进货折扣的线路一律跳过"这条规则生效后，运营手上多了一件必须干的活：给几十条线路录进货价。此前只能一条条点进渠道编辑抽屉去填「进货折扣」，既慢又没人提醒谁还空着——**录漏一条，客户就少一条线路**，而且界面上看不出是"这个模型真没线路"还是"这条线路没录价"。这次补齐四个动作：
  1. **批量录**：渠道列表勾选若干条 → 底部批量操作条新增「设置进货折扣」（`web/src/features/channels/components/data-table-bulk-actions.tsx` + `web/src/features/channels/lib/channel-cost.ts` 的 `handleBatchSetCost`），填一个数一次改一批，留空即清空。后端新增 `POST /api/channel/cost/batch`（`controller/channel_cost.go`，权限 `ChannelSensitiveWrite`）。**只有值真的变了才刷「录入时间」**——把同一个数再存一遍不算重新核对过进价（`model/channel_cost.go` 的 `BatchUpdateChannelCost` 先比后写，返回的是真实改动条数，界面提示的也是这个数而不是勾选数）
  2. **待补清单**：新增 `GET /api/channel/cost/stale`（权限 `ChannelRead`），一次列出「没录的」和「录了超过 30 天没更新的」两类（`days` 可调，1–3650）。入口在渠道页右上角「更多操作 → 进货折扣待补」，弹窗里列出线路名、当前折扣、上次录入时间、以及它为什么被列出来。已停用的线路照样列出——线路一旦恢复启用，成本还是旧的
  3. **列表里看得见**：渠道表格新增「进货折扣」列（`channels-columns.tsx`），已录的显示折扣数与「按标价 XX% 进货」的提示，空着的直接标「未录入」。没值的那批才是真正要紧的，因为它在折扣客户那边等于不存在
  4. **抽屉里提醒该核价了**：渠道编辑抽屉的进货折扣栏，有录入时间时显示「最后更新时间」，超过 30 天再附一句「超过 30 天未更新，上游价格可能已经变了」
- **防手滑**：批量接口把 0.27 填成 27 会被拦下来（> 100 直接拒绝，并提示"请确认没有把 0.27 误填成 27"），填 0 或负数也拒绝。判定口径与 `model/channel.go` 的 `channelCostRatio` 完全一致（空串、非数字、非正数一律算"没录"）；前后端各判一次是刻意的——界面判断只决定显不显示，真正决定"折扣客户能不能走这条线路"的是后端
- **入口**：业务管理 → 渠道 → 右上角「更多操作 → 进货折扣待补」；批量录入在渠道列表勾选后底部的批量操作条 →「设置进货折扣」；单条在渠道行的编辑抽屉里（进货折扣栏）
- 语言：en / zh / zh-TW 各补 16 条；其余 4 个语言由 `i18n:sync` 补英文占位，`missingCount` 归零
- 新增 6 个文件、改 17 个（含 7 个语言文件与 1 个 i18n 同步报告）：后端 `controller/channel_cost.go`、`model/channel_cost.go`、`model/channel_cost_test.go`；前端 `channels/lib/channel-cost.ts`、`channels/lib/__tests__/channel-cost.test.ts`、`channels/components/channel-cost-alerts-dialog.tsx`

### 测试
- 后端 `model/channel_cost_test.go`（新建，4 条用例）：把同一个折扣再存一遍不刷录入时间、传空串清空、只有一条值真变了才写库并返回改动条数、清单把"没录的"与"过期没更新的"都列出来且各带原因
- 前端 `channels/lib/__tests__/channel-cost.test.ts`（新建，5 条用例）：没录／填 0／填 -0.1／填 abc 一律算未配置；`0.270000` 折成 `0.27`；提示用的百分比 `0.27 → 27%`；缺录入时间算过期；超过 30 天算过期
- **先证伪再提交**：把 `isCostRatioStale` 里"没有录入时间算过期"改成 `return false`，两条用例立刻变红（`vitest` 退出码 1）；源码随即还原、复跑全绿
- 验证：`go build ./...` ✅ ｜ `go test ./controller/ ./router/ ./model/` ✅（新增 4 条全过、原有用例无回归）｜ `bun run typecheck` ✅ ｜ `bun run build` ✅ ｜ `bun run test` ✅ 45 文件 / 223 用例（此前 44 / 218）｜ `bun run i18n:sync` ✅（en/zh/zh-TW `missingCount` 归零）
- **须知**：这是完整交付（后端 + 前端），要先 `cd web && bun run build`，再重启后端（`go run main.go --port 3001`）——`web/dist` 是编译期嵌进 Go 二进制的，只重启后端不重建前端产物等于没改。另：新端点的错误文案目前是中文硬编码，与既有多数 channel 端点一致，未接 i18n 机制

### 已知问题（非本次引入）
- **单条编辑没有取值范围校验**：渠道抽屉里的进货折扣输入框只靠 HTML 的 `max=1` 约束，后端 `UpdateChannel` 对 `cost_ratio` 只判断"变没变"、不校验范围，所以把 0.27 填成 27 也能存进去。批量接口这次补了校验（> 100 拒绝），单条那条路径没动——两处口径目前不完全一致，建议后续把校验抽成一处共用
- `go test ./service/` 的 2 条既有失败仍在（详见 v0.19.6 记录），数量未增加

## [v0.19.6] - 2026-09-13

### 修复
- **管理员指定的渠道亏本时补一行留痕**（L2，`middleware/distributor.go`）。令牌上写了 `specific_channel_id` 时，选线路整个绕开候选集与成本过滤——管理员指定的那条线路哪怕亏本也照走（后台要能把某个客户钉在某条线路上，这是有意保留的特权），但这一单**连一行日志都没留**，事后对账看不出「这一单是被人为钉在亏损线路上的」
  - **补的是留痕，不是限制**：这一单照走，只在它确实会亏的时候写一条 warn（渠道号、模型、当前折扣）。只在「真的要中转到上游、会按折扣结算」的请求上判——拉取任务状态之类不产生费用，在那里报「会亏本」是假警报；没折扣的客户（`BuildChannelCostFilter` 返回 nil）本来也不会亏，一并跳过
  - **先证伪再提交**：关掉留痕那一行 → 「亏本时留痕」变红；把判断反转（变成「保本才报警」）→ 三条「不该响」的用例全红，并暴露出 `costFilter` 为 nil 时会空指针崩——于是把 nil 检查从「藏在 `ChannelPassesCostBudget` 对 nil 返回 true 的双重否定里」改成显式
- 新增 1 个文件 4 条用例：`middleware/distributor_specific_channel_cost_test.go`（亏本照走 + 留痕／保本不写／全价客户不写／拉取任务状态不写）
- 验证：`go build ./...` ✅ ｜ `go vet ./middleware/` ✅ ｜ `go test ./middleware/` ✅（新增 4 条全过、原有用例无回归）｜ 前端四项 `typecheck`／`vitest`（44 文件 218 用例）／`i18n:sync`／`build` ✅（本次无前端改动，重跑确认无影响）
- **须知**：这是后端改动，留痕要生效需重启后端（`go run main.go --port 3001`）；前端产物无变化，不必重建

### 已知问题（非本次引入）
- `go test ./service/` 有 2 条既有失败：`TestObserveChannelAffinityUsageCacheByRelayFormat_MixedMode` 与 `..._UnsupportedModeKeepsEmpty`。在未含本次改动的干净检出上同样红，重复跑时红的条数还会变。原因是那套用例用 `time.Now().UnixNano()` 造指纹，Windows 时钟粒度粗，相邻用例撞到同一刻度导致计数互相累加；生产逻辑（entry key 按「规则名＋分组＋指纹」拼）没问题。待定是否修

## [v0.19.5] - 2026-09-13

### 测试
- **「路由策略」抽屉补 6 条用例**（`web/src/features/discounts/components/__tests__/discounts-routing-drawer.test.tsx`，新增）。第 8 件新增的这个抽屉（1 个新文件 + 4 处既有文件 + 7 个语言文件各 19 条）当时一个用例都没写，前端用例数还停在 43 文件 / 212 条，抽屉里的点击流程只能人工点。这次补上——那个「允许走亏损线路」的开关能把客户从"不亏"切成"允许亏着走"，改错了就是白赔钱
  - **覆盖的都是出问题会真金白银的地方**：① 开关落库必须是布尔、没配过的客户按默认口径存（毛利优先 + 不允许）、备注落库前 trim；② 保存后回读后端口径——后端没认这个开关时界面必须跟着回到「不允许」，不能让管理员以为放行了、实际没放行；③ 没选客户时不显示策略区、保存被拦下且不发请求；④ 没有单独配置行的客户「重置」必须灰着（否则会去删一条不存在的行）；⑤ 重置后回到默认口径、重置按钮重新灰掉；⑥ 单选组点已选中那项不该被点空，也不该偷偷换成另一项
  - **先证伪再提交**：故意改坏抽屉三处逻辑（开关落库、保存后回读、重置可用条件），6 条里 4 条立刻变红，随后源码原样还原（`git status` 干净）。写法照同目录的 `discounts-simulate-drawer.test.tsx`：mock `@/lib/api` 的 get／put／delete，挑客户走真实下拉交互
- 验证：`bun run typecheck` ✅ ｜ `bunx vitest run` ✅ 44 文件 / 218 用例（此前 43 / 212）｜ `bun run i18n:sync` ✅ 无词条改动 ｜ `bun run build` ✅ 产物入口 `index.ec6ad7b7a3.js` 与改动前同哈希（只加测试、不动产物，不必重启后端）｜ `go build ./...` ✅
- 改动 1 个文件（新增）：`web/src/features/discounts/components/__tests__/discounts-routing-drawer.test.tsx`

## [v0.19.4] - 2026-09-13

### 修复
- **`auto` 分组下「线路全不保本」的报错不再被吞掉**（`service/channel_select.go`）。auto 分组按客户给的顺序逐组试线路；某一组「有线路、但一条都不保本」时 `model.GetRandomSatisfiedChannel` 会返回一条说明原因的报错，而调用处（`channel, _ = ...`）把它丢掉了——这一组于是被当成"没渠道"，客户最终只看到笼统的「没有可用渠道」，运营也分不清是"这个模型真没线路"还是"线路全亏本、要补进货价或给客户开通亏损放行"
  - **改法（保留回退语义，只补"原因"）**：① 循环里接住 error，只留第一条到 `costBreachErr`（报客户最先想要的那个分组最有用）；② 某一组全亏本**仍然继续试下一个分组**——auto 分组本来就是让客户往下试，后面那组可能是赚的，一遇到就整单失败等于把客户能走的线路也一起堵死；③ 整条 auto 链都试完仍未挑到线路时，把这条报错透出去（替代原来的「nil channel + nil error」）；④ 顺带把那句 debug 日志补上原因，免得运营从日志上也看不出区别
  - **这是个业务选择，先定策略再动手**：§八 L1 摆出的选择是"一个分组全亏时，换下一个分组继续试，还是把报错直接透出去"。定下来的是**两者都要、按顺序**——继续往下试，但整条链都失败时把原因透出。**成功路径一行没动**，能挑到线路时行为与之前逐笔一致
- 验证：新增用例先证伪再改——「两个分组都只有亏本线路」在撤掉修复后为红（`An error is expected but got nil`），修复后转绿；「前一组全亏、后一组保本仍要走后面那组」把回退语义钉住。`go build ./...` ✅ ｜ `go test ./model/ ./controller/` ✅ ｜ `./service/` 仍是那 2 条既有失败、数量未增加 ✅
- 改动 2 个文件：`service/channel_select.go`、`service/channel_select_cost_breach_test.go`（新增）

## [v0.19.3] - 2026-09-13

### 修复
- **稳定优先／没有折扣的老客户，同一个上游优先级里的线路不再按进货折扣拆层**（`model/channel_route_tier.go`）。v0.19.0 引入分层择优时，"哪些线路算同一层"的条件里带上了进货折扣与"成本是否已知"，而这个条件**没有跟着「毛利优先／稳定优先」开关走**：于是在稳定优先下（以及没有折扣、过滤器为 `nil` 的老客户路径上），同一个优先级、进货折扣不同的线路被拆成两层——本该"层内按权重随机分摊"的几条线路，变成固定只走排在前面的那一条
  - **为什么必须修**：对**没绑折扣方案的老客户**这就是行为变化，而 v0.19.0 承诺过"未绑方案的用户行为与改造前逐笔一致"。它不是"少做了一件事"，是"已经改了老客户的行为"
  - **怎么发现的**：不是靠人工点出来的，是复核时读分层条件读出来的；随后**先写用例再改代码**——回归用例先跑成红的（`should have 1 item(s), but has 2`）证实判断成立，改完转绿
  - **改法**：分层条件里的成本部分抽成 `sameRouteTier()` 并用 `prioritizeMargin` 包起来（为 `false` 时只按上游优先级合层）；同步改掉 `TestBuildChannelRouteTiersPriorityStrategy` 里**把拆分当期望**的那条断言（4 层 → 3 层），并新增回归用例锁住"老客户 + 同优先级 + 不同进货折扣 → 仍是一层"
  - **没动什么**：毛利优先（默认）的分层键一个字节没变，折扣客户行为不变；计费侧、前端、`relay/` 零改动

### 验证
- `go build ./...` ✅ ｜ `go test ./model/` ✅ 全包 ｜ `go test ./controller/` ✅ ｜ `go test ./service/` 仍是 2 条既有失败、数量未增加 ✅
- 改动 2 个文件：`model/channel_route_tier.go`、`model/channel_route_tier_test.go`

## [v0.19.2] - 2026-09-13

### 修复
- **「客户折扣方案」的菜单项在 v0.19.1 里仍然看不见**：`web/src/hooks/use-sidebar-config.ts` 的 `URL_TO_CONFIG_MAP` 是一张**白名单**，菜单项只有登记过才会渲染；`/discounts` 从未登记，被 `isModuleEnabled()` 静默判为不可见——**不报错、不留日志、界面无任何提示**。所以 v0.19.1 挪分组并不能让它出现（挪之前也一样看不见）。本次补上 `'/discounts': { section: 'business', module: 'billing' }`，含义是这一项归「计费与定价」模块管，管理员关掉该模块时它一并隐藏
- 同日把这条规则写进 `web/AGENTS.md` §3.8（新增侧边栏入口必须登记三处：路由文件 / 菜单项与 `pathPattern` / **`URL_TO_CONFIG_MAP`**），`04-frontend-plan.md` §八 补第 16 条，`03-step-log.md` 补 S-10 卡

### 验证
- `cd web && bun run build` ✅；**产物核验**（不看界面、直接拆产物）：`web/dist` 的菜单配置为「计费与定价 → …（6 项）+ `Discount Plans`」、白名单表含 `"/discounts":{section:"business",module:"billing"}`；后端 3001 实际下发给浏览器的入口与其分包与本地产物同哈希
- 改动 2 个文件：`web/src/hooks/use-sidebar-config.ts`、`web/AGENTS.md`；无后端改动、无 `relay/` 改动
- **注意交付单位是"前端构建 + 后端重启"**：`web/dist` 是编译期嵌进 Go 二进制的，只重建前端不重启后端，界面不会变

## [v0.19.1] - 2026-09-13

### 变更
- **折扣方案的菜单位置与名字**：从「用户管理」组挪到「计费与定价」组、排最后一项；中文名从「折扣方案」改成「客户折扣方案」
  - **为什么挪**：「分组定价」管的是**一群人**（整个用户分组统一调倍率），「折扣方案」管的是**一个客户或一个经销商**，是同一件事的两个粒度。此前两者分处两个分组，管理员想调价得跑两个地方找，且第一反应是去「计费与定价」，不容易想到折扣在「用户管理」里
  - **为什么不会看混**：挪过去后它排在「计费与定价」第 7 位（最后），与第 4 位的「分组定价」中间隔着「支付网关」和「签到奖励」，视觉上不挨着
  - **为什么改名**：「分组定价」与「折扣方案」两个词长得像，容易以为是同一件事。加「客户」两字点明"这是给单个客户单独设的"。词条键名 `Discount Plans` 保持不变，英文界面仍是 `Discount Plans`，只改简繁中文的译法（`zh.json`／`zh-TW.json`）
  - **页内标题一并变**：折扣页标题与菜单项用的是同一个词条，所以页头同步显示「客户折扣方案」，不会出现菜单一个名、页内另一个名
- 顺带把 `business-settings.config.ts` 顶部注释里列举的顶级路由补上 `/discounts`。该路径本来就在嵌套视图的匹配规则（`pathPattern`）里，无需改动，只是注释没跟上

### 验证
- `bun run typecheck` ✅ ｜ `bun run build` ✅ ｜ `bunx vitest run` ✅ 43 文件 / 212 用例 ｜ `bun run i18n:sync` ✅ ｜ `go build ./...` ✅
- 改动 3 个代码／配置文件：`web/src/components/layout/config/business-settings.config.ts`（挪位置 + 补注释）、`web/src/i18n/locales/zh.json` 与 `zh-TW.json`（改译名）+ `VERSION`；无后端改动、无 `relay/` 改动
- **这一版只挪了位置、改了名，没修白名单**——所以升级到本版仍然看不到那个菜单项（见 v0.19.2）

## [v0.19.0] - 2026-09-13（提交 `d08cca48`，annotated 标签）

### 变更
- **P2 第 8 件 · 路由侧成本过滤完工**（8a 提交 `ddf6ae87` + 本次 8b／8c）：给折扣客户选线路时不再走"会亏本"的线路。这是 P1／P2 队列的最后一件，**9 件至此全部收口**
  - **8a 成本过滤**（`ddf6ae87`）：同一模型有多条线路时，只留「进货折扣 ≤ 客户折扣 − 毛利底线」的。新增 `model/channel_cost_filter.go` + 测试；`model/channel_cache.go` 的 `GetRandomSatisfiedChannel` 加 `costFilter` 入参、候选集算完后插一段过滤；`service/channel_select.go` 新增 `BuildChannelCostFilter`；`middleware/distributor.go` 的"粘连"命中后补一次成本校验（不达标放弃粘连并记 warn）。**进货折扣直接从内存里的渠道对象读，过滤只读内存、不查库**；计费侧一行未改、缓存结构未动
  - **8b 择优**（本次）：候选线路不再只按"上游优先级"分层，改成按（进货折扣，上游优先级）两级键切成**有序的层**——层序就是重试顺序，**层内仍按权重随机**。这样"毛利最高"不等于"永远只走一条线"，不会把流量全压到一条线路上。默认毛利优先；客户被切成"稳定优先"时退回改造前行为；没有配路由策略的客户走默认口径（`model/channel_route_tier.go` 的 `buildChannelRouteTiers`）
  - **8c 报错**（本次）：一条保本线路都没有时不再回一句笼统的"没有不亏本的线路"，而是说清三件事——**为什么失败、还有哪条能走（按亏得最少列前 3 条）、走它每单亏几个百分点**，末尾提示客户联系管理员开通「允许走亏损线路」（`describeCostBreach`）
  - **8c 放行与留痕**（本次）：新建 `discount_routing_policies` 表（一个客户一行：择优策略 + 放行开关 + 操作人 + 备注；**没有配置行就是默认口径**——毛利优先、不允许）。被放行时候选集保持原样、按"亏得最少"优先挑一条，并在消费日志 `other.admin_info.cost_breach` 下留痕（渠道／进货折扣／售价折扣／每单亏损比例），同时记一条与请求关联的后端 warn。放在 `admin_info` 下是沿用 `quota_saturation` 的老做法：非管理员看日志时整块被剥掉，客户不会看到自己被标记"平台在赔钱"
- **运营入口**：折扣页右上角新增「路由策略」抽屉，按客户配择优策略（两档平铺单选）、放行开关和备注，可一键"恢复默认"。后端 3 个端点在 `/api/discount/admin/routing`（GET／PUT／DELETE），与试算／校验同组同鉴权。7 个语言文件各 +19 条
- **Q3 一并定下**：「客户同意走亏损线路」的入口选**后台按客户开开关**，不做"请求里带标记"——那是运营和客户谈出来的商务决定，做成请求标记等于客户能给自己开关。好处是操作人／时间／备注都留在库里

### 验证
- `go build ./...` ✅ ｜ `go test ./model/ -count=1` ✅ ｜ `go test ./controller/ -count=1` ✅ ｜ `go test ./service/ -count=1` 仍是那 2 条既有失败（渠道亲和性缓存，数量未增）｜ `bun run typecheck` ✅ ｜ `bun run build` ✅ ｜ `bun run test` ✅ 43 文件 / 212 用例 ｜ `bun run i18n:sync` ✅
- 新增后端用例：分层择优 3 个（毛利优先的分层顺序／毛利与优先级都相同的线路合进同一层／稳定优先退回改造前行为）、击穿报错 2 个（报错同时给出"为什么失败"与"还有哪条能走、亏多少"／按亏得最少列前 3 条）、路由策略默认口径 3 组、`NewChannelCostFilter` 读取与默认值 1 个
- 改动范围：新增后端文件 4 个 + 后端测试 2 个 + 前端抽屉 1 个；改既有文件后端 9 处／前端 4 处／7 个语言文件；**`relay/` 零改动**，计费侧只在 4 处消费日志各加一行留痕
- **遗留**：① `auto` 分组下详细报错会被既有写法吞掉（`service/channel_select.go` 里那句只取渠道、丢掉 error），最终退化成"没找到可用渠道"；② 用户／令牌**显式指定渠道**（`specific_channel_id`）仍是"照走不拦"，连日志也没留；③ 前端抽屉没有配套用例，仍是 43 文件 / 212 用例；④ 关闭内存缓存的部署（`MemoryCacheEnabled=false`）里成本过滤不生效，与本件 8a 遗留同一条

## [v0.18.1] - 2026-09-12（提交 `91886516`，已推送 origin/main，annotated 标签）

### 修复
- **补 v0.18.0 术语统一漏掉的 13 条**：这批词条的键名里既没有 `channel` 也没有 `vendor`（值里却含「渠道／供应商」），上一轮的按键名批量替换识别不了，需逐条看语境。复核脚本 `.docs/run/term-cgroup.mjs` 把每条判定写死在映射表里，不做规则推断
  - **属上游线路的 6 条改「上游供应商」**：渠道抽屉的字段说明 `Name, provider type, and availability.` → 「名称、上游供应商类型和可用状态。」、`Endpoint, provider-specific settings, and credentials.` → 「接口地址、上游供应商专属设置和凭据。」、`Provider-specific endpoint, account, and compatibility settings.`、`Required provider, authentication, model, and group settings`；新建抽屉引导句 `Name the channel, choose the provider, ...` 里的第二个词（原来只改了前半句，读起来一半新一半旧）；重试说明 `When enabled, if channels in the current group fail, ...` 的「下一组供应商」
  - **属模型厂商的 3 条改「厂商」**：`Filter models by provider, group, type, endpoint, and tags.`、`Refine models by provider, group, type, and tags.`、模型广场搜索框 `Search model name, provider, endpoint, or tag...`（该页 `provider` 实为厂商，取自 `usePricingData()` 的 `vendors`，与同族的 `Filter models by type, endpoint, vendor, group and tags` 口径一致）
- **渠道字段说明里 4 条「提供商」改「上游供应商」**：`Provider type (OpenAI, Anthropic, etc.)`、`API key from the provider`、`Custom API base URL. Leave empty to use provider default.`、`Map request model names to actual provider model names (JSON format)`。这 4 条键名里既没有 `channel` 也没有 `vendor`，但按代码位置确认是渠道表单的字段说明（`web/src/features/channels/constants.ts` 的 `FIELD_DESCRIPTIONS`）；同一张表里的 `NAME`／`MODELS` 早就写成「上游供应商」，留着「提供商」会让渠道抽屉里同屏两种叫法。脚本 `.docs/run/term-residual-pass.mjs`
- **繁体后端语言包补 3 条**：`i18n/locales/zh-TW.yaml` 的 `vendor.name_empty`／`name_exists`／`id_missing` 仍是「供應商」——v0.18.0 只改了简体 `zh-CN.yaml`。现简繁同步为「厂商／廠商」
- **仍然不改（本轮再次确认）**：`Payment Channel`（支付渠道）、OAuth 登录方式那一组（`Add OAuth Provider` 等）、`Hostname or IP of your SMTP provider`；繁中里 OAuth／SMTP 语境沿用的「供應商」也保持原样，别把两件事混成一个词

### 验证
- `bun run typecheck` ✅ ｜ `bun run build` ✅ ｜ `bunx vitest run` ✅ 43 文件 / 212 用例 ｜ `bun run i18n:sync` ✅ ｜ `go build ./...` ✅
- 复核对账（脚本 `.docs/run/term-count.mjs`）：简体语言包里裸露的「供应商」为 0（252 处全部是「上游供应商」的一部分），「渠道」剩 2 条（均为支付网关语境）；繁中「供應商」剩 31 条、简体「提供商」剩 40 条，绝大多数落在 OAuth 登录方式、公司介绍、SMTP、法律条款语境（按既定边界不改）
- **本轮没动、留待确认的 3 条**：含「上游」的泛称文案（模型分析聚合 `Aggregated traffic by upstream model provider`、渠道直通说明 `Forward requests directly to upstream providers...`、模型详情提示 `May be used for training by upstream provider`）。这三条在本仓库里检索不到 `t('...')` 的直接引用（应是经常量表间接取用），**语境没做实之前不动**——延续「键名 + 代码引用位置」双重确认的口径
- 改动仅 4 个语言文件（2 前端 + 2 后端）+ `VERSION`，无代码改动、无 `relay/` 改动
- **遗留同 v0.18.0**：v0.17.0 的 G1 逐笔比对（S-07 卡 6 步剧本）仍未跑，计费总开关保持默认关

## [v0.18.0] - 2026-09-12

### 变更
- **P1 第 9 件 · 术语文案统一（`Vendor`→厂商、`Channel`→上游供应商）**：界面文案里的「渠道」全部改称「上游供应商」，把「供应商」一词腾给模型厂商（统一叫「厂商」）。此前同一个上游账号在不同页面有四种叫法（渠道／上游／线路／Channel），而「供应商」又被同时用来指模型厂商，读起来容易与平台／经销商的三层归属混在一起
  - **简体中文**：216 条句子内嵌的「渠道」（渠道管理／渠道密钥／渠道亲和性／手动禁用渠道／测试并发数……），外加 23 条 `Vendor` 语境的「供应商」，以及 3 条 abilities 相关文案（「修复渠道一致性」→「修复上游供应商一致性」）
  - **繁体中文**：215 条「渠道」与 26 条「供應商」，另补一条此前漏译、一直显示英文的 `Upstream Providers`
  - **法／俄／日／越**：`Channel`／`Channels`／`Channel Affinity`／`Vendor`／`All Vendors`／`Upstream Providers` 六个概念名分别落到各语言（Canal→Fournisseur amont、Канал→Поставщик、チャネル→アップストリームプロバイダー、Kênh→Nhà cung cấp）。句子里散落的同义直译留给母语审校
  - **后端 `i18n/locales/zh-CN.yaml`**：客户调用 API 时可能看到的 16 条报错文案（`channel.*`、`distributor.*`）同步改口径
- 英文包一字未改：语言文件的键就是英文原文，`Channel` 继续作为内部概念名保留
- **边界判定（已排除，勿改）**：`Payment Channel`（支付通道）、`OAuth Provider`（登录方式）、`SMTP provider`（邮件服务商）均不在范围内。按「键名 + 代码引用位置」双重确认——OAuth 那组落在 `system-settings/auth/custom-oauth/`，与上游渠道无关；渠道字段说明（`Provider type`／`API key from the provider`）虽不含 channel 字样，经确认在 `features/channels/constants.ts` 中，属上游语境

### 验证
- `bun run typecheck` ✅ ｜ `bun run build` ✅ ｜ `bunx vitest run` ✅ 43 文件 / 212 用例 ｜ `bun run i18n:sync` ✅ ｜ `go build ./...` ✅
- 改动仅语言文件（7 个前端 + 1 个后端），无代码改动、无 `relay/` 改动
- **遗留**：v0.17.0 的 G1 逐笔比对（S-07 卡 6 步剧本）仍未跑，计费总开关保持默认关

## [v0.17.1] - 2026-09-12

### 修复
- **折扣表单按「经销商」口径校正**：归属与状态从下拉菜单改成平铺单选；分成比例只在「归属＝经销商」时出现，选「平台」时整格不渲染（此前选了经销商也不出现）
- **「免费」不再显示成「可用」**：计费模式的第三项原来复用全局词条 `Free`，而 `Free` 在磁盘空间语境下译作「可用」。改走独立词条 `Free of charge`
- **表格列头的「收费方式」此前一直是英文**：原键 `Billing mode`（大写 M）在语言文件里根本不存在，改用 `Plan billing mode`

### 补录
- **繁体中文补 122 条折扣文案**：方案、规则、试算、绑定四个界面此前在繁中里整片是英文原文（含报错提示与跨行说明）；简体补 6 条（折扣策略设置区「最小毛利比例」那一组）
- **6 个词条此前所有语言包都没有，连英文源文件也没有**：`Manual`／`Migration`／`Customer code` 这类只写在常量映射里、由函数间接取用，而 `i18n:sync` 只在语言文件之间对齐、不扫源码，因此永远补不进来。已一并补上，并按项目规范登记进 `src/i18n/static-keys.ts`
- **术语对齐简体口径**：进货折扣、线路、生效时间、范围取值

### 验证
- `bun run typecheck` ✅ ｜ `bun run build` ✅ ｜ `bun run test` ✅ 43 文件 / 212 用例 ｜ `bun run i18n:sync` ✅ ｜ `go build ./...` ✅
- 各语言包键数一致（5605）；折扣模块在两个中文包里经逐键复核，无英文残留、无缺失
- 改动仅前端折扣模块与语言文件，后端与 `relay/` 零改动
- **仍未做的仍是同一件事**：v0.17.0 的 G1 逐笔比对（S-07 卡 6 步剧本）还没跑，计费总开关保持默认关

## [v0.17.0] - 2026-09-12

### 新增
- **折扣参与计费**（第 7 件，**默认关闭**）：客户请求扣费时乘上他自己的折扣——绑 0.8 方案的客户实付 = 官方标价 × 分组倍率 × 0.8
- **运营设置里的总开关「折扣参与计费」**（系统设置 → 运营设置 → 折扣策略，默认关）：关着时折扣只用于试算与保存前检查，每笔仍按官方标价；打开后也只对「绑定了启用中方案」的客户生效，没绑方案的人照旧
- **消费日志记录折扣**：真打折时日志 `other` 里多出 `discount_ratio`／`discount_source`／`discount_plan_id` 三项；没打折时一个字段都不加，老日志形状不变——这是为了能分清「分组倍率本来就是 0.8」与「分组倍率 1.0 打了 8 折」

### 实现方式
- 折扣不单开一个乘数位，而是**乘进分组倍率**：`GroupRatioInfo.GroupRatio` 从此是「这一单真正生效的倍率」。这样预扣、重试重算、结算、任务计费、实时流式扣费读到的必然是同一个值——分成两处最容易出的错是「预扣按折后算、结算按原价扣」
- 新增 2 个后端文件：`model/discount_billing.go`（把折扣收成一个只读结构 `BillingDiscount`，含总开关与合法性兜底）、`relay/helper/discount.go`（`ApplyUserDiscountRatio`）
- 既有的 `HandleGroupRatio` 只在末尾多一行调用（覆盖预扣、重试、按次计费）；另有 **2 条不走它的计费路径单独接线**：`service/quota.go` 的实时流式预扣、`service/task_billing.go` 的任务 token 重算——漏了前者流式请求不打折，漏了后者任务会把折扣「补扣」回去
- 类型 `GroupRatioInfo` 加 3 个只读字段（折扣乘数／来源／方案 id），不参与金额计算，只供日志与对账
- 前端既有文件改 5 处：运营设置的类型、默认值、分区注册、折扣策略分区（加开关）、该分区测试

### 出错时的口径
- 折扣读不出来、不是数字、落到 (0,1] 之外，一律**回退官方标价**并记一条 `SysError`：折扣子系统故障不能把整条计费链路带停，但也不能静默发生。回退是保守侧（这一单不打折），不会因为一个读不到的数字把客户的钱算错
- 折扣只可能把倍率变小（范围 (0,1]），不放大任何金额，因此不会产生负额度或绕过 `common/quota_math.go` 的饱和审计

### 验证
- `go build ./...` ✅ ｜ `bun run typecheck` ✅ ｜ `bun run build` ✅ ｜ `bun run test` ✅ 43 文件 / 212 用例（新增 2 个 Go 测试文件 8 个用例 + 前端 2 个用例）｜ `bun run i18n:sync` ✅
- `go test ./model/ ./relay/helper/ ./service/ ./controller/`：model、relay/helper、controller 全过；service 有 2 条既有失败（`TestObserveChannelAffinityUsageCacheByRelayFormat_*`，渠道亲和性缓存，与本次无关）——已用 `git stash` 暂存本次改动复跑确认，改造前同样失败
- 语言文件确认只增行不改行：7 个文件各 +2 行、0 删除
- **尚未做**：`README.md` §二的 G1 逐笔比对（未绑方案用户「逐笔与改造前一致」的实测）还没跑。代码已接入，但**总开关默认关**，线上行为零变化

### 文档
- `task-04-p1-discount/03-step-log.md` 补 S-07 记录；`02-work-queue.md`、`04-frontend-plan.md`、`master-plan.md` 同步进度

## [v0.16.0] - 2026-09-12

### 新增
- **折扣方案管理页**（`/discounts`，此前只有「试算」一个入口）：方案列表带名字／ID 搜索、状态与归属筛选、分页，行内可启用停用（带确认）、编辑、删除；右上角「新建方案」+ 原有「试算」并排
- **方案表单**（新建与编辑共用抽屉）：名字、归属（平台／代理商，选代理商才要填代理商用户 ID）、基础折扣、最低折扣、计费方式、分成比例、状态、备注；折扣一律按 0～1 小数填写并转成后端要的 6 位小数字符串
- **保存前检查与二次确认**：编辑既有方案时点保存先调 `POST /api/discount/admin/plans/:id/validate`，有会亏钱的规则就先把"哪个模型／厂商、折扣多少、为什么亏、哪些线路还能走"列出来，按钮变成「仍然保存」，再点一次才真写库——**只提示不阻断**（亏本促销与大客户见面礼都是真实业务）；新建方案还没 ID 时说明"下次保存才会跑检查"，不假装检查过
- **规则抽屉**：给方案配"哪个模型／厂商走多少折扣"，可增删改，带优先级（数字大的先命中）与生效时间；规则加载失败单独给错误态，不吞
- **绑定抽屉**：按用户名搜客户后绑定，列表可解绑；绑定接口只回客户 ID，列表同时显示搜到的用户名与 ID，认不出人时不编造名字
- 7 个语言文件各 +88 条词条（`zh` 已译，其余语言暂为英文）

### 界面口径
- 折扣／比例统一显示成百分比，取值原样按 6 位小数解析；后端 `null` 一律显示「—」，**"没录"不能显示成 0**
- 归属、状态、计费方式、折扣来源、违规原因都走"表驱动"标签：遇见没见过的枚举值原样显示，不吞不改
- 删除方案与解绑客户都要二次确认，确认文案写清后果（"彻底删除"／"解绑后立刻回到原价"）

### 实现方式
- 全部按"能新增就不改写"：新增 15 个文件（`discounts-table/columns/row-actions/primary-buttons/provider/dialogs`、`plan-mutate-drawer`、`plan-delete-dialog`、`rules-drawer`、`bindings-drawer`、`lib/plan-form.ts`、`lib/rule-form.ts`、`lib/ratio.ts` 及 2 个测试）
- 既有文件改 6 处：`discounts/{types,api,constants,index}.tsx`、`lib/index.ts`（导出口）、`routes/_authenticated/discounts/index.tsx`（补搜索参数校验）
- **后端一行未改**，仍只有前端；**仍未接计费**

### 验证
- `bun run typecheck` ✅ ｜ `bun run test` ✅ 43 文件 / 210 用例（新增 2 文件 10 用例）｜ `bun run build` ✅ ｜ `bun run i18n:sync` ✅ ｜ `go build ./...` ✅（确认重新构建的 `web/dist` 能被正常 embed）
- 语言文件确认只增行不改行：7 个文件各 +88 行、0 删除
- `oxlint` 对 `discounts` 仅剩 1 条 `react(only-export-components)`（provider 文件同时导出组件与 hook），与 `users`／`channels`／`models` 等 8 个同类 provider 写法一致，未改
- 人工目视（界面复查）尚未做过

### 实现期经验
- **`bun run format` 会扫全仓库**：`format-with-protected-headers.mjs --write` 不限路径，本次误伤 71 个无关既有文件（business-settings／home／pricing 等），已全部回退。以后只对目标目录做格式检查，别在功能提交前跑全仓库 `--write`
- **i18n 同步脚本不扫源码**：`sync-i18n.mjs` 只在语言文件之间对齐，新词条必须先手工写进 `en.json`／`zh.json`（按字母序逐行插入，避免整体重写触发转义键问题），再跑 `i18n:sync` 补其余语言
- **测试断言别指望长文本整体匹配**：一句提示拆在多个元素里时 `getByText` 匹配不到，把要断言的那段文案单独包一个元素（或改用 `exact: false`）
- **抽屉测试要真操作**：客户选择先点 `combobox` 再点 `role="option"`；只改输入框不选客户会一直停在"请先选客户"

### 文档
- `task-04-p1-discount/03-step-log.md` 补 S-06 记录；`02-work-queue.md`、`master-plan.md` 同步进度

## [v0.15.0] - 2026-09-12

### 新增
- **折扣方案页 `/discounts` 与「试算」抽屉**（侧边栏「业务管理 → 用户管理 → 折扣方案」）：先按用户名搜客户并选中，再填模型名（支持 `claude-*` 这类结尾通配），点「试算」出结果——实际折扣、折扣是谁给的、命中哪条规则、客户挂在哪个方案、毛利底线多少，以及这个模型会走的每条上游线路的进货折扣／毛利／是否达标
- 侧边栏「业务管理 → 用户管理」新增一格「折扣方案」（`Percent` 图标），`pathPattern` 同步加 `discounts`（不加会掉回根导航），侧栏解析测试补 `/discounts` 断言
- 7 个语言文件各 +33 条词条（`zh` 已译，其余语言暂为英文）

### 界面口径
- 与接口同一口径：进货折扣／毛利／是否达标为 `null` 时显示「—」，顶部另给一句「成本未知」——**"不知道成本"不能显示成"赚 0 元"**，前者运营会去补进货价，后者会让他以为这单白赚
- 折扣与毛利按 6 位小数字符串转百分比；毛利低于底线标红，但**只提示不阻断**（亏本促销与大客户见面礼都是真实业务）
- 遇见没见过的来源／范围取值原样显示，不吞不改
- 页面本身只放试算入口 + 一句"方案列表还没做"的说明，不假装已有列表

### 实现方式
- 全部按"能新增就不改写"：新增 `web/src/features/discounts/`（`types.ts` 对齐后端出参、`api.ts`、`constants.ts`、`lib/format.ts`、抽屉组件、页面出口，共 9 个文件）与路由 `web/src/routes/_authenticated/discounts/index.tsx`（管理员守卫，照订阅页写法）
- 既有文件只动 3 处：`business-settings.config.ts` 插一格菜单 + `pathPattern` 一处正则，`sidebar-view-registry.test.ts` 补一行断言；`routeTree.gen.ts` 是生成物
- **后端一行未改**，`relay/` 自然零改动；**仍未接计费**

### 验证
- `bun run typecheck` ✅ ｜ `bunx vitest run` ✅ 41 文件 / 200 用例（新增 2 文件 8 用例）｜ `bun run build` ✅ ｜ `bun run i18n:sync` ✅
- 版权头检查（`scripts/add-copyright.mjs --check`）与格式化检查（`scripts/format-with-protected-headers.mjs --check`）对 `discounts` 均无输出
- 人工目视（界面复查）尚未做过

### 实现期经验
- **`null` 不能显示成 0**：抽屉测试专门断言没录成本时界面上不出现 `0.00%`
- **抽屉类交互的测试要真操作一遍**：用 `fireEvent.click` 打开 Select 再点 `role="option"`；只填模型名会一直停在"请先选客户"，测出来是假绿
- **版权头 `*/` 后面不能空行**：`add-copyright.mjs` 会把空行吃掉，于是版权检查与格式检查同时对同一个文件报错
- **`en.json` 不要整体重写**：文件里有需转义的键（`sync-i18n.mjs` 的 `OBFUSCATED_KEYS`），按字母序逐行插入才能让 `git diff` 只多 33 行

### 文档
- `task-04-p1-discount/03-step-log.md` 新增 S-05 复查卡；`04-frontend-plan.md` 补 §3.4 逐文件清单与 §六.2 词条清单；`02-work-queue.md`、`master-plan.md` 同步进度

## [v0.14.0] - 2026-09-12

### 新增
- **保存前提示接口 `POST /api/discount/admin/plans/:id/validate`**（管理员）：存方案之前算一遍"这个方案会不会亏"，会亏就指出是哪个模型／厂商，并给出最低可用进货折扣与判定用的毛利底线

### 接口口径
- 入参：路径 `:id` 为方案 ID；body **可选**且**允许为空**，可带 `user_id`（按该客户实际折扣算）与 `channel_id`（只算指定线路，不在任何规则范围内时只给提示）
- 出参：`passed` / `min_margin_ratio` / `violations[]`（含 `scope_type`、`scope_value`、`discount`、`reason`、`detail`、`available_channels`）/ `warnings[]`
- 判定分两层：**折扣层**看规则折扣是否低于方案最低折扣、最低折扣是否高于基础折扣（方案级，`scope_type` 为 `plan`）；**成本层**看规则作用范围内"最低进货折扣 ≤ 折扣 − 毛利底线"是否成立——等价于"至少有一条线路顶得住"，绝不因为某条线路亏本就判整个方案亏
- 成本未知（线路没录进货折扣）一律只进 `warnings`，不参与结论，与试算同一口径
- **只提示不阻断**：亏本促销与大客户见面礼都是真实业务，前端标红即可，不拦保存
- 只读接口，不写任何表；**仍未接计费**：`relay/` 一行未改

### 实现方式
- 全部按"能新增就不改写"：新增 `model/discount_validate.go`（规则作用范围还原成模型名集合）、`service/discount_validate.go`（折扣层 + 成本层判定）、`controller/discount_validate.go`（参数解析与错误码）
- `model/discount_simulate.go` 的候选线路查询扩成 `GetDiscountChannelCandidatesByModels`（一次问一批模型），原单模型函数改为调用它；厂商级规则要按整片模型问，逐个问会打出一串查询
- 既有文件仍只动 `router/api-router.go` 一行（挂载新端点）

### 验证
- `go build ./...` ✅ ｜ `go vet` ✅
- `go test ./service/ -run TestValidateDiscount` ✅ 10 例：规则折扣低于最低折扣、最低折扣高于基础折扣、成本击穿、最便宜线路顶得住即通过、两种"打不到线路"、`claude-*` 通配的通过与击穿、没录进货折扣只给 warning、指定线路、停用规则跳过、方案／客户不存在、客户分组口径
- `go test ./controller/ -run TestDiscountValidate` ✅ 参数校验（无效 ID／非正 `user_id`／非正 `channel_id`／非 JSON body 回 400）+ 空 body 合法
- `go test ./service/ ./model/ ./controller/ -run Discount` ✅ 旧折扣测试无回归
- **真接口**：在 3009 端口起临时实例（临时 SQLite，不碰正在跑的 3001），`POST /api/discount/admin/plans/1/validate` 与既有 `GET /api/discount/admin/plans` 都回 401 → 证明新路由确已挂载、鉴权链生效；验完停掉临时实例，3001 未受影响
- 前端本版无改动

### 实现期经验
- **规则停用只能走更新路径**：`discount_rules.status` 列带 `default:1`，新建时传 0 会被数据库默认值盖掉（现象是"新建即停用存不进去"，新建后改成停用是正常的）。要不要去掉这个默认值另议
- `LIKE` 做前缀通配要先粗筛再用 `strings.HasPrefix` 复筛，否则模型名里的 `_` / `%` 会被当成通配符误匹配

### 文档
- `task-04-p1-discount/03-step-log.md` 新增 S-04 复查卡；`01-simulate-api.md` 补 §5.4 实现期口径；`02-work-queue.md`、`master-plan.md` 同步进度

## [v0.13.0] - 2026-09-12

### 新增
- **试算接口 `GET /api/discount/admin/simulate`**（管理员）：给「客户 + 模型」，返回按几折、这个折扣是谁给的、该模型会走哪几条上游线路、每条线路的毛利与是否赔本

### 接口口径
- 入参：`user_id`（必填）、`model`（必填）、`channel_id`（可选；只算指定线路，且它必须在候选里，否则回 400）
- 出参：`user` / `model` / `vendor` / `plan` / `resolution`（含命中规则）/ `cost_known` / `min_margin_ratio` / `channels[]` / `warnings[]`
- 毛利 = 客户折扣 ÷ 进货折扣 − 1（6 位小数）；`passes_floor` 按后台"毛利底线"判断——**只提示不阻断**，亏本促销与大客户补贴仍可做
- 线路没录进货折扣时：该条三个值字段给 `null`、`cost_known` 为 `false`、`warnings` 里指名是哪条线路。**"不知道成本"不能显示成"赚 0 元"**
- 候选线路**直查能力表**（不读渠道缓存，避免缓存刷新时机干扰），同渠道挂多分组时去重，客户分组为 `auto` 时展开成实际可用分组
- 只读接口，不写任何表；**路径由设计稿写的 `/api/discount/simulate` 移到既有 `/api/discount/admin` 组下**，与另外 13 个管理端点同组、同鉴权、同前缀
- **仍未接计费**：`relay/` 一行未改

### 实现方式
- 全部按"能新增就不改写"：新增 `model/discount_simulate.go`（候选线路查询）、`service/discount_simulate.go`（组装与毛利计算）、`controller/discount_simulate.go`（参数解析与错误码）；既有文件**只动 `router/api-router.go` 一行**

### 验证
- `go build ./...` ✅
- `go test ./service/ -run TestSimulateDiscount` ✅ 9 例：命中模型级规则、未录进货价、未绑方案、亏本线路、毛利底线抬高、无可用线路、指定线路与去重、进货价脏数据、客户不存在
- `go test ./controller/ -run TestSimulateDiscount` ✅ 7 例参数校验（参数不对回 400 且提示可读，不落到 500）
- `go test ./service/ ./model/ -run Discount` ✅ 旧折扣测试无回归
- **真接口**：在 3009 端口起临时实例（临时 SQLite，不碰正在跑的 3001），`/api/discount/admin/simulate` 与既有 `/api/discount/admin/plans` 都回 401 → 证明新路由确已挂载、鉴权链生效；验完停掉临时实例，3001 未受影响
- 前端本版无改动（typecheck／build／test／i18n:sync 仍跑一遍确认无影响）

### 文档
- `task-04-p1-discount/03-step-log.md` 新增 S-03 复查卡（做了什么／这步没动什么／怎么验证／怎么人工复查／怎么退回）
- `01-simulate-api.md` 同步路径调整与实现期口径；`02-work-queue.md`、`04-frontend-plan.md`、任务 `README.md`、`master-plan.md` 同步进度与看板

## [v0.12.1] - 2026-09-12

### 修复
- **补齐渠道"进货折扣"的两条文案词条**：`Cost Ratio` 与它下面那句说明，原先只有代码里的 `t('...')` 字面量，没有进任何语言文件——**中文界面会显示英文**。已补进 `en.json`（其余 6 个语言随之同步）并写上中文译文
- **根因**：`bun run i18n:sync` 只做"以 `en.json` 为基准，把缺的键补进其它语言"，**不会**从代码里扫描新写的 `t('...')`。所以新文案必须**手工先进 `en.json`** 再同步；漏了这一步不会有任何报错，静默显示英文
- 这是 v0.12.0 的漏项，不是新功能。由"每步一记"的复查习惯发现（做法见任务台账 S-02 卡的"复查发现"）

### 验证
- `bun run typecheck` ✅ ｜ `bun run build` ✅ ｜ `bun run test` ✅ 39 文件 / 192 用例 ｜ `bun run i18n:sync` ✅（复跑一次，各语言 `missingCount` 全为 0）
- 本版**只动语言文件与文档**，未改任何逻辑与样式

### 文档
- 新增 `task-04-p1-discount/03-step-log.md`：**步骤记录（复查台账）**——每步一张卡，写清做了什么、这步没动什么、怎么验证、怎么人工复查、出问题怎么退回
- 新增 `task-04-p1-discount/04-frontend-plan.md`：**前端影响与重新装配规划**——已动/将动的前端文件、前端装配点地图、8 条 i18n 词条清单、装配后的核销清单
- "每步一记"已写进 `.docs/README.md` §四 标准工作流，之后每个任务都照此执行

## [v0.12.0] - 2026-09-12

### 说明
- **P1 折扣内核再落两块：折扣解析函数（第 1 件）与上游进货价字段／录入（第 2 件）**。解析函数是 `simulate`、保存前校验、以后真正扣费三者的公共地基，它错了后面全错，所以先做、且只做函数与用例；进货价提前做是为了让试算能回答"赚不赚"，**但没有提前"按进货价选线路"**
- **能新增就不改写**：解析逻辑没有写进既有的 `model/discount.go`，而是新增 `model/discount_resolve.go`；毛利底线设置项落在新增的 `setting/operation_setting/discount_setting.go`；后台设置分区同样是新增文件。既有大文件（渠道编辑抽屉 214 KB）只做局部插入
- **进货折扣用指针 `*string` + `varchar(20)`**：指针才能区分"请求里没带这个字段"（保持原值）与"显式清空"；列类型避开 `decimal`，因为 `channels` 是核心大表，SQLite 下 decimal 会让每次启动整表重建（v0.11.0 已记录过同一现象）

### 新增
- `model/discount_resolve.go`：`ResolveUserDiscount(userId, modelName)` 按「模型级规则 → 厂商级规则 → 方案基础折扣 → 1.0」解析并返回来源；通配只支持结尾 `*`
- `model/discount_resolve_test.go`：11 个子测试 + 1 张通配边界表（含"先建后停"才是真实停用路径这条实现期经验）
- `setting/operation_setting/discount_setting.go`：`discount_setting.min_margin_ratio`（毛利底线，默认 0）
- `web/src/features/system-settings/pricing/discount-setting-section.tsx`：运营设置页新增"折扣策略"分区，含 3 个回归用例（回显、留空归一化为 0、超过 1 被拦下）
- `channels.cost_ratio`（进货折扣）与 `channels.cost_updated_at`（更新时间）

### 变更
- `controller/channel.go`：仅当 `cost_ratio` 真的变了才刷 `cost_updated_at`；新建时带非空值同样落时间戳
- `controller/channel_authz.go`：`cost_ratio` 归敏感字段（改它要走敏感变更权限），`cost_updated_at` 归只读字段（由服务端维护，客户端传了也会被清掉）
- 渠道编辑抽屉新增"进货折扣"输入框，旁边注明口径："按平台标价折算，不是上游报价单上的数字"

### 明确没做（仍是 P2）
- **按进货价选线路**：路由侧过滤、择优、"会亏需要客户同意"全部未做。本次只是把进货价记下来，`relay/` 与选线路逻辑一行未改

### 验证
- `go build ./...` ✅ ｜ `go test ./model/ -run "TestResolveUserDiscount|TestMatchDiscountScope"` ✅ ｜ `go test ./controller/` ✅
- `bun run typecheck` ✅ ｜ `bun run build` ✅ ｜ `bun run test` ✅ 39 文件 / 192 用例 ｜ `bun run i18n:sync` ✅（新键已补齐 7 个语言文件）
- 既有失败说明：`go test ./service/` 有两个 channel-affinity 缓存测试失败，已用"临时移开本次新文件后同样失败"的对照实验确认是**既有问题**（两个测试还会互相污染计数），本次未处理

### 遗留
- `simulate`（第 3 件）、保存前 `validate`（第 4 件）、前端试算抽屉与方案管理页（第 5、6 件）、折扣接计费（第 7 件，含 G1 逐笔比对这个唯一串行卡点）待做

## [v0.11.0] - 2026-09-12

### 说明
- **P1 折扣内核 G0 影子阶段第二块落地：折扣方案管理接口**。表结构上一版已就位，本版补齐管理员侧的配置读写，**全程不接计费**（`relay/` 一行未改），未绑定方案的用户计费结果与改造前完全一致
- **绑定与 `users.discount_plan_id` 快路径放在同一事务里写**：后者是后续计费读取折扣的入口，先写绑定再补快路径会留下"配置看起来生效了、计费却读不到"的窗口
- **折扣口径在入口处收紧**：折扣必须大于 0（0 等于免费）且不超过 1（不允许加价）；`min_discount` 不得高于 `base_discount`；落库前统一规范成固定 6 位小数字符串，避免 SQLite 读回 `0.9`、MySQL 读回 `0.900000` 这类跨库差异

### 新增
- `model/discount.go`：折扣方案／规则／客户绑定的读写与口径校验；同一主体多条生效绑定时按来源 `manual > subscription > customer_code > migration` 取用，同级取最新
- `controller/discount.go` + `router/api-router.go`：`/api/discount/admin` 下 13 个管理员端点（方案 6、规则 4、绑定 3）
- 测试：折扣取值边界、绑定来源优先级与生效窗口、绑定／解绑对 `users.discount_plan_id` 快路径的同步

### 验证
- `go build ./...` ｜ `go test ./model/` ｜ `bun run typecheck` ｜ `bun run build` ｜ `bunx vitest run`（38 文件 / 189 用例）
- 以 3007 端口真实启动两次，确认路由注册无冲突、数据库迁移正常；顺带确认一处**既有现象**（非本版引入）：SQLite 下带 `type:int` 标签的表每次启动都会被整表重建，详见 `task-04-p1-discount/README.md` §五

## [v0.10.0] - 2026-09-12

### 说明
- **V2 控制台后台对齐 + V3 模型广场对齐，两个工作流一起落地**。这是 P0 视觉统一阶段继 V1 批次 2 之后的最后两块
- **开工前按 §2.1 三条对冲逐页复核**（V2 的既有要求）。结论：**三条全部天然满足** —— `glass-1..5` 全站零调用、控制台零 `rounded-3xl/4xl`、网格底纹控制台零处；唯一的光晕在概览页，§2.1 明确豁免"概览类卡片与空状态"
- **实际改动比预期小得多**：V1 批次 2 把基元统一之后，凡是用基元的页面已自动跟着统一，本版只需收拾"没用基元、自己手写外壳"的漏网容器。换句话说，V2 字面意思的"全量对齐"在 V1 收口那天就已经完成了大半
- **顺带确立一条规格判定**：表格壳（`rounded-lg`）与卡片（`rounded-2xl`）是**两套并存的正确规格**，不是不一致；本次按此判定保留了所有表格壳

### 变更（10 处 / 9 个文件）
- **V2 控制台（3 处 / 3 个文件）**
  - `features/system-info/components/system-tasks-panel.tsx`、`system-instances-panel.tsx`：面板外壳 `rounded-lg` → `rounded-2xl`，与全站面板基元 `PanelWrapper` 取齐。这两处原本自带 `shadow-card`（即自认为是卡片），圆角却不是卡片规格，是本节仅有的"同类不同规格"
  - `features/usage-logs/components/usage-logs-mobile-card.tsx`（2 处：骨架 + 真实列表）：外壳描边 `border-border/50` → `border-border`。日志页属控制台，按 §2.1「共享基元与控制台用全值」；圆角**保持 `lg` 不改** —— 它与 `data-table/layout/mobile-card-list.tsx` 同属表格壳体系
- **V3 模型广场（6 处 / 5 个文件）**
  - `features/pricing/components/model-card.tsx`：模型卡由 `rounded-xl border`（无底色、无阴影、悬停只变色）改为 `rounded-2xl border-border/50 bg-card shadow-card`，悬停改 `-translate-y-1` + `shadow-card-lift`，与首页促销卡、关于页卡片共用同一套悬浮语言
  - `features/pricing/components/loading-skeleton.tsx`：骨架卡同步 `rounded-2xl border-border/50`，避免加载完成时外形跳变
  - `features/pricing/components/pricing-sidebar.tsx`、`pricing-toolbar.tsx`：容器 `rounded-xl` → `rounded-2xl` 并补 `bg-card` —— 两者含筛选表单，属"表单类"容器，必须实心才能隔开页头的 `brand-glow-page` 光晕
  - `features/pricing/components/model-details.tsx`：详情抽屉定价区块由 `bg-card/60 + rounded-xl + shadow-sm` 改为 `bg-card + rounded-2xl + shadow-card`；同文件"能力／模态"内容块由 `rounded-xl` 改为 `rounded-lg`，与该文件其余 11 处内层块统一
- **顺带覆盖（1 处）**
  - `features/setup/components/complete-step.tsx`：完成页配置卡 `rounded-xl` → `rounded-2xl`（自带 `shadow-card`，同属卡片规格）。`setup` 向导虽不在 `_authenticated` 下，但风格属控制台，一并处理

### 有意不改（已逐条核实）
- **概览页光晕**（`dashboard/components/overview/overview-dashboard.tsx`）：§2.1 原文"控制台仅限概览类卡片与空状态"，属豁免范围
- **`summary-cards.tsx` 的 `color-mix` 分区背景**：已在用 `--overview-accent-*` token。§8.5 记的"属控制台体系（V2 范围）"意指"它已合规、只是不该在 V1 的光晕收敛里动"，本次核实后确认无需修改
- **手写 `<button>` 的 244 个文件**：制度禁止的是"**新增**"，存量不属 V2 范围
- **`data-table` 表格壳系列**：`rounded-lg` 是既有正确规格
- **badge／头像／chip 的 `rounded-4xl`／`rounded-full`**：胶囊语义正确
- **`setup` 其余 3 处小块**（步骤指示、选择块、信息行）：尺寸小、无阴影，属"卡中卡"，保持 `xl`／`lg`

### 需要留意（观感）
- **模型广场是本次变化最明显的一处**：模型卡从"透明无阴影、悬停只变色"变成"实心底色 + 静止阴影 + 悬停上浮"。这与业务方"所有卡片像终端卡一样有阴影并可悬浮"的要求一致，但模型广场一屏卡片多（默认 9 张），建议优先目视
- 圆角变化都很温和：`--radius` 已减半，`rounded-xl` = 11.2px、`rounded-2xl` = 14.4px，本次大部分改动只是 3.2px 的位移

### 验证
- `typecheck` ✅ ｜ `build` ✅ ｜ `vitest` ✅ 38 文件 / 189 用例 ｜ `i18n:sync` ✅（无 diff）｜ `git status` 恰为本次改动的 9 个源码文件
- 本版**纯类名调整**，未新增／未改动任何 i18n key，未改动任何逻辑

### 遗留（待人工）
- **本版建议目视**：模型广场（卡片视图 + 表格视图 + 详情抽屉 + 筛选侧栏）、系统信息页两个面板、调用记录移动端列表、setup 向导完成页
- **V2／V3 追加要求**：三身份可见性回归（未登录／客户／管理员）+ 明暗双主题对照
- **公开页 `rankings/*` 的 4 处无阴影容器**：§8.9 登记的原始待办之一，不在 V2（控制台）也不在 V3（`features/pricing`）范围内，**仍挂账**
- 上一批遗留不变：全站明暗双主题目视回归；"不带转圈的加载态"全仓库复查；fr / ru / ja / vi 术语值层待母语审校；504 / 524 重试免责声明保留；调用记录详情 Pre-consumed / Raw Quota 留待 P5

## [v0.10.3] - 2026-09-12

### 说明
- 修复「同一侧边栏里中文在不同浏览器显示大小不一致」的问题：`--font-sans` 只写了 `'Public Sans', sans-serif`，中文要靠通用的 `sans-serif` 关键字兜底，而通用关键字在 Edge 里会被浏览器自身的字体设置改写

### 变更
- `web/src/styles/theme.css`
  - `--font-sans` 从 `'Public Sans', sans-serif` 扩展为显式列出 CJK 无衬线字体（PingFang SC / Microsoft YaHei / Hiragino Sans GB / Noto Sans SC·TC·JP·KR / Source Han Sans SC·TC / Heiti SC / WenQuanYi Micro Hei），最后才回落到 `sans-serif`
  - 与 `--font-serif` 现有做法保持一致（该栈早已显式列出 CJK 衬线字体，并附注释说明通用关键字在 Windows 上不可靠）

### 验证
- `bun run typecheck` ✅
- `bun run build` ✅（index.02cf610d86.js）
- `bun run test` ✅ 38 文件 / 189 用例
- 双浏览器实测：用无头 Chrome 与「字体全部设为 STXihei」的 Edge 临时配置渲染同一份侧边栏测试页，修复后两边中文渲染一致

### 备注
- 本机 Edge 的 `Preferences` 里 `webkit.webprefs.fonts` 的 standard/sansserif/serif/fixed 四项全部被设为 `STXihei`，这是 Edge 侧差异的来源；Chrome 四项均未设置，走浏览器默认。

## [v0.10.2] - 2026-09-12

### 说明
- 工作区侧边栏二级菜单的缩进与激活态字重继续对齐一级菜单（v0.10.1 的跟进修复）

### 变更
- `components/ui/sidebar.tsx`
  - `SidebarMenuSub` 左内边距由 `px-3` 加大到 `px-4`，二级菜单相对一级菜单的缩进更明显
  - `SidebarMenuSubButton` 补充 `data-active:font-medium`，激活时与一级菜单的字重一致

### 验证
- `go build ./...` ✅
- `bun run typecheck` ✅
- `bun run build` ✅（web/dist 已生成，index.b31e5820f0.js）
- `bunx vitest run` ✅ 38 文件
- `bun run i18n:sync` ✅（无 diff）

### 备注
- 本地 3001 端口的后端进程此前一直在提供旧的前端产物（`go:embed` 在编译期固化 `web/dist`），导致 v0.10.1 的改动在页面上看不到；本次已重启后端并把新产物嵌入。

## [v0.10.1] - 2026-09-12

### 说明
- 修复业务管理、系统管理两个工作区入口的落地 URL，并统一工作区侧边栏二级菜单与一级菜单的视觉规格

### 变更
- `hooks/use-sidebar-data.ts`
  - 业务管理入口 `/business-settings/billing/quota` → `/channels`（工作区侧边栏第一项）
  - 系统管理入口 `/system-settings/auth/oauth` → `/system-info`（工作区侧边栏第一项）
  - 保留 `configUrls` 不变，模块可见性逻辑无变化
- `components/ui/sidebar.tsx`
  - `SidebarMenuSubButton` 行高由 `h-7` 改为 `h-8`，显式使用 `text-sm` 并占满整行，图标颜色随文字状态变化
  - `SidebarMenuSub` 缩进由 `mx-3.5 px-2.5` 加大到 `mx-4 px-3`，层级感更明显
- `features/system-settings/utils/section-registry.ts`
  - `SectionDefinition` 支持可选 `icon` 字段，`getSectionNavItems` 透传图标
- `components/layout/config/business-settings.config.ts`、`components/layout/config/system-settings.config.ts` 及 8 个业务/系统配置注册表
  - 为所有二级菜单项补齐 Lucide 图标

### 验证
- `go build ./...` ✅
- `bun run typecheck` ✅
- `bun run build` ✅（web/dist 已生成）
- `bunx vitest run` ✅ 38 文件 / 189 用例
- `bun run i18n:sync` ✅（无 diff）

## [v0.9.0] - 2026-09-12

### 说明
- **空态收尾下半：编辑器／面板／表格里的手搓空态收进 `EmptyState`。空态这一项到此收完**
- **全量对账**：`v0.7.1` 盘出 **36 处**手搓空态 → 两版合计收进基元 **27 处**（`v0.8.0` 14 处 + 本版 13 处），另 **9 处**逐条写明理由判为不改。36 = 27 + 9，账目闭合
- **修正 `v0.8.0` 条目里的一处计数错误**：该版实际改走基元 15 处（14 个空态 + 1 个加载态），原文误记为 14 处（13 个空态 + 1 个加载态）。计时线同步修正

### 变更（13 处空态 / 10 个文件）
- **系统信息**：`features/system-info/components/system-tasks-panel.tsx`（3 处：整块空态、进行中任务分组、历史任务分组）、`system-instances-panel.tsx`（1 处）
  - 两处的整块空态原本自己画了"灰底圆角方块 + 图标 + 一行文字"，改用基元的 `icon` 参数直接传原有图标（`ListChecks` / `ServerCog`），图标的大小与位置交给基元
  - 两个任务分组的空态保留原来的虚线框（`className='rounded-md border border-dashed'`）
- **模型与分组编辑器**：`features/system-settings/models/model-ratio-visual-editor.tsx`（2 处）、`group-special-usable-editor.tsx`（1 处）、`upstream-ratio-sync-table.tsx`（1 处）
  - `model-ratio-visual-editor` 右侧详情面板的占位里带一颗"添加模型"按钮，改用基元的 `action` 参数，按钮的排版交给基元
  - `upstream-ratio-sync-table` 的空态与它上面的 `LoadingState` 现在边框写法一致（都是 `rounded-md border`）
- **后台计费类可视化编辑器 4 个**：`payment-methods-visual-editor.tsx`、`creem-products-visual-editor.tsx`、`amount-options-visual-editor.tsx`、`amount-discount-visual-editor.tsx`（各 1 处）
  - 前两个的空态文案跟着"有没有在搜索"变，改后仍是同一个三元表达式，只是交给基元的 `title`
- **订阅套餐卡**：`features/plans/components/subscription-plans-card.tsx`（1 处）

### 有意不改（5 处，已在 `ui-consistency.md` §十 登记，逐条写明理由）
- **`features/channels/components/model-mapping-editor.tsx` 与 `components/json-editor.tsx`（各 1 处）**：两处的占位都写在 `flex h-24` 的框里，是表单中"键值对列表"的紧凑占位，上下紧贴着表单控件。换成 200px 会把表单撑开一大块空白，而它们要表达的只是"这里还没有内容"
- **`components/model-group-selector.tsx`（1 处）**：它在模型选择下拉框的列表内部，字号被压到 `text-[12px]`，属下拉列表内的提示，不是区块空态
- **`features/dashboard/components/models/performance-overview.tsx`（1 处）**：它是仪表盘里的一条 KPI 横条，有数据时也是一条约 40px 高的细横条。空态要保持同一形态，改成 200px 的区块会破坏这个组件在仪表盘里的横条布局
- **`features/playground/components/chat/playground-empty-state.tsx`（1 处）**：它不是通用空态，而是一个专门设计的引导块"开始对话"，里面是 2×2 的开场提示按钮，高度按视口算（`min-h-[min(520px,calc(100svh-18rem))]`）。基元只有"标题 + 副标题 + 一块操作区"三段，装不下这个版式

### 需要留意（观感）
- 本批把小卡片里"一行灰字"的空态放大了：`group-special-usable-editor`、`subscription-plans-card`、`amount-options-visual-editor`、`amount-discount-visual-editor` 这几处原来只有约 50px 高，现在是 200px 的区块（与全站其他空态一致）。系统任务面板那两个分组的空态也从约 68px 变成 200px，一屏里会有两块。这是"全站统一成一个高度"的必然结果；如果目视下来觉得偏大，需要单独讨论要不要给基元再加一档更小的尺寸，而不是把这批回退

### 验证
- `typecheck` ✅ ｜ `build` ✅ ｜ `vitest` ✅ 38 文件 / 189 用例 ｜ `i18n:sync` ✅（无 diff）｜ `git status` 恰为本次改动的 10 个源码文件
- 本批 13 处空态全部复用已有 i18n 键，**未新增、未改动任何 key**

### 遗留（待人工）
- **本批建议目视确认**：系统信息页（任务面板的空态与两个虚线框分组）、模型倍率编辑器（左右两栏的空态）、上游倍率同步表、订阅套餐卡、后台计费设置里支付方式／Creem 产品／金额选项／金额折扣四个编辑器、分组特殊可用编辑器
- **空态这一项已收完**，`ui-consistency.md` §七 的空态清单可以划掉；**V1 批次 2 到此全部完成**
- **待复查（仍未做）**：`v0.7.0` 的排查口径是"手写转圈"，`v0.8.0` 发现"只写一行 Loading 文案、不带转圈"的加载态会被这个口径漏掉。全仓库按新口径复查这件事还挂着
- 上一批遗留不变：全站明暗双主题目视回归（`Card` 有 20 个 import 点）
- 更早批次遗留不变：fr / ru / ja / vi 术语值层待母语审校；504 / 524 重试免责声明保留；调用记录详情的 Pre-consumed / Raw Quota 等字段留待 P5

## [v0.8.0] - 2026-09-12

### 说明
- **空态收尾上半：把弹窗里的手搓空态收进 `EmptyState`**。`v0.7.1` 全量盘点出 36 处手搓空态，本版收掉 15 处（11 个文件），另有 4 处判为不改，剩下 18 处属编辑器／面板／表格类，留作下一批
- **顺带抓到一处漏网的手写加载态**：渠道亲和性缓存弹窗里"加载中"是自己写的一行灰字 `Loading...`（没有转圈图标，所以 `v0.7.0` 按"手写转圈"排查时漏掉了）。本版修掉这一处，并把"只写一行 Loading 文案、不带转圈"这个口径记进下一批的复查项
- **共享组件内部也收了一处**：`dashboard` 的面板外壳（`panel-wrapper.tsx`）在 `empty` 时是自己画的一行居中灰字；改走基元后，所有用到它的仪表盘面板空态一起统一

### 变更
- **15 处改走基元**（14 处空态 + 1 处加载态），分布在 11 个文件：
  - `features/dashboard/components/ui/panel-wrapper.tsx`（共享壳，`empty` 分支）
  - `features/wallet/components/dialogs/billing-history-dialog.tsx`
  - `features/usage-logs/components/dialogs/user-info-dialog.tsx`、`audio-preview-dialog.tsx`
  - `features/models/components/dialogs/missing-models-dialog.tsx`、`upstream-conflict-dialog.tsx`（2 处）
  - `features/users/components/dialogs/user-binding-dialog.tsx`
  - `features/channels/components/dialogs/fetch-models-dialog.tsx`（2 处）、`ollama-models-dialog.tsx`（2 处）、`multi-key-manage-dialog.tsx`
  - `features/system-settings/general/channel-affinity/cache-stats-dialog.tsx`（空态 + 漏网的加载态）
- **档位统一按"容器"挑**：弹窗内与滚动列表内一律 `size='sm'`（200px），与同一批弹窗里的 `LoadingState size='sm'` 对齐
- **虚线框按原样保留**：`upstream-conflict-dialog` 的两处空态原本是虚线框，基元默认不画边框，改用 `className='rounded-md border border-dashed'` 显式保留，观感不变
- **面板外壳高度不变**：`panel-wrapper` 的 `height`（默认 `h-64`）仍由外层容器控制，基元在容器里撑满，面板整体高度与改造前一致

### 有意不改（已在 `ui-consistency.md` §十 登记，逐条写明理由）
- **`features/channels/components/dialogs/upstream-update-dialog.tsx`（2 处）**：两处空态都在 `h-[280px]` **固定高度**的滚动列表里（"待添加模型"／"待移除模型"）。换成 200px 的整块占位，会在 280px 的框里留下大片空白
- **`features/channels/components/dialogs/param-override-editor-dialog.tsx`（1 处）**：它是编辑器左侧窄栏列表里的"无匹配规则"提示，字号被特意压到 `text-xs`，属列表内提示而非区块空态
- **`features/channels/components/dialogs/advanced-custom-editor-dialog.tsx`（1 处）**：这个占位是"标题 + 描述 + 一行等宽字体的路径"三段式，基元目前只有标题与副标题两段，装不下第三行；要装下得把基元的 `description` 放宽成 `ReactNode`，属改基元，单独评估

### 验证
- `typecheck` ✅ ｜ `build` ✅ ｜ `vitest` ✅ 38 文件 / 189 用例 ｜ `i18n:sync` ✅（无 diff）｜ `git status` 恰为预期的 11 个源码文件
- 本批 13 处空态全部复用已有 i18n 键，**未新增、未改动任何 key**

### 遗留（待人工）
- **本批建议目视确认**：仪表盘各面板在无数据时（面板外壳这次改动面最大）；账单记录、用户信息、音频预览、缺失模型、上游冲突、用户绑定、抓取模型、Ollama 模型、多密钥这几个弹窗的空态；渠道亲和性缓存弹窗（顺带改了加载态）
- **下一批（空态收尾下半，清单已点全）**：15 个文件 / 18 处，编辑器／面板／表格类——`system-tasks-panel`（3 处）、`system-instances-panel`、`model-ratio-visual-editor`（2 处）、`group-special-usable-editor`、`upstream-ratio-sync-table`、`model-mapping-editor`、支付方式／Creem 产品／金额选项／金额折扣四个可视化编辑器、`subscription-plans-card`、`performance-overview`、`json-editor`、`model-group-selector`、`playground-empty-state`（需评估）
- **待复查**：`v0.7.0` 的排查口径是"手写转圈"，本批证明"只写一行 Loading 文案、不带转圈"的加载态会被漏掉。下一批顺带全仓库复查一次这个口径
- 上一批遗留不变：全站明暗双主题目视回归（`Card` 有 20 个 import 点）
- 更早批次遗留不变：fr / ru / ja / vi 术语值层待母语审校；504 / 524 重试免责声明保留；调用记录详情的 Pre-consumed / Raw Quota 等字段留待 P5

## [v0.7.1] - 2026-09-12

### 说明
- **上次登记的"三处收尾"里有一处是错误态、一处是写死的灰色，本版收掉；顺带修掉一个浅色主题下的真缺陷**
- **真缺陷（浅色主题下看不见日志）**：看日志的弹窗里，日志正文用的是写死的 `text-gray-200`（很浅的灰）。那个日志框的底色是跟随主题的（`bg-muted`）——深色主题下深底浅字没问题，**浅色主题下浅底浅字，日志基本读不出来**。同框里另外 3 条提示文字用的 `text-gray-400` 同样在浅色下偏灰
- **一处事实更正**：`v0.7.0` 的说明写了"V1 批次 2 至此清空"，不准确。本版做了一次全量盘点（只看不改），**控制台里还剩 24 处手搓空态、7 处手搓错误态**（其中 3 处在共享组件内部）。本版只收错误态，空态留作下一批

### 变更
- **看日志弹窗的写死灰色改走主题色**（`view-logs-dialog.tsx`）：日志正文 `text-gray-200` → `text-foreground`（浅色下从"几乎看不见"变成正常可读）；"没有容器 / 请选择容器 / 没有日志"3 条提示 `text-gray-400` → `text-muted-foreground`
- **详情加载失败 → `ErrorState`**（`view-details-dialog.tsx`）：此前是居中一行灰字，现在带警示图标，并补上此前缺失的**重试按钮**（复用页头那颗"刷新"按钮的同一个处理函数）
- **2FA 设置数据加载失败 → `ErrorState`**（`two-fa-setup-dialog.tsx`）：此前是居中一行灰字，现在与全站错误态同形
- `VERSION`：`v0.7.0` → `v0.7.1`

### 有意不改（已在 `ui-consistency.md` §七 登记，逐条写明理由）
- **`chat/$chatId.tsx` 的 4 处**（2 个"找不到预设"整页占位 + 2 个页面级宽 `Alert` 错误）：属整页级状态，现用 `h-12 w-12` 大图标与宽横幅；换成基元后图标缩到 24px、宽横幅变成居中块，是明显的观感降级，不适合顺手改。与空态批次一起定
- **`deployment-access-guard.tsx` 的"服务未启用"与"连接失败"两块**：它们是同一张整页引导屏的两个分支（64px 彩色图标盒 + 标题 + `Alert` + 全宽按钮）。只改一处会让两个兄弟屏互相不一致；要改就得连"服务未启用"（它不是错误态）一起改，属页面级对齐的活
- **`telegram-bind-dialog.tsx` 的错误块**：它和 Telegram 授权按钮容器同处一张 `rounded-lg border p-6` 卡片内，属"卡片内行内错误"（一行红字 + 重试按钮）；换成整块 200px 占位会把卡片撑开变形
- **`prefill-group-management-dialog.tsx` 的错误**：它是列表**上方**的行内 `Alert`，出错时下面的列表仍照常渲染；整块占位语义不对
- **`image-dialog.tsx` 的图片加载失败**：它是叠在图片上的绝对定位覆盖层，不是区块占位
- **`flow-charts.tsx`**：维持 `v0.6.0` 的登记（靠 `h-full` 在图表容器里撑满，改造要动基元的包装层）

### 验证
- `typecheck` ✅ ｜ `build` ✅ ｜ `vitest` ✅ 38 文件 / 189 用例 ｜ `i18n:sync` ✅（无 diff）｜ `git status` 恰为预期的 3 个改动
- 复查全仓库已无 `text-gray-*` 这类写死的灰色

### 遗留（待人工）
- **本批建议切浅色主题目视确认**：某次部署的「查看日志」弹窗里日志正文的可读性；「查看详情」弹窗在详情请求失败时的错误块；个人资料 → 设置 2FA 时数据没回来的错误块
- **下一批（空态收尾，清单已点全）**：控制台里 24 处手搓空态，分布在约 20 个文件——多数是 `py-8/10/12 text-center` 的一行提示，也有几个已是"图标 + 标题 + 副标题"的完整空态（`chat/$chatId`、`missing-models-dialog`、`billing-history-dialog`、`system-tasks-panel`、`system-instances-panel`）。**优先做 2 处在共享组件内部的**（`features/dashboard/components/ui/panel-wrapper.tsx` 的 `empty` 渲染、`components/ai-elements/conversation.tsx`），修一处惠及多个调用点
- **顺带发现的 i18n 缺陷**（不属视觉统一，另行处理）：`prefill-group-management-dialog.tsx` 的 `'Please retry or refresh the page.'` 与 `chat/$chatId.tsx` 的 `'Unable to generate chat link. Please check your API keys.'` 是写死的英文，没走 `t()`
- 上一批遗留不变：全站明暗双主题目视回归（`Card` 有 20 个 import 点）
- 更早批次遗留不变：fr / ru / ja / vi 术语值层待母语审校；504 / 524 重试免责声明保留；调用记录详情的 Pre-consumed / Raw Quota 等字段留待 P5

## [v0.7.0] - 2026-09-12

### 说明
- **14 处手搓的"整块加载态"全部收敛到 `LoadingState`，V1 批次 2 至此清空**（注：此说不准确——空态与错误态尚有遗留，见 `v0.7.1`）：改完之后全站**不再有任何手写的加载转圈**——不管页面级、弹窗内、还是表格占位，"加载中"长什么样由同一个组件决定
- 顺带清掉最后 2 个"用 CSS 画的圆圈"（`border-2 border-t-transparent` 那种假 spinner）
- 额外修好一处**主题不友好**：`view-logs-dialog` 的加载图标原本写死 `text-gray-400`，浅色主题下几乎看不见；改走基元后自动跟随主题（`text-muted-foreground`）

### 变更
- **页面级 2 处**（`chat2link.tsx` / `chat/$chatId.tsx`）：`size='lg'`。外面套一层 `flex h-full flex-col`，让基元内部的 `flex-1` 真正撑满高度，从而保持原来"整屏垂直居中"的效果（不套这层会退化成顶部一个 400px 方块）
- **弹窗内 11 处**（`user-binding-dialog` / `user-info-dialog` / `multi-key-manage-dialog` / `fetch-models-dialog` / `tag-batch-edit-dialog` / `missing-models-dialog` / `update-config-dialog` / `view-details-dialog` / `view-logs-dialog` / `extend-deployment-dialog` / `two-fa-setup-dialog`）：`size='sm'`（200px）。此前是 `py-8` / `py-10` / `py-12` 三种算出来的高度（约 88–128px），换成统一档位后弹窗内会略高一点、但再无跳动
- **表格占位 1 处**（`upstream-ratio-sync-table.tsx`）：默认 `md` + `className='rounded-md border'`，保留原来那个方框（同分支的"无差异"空态仍是 `h-64`，未动）
- **删掉最后 2 个 CSS 圆圈**：`two-fa-setup-dialog` 的 32px 粗环（块级）、`upstream-ratio-sync.tsx` 按钮内的 16px 细环（行内）→ 均换成 `Loader2`
- **`Loader2` 导入清理**：7 个文件（`chat2link` / `chat/$chatId` / `user-binding-dialog` / `user-info-dialog` / `upstream-ratio-sync-table` / `missing-models-dialog` / `multi-key-manage-dialog`）改完后不再直接使用 `Loader2`，导入一并删掉；其余 7 个文件里 `Loader2` 仍被按钮内的行内转圈使用，保留
- `VERSION`：`v0.6.0` → `v0.7.0`

### 有意不改（已在 `ui-consistency.md` §七 登记）
- **按钮内的行内转圈不动**（`<Loader2 className='mr-2 h-4 w-4 animate-spin' />`）：这是"按钮正在提交"的指示，属按钮语言，不是块级加载态
- **`deployment-access-guard.tsx` 的加载屏不动**：它不是"一个转圈"，而是"转圈 + 多步进度列表"（`LoadingStep` 子组件），结构上不属于本批的简单块级加载态
- **`two-fa-setup-dialog.tsx` 的 "Failed to load setup data" 分支不动**：那是错误提示文案不是加载态，归下一批（错误态收尾）

### 验证
- `typecheck` ✅ ｜ `build` ✅ ｜ `vitest` ✅ 38 文件 / 189 用例 ｜ `i18n:sync` ✅（无 diff）｜ `git status` 恰为预期的 15 个改动
- 复查全仓库已无 `h-8 w-8 animate-spin` / `h-6 w-6 animate-spin` / `size-6 animate-spin` / `animate-spin rounded-full` 四种手写写法
- 未涉及 `relaykit/`、计费、数据结构、鉴权、i18n 文案

### 遗留（待人工）
- **本批需目视确认**：聊天跳转页与 `chat/$chatId` 的整屏加载；13 个改动过的弹窗在"加载中 → 加载完"切换时的高度跳动（弹窗内统一为 200px，比原先略高）
- **顺带发现的下一批**：`deployment-access-guard` 的多步加载屏与 `two-fa-setup-dialog` 的手搓错误态；`view-logs-dialog` 里还有 4 处 `text-gray-400` / `text-gray-200` 的提示与日志文字在浅色下偏灰，应改 `text-muted-foreground`
- 上一批遗留不变：全站明暗双主题目视回归（`Card` 有 20 个 import 点）
- 更早批次遗留不变：fr / ru / ja / vi 术语值层待母语审校；504 / 524 重试免责声明保留；调用记录详情的 Pre-consumed / Raw Quota 等字段留待 P5

## [v0.6.0] - 2026-09-12

### 说明
- **V1 批次 2 第 5 项（基元层部分）落地**：空态／加载态／错误态三个基元对齐成一套几何，并删掉 3 处重复实现
- **调研结论推翻了 v0.5.0 里的估算**：所谓"58 个文件"是**引用面**，不是改动面。实测下来：
  - **表格壳早就统一了** —— `DataTablePage` 一个组件管住全部列表页，无需改动
  - **页头也早就统一了** —— `SectionPageLayout` 有 14 个控制台页面在用，`features/` 下**没有任何一页手写标题栏**（搜遍 `<h1` / `<h2` 只有公开页与弹窗内部标题）
  - **真正散装的只有空态／加载态／错误态**，而且毛病不是"基元不好"，是**基元没人用**：共享的 `EmptyState` 有 **0 个页面**在用、`LoadingState` 只有 1 个、`ErrorState` 只有 4 个；同时 13 处直接在拼 `Empty` 子组件、文件内还私藏了 2 个重名的 `EmptyState`
  - 这正是 `ui-consistency.md` §3.5 早就写下的判断："不是缺组件，是缺'唯一实现'的约束"
- **观感变化（需人工看）**：加载态从"裸转圈"变成"灰底圆角图标徽章 + 文案"；`login-sessions` 的错误图标变红；若干对话框的空态高度统一到 200px

### 变更
- **`EMPTY_SIZE` 三档成为全站唯一高度来源**（`web/src/components/ui/empty.tsx`）：`sm` 200px / `md` 300px（默认）/ `lg` 400px。此前散落着 `min-h-32`(128) / `min-h-48`(192) / `py-8`(≈214) / `py-10` / `py-12` / `min-h-[300px]` / `h-[400px]` 等各写各的
- **`empty-state.tsx` / `error-state.tsx` 新增 `size` prop**（默认 `md`）；并修掉一个潜在 bug：`FadeIn` 多包了一层普通 div，导致 `Empty` 上的 `flex-1` 一直**废着**（撑不开父容器），现在 `FadeIn` 那层带上 `flex` 布局
- **`loading-state.tsx` 重建为与上两个共用同一套几何**：此前它是自己一套裸 spinner 居中布局（`gap-3` + 裸图标 + `<p>`），与空态／错误态长得不一样；现在同样是"图标徽章 + 次级文案"。`inline` 变体保持不变（那是行内转圈，不是一类东西）
- **`table-empty.tsx`**：高度改用 `EMPTY_SIZE.lg`，`children` 收进 `EmptyContent`（此前直接挂在 `Empty` 下，间距与 `EmptyState` 不一致）
- **`notification-popover.tsx` 删掉文件内的第三个 `EmptyState` 定义**：它 named 与共享件重名，把共享的那个挡住了；加载分支改用 `LoadingState`
- **`login-sessions-card.tsx` 手搓的错误态 → `ErrorState`**：图标、标题、描述、重试按钮此前是逐行手写的，与 `ErrorState` 完全重合。改为基元后错误图标会带 `text-destructive`（红色）、重试按钮统一为 `outline` / `sm`
- **Hugeicons → Lucide**（`login-sessions-card.tsx` / `login-session-item.tsx`）：全站 265+ 个文件用 Lucide，业务页面里只有 3 个用 Hugeicons（另 21 个是 `components/ui/` 下 shadcn 基础件，属正常）。本次换掉这 2 个
- **`prefill-group-management-dialog.tsx`**：手搓加载态 → `LoadingState`；修掉空态结构错误（`EmptyMedia` 此前挂在 `EmptyHeader` **外面**，间距与别处不一致）；`border-dashed` → `border`
- **空态高度归一**（`access-token-dialog` / `missing-models-dialog` / `codex-usage-dialog`）：`py-8`(≈214px) / `min-h-32`(128px) / 不设(≈150px) → 统一 `EMPTY_SIZE.sm`
- `VERSION`：`v0.5.0` → `v0.6.0`

### 有意不改（已在 `ui-consistency.md` §七 登记）
- **模型广场（`features/pricing`）的空态不动**：`ui-consistency.md` §四 把它列为 **V3**，且它用的是公开页的"大图标 + 虚线框"风格，与控制台是两个体系，硬改会把 V3 撕开一半
- **`rankings` / `about` 等公开页空态同理**：属公开页风格，留给对应工作流
- **`card-grid.tsx` / `mobile-card-list.tsx` / `api-keys-table.tsx` / `redemptions-mobile-list.tsx` / `usage-logs-mobile-card.tsx` 这 5 处不动**：它们已经用的是同一套 `Empty*` 子组件 + 外层带框，本身就是齐的（只是 `border-none` 现在是废类，无害）
- **`flow-charts.tsx` 的空态不动**：它靠 `h-full` 在图表容器里撑满，改造要动基元的 `FadeIn` 包装层，风险大于收益
- **`data-table` 的 `emptyIcon` prop 未删**：实测全站**零调用点**，是死 prop；但删它属公开 API 变更，另行处理

### 下一批（清单已列全）
**14 处手搓的"整块加载态"** —— 尺寸 `h-8 w-8`(6 处) / `h-6 w-6`(4 处) / `size-6`(1 处) / 纯 CSS 圆圈(1 处) 混用，高度 `py-8` / `py-10` / `py-12` / `h-64` 也各不同：
`chat2link.tsx` · `chat/$chatId.tsx` · `user-binding-dialog.tsx` · `user-info-dialog.tsx` · `upstream-ratio-sync-table.tsx` · `tag-batch-edit-dialog.tsx` · `multi-key-manage-dialog.tsx` · `fetch-models-dialog.tsx` · `two-fa-setup-dialog.tsx` · `view-logs-dialog.tsx` · `view-details-dialog.tsx` · `update-config-dialog.tsx` · `missing-models-dialog.tsx` · `extend-deployment-dialog.tsx`
（14 个文件每个都要动 import，故未混在本次；本次已用 `prefill-group-management-dialog` 验证过改法）

### 验证
- `typecheck` ✅（首轮抓到 `EMPTY_SIZE` 重复导出，已修）｜ `build` ✅ ｜ `vitest` ✅ 38 文件 / 189 用例 ｜ `i18n:sync` ✅（无 diff）｜ `git status` 恰为预期的 12 个改动
- 未涉及 `relaykit/`、计费、数据结构、鉴权、i18n 文案

### 遗留（待人工）
- **本批需目视确认**：概览页通知弹层（`notification-popover`）的加载／空态、个人资料页「登录会话」卡片的错误态与空态、模型／渠道的若干对话框（预填充组、缺失模型、Codex 用量、访问令牌）的空态高度
- 上一批遗留不变：全站明暗双主题目视回归（`Card` 有 20 个 import 点）
- 更早批次遗留不变：fr / ru / ja / vi 术语值层待母语审校；504 / 524 重试免责声明保留；调用记录详情的 Pre-consumed / Raw Quota 等字段留待 P5

## [v0.5.0] - 2026-09-12

### 说明
- **V1 批次 2 收口（可数清的部分）**：`Card` 基元统一描边手段（`ring` → 真 `border`）、`Empty` 基元的假虚线框修复、`panel-wrapper` 与基元外框值对齐
- V1 批次 2 共 5 项，本版完成第 4 项。第 5 项（表格壳／空态／加载态／错误态／页头收敛）经调研发现工作量远大于表面（牵涉 58 个文件的引用面、且多为"每页都要动"），拆为独立批次另行开工
- **观感影响极小**：暗色下 `ring-foreground/10` 与 `border-border` **数值等价**（均为白色 10%），浅色下有极轻微变化（近黑 10% → `oklch(0.93)` 全值）

### 变更
- **`Card` 基元换成真边框**（`web/src/components/ui/card.tsx`）：`rounded-xl` + `ring-1 ring-foreground/10` → `rounded-2xl` + `border border-border`；`CardHeader` / `CardFooter` 与首尾图片的圆角同步 `xl → 2xl`。改用真 `border` 后，调用点写的 `border-dashed` / `border-border/60` 之类"只改样式或颜色、不改宽度"的类**首次真正生效**（此前因基元无边框宽度而全部空转）
- **`Empty` 基元删掉假虚线框**（`web/src/components/ui/empty.tsx`）：基元只写 `border-dashed` 却没写 `border`，边框宽度为 0，该虚线**从未在任何页面显示过**；同时删掉 `components/empty-state.tsx` 的 `bordered` prop —— 它全仓库零调用点，存在的唯一目的就是绕过这个 bug
- **`panel-wrapper` 外框显式化**（`features/dashboard/components/ui/panel-wrapper.tsx`）：裸 `border` → `border border-border`。它的 `rounded-2xl + border + bg-card + shadow-card` 本来就是目标形态，`card.tsx` 本次只是向它看齐
- **首页步骤卡同步**（`features/home/components/sections/how-it-works.tsx`）：手写复刻的 `bg-card ring-1 ring-foreground/10 rounded-2xl shadow-card` → `bg-card border border-border rounded-2xl shadow-card`
- `VERSION`：`v0.4.4` → `v0.5.0`

### 有意不改（已在 `ui-consistency.md` §七 登记）
- **`titled-card` 与 `panel-wrapper` 都保留**：前者是无状态且带图标的用户设置卡，后者自带 `loading` / `empty` / `height`，服务对象不同。把后者改造为 `Card` 基元会牵动 dashboard 4 个面板的字号与间距，收益仅是少写一个 div
- **浮层的 `ring-1 ring-foreground/10` 不动**（`select` / `popover` / `menu` / `navigation-menu` / `sidebar`）：那是浮层投影语言，不属于卡片描边
- `legal-document.tsx:75` 的 `<Card className='border-dashed'>` **保留**：基元补上边框宽度后它首次真正生效，文档加载失败态将显示为虚线框（业务方 2026-09-12 拍板保留）
- 已核验 `faceted-filter.tsx`（`Button` 基类自带 `border`）与 `preset-selector.tsx`（`SettingsControlGroup` 自带 `border`）两处虚线**不是**空转，无需改动

### 验证
- `typecheck` ✅ ｜ `build` ✅ ｜ `vitest` ✅ 38 文件 / 189 用例 ｜ `i18n:sync` ✅（无 diff）｜ `git status` 恰为预期的 5 个改动
- 未涉及 `relaykit/`、计费、数据结构、鉴权、i18n 文案

### 遗留（待人工）
- **全站明暗双主题目视回归未做**：`Card` 有 20 个 import 点，本批改变了全站卡片圆角（`xl → 2xl`）与描边实现。建议重点看：概览页统计卡、账单／充值卡、安全设置卡、模型广场卡片
- V1 批次 2 第 5 项（表格壳／页头收敛）拆为下一批
- 更早批次遗留不变：fr / ru / ja / vi 术语值层待母语审校；504 / 524 重试免责声明保留；调用记录详情的 Pre-consumed / Raw Quota 等字段留待 P5

## [v0.4.4] - 2026-09-12

### 说明
- **V5 人工验收第一轮修复**：首页 Hero 的滚动 pin 只在桌面启用，窄屏（`< 1024px`）退化为可滚动的普通流式区块
- 375 / 768 两档实测到的"标题与按钮消失""内容顶到导航栏下"是同一根因的三个表现，本版一并修掉

### 修复
- **窄屏首屏内容被裁切**（`features/home/components/sections/hero.tsx`）：pin 容器是 `h-svh + overflow-hidden` 且内容居中，窄屏双列塌成单列后内容约 1000px 高、手机视口仅约 670px，居中溢出被上下同时裁掉 —— 375 档只剩「AI 智能体」标签可见，标题与按钮整组消失。修法：新增 `isPinned = useMediaQuery('(min-width: 1024px)') && !reduceMotion`；非 pin 时 section 不挂 `hero-scroll-stage`，内层由 `sticky h-svh overflow-hidden` 退回 `relative`，内容层改用 `min-h-svh py-28` 自然撑开，滚动变换（缩放／位移／淡出／模糊）一并置空
- **首屏与悬浮导航栏重叠**（同上）：`PublicLayout` 在 `container={false}` 下不预留 header 高度（`headerOffset` 默认 `!ownsLayout`），Hero 自身也没补，宽屏只是靠居中"碰巧"躲开。修法：非 pin 分支补 `py-28` 明确避让
- **矮视口的同类裁切**（同上）：`items-center` 改为内容层 `my-auto` —— auto margin 在内容超出容器时会归零，标题从顶部开始显示，不再被上下两端同时切掉

### 变更
- **CSS 舞台断点同步**（`web/src/styles/index.css`）：`.hero-scroll-stage` 的 travel 由「`< 768px` 60svh ／ `≥ 768px` 100svh」改为单一 `100svh + 100svh`（该类现在只可能在 `≥ 1024px` 出现）；`.hero-content-overlap` 在 `max-width: 1023px` 与 `prefers-reduced-motion` 下归零。断点与 `hero.tsx` 的媒体查询必须同步，已在两处注释互指
- `VERSION`：`v0.4.3` → `v0.4.4`

### 验证
- `typecheck` ✅ ｜ `build` ✅ ｜ `vitest` ✅ 38 文件 / 189 用例
- 未涉及 `relaykit/`、计费、数据结构、鉴权、i18n 文案

### 遗留（待人工复验）
- 375 / 768 两档需复验；桌面（`≥ 1024px`）pin 动画、明暗双主题逐页回归与 V1 批次 2 剩余两项仍待 V5 收口

## [v0.4.3] - 2026-09-12

### 说明
- **V1 批次 2 的零观感部分落地**：把品牌渐变与页头光晕从"逐处复制的字面串"收敛为两个 `@utility`，并删掉一个零引用孤儿件
- 本批**不改变任何观感**，故只递增 PATCH：`brand-gradient-text` 用 `@apply` 复刻原类串、产物 CSS 逐属性一致，`brand-glow-page` 的三层色值原样搬入。V1 批次 2 共 5 项，本版完成 3 项；剩余两项（`card.tsx` 基元统一、表格壳／空态收敛）会改变全站观感，待 V5 人工验收后单独升 MINOR
- 提交 `a9a9e6cb` ／ 标签 `v0.4.3`（10 文件，+46 / −97）

### 变更
- **品牌渐变收敛为 `.brand-gradient-text`**（`web/src/styles/index.css`）：替换 **5 处**逐字重复的 `bg-gradient-to-r from-blue-400 via-violet-400 to-purple-500 bg-clip-text text-transparent` —— `hero.tsx` / `cta.tsx` / `promotions.tsx` / `how-it-works.tsx` 与 `about/index.tsx` 的 `GRADIENT_TEXT` 常量。此后换主题预设只需改一处
- **页头光晕收敛为 `.brand-glow-page`**（同上）：`rankings/index.tsx` 与 `pricing/index.tsx` 中**逐字符相同的 30 行 inline style**（3 层径向渐变 + mask 40% 淡出）改为一个类，两处各减 11 行
- `VERSION`：`v0.4.2` → `v0.4.3`

### 移除
- `web/src/components/layout/components/glow.tsx`：未从 `layout/index.ts` 导出、全仓库零引用，其 `amber-500` / `yellow-400` 是与基线 A 蓝紫主色冲突的历史残留

### 有意不改（已在 `ui-consistency.md` §七 登记）
- **`cta.tsx` 的 2 层 mesh 光晕保留 inline**：色值（`oklch(0.7 0.15 250)` / `oklch(0.65 0.12 200)`）与 `brand-glow-page` 不同族，且**仅 1 个调用者**，按 `AGENTS.md`「避免只被单一调用者使用的辅助函数」不抽象
- **不使用 `--chart-1/2/3`**：`rankings` / `pricing` 光晕的三层色值与 `--chart-*` 数值完全相同，但 `--chart-*` 在 `dark` 下被重新定义，而原 inline style 在暗色下只改 opacity、不改颜色，直接替换会改变暗色观感
- `config-drawer.tsx` 的彩虹渐变是**主题预设色板预览**（需显示 5 种不同色）；`summary-cards.tsx` 的 `color-mix` 已使用 `--overview-accent-*`，属控制台体系（V2 范围）

### 验证
- `typecheck` ✅ ｜ `build` ✅ ｜ `vitest` ✅ 38 文件 ｜ `i18n:sync` ✅（无 diff）｜ `git status` 恰为预期的 9 处改动
- 另核验产物 CSS 中两个新 `@utility` **均已生成**（`@utility` 按需输出，不生成会静默丢样式）
- 未涉及 `relaykit/`、计费、数据结构、鉴权

### 遗留（待人工）
- **明暗双主题逐页目视回归未完成**：本批为零观感改动，仍建议随 V5 验收一并扫一眼首页渐变文字
- V1 批次 2 剩余两项：`components/ui/card.tsx` 基元统一（`rounded-xl + ring-1` → `rounded-2xl + border border-border/50 bg-card`）、表格壳／空态／加载态／错误态／页头收敛到基元

## [v0.4.2] - 2026-09-12

### 说明
- 修复浅色主题下 Hero 与悬浮导航栏的文字不可见；并调整首页视觉（三步配图换 jpg、终端卡半透明与延迟浮现）
- 本版本是 v0.4.0 那批视觉改动的**浅色主题续修**：全站阴影、圆角减半、背景离白三处都改变了浅色下的明度关系，而浅色回归未在 v0.4.0 内完成

### 修复
- **浅色主题下 Hero 文字不可见**（`features/home/components/sections/hero.tsx`）：Hero 是深空底图，用 `dark` 类把深色 token 限定在本 section 内。但 `dark` 只切换 CSS 变量、**不重新锚定 `color`** —— 裸文本节点（标题首行、智能体卡名称、outline CTA 的 label）的颜色是从 `body` 继承的，浅色主题下 `body` 已解析为近黑，于是在黑色底图上集体消失。修法：`dark` 旁边补 `text-foreground`，让继承节点改读本 section 的深色 `--foreground`
- **同一根因出现在悬浮导航栏**（`components/layout/components/public-header.tsx`）：`overDarkSurface && 'dark'` → `'dark text-foreground'`。受影响的是 logo 旁的站名（裸文本节点）与 `ghost` 图标按钮（图标走 `currentColor`）

### 变更
- **终端卡改半透明**（`hero-terminal-demo.tsx`）：`bg-white/95` → `/85`，深色 `#0b0f17/95` → `/80`。它压在星云底图上，近实心会遮住首屏构图；`shadow-card-lift` 不变
- **终端卡延迟浮现**：`hero.tsx` 右栏由 `landing-animate-fade-up` + `320ms` 改为 `landing-animate-cinematic-emerge` + `2000ms`；新增该动画（`styles/index.css`）——1.8s、`cubic-bezier(0.32, 0.72, 0.24, 1)`，自 `translate3d(0, 24px, 0) scale(0.985)` 起，并已并入 `prefers-reduced-motion` 禁用列表
- **三步配图换图**（`sections/how-it-works.tsx`）：`/home/step-{1,2,3}-*.webp` → `/home/3-step/{1-config-picture,2-onnection,3-monitor-picture}.jpg`；原 3 张 webp 随之删除
- `VERSION`：`v0.4.1` → `v0.4.2`

### 遗留（待清理）
- `web/public/home/3-step/2-onnection.jpg` 文件名拼写有误（应为 `connection`）；代码与文件一致，不影响运行
- `web/public/home/hero-bg.png` 为 **6.47 MB** PNG，是首屏唯一大图，未压缩亦未转 webp

### 验证
- `typecheck` ✅ ｜ `build` ✅ ｜ `vitest` ✅ 38 文件 / 189 用例 ｜ `i18n:sync` ✅（2026-09-12 收口复跑）
- 未涉及 `relaykit/`、计费、数据结构、鉴权

## [v0.4.1] - 2026-09-12

### 说明
- v0.4.0 推送后立即修复 `VERSION` 换行策略问题，避免 Windows 开发者在 `core.autocrlf=true` 环境下把版本号文件提交为 CRLF，导致 `makefile` 读取注入的前端版本号带 `\r`

### 变更
- `.gitattributes`：新增 `VERSION text eol=lf`，强制版本号文件始终 LF
- `VERSION`：`v0.4.0` → `v0.4.1`

### 验证
- 执行 `git add --renormalize VERSION` 后重新 checkout，确认 `VERSION` 字节为 `76 30 2e 34 2e 31 0a`（LF，无 CR）✅

## [v0.4.0] - 2026-09-12

### 说明
- **P0 第三批「视觉统一」落地**：以公开首页本体为基准 A，把卡片基元、阴影、圆角、图标容器收敛到一套语言（V1 批次 1 + V5），同时完成关于页公司资料（V4）、Hero 滚动舞台与 ICP 备案号配置化
- 本段覆盖 v0.3.0 之后的全部工作区改动；批次明细见 `.docs/frontend/ui-consistency.md` §七（V1）/ §八（V5）
- 本版本把此前散落在工作区、跨多次会话的成果一次性固化为可回退基线 —— 此前 Hero 改动与 V5 成果叠在同一批未提交改动里，导致无法按任务粒度回滚

### 新增
- **ICP 备案号配置化**（完整链路）：`common/constants.go` 新增 `var Icp = ""`（默认空，刻意不硬编码号码）；`model/option.go` 注册为可配置项并经 `controller/misc.go` 的 `GetStatus()` 下发；前端补齐 `use-system-config` / `system-config-store` / `SystemStatus` 类型链路，`footer.tsx` 新增 `IcpNotice`（空值不渲染，非空时链接 beian.miit.gov.cn）；后台入口在「业务管理 → 站点 → 系统信息 → ICP 备案号」，保存后刷新 status
- **首页促销区块（V5 / S3）**：`sections/features.tsx` 整段替换为 `sections/promotions.tsx` —— 6 张热卖模型卡分「全球旗舰 / 国产旗舰」两组，含贴角折扣徽章、厂商标识、三档牌价与牌价脚注；新增 `promo-model-card.tsx` / `model-logos.tsx`
- **首页智能体卡片（V5 / S1）**：Hero 原「支持应用」三个胶囊改为 6 张精选智能体卡片 + 1 张虚线「查看更多」，弹窗按 3 列列出全部 15 个；新增 `hero-agents.tsx` / `agent-link-card.tsx` / `agent-logos.tsx` / `agents-more-dialog.tsx`
- **Hero 滚动舞台**：首屏大图改为滚动 pin 设计（新增 `.hero-scroll-stage` / `.hero-content-overlap`），内容随滚动缩放、上移、淡出与模糊；`public-header.tsx` 新增 `overDarkHero`，以 `IntersectionObserver` 跟随 Hero 实际几何，使悬浮导航在深色大图期间保持深色 token
- **全站卡片阴影 token**：`theme.css` 新增 `--elevation-card` / `--elevation-card-lift`（明暗两套取值），在 `@theme inline` 映射为 `shadow-card` / `hover:shadow-card-lift`；接入 `card.tsx`、`panel-wrapper.tsx`、首页各卡、关于页与控制台 6 处卡片壳
- **首页回归测试**：新增 `home/components/__tests__/` 下 4 个测试文件（`promotions` / `hero-agents` / `layout` / `how-it-works`）

### 变更
- **关于页＝微观互联公司页（V4）**：后台未配置 About 内容时渲染 CompanyProfile（概述 / 战略转型 / 公司简介 / 国家级合作与资质 / 核心产品 / 发展历程 / 愿景 / 声明）；上游署名与许可声明（New API / QuantumNous / One API / AGPL）从关于页底部收敛到全局 Footer 并压成一行小字，Footer 品牌区（logo + 站名 + tagline + copyright + 备案号）同步强化
- **首页文案与节奏（V5）**：徽章改「Token 工厂时代来临」，标题改「一个 API 网关 / 轻松享用海量 AI 模型」，删除中段说明与 S5 副文案；`SECTION_PY` 由 `py-16 md:py-20 lg:py-24` 二次下调至 `py-12 md:py-14 lg:py-16`，S3/S4/S5 接缝降至 96 / 112 / 128px；S2 统计口径改为 33+ / 120+ / 20+ / 100+
- **全站圆角减半**：`--radius` 由 `1rem` 改为 `0.5rem`（组件零改动，全局派生）；按钮圆角 16px → 8px、`rounded-2xl` 卡片 28.8px → 14.4px
- **图标容器去描边**：全站 8 处内联「图标外套描边方框」去掉边框；其中引导步骤容器底色 `bg-background` → `bg-muted`（需遮住背后的连接竖线），完成态语义改由 `bg-success/10` 承载
- **按钮尺寸统一**：`button.tsx` 新增 `xl`（`h-11 gap-2 px-5`），`hero.tsx` 5 处就地尺寸覆盖随之移除（V1 批次 1）
- **浅色主题背景离白**：`:root` 的 `--background` 由纯白改为 `oklch(0.98)`，让浅色卡片靠明度分层而非仅靠柔光；骨架屏 base/highlight 同步降一档

### 移除
- 首页 5 个孤儿件：`gateway-card` / `feature-item` / `stat-item` / `icon-card` / `scrolling-icons`
- `home/constants.ts` 的 `DEFAULT_FEATURES` / `getDefaultFeatures`；3 个首页孤儿 i18n 键（`How It Works`、`Three steps to get started`、S5 副文案）

### 文档
- `ui-consistency.md`：新增 §七「V1 执行批次」、§八「首页改造（V5）细则」§8.1–§8.12
- `VERSION` 升 `v0.4.0`

### 验证
- `go build ./...` ✅ ｜ `bun run typecheck` ✅ ｜ `bun run build` ✅ ｜ `bunx vitest run` ✅ 38 文件 / 189 用例
- 未涉及 `relaykit/`，未执行该模块独立构建；计费、数据结构、鉴权未改动

### 遗留（待人工）
- **明暗双主题逐页目视回归未完成**：首页首屏（CTA 与智能体是否露出）、控制台卡片静止即带阴影的观感、圆角减半后的整体气质、促销卡暖色徽章与基线 A 蓝紫的搭配
- 深色主题下智能体卡片品牌色的对比度待确认（改 `agent-logos.tsx` 一处即可）

## [v0.3.0] - 2026-09-11

### 说明
- **P0「门面与表述层」收口**：表述层收口（模型广场 / 建 Key / 调用记录 / 个人设置）+ 客户端门面落定（导航 3 组 + 「我的」四页）+ 术语统一与教学段落清理 + §5.6 三项骨架债；清单与验收口径见 `.docs/master-plan.md`
- 本段覆盖 v0.2.0 之后的全部工作区改动（含两次会话：门面收口 + 术语/债收尾）
- 术语统一范围说明：值层替换本轮落在 **zh / zh-TW**（客户可见面 67–68 条）；fr / ru / ja / vi 保持原文，原因是这些语言的被替换词需按性数格重写，机器替换会产出语法错误，留待母语审校（键与键序不变，`i18n:sync` 通过）

### 新增
- **客户端「我的」门面落定（§1.5）**：新增 `/plans` 套餐页（`subscription-plans-card` 由钱包页拆出）、`/earnings` 收益页（推广佣金卡 + 邀请链接 + 邀请数 + 转出到余额）、`/billing` 账单占位页（经销商专享，待 P3 开放）；客户端侧栏定型为 3 组（常规 6 / 我的 4 / 管理 2）
- **侧栏开关补项**：业务管理 → 站点 → 侧边栏模块新增 `plans` / `earnings` / `billing` 三项默认配置与面板元数据

### 变更
- **模型广场表述层收口**（承接 v0.2.0 的「去分组」）：移除「标准 / 充值」切换、「/1K」单位切换（统一按每 1M tokens）、「Pricing Type」筛选；详情抽屉移除分组定价区块，统一展示官方价格；性能页「Per-group performance」表改为聚合展示，API 页限额说明去掉分组口径
- **未选中分组不再回落到档位价**：`getDisplayGroupRatio`（未选中分组时返回该模型最低倍率）已移除——未登录访客此前看到的不是官方价，而是接近批发的档位价
- **建 Key 去分组**：创建 / 编辑令牌抽屉移除「选择分组」与自动分组顺序编辑器；令牌列表移除「分组」列（含倍率徽章）；调用记录令牌单元格的分组名与倍率保持仅管理员可见
- **调用记录合并**：侧栏「通用 / 绘画 / 任务」三项收敛为一项并在页内切换；`/usage-logs/$section` 路由与逐类授权不变，未授权分类不出现在切换栏
- **导航白名单化**：未登记在 `URL_TO_CONFIG_MAP` 的路由默认不出现在导航（此前默认可见）
- **钱包页瘦身**：只剩余额、充值、兑换码、支付方式与账单历史
- **i18n**：新增 `Earnings`、`Dealer Billing`、`Dealer Billing Placeholder Description` 与去掉「Limits apply per token group.」的限额说明；`Personal` 改为「我的」、`Profile` 改为「资料」（zh / zh-TW）；7 套语言同步
- **术语统一（客户可见面）**：按 `06` §4.6 替换客户能看到的用词——「额度 / 配额」→「余额」、「费用 / 消耗」→「花费」、「令牌」→「API Key」、「使用日志 / 日志详情」→「调用记录」、「邀请额度 / 奖励」→「推广佣金」（zh 67 条 / zh-TW 68 条）；新增术语键前先进 §4.6 再落 i18n 的约定不变
- **「渠道」对外文案改「上游供应商」**（§5.6 债务 5）：`Channel` / `Channels` / `Channel ID` / `Channel Affinity` 及启用说明共 5 条；系统设置内更深一层的渠道文案随 P1 分层处理
- **`docs_link` 搬迁**（§5.6 债务 3）：从 `business-settings/billing/sections/quota-settings-section.tsx`（紧邻充值链接，看起来像充值项）迁出，新增独立区块 `business-settings/site/sections/docs-link-section.tsx` 并注册在站点工作区「顶栏导航」之后；沿用 `general_setting` 选项机制（按 key 提交），不与 `HeaderNavModules` 的整包 JSON 提交混用；`BillingSettings` 去掉该字段、`SiteSettings` 新增该字段
- **`.gitignore` 补 i18n 产物**：新增 `web/src/i18n/locales/_extras/`、`_reports/*.untranslated.json`（§5.6 债务 4）
- **i18n 同步修正**：`en.json` 中 `Supports one-click configuration and perfectly adapts to Microsslink multi-protocol configuration.` 被 `i18n:sync` 用中文值回填，已改回英文源串

### 移除
- **计费机制「教学段落」**（`06` §4.6）：3 个孤儿键（分组两种角色 / 令牌分组计费规则 / 三档分组示例）+ 在用的「常见误区：用户分组基础倍率不是个人折扣」（`system-settings/models/group-ratio-form.tsx` 的 Usage guide）
- **未删除项（有意保留，待定）**：504/524 重试免责声明属破坏性操作的安全披露（管理端、非客户侧计费教学），未随本项删除
- 个人设置中的「侧边栏个人设置」卡（`features/profile/components/sidebar-modules-card.tsx` 及 `permissions.sidebar_settings` 判定）：客户不再自配系统模块，开关统一收敛到管理端
- 随之孤立的分组展示件与其测试：`api-key-group-cell`、`api-key-group-combobox`、`auto-group-order-editor`、`auto-group-visuals`（含 3 个测试文件）、`model-details-uptime-sparkline`
- 钱包页迁出并删除的原文件：`subscription-plans-card`、`affiliate-rewards-card`、`transfer-dialog`、`use-affiliate`、`lib/affiliate.ts`（迁至 `features/plans`、`features/earnings`）

### 文档
- `master-plan.md`：§5.1 第 9–12 项、第 7 项标记完成并补执行备注；§5.4 / §5.5 改按「P0 已预置」表述；§5.6 四项债全部消账；§2.4 债务 2 标记已消除、债务 3 路径修正；§八 P0 / P3 验收口径更新；§十 增本次变更行
- `VERSION` 升 `v0.3.0`（P0 收口里程碑）

### 验证
- `go build ./...` ✅ ｜ `cd relaykit && GOWORK=off go build ./...` ✅ ｜ `bun run typecheck` ✅ ｜ `bun run build` ✅ ｜ `bunx vitest run` ✅ 37 文件 / 196 用例 ｜ `bun run i18n:sync` ✅

## [v0.2.0] - 2026-09-11

> **基准版本**：后续 P0–P5 改造以本版本为基线（标签 `v0.2.0` ／ 提交 `5490a6b9`）。
> 总体计划见 `master-plan.md`。

### 新增
- **布局骨架统一**（决策 D15）：新增 `Container` 组件作为全站唯一宽度声明点；`PublicLayout` / `Main` 由布尔开关改为档位（`narrow` / `default` / `wide` / `full`）；后台统一为 1536 居中画布，页面不再自行声明宽度
- **导航工作区重构**：新增业务管理工作区（`/business-settings/*`，由原 `/system-settings/{billing,content,site}` 迁移而来）；新增「准入与限制」页；`SidebarView` 增加 `requiredRole` 视图级角色门槛
- **渠道与模型归属修正**：从系统管理移入业务管理（两者路由守卫均为 `ADMIN`，原归属导致普通管理员在侧边栏无任何入口）
- **用户级导航授权**：后端新增 `model.UpdateUserSidebarModules()`（加锁读改写，避免整快照覆盖）并在管理端创建/编辑用户时下发 `sidebar_modules`；前端新增用户授权矩阵与个人可见性设置
- **业务管理代码目录对齐**：`features/system-settings/` 下的业务管理功能迁入 `features/business-settings/`（billing / site / policies / content）；共享的 settings 框架、`models/ratio-settings-card`、`integrations/utils` 保留原位并改为绝对路径导入
- **P0 起步 · 模型广场去分组**：移除侧栏「Groups」「Pricing Type」筛选、表格「Groups」列、卡片主分组标签与倍率徽章；价格口径未变（无分组选中时仍走原倍率解析）

### 文档
- 建立工作文档中心 `.docs/`：新增文档索引（`README.md`）、版本管理制度（`governance/version-policy.md`）、本变更记录
- 新增任务 01「改造准备工作」文档目录
- 新增任务 02「业务目标与方案设计」文档目录（8 份设计文档、决策 D1–D15、分期 P0–P5）
- 新增任务 03「前端骨架与导航重构」文档目录
- 新增前端布局规范 `frontend/layout-system.md`
- **新增总体改造计划 `master-plan.md`**：汇总改造目标、基准版本、前后端任务清单与协作纪律

### 验证
- `go build ./...` ✅ ｜ `bun run typecheck` ✅ ｜ `bun run build` ✅ ｜ `bunx vitest run` ✅ 17 用例 ｜ `bun run i18n:sync` ✅

## [v0.1.0] - 2026-09-11

### 新增
- **品牌改造**：前端运行时品牌由 NewAPI 替换为 **Microsslink | 微观互联**
  - 前端：`web/index.html` 标题与 meta、`web/src/lib/constants.ts` 的 `DEFAULT_SYSTEM_NAME`、页脚、关于页、首页 hero、站点设置默认值、`web/src/assets/logo.tsx`
  - 后端：`common/constants.go` 的 `SystemName` 改为 `Microsslink`；`setting/operation_setting/general_setting.go` 的 `DocsLink` 默认值清空
  - i18n：`web/src/i18n/locales/` 下 en / zh / zh-TW / fr / ru / ja / vi 七个语言文件
- **品牌资产**：`web/public/logo.png`（180×180）、`web/public/favicon.ico`（16/32/48）

### 说明
- 基线来自上游 new-api 提交 `2d8e50bf`
- 保留全部上游保护信息（项目名、组织名、版权声明、Go module path）
- 详细改动清单见 `.docs/rebrand/summary.md`

### 环境
- 建立本地开发配置：根目录 `.env`（`DEBUG=true`、`MEMORY_CACHE_ENABLED=true`）、`web/.env`（`VITE_REACT_APP_SERVER_URL=http://localhost:3001`）
- 开发端口约定：后端 `3001`、前端 `5173`，详见 `.docs/env/setup.md`
