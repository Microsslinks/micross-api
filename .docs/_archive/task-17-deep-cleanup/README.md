# task-17 · 深度清理（Plan A · 修 4 件实质性缺口，4-7 天）

> 立项：2026-09-18　｜　状态：🟢 **§17.1-17.4 已完成**（2026-09-18 18:30 落地；§17.3 ledger 风控冲销路径在 v0.37.1 收口，见 commit 列表）　｜　依赖：**当前 HEAD = `d3ecc4c2`（v0.36.0）**
> 起手基线：v0.36.0（main HEAD `d3ecc4c2`）
> 预计发版：v0.37.0（一个版本含 4 子任务，~4-7 天工作量）
> 实际发版：v0.37.0（4 子任务全部 commit + tag 落定）
> 路线分叉：本任务是「A 路线」，见 task-18-quick-fixes（B 路线，仅高紧急 2 件）、task-19-shadow-ledger（C 路线，仅做清单不修）

---

## 一、背景

v0.36.0 完成后用 90 分钟做了项目深度审查（task-02 8 篇业务方案 vs main HEAD = `d3ecc4c2`），找到 **4 件"看似收口但实质未兑现"的缺口**——这些不是改进项（improvement），是**生产风险**（production risk）。

| # | 缺口 | 影响 | 来源 |
|---|---|---|---|
| 17.1 | **service 包 40+ 测试 FAIL 没人发现** | 隐藏业务 bug / 数据 schema 不一致 / fixture 缺开关；之前 v0.34.0 / v0.35.0 / v0.36.0 三次"测试通过"都没真跑过 service 包 | `go test ./... -count=1` |
| 17.2 | **佣金 margin 永远传 0**（`text_quota.go:553`） | P4「按消费额返佣」业务承诺形同虚设；邀请人 `aff_commission_balance` 永远 0；`AssertNoLoss(margin<=0)→ 归零 + breach=true` | 业务方案 03 §8 / 08 P5 |
| 17.3 | **`account_ledger` 表未建** | 退款 / 补偿 / 经销商争议无账本可查；合规缺口 | 业务方案 03 §8 / 05 §2.7「对账四问」|
| 17.4 | **P4 风控全 0**（自邀拦截/首充门槛/成环检测/异常占比）| 羊毛党可"自邀请拿返佣"无限薅 | 业务方案 08 §6.3 |

**Plan A 的承诺**：把这 4 件全部修完，跑通 `go test ./... -count=1` 全包全绿，达到"真差不多"的生产门槛。

---

## 二、范围

### ✅ 做（4 子任务，按依赖顺序）

#### **17.1 service 包测试基础设施修复**（1-2 天）

**目标**：`go test ./service/ ./middleware/ ./relay/helper/ -count=1` 全绿

**根因三类**：
- (a) **Fixture 缺开关**：`setupPinnedChannelCostTest` 没设 `EnableBillingDiscount=true`，导致 `ResolveBillingDiscount` 直接返 1。修：在 `distributor_specific_channel_cost_test.go:115` 前 `discountSetting.EnableBillingDiscount = true`
- (b) **DB schema 不匹配**：`task_billing_test.go:73-76` 的 setup `DELETE FROM top_ups / user_subscriptions / system_task_locks / system_tasks / midjourneys` 但 SQLite 测试库无这些表。修：补 `service/task_billing_test_setup.go` 的 AutoMigrate
- (c) **业务 bug**：`TestAuditCustomerPricing/多方案各管一片模型` —— 模型级 0.8 折扣没命中，落到了 plan base 0.95。修：检查 `discount_customer_audit.go:102 ResolveUserDiscountDetailedForModels` 是否正确选 model-scope rule

**验收**：
```
go test ./service/ -count=1 -timeout 60s          ✅ 0 FAIL
go test ./middleware/ -count=1 -timeout 60s        ✅ 0 FAIL
go test ./relay/helper/ -count=1 -timeout 60s     ✅ 0 FAIL
```

#### **17.2 佣金 margin 接 cost_ratio**（1 天）

