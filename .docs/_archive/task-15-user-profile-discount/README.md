# task-15 · 用户详情页里的折扣方案展示与编辑

> 立项：2026-09-16　｜　状态：⚠️ **折扣 Tab 已落地**（merge `31f89529`，2026-09-17 合 main；详情页加 Discount section——bindings + inviter + commission balance + subject_type 共 4 字段），**「重置邀请人」按钮按 task-16 README 计划留到 task-16**（已落地于 `aa214067`）；经销商归属 / 佣金数据依赖未触（任务边界重定义）　｜　依赖：**task-08 / task-11**（经销商归属 / 佣金数据有依赖）
> 起手基线：当前 HEAD
> 预计发版：v0.34.x（含 task-16 合并后实际完成）

---

## 一、背景

`.docs/summary.md` §5.3 第 6 项原话："P1 遗留：用户详情页里的折扣方案展示与编辑 —— 原方案列了这项，本次是在折扣页里做绑定，用户详情页那一路没动。"

折扣绑定在 `web/src/features/discounts/`（`/discounts` 路由，方案 + 绑定抽屉）已完善，但**从用户侧查看某客户"挂了几套方案、谁是邀请人、佣金余额"** 这条路径没做。

---

## 二、范围

### ✅ 做
- `web/src/features/users/components/users-detail-tabs.tsx`（或新建 tab）加"折扣"标签页
- 显示该客户的：
  - 生效中的 `discount_bindings`（来自 v0.30.0 多方案绑定）
  - 每个绑定对应的方案 + 来源（`manual` / `agent` / `customer_code` / `subscription` / `migration`）
  - 邀请人（`users.inviter_id`）+ 邀请码（`aff_code`）+ 邀请人数（`aff_count`）
  - 佣金钱包余额（task-11 完成后：`aff_commission_balance`）
  - 归属（`subject_type` / `parent_agent_id` —— task-08 完成后）
- 管理员可在此页面直接：
  - 解绑某条 binding
  - 加一条 binding（跳到 `/discounts` 抽屉）
  - 重置邀请人（仅超管）

### ❌ 不做
- 不动"用户列表"页面（已有「客户类型」列 + 行菜单）
- 不动"折扣方案"管理页（已有完整 CRUD）
- 不在用户详情页加"消费明细"（那是另一条 task）

---

## 三、依赖与阻塞

- **强依赖**：task-08（`parent_agent_id` 与归属字段已就绪）、task-11（佣金数据已展示）
- **弱依赖**：无
- **可独立**：只看 binding 部分可以单独发版，task-08 / 11 完成的字段再加

---

## 四、验收

| 检查 |
|---|
| 用户详情页加"折扣"标签，与其他 tab 并列 |
| 列出的 binding 与 `/discounts` 抽屉一致 |
| 解绑动作要走 `DELETE /api/discount/admin/bindings/:id`（已有） |
| 邀请人 + 邀请码 + 邀请人数正确显示 |
| 佣金钱包余额（task-11 完成后）正确显示 |
| 仅超管可见"重置邀请人"按钮 |
| `bun run i18n:sync` 0 缺失 0 多余 |

---

## 五、起手动作

```bash
# 1. 等 task-08 / task-11 收口
# 2. 打 baseline
git tag -a "baseline/pre-user-profile-discount" -m "..." HEAD

# 3. 切分支
git checkout -b task-15-user-profile-discount

# 4. 顺序
#   1) web/src/features/users/components/users-detail-tabs.tsx 加新 tab
#   2) 新建 DiscountTab 内容
#   3) i18n 7 语言
#   4) typecheck / lint / build
```

---

## 六、风险与边界

| 风险 | 缓解 |
|---|---|
| 用户详情页 tab 太多影响性能 | 该 tab 数据走 lazy load，进入 tab 才请求 |
| "重置邀请人"被滥用 | 仅超管可见 + 二次确认 + 留痕 |

---

## 七、参考资料

- `web/src/features/users/components/users-detail-tabs.tsx`（如有）
- `web/src/features/discounts/`（v0.30.0+ 多方案绑定抽屉）
- `model/user.go:99-108`（绑定与归属字段）
- `controller/discount.go`（binding CRUD）