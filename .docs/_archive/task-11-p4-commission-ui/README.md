# task-11 · P4 邀请佣金 UI 与不可提现

> 立项：2026-09-16　｜　状态：✅ **未开工**（与代码事实一致——收益页仍只显示 `aff_quota` 推荐卡片，无佣金流水表、无不可提现拦截）　｜　依赖：**task-10**（数据 + 4 个端点）
> 起手基线：当前 HEAD（task-10 收口后）
> 预计发版：v0.38.0

---

## 一、一句话

把上游已有的"邀请奖励"卡片（`web/src/features/earnings/affiliate-rewards-card.tsx`）改成"佣金钱包"：数据源从 `aff_quota` 切到 `aff_commission_balance`（task-10 加的新字段）；加流水表；拦截提现。

---

## 二、范围

### ✅ 做

#### 2.1 数据源切换
- `web/src/features/earnings/lib/affiliate.ts` 把 `aff_quota` / `aff_history_quota` 全替换成 `aff_commission_balance`（task-10 端点）
- `web/src/features/wallet/types.ts` `Affiliate*` 类型调整（加 `commissionBalance` / `commissionRecords`）

#### 2.2 卡片改名 + 文案
- `affiliate-rewards-card.tsx` 改 `commission-rewards-card.tsx`（**保留原文件名 + 重导出别名**，避免上游 merge 冲突）
- 文案："可用佣金" → "可入账佣金"；"邀请奖励" → "返佣"
- i18n 7 语言同步

#### 2.3 流水表（与 wallet 流水共用零件）
- 新建 `commission-records-table.tsx`，复用 `TransactionTable` 模式（与 `wallet/components/transactions-table.tsx` 同结构）
- 字段：日期 / 被邀请人 / 模型 / 消费金额 / 返佣率 / 返佣额 / 是否降级（commission_breach 标红）
- 分页（默认 20/页）
- 接入 `web/src/features/earnings/index.tsx`（已有 `affiliate-rewards-card.tsx` 的父组件）

#### 2.4 不可提现拦截
- 修改 `controller/user.go` `TransferAffQuota`（上游已有）→ 新建 `TransferCommissionToQuota`
- 老 `TransferAffQuota` 保留兼容（已发出去的 `aff_quota` 还能转，但前端入口下线）
- `wallet/topup/withdraw` 链路显式拦截 `aff_commission_balance`：识别为"非可提现余额"，提现页不显示
- `web/src/features/wallet/` 提现对话框：佣金余额灰显 + tooltip "Commission is not withdrawable"
- i18n：增加 "Commission is not withdrawable" / "可入账，不可提现" 等 7 语言

#### 2.5 端点补全
- `GET /api/user/aff/commission/balance`（已有，task-10 加）
- `GET /api/user/aff/commission/records?page=1&page_size=20`（已有，task-10 加）
- `GET /api/user/aff/commission/summary`（已有，task-10 加）

### ❌ 不做
- 邀请链接生成（不在 P4 范围）
- 邀请人查看"被邀请人列表"（上游也没做）
- 跨用户转佣金（业务方未提，不开）
- 历史 `aff_quota` 余额清零（保留兼容，原 `TransferAffQuota` 仍可调）

---

## 三、验收

| 检查 | 命令 / 操作 |
|---|---|
| 端到端 | 注册 → 邀请 → 被邀请人调 API → 邀请人"返佣钱包"立即 +金额 |
| 流水展示 | 列表、筛选、分页、详情展开均正常 |
| 不可提现 | 提现对话框内 `aff_commission_balance` 行灰显、悬停提示"不可提现"；走一遍"申请提现"链路后端要拒绝 |
| 数据兼容 | 老用户的 `aff_quota`（注册一次性返利）仍可点"转入主余额"走老路径 |
| i18n | `bun run i18n:sync` 0 缺失 0 多余 |
| 视觉 | 沿用 `web/src/features/earnings/` 现有视觉，与 wallet 页同色系（task-10 之后确认色板）|
| 构建 | `go build ./...` + `bun run build` + `bun run typecheck` |

---

## 四、起手动作

```bash
# 1. 拉 task-10 收口后的 HEAD
git checkout main
git pull

# 2. 打 baseline
git tag -a "baseline/pre-p4-commission-ui" -m "..." HEAD

# 3. 切分支
git checkout -b task-11-p4-commission-ui

# 4. 顺序
#   1) 先把数据源切到 commission 端点（lib + api + types）
#   2) 卡片改名 + 文案（保留原文件做重导出别名）
#   3) 流水表（先做表格本身，再做卡片）
#   4) 提现拦截（后端 controller + 前端钱包页）
#   5) i18n
#   6) typecheck / lint / build
```

---

## 五、风险与边界

| 风险 | 缓解 |
|---|---|
| 改 `affiliate-rewards-card.tsx` 文件名会被上游 merge 引入冲突 | **保留文件名 + 重导出别名**，新名只在内部用 |
| 老用户 `aff_quota` 余额突然不可见 | 保留 `TransferAffQuota` 端点 + 加一行 i18n 说明"余额将在 X 后下线"|
| 流水表性能 | 加 `(inviter_id, settled_at DESC)` 索引；查询时分页只 SELECT 必要列 |
| 提现拦截漏掉某条路径 | 用 grep 找所有 `wallet/withdraw` 入口，逐条加判断 |

---

## 六、依赖与阻塞

- **强依赖**：task-10（4 个端点必须先有）
- **依赖**：wallet 提现链路 + `web/src/features/earnings/` 已有的视觉风格
- **阻塞下游**：task-15（用户详情页佣金卡片可能要展示本任务的口径）
- **不依赖**：task-07 / task-08 / task-09 / task-12+

---

## 七、参考资料

- `web/src/features/earnings/`（v0.32.0 钱包改版时的产物）
- `web/src/features/wallet/`（提现对话框、流水表零件）
- `controller/user.go:450` `TransferAffQuota`（上游端点）
- `model/user.go:603` `TransferAffQuotaToQuota`（上游方法）
- `.docs/summary.md` §5.3 第 4 项：「收益页加佣金流水表」