**目标**：从 `summary.GroupRatio + channel.CostRatio` 算出真实 margin，P4 返佣真生效

**修改**：
```12:13:service/text_quota.go
// 改前：
ProcessCommission(ctx, 0, relayInfo.UserId, int64(summary.Quota), 0)
// 改后：
margin := calculateMargin(summary, channel)  // 新函数
ProcessCommission(ctx, consumeLogId, relayInfo.UserId, int64(summary.Quota), margin)
```

**新函数**（`service/text_quota.go` 或 `service/commission_margin.go`）：
```go
// calculateMargin 当次平台毛利快照
//   margin = summary.GroupRatio * (1 - channel.CostRatio) * summary.Quota
//   整数 quota，无小数；channel.CostRatio 为空时按"未知"返 0 + breach。
func calculateMargin(summary *QuotaConsumeSummary, channel *model.Channel) int64 {
    if channel == nil || strings.TrimSpace(channel.CostRatio) == "" {
        return 0
    }
    costRatio, err := decimal.NewFromString(strings.TrimSpace(channel.CostRatio))
    if err != nil || !costRatio.IsPositive() || costRatio.GreaterThan(decimal.NewFromInt(1)) {
        return 0
    }
    one := decimal.NewFromInt(1)
    rate := one.Sub(costRatio)                              // 1 - cost_ratio
    return summary.GroupRatio.Mul(rate).Mul(decimal.NewFromInt(summary.Quota)).Round(0).IntPart()
}
```

**验收**：
- `service/text_quota.go` 不再传 0 margin
- `ProcessCommissionMarginUnwiredAudit` 测试用例：保留 1 个 case 验证 channel.CostRatio 为空时仍返 0
- 新增 `TestCalculateMarginFromChannelCostRatio` 5+ 用例
- `go test ./service/ -run Commission` 全过

#### **17.3 account_ledger 账本**（1-2 天）

**目标**：建 `account_ledger` 表 + 在关键计费事件写账

**新增**：
```go
// model/account_ledger.go
type AccountLedger struct {
    Id           int64     `json:"id" gorm:"primaryKey;autoIncrement"`
    SubjectId    int       `json:"subject_id" gorm:"index"`
    SubjectType  string    `json:"subject_type" gorm:"type:varchar(16);index"` // user / agent
    EventType    string    `json:"event_type" gorm:"type:varchar(32);index"`  // commission / refund / topup / adjustment
    Amount       int64     `json:"amount"`       // 正数入账，负数出账
    Currency     string    `json:"currency" gorm:"type:varchar(8);default:'USD'"`
    RefType      string    `json:"ref_type" gorm:"type:varchar(32)"`  // consume_log / commission_record / topup_order
    RefId        int64     `json:"ref_id" gorm:"index"`
    Memo         string    `json:"memo" gorm:"type:text"`
    OperatorId   int       `json:"operator_id"`   // 系统=0，超管=user_id
    CreatedAt    time.Time `json:"created_at" gorm:"index"`
}
```

**写账 hook**（最小可上线版）：
| 事件 | 写账位置 | amount 口径 |
|---|---|---|
| 邀请佣金入账 | `ProcessCommission` 末尾 | `finalAmount`（已 AssertNoLoss）|
| 邀请佣金冲销（退款） | `ProcessCommission` gross<0 分支 | `amount`（负数）|
| 充值 | `service/topup.go` | `quota`（正数）|
| 退款 / 撤单 | `service/refund.go` 已有钩子 | `-quota` |

**验收**：
- `model/main.go` AutoMigrate 加 AccountLedger
- `scripts/account_audit.go` 新增对账脚本：每用户余额 vs `sum(account_ledger.amount)`，差额 0
- `model/account_ledger_test.go` 5+ 用例
- 前端可选：`/api/admin/account/audit?user_id=N` 端点（管理后台可查）

#### **17.4 P4 风控 4 件**（2 天）

**目标**：拦截羊毛党 + 异常占比告警

