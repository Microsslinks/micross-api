# task-10 · P4 邀请佣金核心（数据 + 计费 + 不赔本）

> 立项：2026-09-16　｜　开工：2026-09-16　｜　状态：⚠️ **口径已敲定（A① B② C①），代码未动**（2026-09-16 业务方敲定口径；仓库中 `model/commission*.go` / `service/commission*.go` / `controller/commission*.go` 全 0 文件，`commission_records` / `account_ledger` 表未建，按消费额返佣逻辑未实现）
> 起手基线：tag `baseline/pre-task-10-p4-commission-core`（该 tag 现指向无关 commit `ed833ae3`，**已被本任务清掉**，重新打 tag 见 `task-17-doc-sync`）
> 预计发版：v0.37.0（核心层，独立可发版；UI 与不可提现放 task-11）

---

## 一、一句话

被邀请人每次成功调用 API 后，**按消费金额 × 邀请人返佣率**给邀请人返一笔佣金，**实时校验不赔本**，落入独立的"佣金钱包"（不可提现、只进不出）。

---

## 二、先与业务方定的 3 个口径

| 口径 | 选项 | 推荐 | 阻塞点 |
|---|---|---|---|
| **A. 返佣率** | ① 全局一个数（运营填）② 按邀请人级别（VIP 高 / 普通低）③ 按被邀请人消费档 | **①**（最简单，先做） | 运营在「系统设置」加字段 |
| **B. 返佣分母**（按什么金额算比例）| ① L1 官方标价 ② 平台实收（l1 × 客户折扣） ③ L2 进货价（一定不赔） | **②**（贴近业务观感；③ 也是对的但难解释） | 数学上：①+② 都可能 > ③，需要不赔本拦截 |
| **C. 不赔本粒度** | ① 逐笔实时校验 `佣金 ≤ 平台毛利`② 周期性对账③ 不校验 | **①**（实时，与 v0.19.6 "指定渠道亏本留痕" 同模式） | 需要在计费主流程加一道拦截 |

业务方 30 分钟电话能敲完这三件事。

---

## 三、范围（本任务）

### ✅ 做
- `model/commission.go` 新增 `commission_records` 表
  - `id` / `inviter_id` / `invitee_id` / `consume_log_id` / `gross`(消费金额) / `rate`(返佣率) / `amount`(返佣额) / `currency` / `settled_at`
  - 三库兼容 schema
- `model/user.go` 加 `aff_commission_balance INT` 字段（**独立钱包，与主余额分开**，迁移脚本加列）
- `service/billing_post_process.go`（v0.17.0 起存在）改：扣费完成后 → 写一条 `commission_records` + `aff_commission_balance += amount`
- `service/commission.go` 新增：`CalculateCommission(consumeLog)` —— 按口径 B × A 计算金额
- `service/commission.go` 新增：`AssertNoLoss(consumeLog, commission)` —— 若 `commission > 平台毛利 - 安全垫`，降级到上限 + 写审计
- `controller/commission.go` 新增 4 个端点（task-11 用，先建接口）：
  - `GET /api/user/aff/commission/balance`（独立钱包余额）
  - `GET /api/user/aff/commission/records`（流水，分页）
  - `GET /api/user/aff/commission/summary`（累计）
  - `POST /api/admin/commission/rate`（运营设置全局返佣率）
- `setting/operation_setting.go`（已存在）加 `CommissionRate DECIMAL(6,6) DEFAULT 0` 字段（运营在系统设置改）
- i18n 7 语言同步新增键

### ❌ 不做（task-11 范围）
- UI 卡片 / 流水表
- 不可提现拦截（task-11 的 `TransferCommissionToQuota`）
- 把现有 `aff_quota` 改名（保留兼容）

---

## 四、数据流

