# task-07 · 外围重塑（README / Docker / AGENTS / .env）

> 立项：2026-09-16　｜　状态：⚠️ **半成品**（Docker / AGENTS / makefile 已改；README 「本项目差异」章节 5 语言全部缺失；`docs/installation/BT.md` 仍引 `calciumion/new-api`；`new-api.service` systemd 单元未建立）　｜　依赖：**无**
> 起手基线：`baseline/playground-ui-redesign`（`bde2d08c`）
> 预计发版：v0.34.0（5 个子任务可能拆 1–3 个版本）

---

## 一、一句话

把仓库"门面文件"里仍写着 `New API` / `calciumion` / `Calcium-Ion` / `new-api` / `one-api` 的部分，**在不破坏 AGPLv3 §7 与上游 NOTICE 段的前提下**，标识为本项目（MicrossAPI / 微观互联）。

---

## 二、范围

### ✅ 做
- 5 个 README 加"本项目差异"章节，明确 MicrossAPI 身份与改造脉络（**上游 NOTICE / 徽标 / 链接必须原样保留**）
- `Dockerfile` / `docker-compose.yml` / `docker-compose.dev.yml` / `Dockerfile.dev` 的 image 标签、容器名、网络名改 micross-api
- `new-api.service`（systemd 单元文件）的 Description / WorkingDirectory / ExecStart 同步
- `makefile` 的 `DEV_API_SERVICE` / `DEV_POSTGRES_DB` / `DEV_SQLITE_PATH` 默认值
- 根 `AGENTS.md` 标题改 "MicrossAPI" + Overview 段加"二次改造自 QuantumNous/new-api"
- `web/AGENTS.md` 同上
- `.env.example` 编码修复（先 `file .env.example` 确认是 GBK/UTF-8，重写为 UTF-8）

### ❌ 不做（红线）
- 不动 `LICENSE` / `NOTICE` / `THIRD-PARTY-LICENSES.md`
- 不改 README 中上游指向 `https://github.com/QuantumNous/new-api` 的链接
- 不删 README 中"Frontend design and development by New API contributors."这段
- 不改 Go module path（`github.com/QuantumNous/new-api`，AGENTS.md 红线）
- 不重命名根目录 `relaykit/`（独立可构建子模块）

---

## 三、验收

| 检查 | 命令 |
|---|---|
| README 无新引入的上游链接 | `git diff --stat README*.md` + 人工逐条核对 NOTIC E段 |
| Docker 编排能用 | `docker compose config -q` 无报错 |
| 容器名/网络名干净 | `grep -rE "new-api|calciumion" docker-compose*.yml Dockerfile*` 只剩注释/历史引用 |
| AGENTS.md 标题 | `head -3 AGENTS.md` 含 "MicrossAPI" |
| .env.example 是 UTF-8 | `file .env.example` 显示 `UTF-8` |
| 后端不动 | `git diff --stat controller/ model/ relay/ relaykit/ service/` 必须为空 |
| 构建 | `go build ./...` + `cd web && bun run build` 均过 |
| typecheck / lint | `bun run typecheck` / `bun run lint` 均过 |

---

## 四、起手动作

```bash
# 1. 打 baseline
git tag -a "baseline/pre-rebrand-peripherals" -m "..." bde2d08c

# 2. 切新分支
git checkout -b task-07-rebrand-peripherals

# 3. 按以下顺序改
#   1) README*.md  ── 5 语言加"本项目差异"章节
#   2) Dockerfile / docker-compose*.yml
#   3) new-api.service
#   4) makefile
#   5) AGENTS.md / web/AGENTS.md
#   6) .env.example（先确认编码）
```

**建议拆版本**：v0.34.0（README+Docker+systemd+makefile）→ v0.34.1（AGENTS+i18n）→ v0.34.2（.env 修复）。每个版本独立可回滚。

---

## 五、风险

- 改了 README 触发 GitHub badge cache 失效 → 不是代码问题
- `makefile` 默认值改了会让本地开发者踩坑 → 在 commit message 与 docs 里说清
- `.env.example` 编码修复需要确认上游原本用什么编码（GBK / Big5 都可能），如果本地是 GBK 改 UTF-8 后再被非 UTF-8 终端 cat 会乱码 → 写 commit message 提醒

---

## 六、依赖与阻塞

- **依赖**：无（纯文档与编排）
- **阻塞下游**：无（但能影响"仓库门面"，优先级 🔴 P0）
- **不能并行**：与 task-08/09/10/... 可以完全并行（互不依赖）

---

## 七、参考资料

- `.docs/AGENTS.md`（项目红线）
- `.docs/governance/version-policy.md`（基线 tag 规则）
- `.docs/rebrand/`（v0.1.0 已做的品牌改造，留作范本）