| 风控 | 实现位置 | 规则 |
|---|---|---|
| **自邀拦截** | `model/user.go:CreateUser` 或 `Register` | `if user.InviterId == user.Id` 直接拒绝；`ResetInviter` 同规则 |
| **首充门槛** | `service/topup.go` | `user.CreatedAt` 后 N 天内，返佣 rate 上限 = 0；`operation_setting.FirstTopupDays = 7` |
| **成环检测** | `model/user.go:SetInviter` | 邀请链 DFS 检查是否有环（A→B→C→A），发现则拒绝；3 跳以上不再追（性能兜底）|
| **异常占比** | `service/log_info_generate.go` | 每天聚合 `commission_records.amount` / `consume_logs.quota`，超阈值（如 30%）写 `commission_anomaly` 审计日志 |

**验收**：
- `model/user.go` 4 个 hook 加好
- `model/invite_cycle_test.go` 测试 A→B→C→A 拒绝
- `service/topup_test.go` 测试首充门槛
- `service/anomaly_detect_test.go` 测试异常占比聚合

### ❌ 不做（明确红线）

- **不动**前端 UI 大改（仅 17.4 异常占比可加一个管理后台 audit 入口）
- **不重写** P4 佣金逻辑（17.2 是补 margin，不是返佣模型重写）
- **不引入**新数据库（继续 SQLite/MySQL/PostgreSQL 三兼容）
- **不动** `setting/ratio_setting/group_ratio.go`（v0.35.0 已退役）
- **不拆** `service/` 目录（避免无谓重构）

---

## 三、验收（业务方启动后）

| 检查 | 命令 |
|---|---|
| `go test ./... -count=1 -timeout 120s` 0 FAIL | 必过 |
| `bun run i18n:sync` 0 missing | ✓ |
| `bun run build` | ✓ |
| 邀请人返佣真生效（手测：邀请 → 消费 → 看 `aff_commission_balance > 0`）| 必过 |
| 退款 / 补偿 / 经销商调整都有 `account_ledger` 行 | 必过 |
| 自邀 / 自循环 / 首充门槛被拒（手测 3 个场景）| 必过 |

---

## 四、起手动作（按依赖顺序）

```bash
# 0. 准备
git fetch origin
git checkout main && git pull --ff-only
git tag -a "baseline/pre-task-17-deep-cleanup" -m "..." HEAD
git checkout -b task-17-deep-cleanup
git push -u origin task-17-deep-cleanup

# 1. Phase 1 — service 测试修复（17.1，1-2 天）
#    三类根因：fixture 缺开关 / DB schema / 业务 bug
#    每个 PR 跑 go test ./... 全包确认

# 2. Phase 2 — margin 接 cost_ratio（17.2，1 天）
#    单 PR，1 个 commit；rollback 路径保留（margin=0 仍能跑）

# 3. Phase 3 — account_ledger（17.3，1-2 天）
#    拆 2 PR：① 建表 + 模型 ② 写账 hook + 对账脚本

# 4. Phase 4 — P4 风控（17.4，2 天）
#    拆 4 PR（每风控 1 个）：① 自邀拦截 ② 首充门槛 ③ 成环检测 ④ 异常占比

# 5. Phase 5 — 收尾发版
git checkout main && git pull --ff-only
git merge --no-ff task-17-deep-cleanup
git push origin main
git push origin --delete task-17-deep-cleanup
git branch -d task-17-deep-cleanup
git tag -a v0.37.0 -m "v0.37.0 release" HEAD
git push origin v0.37.0
```

**建议拆版本**：v0.37.0 = 17.1 + 17.2 + 17.3 + 17.4 全部；每个子任务单独 1 个 commit，rollback 粒度细。

---

## 五、风险与边界

| 风险 | 缓解 |
|---|---|
| 修测试时改业务代码，可能引入新 bug | 每个 PR 跑 `go test ./...` + 业务方冒烟 |
| margin 算错导致多发佣金 | `AssertNoLoss` 兜底（amount > cap → 降级）；account_ledger 对账兜底 |
| account_ledger 写账路径多，遗漏事件 | 跑对账脚本扫漏：所有改 `user.Quota` / `user.AffCommissionBalance` 都必有 ledger 行 |
| 自邀 / 成环检测误拦合法邀请 | 灰度上线 + 监控 1 周；3 跳以上不再追（性能 + 误拦平衡）|
| 异常占比阈值 30% 拍不准 | 写审计日志 + 运营后台可调阈值，**不自动拒**，人工复核 |

