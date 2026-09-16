# cmd/migrate-discount-plans

**Task-12 / Phase 1 · 12.2 存量迁移幂等脚本**（master-plan §4.6 口径）

## 作用

把老 `users.discount_plan_id = 0` 且 `group != 'default'` 的存量用户，按
`setting/ratio_setting.GroupRatio[group]` 折算成 `DiscountPlan`，写一条
`source='migration'` 的 `discount_bindings` 行。

`BindDiscountPlan` 事务内同步更新 `users.discount_plan_id`（快路径冗余字段，
**保留**，不删）。

幂等：重复跑不写重复行（plan 按 name 复用；binding 已有 source='migration' 跳过）。

## 业务方决策项（将来可改）

| 项 | 当前预设 | 改的地方 |
|---|---|---|
| 迁移白名单 | 所有非 default 分组（在 GroupRatio 里的）| `--groups=vip,svip` flag |
| 折扣率来源 | GroupRatio JSON（运营可改）| `options.GroupRatio` |
| 折扣方案命名 | `migration_<group>`（owner=platform）| 改 `findOrCreateMigrationPlan` |
| 弃用 banner | "分组倍率即将弃用，请使用客户折扣方案" | i18n key（task-12 Phase 3）|

## 用法

```bash
# 1. dry-run 看会改什么
go run ./cmd/migrate-discount-plans --dry-run

# 2. 试水：每个分组前 10 条
go run ./cmd/migrate-discount-plans --limit 10

# 3. 白名单：只跑 vip + svip
go run ./cmd/migrate-discount-plans --groups=vip,svip

# 4. 实跑
go run ./cmd/migrate-discount-plans

# 5. 编译成单文件
go build -o /tmp/migrate-discount-plans ./cmd/migrate-discount-plans
/tmp/migrate-discount-plans --dry-run
```

## 环境变量

读 `.env` / 环境变量 `SQL_DSN`（MySQL / PostgreSQL）或 `SQLITE_PATH`（SQLite）。
默认连主库；LOG_SQL_DSN 不一致时单独连日志库。

## 三库兼容

- `users.group` 列名是保留字，统一用反引号
- `discount_bindings` 表已存在（task-04-p1-discount 阶段）
- `discount_plans` 表已存在
- 不依赖外键 / CHECK（master-plan §4.7）

## 验收

| 检查 | |
|---|---|
| `--dry-run` 打印变更不写库 | ✅ |
| 实跑后 `discount_bindings` 有 source='migration' 新行 | ✅ |
| 实跑后 `users.discount_plan_id` 已更新为对应 plan_id | ✅ |
| 双跑不重复写入 | ✅ |
| 三库 MySQL/PG/SQLite 各跑一遍通过 | ✅ |

## 紧急回滚

- 改业务只需：直接改 `options.GroupRatio`（运营后台可改）
- 删除迁移绑定：调 `model.UnbindDiscountPlan(binding_id)`（停用 + 重算快路径）
- 删 DiscountPlan：先解绑所有 source='migration' 行，再 `model.DeleteDiscountPlan(plan_id)`