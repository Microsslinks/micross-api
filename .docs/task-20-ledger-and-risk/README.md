# task-20 · ledger 全类型写账 + P4 风控核心（8 子项）

> 立项：2026-09-18　｜　状态：🟢 **§20.1-§20.8 已完成**（2026-09-18 同日落地）　｜　依赖：v0.37.0（task-17 Plan A 完成）
> 起手基线：v0.37.0（main HEAD `95744b46`）
> 预计发版：v0.37.1（一个版本含 8 子任务）
> 实际发版：v0.37.1（main HEAD `008c9a40`）
> 上游依赖：task-10 P4 邀请佣金核心（`account_ledger` 表 + `commission_records` 表基础）/ task-17 §17.3 account_ledger 统一账本 + §17.4 admin commission audit log

---

## 一、背景

task-17 Plan A 落地 v0.37.0 时，§17.3 account_ledger 统一账本只接入了 **commission** 一个事件类型（commission 写账）；§17.4 admin commission audit log 只做 admin 操作审计。v0.37.0 文档"仍未兑现"段列的两组事实缺口（业务方案 05 §2.7「对账四问」+ 业务方案 06 §6「P4 风控」）必须在本任务收口：

| # | 缺口 | 影响 | 来源 |
|---|---|---|---|
| 20.1 | **topup 充值完成未写 ledger** | 充值金额变动无账本；财务对账无从下手 | 业务方案 05 §2.7 |
| 20.2 | **refund 退款未写 ledger** | 退款金额变动无账本；账实不符 | 业务方案 05 §2.7 |
| 20.3 | **agent_quota_grant 经销商发额度未写 ledger** | 经销商给客户发额度无账本；下游争议无证据 | 业务方案 05 §2.7 |
| 20.4 | **注册阶段自邀拦截全 0** | 脚本注册→立刻互邀拿返佣，薅羊毛 | 业务方案 06 §6.3 第 1 件 |
| 20.5 | **commission 实时风控叠加全 0**（ring + first-topup）| 邀请人没消费也能拿返佣；invite-ring 闭环薅 | 业务方案 06 §6.3 第 2+3 件 |
| 20.6 | **commission_reverse 风控冲销未做** | 历史薅羊毛数据无清理路径 | 业务方案 05 §2.7 / 06 §6 |
| 20.7 | **admin HTTP + 自动 cron 未挂** | 风控函数有但没人调用，等于没做 | 工程闭环 |
| 20.8 | **admin UI 未做** | admin 没界面调 §20.7 的 HTTP 端点 | 用户体验 |

**任务承诺**：把这 8 件全部做完后，业务方案 05 §2.7「对账四问」可逐条回复，06 §6 「P4 风控」4 件核心（自邀拦截 + 实时叠加 + 风控冲销 + 自动化）全部兑现。

---

## 二、范围

### ✅ 做（8 子任务，按依赖顺序）

#### **20.1 topup 写 ledger**（30 分钟）

**目标**：`service/topup.go:CompleteTopUp` 事务内 `RecordAccountLedger(event=topup, amount=+quota, balance_after=...)`。

**关键**：balance_after 用与 commission 同款的"行锁 SELECT"算法（task-17 §17.3 沉淀）——避免被并发 UPDATE 抹掉。

#### **20.2 refund 写 ledger**（30 分钟）

**目标**：`service/refund.go:Refund` 写 `event=refund, amount=-quota` 负向 ledger。

**特殊**：refund 与 commission 复合事务（refund 一笔消费会触发 commission 负向 amount 写 commission ledger）；refund ledger 行独立写一条，与 commission ledger 不混。

#### **20.3 agent_quota_grant 写 ledger**（30 分钟）

**目标**：`service/agent_quota_grant.go:IssueQuotaToCustomer` 写 `event=agent_quota_grant`。

**特殊**：带 operator_id 字段区分"admin 操作"vs"经销商 system 自动"——后者 operator_id=0。

#### **20.4 注册阶段自邀拦截**（半天）

**目标**：`model/user.go:ValidateInviterForRegistration(inviterId, email, now)` 三重门槛：

- **年龄**：inviter 的 `created_at` ≥ 24h（防批量脚本注册→立刻互邀）
- **邮箱域名**：inviter 与 invitee 邮箱 @ 后内容相同则拒（防主控账号与新账号共用 @example.com 伪装）
- **邀请频次**：inviter 24h 内邀请 ≤ 5 人（防主控账号每天拉 100 个新号）

每个门槛一个 sentinel error（`ErrInviterTooNew` / `ErrInviterSameEmailDomain` / `ErrInviterTooActive`），controller 端 `errors.Is` 翻译为 i18n message。

