# task-13 · 母语审校（fr / ru / ja / vi）

> 立项：2026-09-16　｜　状态：⚠️ **业务热路径已审校**（merge `55d87f98`，2026-09-17 合 main；task-15/16 与折扣/经销商相关新键 4 语言全部真翻译），**老键 fr/vi 残留 12+ 处英文 fallback**（如 "My Customers" / "No customers yet" / "Allow cost breach" / "Add rule" / "Administration" / "AI driven" 等），**未达 README「母语审校」门槛**　｜　依赖：**外部资源（母语审校者）**
> 起手基线：当前 HEAD
> 预计发版：审校完成一批后 1 个版本（v0.x.x）

---

## 一、背景

P0 收口时（v0.3.0）术语统一在 7 语言各落了 6 个概念名（`channel`/`vendor`/`ratio`/`quota` 等）。v0.18.0 的"渠道→上游供应商"全文替换**只在中文做了**（216 句简 + 215 句繁）；fr / ru / ja / vi 只落了概念名，**句子里的同义直译没做**。

`task-04-p1-discount/README.md` 与 `.docs/summary.md` §5.3 第 6 项已声明：**这条需要母语者，AI 代劳不了**。

---

## 二、范围

### ✅ 做
- 逐语言过 `web/src/i18n/locales/<lang>.json`：
  - 找出**疑似机器翻译**的句子（中文价值对比、英文母版都能找到来源）
  - 给每个问题句子写母语改写建议
  - 复核一遍含义准确性

### ❌ 不做
- 不动 `en.json` / `zh.json` / `zh-TW.json`（这 3 个已人工或半人工过）
- 不新增词条（只改措辞）
- 不重写全部句子（只标记问题句子 + 改最严重的）

---

## 三、4 语言的现状概览

| 语言 | 文件大小（行）| 主要问题（估测）|
|---|---|---|
| `fr.json` | ~5 700 | "上游供应商" 翻译有时用 `canal`（旧词）有时用 `fournisseur en amont`（新词），不一致 |
| `ru.json` | ~6 500 | 句子结构偶尔是中译俄的"翻译腔"，术语用对了但语气不像 SaaS 产品 |
| `ja.json` | ~6 200 | 敬语层级混乱，管理员 / 客户 / 经销商三套语气混在一起 |
| `vi.json` | ~6 000 | 越南语外来词音译选择不统一（"余额" `số dư` / `credit` / `balance` 都出现过）|

**审计顺序建议**：fr → vi → ja → ru（按"机器翻译痕迹明显程度"排序，估测）。

---

## 四、起手动作

```bash
# 1. 联系母语审校者（朋友 / 社区 / 外包）
#    需要：fr / vi / ja / ru 各 1 人，每语言 4–8 小时工作量
# 2. 给审校者一个工作清单：本文 §五
# 3. 审校者返回 PR / diff
# 4. 合并进 main
```

### 4.1 找审校者
- **fr**：法裔加拿大开发者社区 / Fiverr 上找法语母语 + IT 经验
- **vi**：越南 AI 社区 / 越南留学生群
- **ja**：日本 SaaS 公司（如 SmartBank 等有 i18n 经验的人）
- **ru**：俄罗斯开发者社区（Хабр / Telegram 上的俄语翻译组）

### 4.2 工作交接清单（给审校者）
```
1. 拉 main 分支
2. 打开 web/src/i18n/locales/<lang>.json
3. 找一个中文价值键（zh.json 里同 key）作为 source of truth
4. 找出所有"读起来像机翻"的句子
5. 用母语重写，保持：
   - 长度不超过原文 ±20%
   - 不要新增/删除键，只改 value
   - 保持变量插值（{varName}）原样
6. 提交 PR 到 <分支>，标题 "i18n(<lang>): native review pass"
```

---

## 五、验收

| 检查 | |
|---|---|
| 4 个 PR 合并到 main | |
| `bun run i18n:sync` 0 缺失 0 多余（键集不能变）| |
| 每个 PR 至少改动 50 个 value | |
| 自然读起来"不像机翻"（随机抽 20 条给人读，无人说"怪"） | |
| 构建 / lint / typecheck 过 | |

---

## 六、风险与边界

| 风险 | 缓解 |
|---|---|
| 审校者把句子改太长，超出 UI 宽度 | 在 i18n 文档加约束："长度不超过原文 ±20%" |
| 审校者改了变量插值导致渲染错 | 强调"不要新增/删除键，只改 value；变量原样" |
| 审校者语言习惯与目标用户群不符 | 优先找**有 SaaS / AI 产品 i18n 经验**的 |
| 4 个 PR 各自独立无法合并 | **不强求同步**，每语言单独发版 |

---

## 七、依赖与阻塞

- **依赖**：外部资源（母语审校者）
- **不依赖**：任何代码改动（纯文案调整）
- **阻塞下游**：无（独立任务）
- **可与任何其他任务并行**

---

## 八、参考资料

- `web/src/i18n/locales/{fr,ru,ja,vi}.json`
- `web/src/i18n/locales/en.json`（作为 source of truth 的参考）
- `web/src/i18n/locales/zh.json`（中文价值参考）
- `.docs/translation-glossary.md`（已有术语表）
- `.docs/translation-glossary.fr.md` / `.docs/translation-glossary.ru.md`（法语 / 俄语专属术语表）