# task-14 · 既有技术债清理（SQLite int / 缓存关闭 / 脆弱测试）

> 立项：2026-09-16　｜　状态：✅ **已完成**（3 个独立 merge：`a4b478b0` task-14.1 移除 `type:int` GORM 标签 + AST 静态扫描 CI 网关；`5c009344` task-14.2 修复 `MemoryCacheEnabled=false` 路径下成本过滤不生效；`64f7c860` task-14.3 用 `crypto/rand` 造指纹修 3 条时序脆弱测试）　｜　依赖：**无**
> 起手基线：当前 HEAD
> 预计发版：3 个独立版本

---

## 一、背景

`.docs/summary.md` §六 与 `.docs/master-plan.md` §〇 阻塞项已记录这三条**既有问题，与本次改造无关**，但是真实存在的债。本任务把它们集中收掉。

---

## 二、3 块内容（独立 PR / 独立版本）

### 14.1 · SQLite 下 `type:int` 整表重建

**问题**：GORM 上 `type:int` 标签的表（`users` / `vendors` / `models` / `perf_metrics` 等十余张）在 SQLite 上**每次启动整表重建**（AutoMigrate 检测到类型变化）。在 MySQL / PostgreSQL 上是 `int(11)` / `int4`，没问题。

**触发条件**：本地用 SQLite 启动时日志里看到 `ALTER TABLE` 重建。

**根因**：GORM 在 SQLite 上把 `int` 当成 `integer`，如果迁移前后定义不同就重建。

**修复方向**：
- 选 A：去掉 `type:int` 标签，让 GORM 用数据库默认类型（SQLite 上是 `integer`，MySQL 上是 `int`）—— **改动最小，但要测三库**
- 选 B：用 `type:bigint`（三库都兼容 bigint）—— **更稳，但 BIGINT 占用更大）
- 选 C：完全不用 tag，写自定义 migrator（最大工作量，最稳）

**推荐选 A**。

**涉及改动**：
- `model/user.go` / `model/vendor.go` / `model/model.go` / `model/perf_metrics.go` 等十余文件
- `bin/migration_v0.x-v0.y.sql`（如需硬迁移）
- 三库兼容性单测（已有套路）

**验收**：
- 升级前 DB 启动 10 次无 `ALTER TABLE` 重建
- 三库创建表结构一致（`show create table users`）
- 单测过：`go test ./model/...`

---

### 14.2 · 关闭内存缓存时成本过滤不生效

**问题**：`MemoryCacheEnabled=false` 时，路由侧成本过滤不生效（v0.19.0 引入）。本项目 `.env` 开着缓存不受影响，但用户用「关缓存」部署时这是 bug。

**根因**：选线路走的是 `service/channel_select.go` 内存候选集分支，关了缓存就 fall through 查库分支，但查库分支里没接 v0.19.0 加的成本过滤。

**修复方向**：
- 把 v0.19.0 的成本过滤函数抽出来独立调用
- 在 `service/channel_select.go` 查库分支前也调一次

**涉及改动**：
- `service/channel_select.go`（查库分支）
- 复用 v0.19.0 已有的 `FilterByCost` 函数

**验收**：
- 关 `MemoryCacheEnabled=false` 后跑测试套件：成本过滤与开缓存行为一致
- 单测覆盖：关缓存 + 多条候选线路 + 一条保本 / 一条亏本

---

### 14.3 · `service/` 时序脆弱测试

**问题**：`go test ./service/` 有 2 条用例用 `time.Now().UnixNano()` 造指纹，Windows 时钟粒度粗，相邻用例撞刻度导致失败。

**根因**：测试代码 `time.Now().UnixNano()` 间隔小于系统时钟分辨率。

**修复方向**：
- 改用 `crypto/rand` 造指纹（不依赖时间）
- 或用 `time.Sleep(time.Microsecond)` 强制间隔

**涉及改动**：
- `service/*_test.go`（定位那 2 条用例）

**验收**：
- 跑 100 次 `go test -count=100 ./service/` 全过

---

## 三、范围与不动边界

### ✅ 做
- 14.1 / 14.2 / 14.3 任一项都可独立发版
- 建议开工序：14.2（影响小）→ 14.3（影响小）→ 14.1（影响大）

### ❌ 不做
- 不重构整个 GORM 标签策略（独立项目）
- 不重写路由选线路逻辑（v0.19.0 设计是对的）
- 不动上游测试代码（除非是本项目自己加的）

---

## 四、验收

| 检查 | |
|---|---|
| 14.1：SQLite 启动无 ALTER TABLE 重建 | |
| 14.2：关缓存场景下成本过滤生效 | |
| 14.3：`go test -count=100 ./service/` 全过 | |
| 三库兼容：构建过、单测过 | |
| `relay/` 不动 | |

---

## 五、起手动作

```bash
# 每块独立：

# 14.1
git checkout -b task-14.1-sqlite-int-tags
# 改 model/*.go 的 type:int 标签
# 三库测一遍

# 14.2
git checkout -b task-14.2-cache-disabled-cost
# 改 service/channel_select.go 查库分支

# 14.3
git checkout -b task-14.3-flaky-tests
# 改 service/*_test.go 用 crypto/rand 替代 time.Now()
```

---

## 六、风险与边界

| 风险 | 缓解 |
|---|---|
| 14.1 改了 `type:int` 后老 SQLite 库读不到 | 写迁移脚本，**先扩展 → 再删除旧标签** |
| 14.2 改了查库分支影响生产性能 | 成本过滤是廉价函数，O(候选数)，可接受 |
| 14.3 测试代码改了语义 | 跑生产 shadow run 验证 |

---

## 七、依赖与阻塞

- **依赖**：无
- **阻塞**：无（与其他任务完全独立）
- **不阻塞 P3 / P4 / P5**

---

## 八、参考资料

- `.docs/summary.md` §六「还有几笔欠账」
- `.docs/master-plan.md` §〇「阻塞项」第 ① ② 条
- `service/channel_select.go`（v0.19.0 引入，14.2 修）
- `model/user.go:103-106`（14.1 修）
- `service/*_test.go`（14.3 修）