**重要**：`controller/auth.go:Register` 调用校验；`admin.ResetInviter` 路径不动——admin 信任不受风控约束。

#### **20.5 实时风控叠加 ring + first-topup**（半天）

**目标**：`service/commission.go:ProcessCommission` 在 AssertNoLoss 前叠加两道拦截：

- `model/user.go:DetectInviteRing(inviterId)` DFS 沿 inviter_id 链上溯，5 跳上限检测环（A→B→A 或更长）；命中时 finalAmount=0 + breach=true
- `model/user.go:GetUserTotalConsumeQuota(inviterId)` SUM `logs.type=LogTypeConsume`；小于 5000 quota 时 finalAmount=0 + breach=true

两规则任一命中写 audit log；amount=0 的 breach 改写"margin_unwired"文案让运维识别。

#### **20.6 风控冲销（ReverseCommission）**（半天）

**目标**：`service/commission.go:ReverseCommission(ctx, recordID, reason, operatorID)`：

- `commission_records` 新增 4 字段：`Reversed bool indexed` / `ReversedAt` / `ReversedBy` / `ReverseReason`
- 事务内：行锁 commission_record + `UPDATE users.aff_commission_balance -= amount` + 标 Reversed + 写 `event=commission_reverse, amount=-record.Amount` ledger
- `ErrCommissionAlreadyReversed` sentinel 让二次撤销返 HTTP 409
- `operatorID=0` 标识"系统自动"（与 admin 人工操作区分）

#### **20.7 admin HTTP + 风控扫描 cron**（1 天）

**目标**：把 §20.6 的 ReverseCommission 挂到 HTTP 路由 + 周期 cron 调。

- `GET  /admin/commission/records` —— 分页 + 过滤（inviter_id / invitee_id / reversed / breach）
- `POST /admin/commission/records/:id/reverse` —— 调 ReverseCommission；二次撤销 HTTP 409；写 logs 表 LogTypeManage 行
- `StartCommissionRiskScanTask()`：6h tick + 1000/批 + 7 天扫描窗口 + `IsMasterNode` 检查 + `sync.Once` 防重入
- `runCommissionRiskScanOnce()`：`SELECT FOR UPDATE`-like（每条 re-load 防 race）+ 跳过已 Reversed + DetectInviteRing/GetUserTotalConsumeQuota 检测 + 命中自动 ReverseCommission(recordID, "auto-risk: ...", 0)

#### **20.8 admin UI 前端**（半天）

**目标**：`web/src/features/admin-commission/` + 路由 `/admin/commission`：

- 列表 + 过滤（inviter_id / invitee_id / reversed / breach）+ 分页
- 每行"撤销"按钮；撤销对话框确认 + reason 必填
- 乐观刷新（撤销成功后不重发 GET，保留分页位置）
- 0 lint errors / 0 typecheck errors（oxlint + tsgo -b）

**架构**：
```
web/src/features/admin-commission/
├── api.ts                                adminListCommissionRecords / adminReverseCommissionRecord
├── index.tsx                             AdminCommissionPage 主页面
├── types.ts                              AdminCommissionRecord / AdminCommissionListFilters
└── components/
    ├── admin-commission-table.tsx       列表 + 分页
    ├── filters-bar.tsx                   过滤栏
    └── reverse-dialog.tsx                撤销确认对话框
web/src/routes/_authenticated/admin-commission/index.tsx     路由 + ROLE.ADMIN 守卫
```

---

## 三、commit 表

| 时间 | commit | section | 内容 |
|---|---|---|---|
| 2026-09-18 | `46138a4c` | §20.1 | topup 写 ledger |
| 2026-09-18 | `783b12c7` | §20.2 | refund 写 ledger |
| 2026-09-18 | `ac858617` | §20.3 | agent_quota_grant 写 ledger |
| 2026-09-18 | `2577434b` | §20.4 | 注册阶段自邀拦截 |
| 2026-09-18 | `1b32c960` | §20.5 | 实时风控叠加 ring + first-topup |
| 2026-09-18 | `457f0489` | §20.6 | ReverseCommission + Reversed 三件 |
| 2026-09-18 | `008c9a40` | §20.7 | admin HTTP + 风控扫描 cron |
| (本 README commit) | `008c9a40` | §20.8 | admin UI 前端（与 §20.7 同 commit 一起提交） |

§20.8 admin UI 与 §20.7 admin HTTP 同 commit（`008c9a40`），因为前端强依赖后端端点路径，分开 commit 体验割裂。

---

## 四、踩过的坑