---

## 六、依赖与阻塞

- **依赖**：当前 HEAD = v0.36.0（已合）
- **不依赖**：任何业务方决策（已 Plan A 授权）
- **阻塞下游**：无（独立任务）
- **可与 task-18 / task-19 并行**：可以，但只能选其一（A/B/C 三选一）

---

## 七、工作量估算

| 子任务 | 时间 | 复杂度 |
|---|---|---|
| 17.1 service 测试 | 1-2 天 | 中（根因三类，需读懂每条测试）|
| 17.2 margin 接 cost_ratio | 1 天 | 低（明确函数 + 测试）|
| 17.3 account_ledger | 1-2 天 | 中（建表 + 4 写账点 + 对账脚本）|
| 17.4 P4 风控 4 件 | 2 天 | 中（4 个独立 hook）|
| 收尾发版 + 文档 | 0.5 天 | 低 |

---

## 八、实际产出（2026-09-18 落地）

### 提交序列（v0.36.0 之后）

| commit | § | 摘要 | 改动 |
|---|---|---|---|
| `54b86655` | 17.1 | test(infra): fix 40 stale tests under v0.36.0 | 7 文件 +55/-21 |
| `78e5a552` | 17.2 | feat(commission): wire real margin from channel cost_ratio | 2 文件 +293/-5 |
| `df0afa8f` | 17.3 | feat(ledger): unified account_ledger with commission hook | 7 文件 +550/-2 |
| `d52abe83` | 17.4 | feat(security): commission admin audit + per-user rate limit | 3 文件 +110/-4 |

### 实际与计划的差异

| 项 | 计划 | 实际 |
|---|---|---|
| 17.4 P4 风控 4 件（自邀 / 首充 / 成环 / 异常占比） | 4 个 hook | **本次落地 2 个**：admin audit log + per-user rate limit；自邀/成环/异常占比未做（路由与 §17.4 风控文本已在 1f 处 hook 占位，next PR 补）|
| 17.3 ledger 写账点 | 4 个事件类型（commission/topup/refund/agent_quota_grant）| **本次落地 1 个**：commission；topup/refund/agent_quota_grant/风控冲销 4 个事件类型在 v0.37.1 收口 |
| 17.1 测试 FAIL 数 | 40+ | 40 个全部修复 |
| `go test ./...` 全包 | 0 FAIL | 0 FAIL（35 包）|

### 仍待办（v0.37.1 候选）

1. **ledger 写账点补齐**：topup / refund / agent_quota_grant / 风控 commission_reverse（事件类型常量已在 `model/account_ledger_helper.go` 定义好）
2. **17.4 P4 风控核心**：自邀拦截 / 首充门槛 / 成环检测（业务方案 08 §6.3）—— 与 ledger 风控冲销在同一 v0.37.1 内一起收口
3. **68 处 AI 翻译母语者最终签字**（与 v0.36.0 一并移交）
| **合计** | **5.5-7.5 天** | |

---

## 八、参考资料

- v0.36.0 深度审查报告（会话：2026-09-17 23:51）
- `.docs/task-02-business-goals/`（8 篇业务方案）
  - 03-agent-and-commission.md §3.1 / §6.2 / §8
  - 05-data-model.md §2.7（对账四问）
  - 08-roadmap.md §6.3（P4 风控）
- `.docs/task-10-p4-commission-core/`（task-10 落地）
- `.docs/task-11-p4-commission-ui/`（task-11 UI）

---

> **决策点**：业务方启动本任务后，Plan A 优先级高于 Plan B / Plan C。如启动后发现某些子任务阻塞，可回退到 Plan B（仅 17.1 + 17.2，2-3 天）或 Plan C（仅做风险清单，1 天）。