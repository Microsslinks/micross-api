# task-16 · 重置邀请人（超管接口 + 详情页按钮）

> 立项：2026-09-17　｜　状态：✅ **已完成**（commit `aa214067`，2026-09-17 合 main；后端 `ResetInviter` + `AdminResetUserInviter` + `POST /api/user/:id/reset_inviter` 路由，前端详情页「重置邀请人」按钮 + AlertDialog + i18n 7 语言 native 化；6 个 model 单测 + 4 个 controller 集成测全过；撞破并修正 3 处隐藏约束：User 无 `updated_at` 列、GORM `Take` 默认带 `deleted_at IS NULL`、项目约定所有 error 返回 HTTP 200 + `success:false`）　｜　依赖：**task-15**（已合）
> 起手基线：当前 HEAD（task-15 merged `31f89529`）
> 预计发版：v0.34.x（独立小版本，~1-2 天工作量）

---

## 一、背景

`task-15-user-profile-discount/README.md` §四「重置邀请人」验收条目原文：

> 仅超管可见"重置邀请人"按钮

task-15 在做用户详情页折扣区时主动把这个按钮**留到了本任务**（scope 控制）：
- README §二「❌ 不做」清单没列它，但实际实现里只放占位 / 没接后端
- README §六「风险与边界」写了「重置邀请人：仅超管可见 + 二次确认 + 留痕」

这次把这条补齐。

**业务动机**：邀请关系一旦配错（用户填错邀请码、运营误改、迁移脚本写偏），运营必须能解绑并重新指派——目前只能改库，没任何受控接口。客户也会投诉「明明是我朋友邀请的，佣金怎么没到 TA 那」。

---

## 二、范围

### ✅ 做

#### 后端（3 块）
1. **`model/user.go`** 新增方法 `ResetInviter(ctx, targetUserId, newInviterId, actorUserId, reason)`：
   - 校验：target 用户存在且未删除；newInviterId 有效（0 = 清空；>0 时该 inviter 也必须存在且未删除）
   - 校验：inviterId 不能等于 targetUserId 自己（不允许自邀请）
   - 写库：`UPDATE users SET inviter_id = ?, updated_at = ? WHERE id = ? AND deleted_at IS NULL`
   - **留痕**：写 `system_operation_logs`（已有表）—— 记录 actor/target/old_inviter/new_inviter/reason/IP
2. **`controller/user.go`** 新增 handler `AdminResetUserInviter`：
   - 入参：`{inviter_id: number, reason: string}`
   - 守卫：`role == RoleRootUser`（仅超管，与 README §六一致）
   - 调上面 `ResetInviter` 模型方法
   - 出参：成功 → 返回更新后的 inviter 信息；失败 → 标准 ApiError
3. **`router/api-router.go`** 注册路由：`POST /api/user/:id/reset_inviter`（在 adminRoute 组下，已自带 `AdminAuth` 普通管理员守卫；handler 内再做 RoleRootUser 校验）

#### 前端（3 块）
1. **`web/src/features/users/api.ts`** 新增 `resetUserInviter(userId, payload)`：
   - 入参 `{inviter_id: number, reason: string}`；返回 ApiResponse<Partial<User>>
2. **`web/src/features/users/components/user-discount-section.tsx`** 加按钮 + 二次确认对话框：
   - 按钮：仅 `currentUser.role === ROLE.SUPER_ADMIN` 时渲染（与后端守卫对齐）
   - 点击 → 弹 AlertDialog 要求填「新的 inviter_id」+「原因（必填 ≥10 字符）」
   - 提交 → 调 `resetUserInviter` → toast → refresh
3. **i18n 7 语言补齐**（~12 个 key）：
   - `Reset inviter`、`New inviter ID`、`Reason (required, ≥10 chars)`、`Inviter reset`（成功 toast）、`Failed to reset inviter`、`Are you sure you want to reset the inviter?` 等

#### 测试（2 块）
1. **`model/user_reset_inviter_test.go`** —— 单元测试 `ResetInviter`：
   - happy path：合法 inviter 替换
   - 自邀请：inviter_id == target_user_id → 拒绝
   - 清空：inviter_id == 0 → 写 0（NULL 等价）
   - inviter 不存在 → 拒绝
   - target 是软删除的 → 拒绝
2. **`controller/user_reset_inviter_test.go`** —— HTTP 集成测试 `AdminResetUserInviter`：
   - 普通管理员（role=10）调用 → 403
   - 超管（role=100）调用 happy path → 200
   - 超管 + target 不存在 → 404
   - 入参 reason < 10 字符 → 400