1. **`httptest.NewRequest` + `c.ShouldBindQuery` 不解析 query string**：
     - controller 最初 `ShouldBindQuery(&req)` 解析 query tag，测试里设了 `?page=1&page_size=2` 但 filter 全是零值。
     - 修：改用 `c.Query("page") + strconv.Atoi`，与现有 `ListAffCommissionRecords` 同一风格。

2. **`seedCommissionRecord` 与现有 helper 冲突**：
     - controller 测试 setup 已有同名的 `seedCommissionRecord(*gorm.DB, model.CommissionRecord)`。
     - 修：重命名为 `seedAdminCommissionRecord` + 独立 `adminTestSeq`。

3. **`setupCommissionControllerTest` 没 migrate `account_ledger` 表**：
     - §20.6 ReverseCommission 写 ledger 报 "no such table"。
     - 修：`seedAdminCommissionRecord` 内幂等 `db.Migrator().HasTable(&model.AccountLedger{})` + 必要时 `AutoMigrate`。

4. **`commission_records.consume_log_id` UNIQUE 冲突**：
     - 测试 seed 用了 id=0 默认值；多条记录撞唯一约束。
     - 修：`ConsumeLogId` 用 `int64(adminTestSeq) * 1000000` 保证唯一。

5. **`ReverseCommission(c *gin.Context, ...)` 在 cron 不可调**：
     - cron 调用没有 gin.Context。
     - 修：参数类型 `*gin.Context` → `context.Context`（ctx 当前未使用，但接口稳定比留死 gin.Context 友好）。

6. **§20.6 测试漏 `event_type=commission_reverse` 过滤**：
     - First 测试用 `Where(ref_type+ref_id)` 命中了 `commission` 那条 ledger 行（ProcessCommission 已写过同 ref 的 commission 行）。
     - 修：加 `Where("event_type = ?", AccountEventCommissionReverse)` 过滤。

7. **§20.4 测试 id 冲突导致 `TestValidate*Inviter` 间歇 FAIL**：
     - `seedInviterWithCreatedAt` 用硬编码 id=6001-6005；前面跑过的 agent 测试用 SQLite auto-increment 已分配 6001。
     - 修：去掉显式 id，让 SQLite 自增 + 用 email 衍生 unique username + aff_code。

---

## 五、结果

- `go test ./... -count=1 -timeout 120s`：**35 包 0 FAIL**。
- 后端 ~1200 行（含 32 个新测试）+ 前端 ~700 行（含 1 个新 feature + 路由）。
- 业务方案 05 §2.7「对账四问」+ 业务方案 06 §6「P4 风控」全部兑现。

---

## 六、仍未兑现 → v0.37.2 候选

| 候选 | 原因 |
|---|---|
| admin UI 前端 i18n 7 语言母语化 | fr/vi/ja/ru 翻译（task-13/i18n 标准流程） |
| `commissionRiskScanTickInterval` / `commissionRiskScanBatchSize` / `firstTopupThresholdQuota` 配置化 | 当前硬编码；上线后看 CPU/IO 再调 |
| 风控白名单机制 | 公司多部门账号互邀合理 ring（如 `model.RiskWhitelist` 表 + `service.IsWhitelistedRing()`） |
| 手动触发扫描的 admin HTTP 端点 | `POST /admin/commission/scan/run` 让运营立刻跑一次 |
| 用户主动通知 | 撤销后没自动发邮件/站内信告知 inviter |

---

## 七、相关文档

- 业务方案：
  - `.docs/task-02-business-goals/05-data-model.md` §2.7（对账四问）
  - `.docs/task-02-business-goals/06-p4-commission.md` §6（P4 风控）
  - `.docs/task-02-business-goals/08-roadmap.md` §6.3（P4 风控路径）
- 落地依赖：
  - `.docs/task-10-p4-commission-core/`（task-10 commission 数据 + 计费）
  - `.docs/task-17-deep-cleanup/`（task-17 §17.3 ledger 统一账本 + §17.4 admin audit 基础）
- 文档同步：
  - `.docs/CHANGELOG.md` v0.37.1 段
  - `.docs/master-plan.md` 当前版本 v0.37.1 + 版本台账 + 最近同步

---

> **决策点**：业务方启动本任务后，§20.1-§20.6 顺序不可调换（依赖关系：commission=errors 应写入 §20.6 风控冲销；§20.5 实时叠加接 §17.4 audit log）；§20.7 + §20.8 顺序不可调换（admin UI 强依赖 admin HTTP 端点）。如果上线后发现 cron 频次不合适，调 `commissionRiskScanTickInterval` 常量即可，不影响接口契约。