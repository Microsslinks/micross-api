# task-12 · P5 订阅绑折扣 + 存量迁移 + 倍率退役

> 立项：2026-09-16　｜　状态：⚠️ **12.1 + 12.2 已完成**（merge `53551ddb`，2026-09-17 已合 main；`SubscriptionPlan.DiscountPlanId` 字段、`bindSubscriptionDiscountTx`、迁移脚本均落地），**12.3 倍率退役未做**（`model/group.go` 无 `DeprecatedAt`、分组倍率视觉编辑器无弃用 banner、`GroupRatio` 未强制 1；i18n 「分组倍率即将弃用」文案已就绪但前端无组件引用）　｜　依赖：12.1/12.2 已无依赖，12.3 待定
> 起手基线：N/A
> 预计发版：业务方提出后 1–2 个版本（v0.39.x 范围）

---

## 一、背景

P1（折扣内核）+ P2（成本路由）已收口，task-10（P4 佣金）如果走"全局一个数"路径也会就绪。**P5 不再被任何上游阻塞**，但业务方目前没明确"订阅"这个产品形态怎么定义。

---

## 二、3 块内容

### 12.1 订阅绑折扣
**目标**：用户买订阅后，计费时按订阅档位取折扣（与"客户折扣方案"并行，订阅折扣来源 = `subscription`）

**待业务方**：
- 订阅有哪些档位？（月卡 / 季卡 / 年卡 / 终身？价多少？）
- 订阅折扣率怎么写？是订阅自带还是运营配？
- 订阅能否与折扣方案同时生效？（推荐：订阅优先，但允许运营在某档位上"关闭订阅折扣"）

**涉及改动**（暂定）：
- `model/subscription.go`（可能不存在）新建
- `service/discount_resolve.go` 加"按订阅取折扣"分支（来源 = `subscription`，与 `manual > agent > subscription > customer_code > migration` 优先级一致）
- `controller/subscription.go` 订阅 CRUD
- `web/src/features/subscription/` 新建（目前不存在）
- `billing_post_process.go` 订阅续费 / 到期处理

### 12.2 存量迁移幂等脚本
**目标**：把老 `users.discount_plan_id`（已废弃列）+ 老 `vip/svip` 分组倍率（已废弃档）按新口径迁移到 `discount_bindings`，**幂等、可双跑、不丢数据**

**待业务方**：
- 哪些分组要迁移？哪些保留？
- 迁移的折扣率怎么定？（推荐：保留原 `GroupRatio`，写到 `manual` 来源的绑定）

**涉及改动**：
- `bin/migration_v0.x-v0.y.sql`（数据库迁移）
- `cmd/migrate/main.go`（可能不存在，新建一个幂等迁移工具）
- 三库兼容 + 灰度开关
- dry-run 模式 + 复核脚本

### 12.3 倍率退役
**目标**：vip/svip 这种分组倍率在 UI 上标"已弃用"，运营后台不展示，但代码继续兼容以免报错

**涉及改动**：
- `model/group.go` 加 `DeprecatedAt *time.Time`
- `web/src/features/channels/` 渠道倍率展示页加弃用 banner
- i18n："分组倍率即将弃用，请使用客户折扣方案"

---

## 三、范围与不动边界

### ✅ 做（按业务方启动的子任务）
- 12.1 / 12.2 / 12.3 任一项启动时单独立项
- **建议开工序**：12.2（迁移）→ 12.1（订阅）→ 12.3（退役，迁移完成后才能让旧字段彻底标记为弃用）

### ❌ 不做
- 业务方未启动前，本任务不主动开任何代码改动
- 不擅自把"订阅"产品形态定义出来（这需要业务方定）

---

## 四、验收

| 检查 | |
|---|---|
| 业务方已在 `.docs/master-plan.md` §〇「待业务方决策」明确 P5 启动项 | |
| 迁移脚本三库幂等 | |
| 订阅扣费与 P4 佣金接入后走同一份 `ResolveBillingDiscount` | |
| 倍率退役不影响现有订单（兼容） | |

---

## 五、起手动作

```bash
# 业务方启动前：本任务唯一动作是
# 1. 把 3 块内容写清楚（本文 §二）
# 2. 在 .docs/master-plan.md §〇「待业务方决策」追加 P5 启动项

# 业务方启动后：按 §三 的 12.1/12.2/12.3 任一项拆子任务
```

---

## 六、依赖与阻塞

- **依赖**：业务方需求
- **弱依赖**：task-10（P4 佣金数据 — 因为订阅折扣率可能要进同一份多方案裁决）
- **阻塞下游**：无（终态之一）

---

## 七、参考资料

- `.docs/master-plan.md` §〇「P5 订阅与迁移 · 0%」
- `service/discount_resolve.go`（已有 `manual > agent > subscription > customer_code > migration` 优先级）
- `model/group.go`（vip/svip 分组倍率定义）
- `bin/migration_v0.2-v0.3.sql`（历史迁移脚本范本）
- `.docs/task-02-business-goals/05-data-model.md`（订阅数据结构建议）