### ❌ 不做
- 不做"批量重置"接口（一次只重置一个用户，避免误操作面太大）
- 不做"邀请链重算"（重置后下游 aff_quota / aff_history 不会重算，与 README §六「不重写历史」一致）
- 不在用户列表页加"批量重置"行菜单（保持单点可控）
- 不写"邀请人变更历史"详情页（运营只在乎 system_operation_logs 里有留痕就够）
- 不动 `invite_code`（那是用户自己的码，不在"邀请人"概念里）

---

## 三、相关代码位置

| 模块 | 文件 | 行/锚点 |
|---|---|---|
| 用户 model | `model/user.go` | `TransferAffQuotaToQuota` 方法在 603 行附近，ResetInviter 加它后面 |
| 用户 controller | `controller/user.go` | 详情页 payload map 在 530 行；新 handler 加到文件末尾的「admin user actions」区 |
| 用户路由 | `router/api-router.go` | `:id/reset_passkey` 在 162 行；新路由放它下面 |
| 用户前端 API | `web/src/features/users/api.ts` | `resetUserPasskey` 在 269 行；新函数加它下面 |
| 用户前端详情 section | `web/src/features/users/components/user-discount-section.tsx` | 在「归属」区之后加按钮（同行布局） |
| 用户超管 role 守卫 | `web/src/lib/roles.ts` | `ROLE.SUPER_ADMIN === 100` 已有 |
| 系统操作日志 | `model/system_operation_logs.go` | 已存在，复用 |

---

## 四、风险与边界

| 风险 | 缓解 |
|---|---|
| 超管误操作（清空邀请人）| 二次确认 + reason ≥10 字符必填 + 留痕 + 没有批量接口 |
| 重置后下游 aff_quota 不重算，下游用户的佣金会错位 | README §三 不做清单明示「不重算历史」；运营补单走原有的口径 |
| 后端守卫只检查 handler 入口 role==100，前端按钮隐藏；但前端绕过能调 API | 后端守卫是硬约束（与 task-15 同款做法）；前端只是 UX |
| reason 字段被滥用写敏感信息 | reason 进 system_operation_logs 后做长度/敏感词检查（≤200 字符） |
| `inviter_id = 0` 与 NULL 在 Go 里语义不一致 | DB 列是 `int`，用 0 表示"清空"；model 写库时把 0 写 0（NULL 等价于 0）；前端展示时 0 显示为"无邀请人" |

---

## 五、验收（业务方启动后）

| 检查 | |
|---|---|
| 后端 `model/user.go` `ResetInviter` 方法存在且单元测试全过 | |
| 后端 `controller/user.go` `AdminResetUserInviter` handler 注册到 `POST /api/user/:id/reset_inviter` | |
| 普通管理员（role=10）调 → 403 | |
| 超管（role=100）调 happy path → 200 且 `users.inviter_id` 真的改了 | |
| system_operation_logs 出现一条 actor/target/old/new/reason 记录 | |
| 用户详情页 Discount section 出现「重置邀请人」按钮（仅 role===100 渲染）| |
| 点击按钮 → 弹 AlertDialog → 填 inviter_id + reason → 提交 → toast 成功 + 页面刷新 | |
| 不填 reason 或 < 10 字符 → 提交按钮 disabled + 红字提示 | |
| `pnpm i18n:sync` clean（0 missing / 0 extras）| |
| `go build ./...` + `go test ./model/... ./controller/...` 全过 | |

---

## 六、依赖与阻塞

- **强依赖**：task-15（用户详情页折扣区已就绪，且只差这一个按钮 + 后端接口）
- **弱依赖**：无
- **阻塞下游**：无（独立任务）
- **可与任何其他任务并行**

---

## 七、参考资料

- `task-15-user-profile-discount/README.md` §四「仅超管可见重置邀请人按钮」
- `model/user.go` `TransferAffQuotaToQuota`（类似 db update + 事务模式可参考）
- `controller/user.go` `AdminResetPasskey` / `AdminDisable2FA`（admin 操作 handler 模式可参考）
- `router/api-router.go:162` `DELETE /api/user/:id/reset_passkey`（路由注册模式可参考）
- `web/src/features/users/components/dialogs/user-binding-dialog.tsx`（AlertDialog + form 校验模式可参考）
- `model/system_operation_logs.go`（留痕表）