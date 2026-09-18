# task-08 · P3 经销商代注册归属

> 立项：2026-09-16　｜　状态：✅ **未开工**（业务口径已收口，零业务依赖；所有声明端点 `POST /api/agent/customers` / `reset-password` / `disable` 仓库中均 0 匹配）　｜　依赖：**无**
> 起手基线：当前 HEAD（v0.34.x）— **不依赖 task-07**，可单独发版
> 预计发版：v0.35.0

---

## 一、一句话

经销商替下属客户开账号，账号建好的同时自动归属到这个经销商名下（同事务），并支持重置密码 / 停用。

---

## 二、范围

### ✅ 做
- 新增 `POST /api/agent/customers` —— 经销商代注册
  - 必填：邮箱、用户名、初始密码（随机生成也行）、角色（默认普通用户）
  - **同事务**：写 `users` + `parent_agent_id` + 默认配额 + 默认边栏配置
  - 冲突校验：邮箱冲突、用户名冲突、当前经销商是否有"代注册"权限
- 新增 `POST /api/agent/customers/:id/reset-password`
- 新增 `POST /api/agent/customers/:id/disable`
- 复用 `model/user.go` 的 `Insert(inviterId int)`，扩展参数接 `parentAgentId int`
- 在 `web/src/features/dealer/`（目前只有空账单占位）新建：
  - `MyCustomersPage`（已存在 `agent-customers-panel.tsx`，加"代注册"按钮 + 行菜单）
  - `RegisterCustomerDialog`（邮箱 + 用户名 + 初始密码，提交前显示"将归属到你名下"）
  - `CustomerRowActions`（重置密码 / 停用）
- 侧栏 `use-sidebar-data.ts` 经销商分组加"我的客户" — **白名单三处**：路由 / `pathPattern` / `URL_TO_CONFIG_MAP`（v0.19.2 教训）
- i18n 7 语言同步

### ❌ 不做
- 邀请链接生成（task-09 折算同步做或单独做）
- 经销商发额度（v0.25.0 已做）
- 经销商改价（v0.25.0 已做）
- 跨经销商转移客户归属（业务方未提，不开）

---

## 三、验收

| 检查 | 命令 / 操作 |
|---|---|
| 接口签名对齐 | 走 `router/api-router.go` 注册 `POST /api/agent/customers` |
| 权限门 | 只有 `users.subject_type='agent'` 且启用了代注册的能调 |
| 同事务 | DB 模拟失败（用户名已存在）→ `parent_agent_id` 也没写入 |
| 停用可逆 | `disable` 后可再 `enable`（v0.21.0 已支持） |
| 列表可见 | 代注册的客户立即出现在"我的客户"列表 |
| 前端 | 「代注册」按钮可见、行菜单「重置密码」/「停用」可用 |
| i18n | `bun run i18n:sync` 7 语言键集一致：缺失 0 / 多余 0 |
| 构建 | `go build ./...` + `bun run build` + `bun run typecheck` |

---

## 四、起手动作

```bash
# 1. 打 baseline
git tag -a "baseline/pre-p3-customer-registration" -m "..." HEAD

# 2. 切分支
git checkout -b task-08-p3-customer-registration

# 3. 顺序
#   1) controller/agent_customers.go 新增 RegisterCustomer / ResetPassword / Disable
#   2) model/user.go 扩展 Insert(parentAgentId)
#   3) router/api-router.go 注册 3 个端点（守卫 = agent + 已确认合规）
#   4) web/src/features/dealer/ 新建 dialog + 加按钮
#   5) use-sidebar-data.ts 改三处
#   6) i18n 7 语言
#   7) bun run i18n:sync + bun run typecheck + bun run build
#   8) go build ./... + go test ./model/... ./controller/...
```

---

## 五、风险与边界

- **密码哈希策略**：代注册初始密码走与正常注册相同的 `crypto.BcryptHash`
- **通知方式**：要不要给被注册的客户发邮件？目前上游 `oauth` 路径不发邮件，**保持不发**（避免引入邮件服务依赖）
- **重复代注册同一邮箱**：拒绝并说清理由，不静默吞掉
- **事务粒度**：写 `users` + `parent_agent_id` + 默认配额 + 边栏配置，**任何一步失败整笔回滚**

---

## 六、依赖与阻塞

- **依赖**：`agent_profiles` 表（v0.22.1 已就绪）
- **依赖**：i18n 加新键
- **阻塞**：task-09（发放折算）的"经销商后台发额度"页面可能要加一个"由代注册自动发"的入口，**但可以在 task-09 内处理**
- **不依赖**：task-07（外围重塑）、task-10+（P4 佣金）

---

## 七、参考资料

- `controller/agent*.go`（v0.21.0–v0.27.0 共 7 版）
- `model/agent*.go`（`agent_customer.go` / `agent_wholesale.go` / `agent_code.go`）
- `web/src/features/dealer/`（v0.25.0 起建立的，目前只有"我的客户"页 + 提现占位）
- `.docs/master-plan.md` §〇「P3 剩余两件」