```
[用户成功调用 API]
  ↓
[扣费] user.Quota -= cost （v0.17.0）
  ↓
[写消费日志] consume_logs INSERT （v0.17.0）
  ↓
[新] 查邀请关系：users.inviter_id WHERE user = consume.user_id
  ↓
[新] CalculateCommission：
   gross = consume.total_cost (口径 B=实收)
   rate = operation_setting.CommissionRate (口径 A)
   amount = gross × rate
  ↓
[新] AssertNoLoss：
   margin = consume.l1_price - consume.cost_ratio * l1_price (毛利)
   if amount > margin × 0.8:  amount = margin × 0.8 ; 写 audit "commission_breach"
  ↓
[新] 写 commission_records INSERT
  ↓
[新] user.aff_commission_balance += amount  (邀请人)
```

---

## 五、验收

| 检查 | 命令 / 操作 |
|---|---|
| 表创建成功 | `bun run` 等价的迁移：`make` 启动后三库都能 `SELECT * FROM commission_records LIMIT 1` |
| 旧库不丢字段 | 升级前已注册的老客户 `aff_commission_balance` 默认 0 |
| 计费链路返回金额不变 | 跑 G1 比对脚本（`service/discount_billing_test.go` 类似套路）：100 笔 API 调用，返佣数据对得上 |
| 不赔本校验触发 | 写一个用例：`margin = 10`，`rate = 0.5` → `amount = 5`，正常；`rate = 1.5` → `amount` 降级到 8 |
| i18n | `bun run i18n:sync` 0 缺失 0 多余 |
| `relay/` 不动 | `git diff --stat relay/` 必须为空 |
| 构建 | `go build ./...` + `bun run build` + `bun run typecheck` |

---

## 六、起手动作

```bash
# 1. 先与业务方敲定 §二 的 3 个口径（必须）

# 2. 打 baseline
git tag -a "baseline/pre-p4-commission-core" -m "..." HEAD

# 3. 切分支
git checkout -b task-10-p4-commission-core

# 4. 顺序
#   1) model/commission.go + 迁移（注意 type:int 不要命中，A 类债问题）
#   2) model/user.go 加字段
#   3) service/commission.go CalculateCommission + AssertNoLoss
#   4) service/billing_post_process.go 接入
#   5) controller/commission.go 4 个端点
#   6) setting/operation_setting.go 加字段
#   7) i18n 7 语言
#   8) 测试：commission_test.go + 不赔本单测
#   9) G1 比对脚本（取生产一周的日志，离线 replay）
```

---

## 七、风险与边界

| 风险 | 缓解 |
|---|---|
| 计费主流程加新逻辑会让所有请求慢 N 毫秒 | CalculateCommission 用邀请关系缓存（与 `inviter_id` 索引同事务读，O(1)）；AssertNoLoss 只在 `cost > 0` 时跑 |
| 「不赔本」降级导致返佣少 → 客户投诉 | 与 v0.19.6 留痕同模式；运营后台能看到降级日志 |
| 邀请关系在用户改名 / 注销时漂移 | `users.inviter_id` 写入后不改；注销走 `deleted_at`，不在返佣链里 |
| 同一笔消费多次返佣（重试场景） | 在 `commission_records` 加唯一约束 `(consume_log_id, inviter_id)` |

---

## 八、依赖与阻塞

- **强依赖**：业务方敲定 §二 的 3 个口径
- **弱依赖**：task-08（P3 代注册）— 代注册客户的 `parent_agent_id` 与 `inviter_id` 是两个独立字段，不冲突
- **阻塞下游**：task-11（UI + 不可提现）依赖本任务的 4 个端点
- **不依赖**：task-07 / task-09 / task-12+

---

## 九、参考资料

- `model/user.go:102-106`（现有的 `aff_*` 字段）
- `model/user.go:588-601` `inviteUser()`（现有的注册返利实现，可参考但不直接复用）
- `service/billing_post_process.go`（v0.17.0 起存在）
- `setting/operation_setting.go`（已有系统设置机制）
- `.docs/master-plan.md` §〇「待业务方决策 · P3 发放折算口径」（同处需要业务方定口径的位置）