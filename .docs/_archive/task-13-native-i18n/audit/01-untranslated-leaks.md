# 01 · 未翻译占位符 leak 清单（task-13 phase 1）

> 创建：2026-09-16（修正：同步 sync 报告真实数字）
> 定义：`untranslatedCount`（来自 `web/src/i18n/locales/_reports/_sync-report.json`）
> 即 value 字段与 en.json 原文相同的键

## 总数（sync 报告权威）

| 语言 | untranslatedCount | 与本文件"🔴 必改"重叠 |
|---|---|---|
| en | 0 | 0 |
| fr | **171** | 50 |
| ja | **431** | 50 |
| ru | **432** | 50 |
| vi | **172** | 50 |
| zh-TW | 0 | 0 |
| zh | 13 | 0 |
| **4 lang 总计** | **1206** | **~200** |

> 数字解释：i18n:sync 工具对比 value 字段与 en.json；差异数为该 lang 的"未翻译/不同翻译"总数。
> 不全等于"未翻译"——可能含"被改写但差异较大"的 key。

## 🔴 必改 leak（核心 8 概念，4 语言共同）

### Commission（2 keys × 4 lang = 8 处）

| key | zh 翻译 | 4 lang value |
|---|---|---|
| Commission ratio | 分成比例 | "Commission ratio" |
| Commission ratio must be a number between 0 and 1 | 分成比例必须是 0 到 1 之间的小数 | "Commission ratio must be a number between 0 and 1" |

### Discount Plan（2 keys × 4 lang = 8 处）

| key | zh 翻译 | 4 lang value |
|---|---|---|
| Discount Plans | 客户折扣方案 | "Discount Plans" |
| Discount plan | 折扣方案 | "Discount plan" |

### Channel 复合句（2 keys × 4 lang = 8 处）

| key | zh 翻译 |
|---|---|
| Channels without a cost ratio are skipped when serving customers on a discount. | 未录入进货折扣的渠道，在给折扣客户选线路时会被跳过。 |
| Channels without a cost ratio are skipped ... {{days}} ... | 未录入进货折扣的渠道... {{days}} 天没更新，也可能已经和上游价格对不上了。 |

### Vendor（2 keys × 4 lang = 8 处）

| key | zh 翻译 |
|---|---|
| Vendor-level rule | 厂商级规则 |
| Vendor list price · per 1M tokens | 厂商挂牌价 · 每 100 万 token |

### Dealer（6 keys × 4 lang = 24 处）

| key | zh 翻译 |
|---|---|
| Dealer price | 经销商价 |
| Dealer status removed | 已取消经销商身份 |
| Dealer settings saved | 经销商设置已保存 |
| Dealer Settings | 经销商设置 |
| Dealer | 经销商 |
| Dealer wholesale price | 经销商拿货价 |

### Quota（1 key × 4 lang = 4 处）

| key | zh 翻译 |
|---|---|
| Quota issued | 已发放额度 |

**核心 8 概念总计**：~17 keys × 4 lang = **~60 处**（与本文件"🔴 必改"重叠）

---

## 🟡 非核心 leak（每语言 100-380 处）

来源：
- 后端 i18n 错误码（大部分未翻译）
- 后端 admin 提示文案
- debug/技术 tooltip
- 测试/演示文案
- 第三方集成字符串（Stripe / Waffo / Creem / Pancake 文档/参数）

**典型分布估算**：
- fr：~120 处（171 - 50 = 121）
- vi：~120 处
- ja：~380 处（**2 倍**）
- ru：~380 处（**2 倍**）

> **重要发现**：ja / ru 的非核心 leak 数量是 fr/vi 的 3 倍以上。
> 这两个语言的"低频未翻译"问题更严重——需要 Phase 2 进一步确认是"漏翻译"还是"已重写差异较大"。

---

## zh 13 处"未翻译"

`untranslatedCount=13` 在 zh 是异常——zh 通常全译。可能原因：
1. en.json 与 zh.json 字面相等（变量插值完全相同）—— 实际不算"未翻译"
2. 一些术语英文没翻（少数）
3. placeholder 占位符（如纯数字 / 时间串）

Phase 2 完整扫描时一并确认。

---

## 后续

- Phase 2 机翻特征扫描会给出**非核心 leak 的具体列表** + 区分"真未翻译"vs"差异较大"
- Phase 4 AI 改写候选**优先改本表 🔴 必改 leak**（~60 keys / 语言）
- Phase 5 业务方签字决定是否扩到 🟡 非核心 leak
- Phase 6 写入 i18n 文件
- Phase 7 母语者最终签字

---

## 验收

- [x] 4 语言 × 8 概念术语表（`web/src/i18n/translation-glossary.md`）
- [x] 8 概念 × 4 文件不一致列表（`01-terminology-map.md`）
- [x] 🔴 必改 leak 清单（本文件核心部分，约 60 keys / 语言）
- [x] 真实 untranslatedCount（来自 sync 报告）已记录
- [ ] Phase 2 完整 leak 列表（待 Phase 2 完成）