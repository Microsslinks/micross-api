# 01 · 术语一致性地图（task-13 phase 1）

> 创建：2026-09-16
> 数据来源：`web/src/i18n/locales/{fr,ru,ja,vi}.json`
> 工具：`ripgrep` 模式匹配 + 人工归类
> source of truth：`web/src/i18n/locales/zh.json`

## 摘要

| 概念 | zh 价值 | fr | ru | ja | vi | 严重程度 |
|---|---|---|---|---|---|---|
| Channel | 上游供应商 | 新词 2 + 老词 30 | 同 | 同 | 同 | 🔴 严重（每语言 30 处混用）|
| Vendor | 厂商 | 自身 2 词混用 + 2 leak | 同 | 一致（ベンダー）| 自身 2 词混用 | 🟡 中（fr/vi 各 4 处不一致）|
| Commission | 分成比例 | **全 leak** | **全 leak** | **全 leak** | **全 leak** | 🔴 严重（4 lang × 2 key = 8 处）|
| Dealer | 经销商 | 1 译 + 6 leak | 1 译 + 6 leak | 1 译 + 6 leak | 1 译 + 6 leak | 🔴 严重（每语言 6 处）|
| Discount Plan | 折扣方案 | **全 leak** | **全 leak** | **全 leak** | **全 leak** | 🔴 严重（4 lang × 2 key = 8 处）|
| Wallet | 钱包 | ✅ 一致 | ✅ 一致 | ✅ 一致 | ✅ 一致 | ✅ 干净 |
| Quota | 余额 | 部分 leak | 部分 leak | 部分 leak | 部分 leak | 🟡 中（每语言 1-2 处）|
| Invitation | 邀请 | ✅ 翻得 OK | ✅ 翻得 OK | ✅ 翻得 OK | ✅ 翻得 OK | ✅ 干净 |

**总计待改**：约 **80-120 处**（每语言 20-30 处），其中 50-60 处是**完全未翻译的占位符 leak**。

---

## 1 · Channel 概念（最严重）

### fr.json（line 1061-1092）

| key | 当前 value | 分类 |
|---|---|---|
| Channel | Fournisseur amont | ✅ 新词 |
| Channel {{name}} | Canal {{name}} | ❌ 老词 |
| Channel {{name}} model {{model}} | Canal {{name}} modèle {{model}} | ❌ 老词 |
| Channel Affinity | Affinité fournisseur amont | ✅ 新词 |
| Channel affinity reuses ... | L'affinité de canal réutilise ... | ❌ 老词 + 英文术语 |
| Channel Affinity: Upstream Cache Hit | Affinité de canal : hit de cache en amont | ❌ 老词 |
| Channel consistency repaired: ... | Cohérence des canaux réparée : ... | ❌ 老词（复数）|
| Channel copied successfully | Canal copié avec succès | ❌ 老词 |
| Channel created successfully | Canal créé avec succès | ❌ 老词 |
| Channel deleted successfully | Canal supprimé avec succès | ❌ 老词 |
| Channel disabled successfully | Canal désactivé avec succès | ❌ 老词 |
| Channel enabled successfully | Canal activé avec succès | ❌ 老词 |
| Channel Extra Settings | Paramètres supplémentaires du canal | ❌ 老词 |
| Channel health checks | Contrôles de santé des canaux | ❌ 老词（复数）|
| Channel ID | ID du Canal | ❌ 老词 |
| Channel ID is required | L'ID du canal est requis | ❌ 老词 |
| Channel key | Clé du canal | ❌ 老词 |
| Channel key unlocked | Clé de canal déverrouillée | ❌ 老词 |
| Channel Management | Gestion des canaux | ❌ 老词（复数）|
| Channel models | Modèles de canaux | ❌ 老词（复数）|
| Channel name is required | Le nom du canal est requis | ❌ 老词 |
| Channel test completed | Test du canal terminé | ❌ 老词 |
| Channel test concurrency | Parallélisme des tests de canaux | ❌ 老词（复数）|
| Channel test concurrency must be 1-32 | Le parallélisme des tests ... | ❌ 老词 |
| Channel test mode | Mode de test des canaux | ❌ 老词 |
| Channel type is required | Le type de canal est requis | ❌ 老词 |
| Channel updated successfully | Canal mis à jour avec succès | ❌ 老词 |
| Channel-specific settings (JSON format) | Paramètres spécifiques aux canaux ... | ❌ 老词 |
| Channel: | Canal : | ❌ 老词 |
| channel(s)? This action cannot be undone. | canal(aux) ? Cette action ne peut pas être annulée. | ❌ 老词 |
| Channels | Fournisseurs amont | ✅ 新词 |
| Channels deleted successfully | Canaux supprimés avec succès | ❌ 老词（复数）|

**fr 统计**：32 keys，2 ✅ + 30 ❌

### ru.json（line 1061-1092）

| key | 当前 value | 分类 |
|---|---|---|
| Channel | Поставщик | ✅ 新词 |
| Channel {{name}} | Канал {{name}} | ❌ 老词 |
| Channel Affinity | Привязка к поставщику | ✅ 新词（混："Affinity" 译"Привязка" 而非 "Аффинитет"）|
| Channel affinity reuses ... | Привязка к каналу повторно использует ... | ❌ 老词 |
| Channels | Поставщики | ✅ 新词（复数）|
| （其余 28 keys）| Канал / каналов / каналы | ❌ 老词 |

**ru 统计**：32 keys，3 ✅ + 29 ❌

### ja.json（line 1061-1092）

| key | 当前 value | 分类 |
|---|---|---|
| Channel | アップストリームプロバイダー | ✅ 新词 |
| Channels | アップストリームプロバイダー | ✅ 新词 |
| Channel Affinity | アップストリームプロバイダーアフィニティ | ✅ 新词 |
| Channel test concurrency | チャンネルテストの同時実行数 | ❌ 老词（注意：用片假名"チャンネル"而非"チャネル"，**自相矛盾**）|
| Channel test concurrency must be 1-32 | チャンネルテストの同時実行数は1～32にしてください | ❌ 老词（同样"チャンネル"）|
| （其余 28 keys）| チャネル... | ❌ 老词 |

**ja 统计**：32 keys，3 ✅ + 29 ❌
**ja 附加**：老词内部还不统一（"チャネル" vs "チャンネル"）

### vi.json（line 1061-1092）

| key | 当前 value | 分类 |
|---|---|---|
| Channel | Nhà cung cấp | ✅ 新词 |
| Channels | Nhà cung cấp | ✅ 新词（无复数变化，符合越南语）|
| Channel Affinity | Ưu tiên nhà cung cấp | ✅ 新词 |
| Channel affinity reuses ... | Ưu tiên kênh sẽ sử dụng lại kênh thành công gần nhất ... | ❌ 老词 |
| （其余 28 keys）| Kênh... | ❌ 老词 |

**vi 统计**：32 keys，3 ✅ + 29 ❌

### 其他含 "channel" 的英文未翻译 keys（4 lang 全部 leak）

- line 1449: `Channels without a cost ratio are skipped when serving customers on a discount.`
- line 1450: `Channels without a cost ratio are skipped when serving customers on a discount, and a cost ratio left unchanged for over {{days}} days may no longer match the upstream price.`

---

## 2 · Vendor 概念

### fr.json（line 294 / 5434-5438 / 5678）

| key | 当前 value | 分类 |
|---|---|---|
| Vendor-level rule | Vendor-level rule | ❌ 未翻译 |
| Vendor | Fabricant | ⚠️ 译"制造商"（应是 Fournisseur）|
| Vendor deleted successfully | Fournisseur supprimé avec succès | ✅ Fournisseur |
| Vendor Name * | Nom du fournisseur * | ✅ Fournisseur |
| Vendor: | Fournisseur : | ✅ Fournisseur |
| Vendors ranked by ... | Fournisseurs classés par ... | ✅ Fournisseur |
| Vendor list price · per 1M tokens | Vendor list price · per 1M tokens | ❌ 未翻译 |

**fr 统计**：7 keys，4 ✅ + 3 ❌（其中 1 个 fr 内部混用 Fabricant/Fournisseur）

### ru.json（同位置）

| key | 当前 value |
|---|---|
| Vendor-level rule | Vendor-level rule ❌ |
| Vendor | Производитель ⚠️（制造商）|
| Vendor deleted successfully | Поставщик успешно удалён ✅ |
| Vendor Name * | Название поставщика * ✅ |
| Vendor: | Поставщик: ✅ |
| Vendors ranked by ... | Поставщики, ранжированные ... ✅ |
| Vendor list price · per 1M tokens | Vendor list price · per 1M tokens ❌ |

**ru 统计**：7 keys，4 ✅ + 3 ❌

### ja.json

| key | 当前 value |
|---|---|
| Vendor-level rule | Vendor-level rule ❌ |
| Vendor | ベンダー ✅ |
| Vendor deleted successfully | ベンダーが正常に削除されました ✅ |
| Vendor Name * | ベンダー名 * ✅ |
| Vendor: | ベンダー: ✅ |
| Vendors ranked by ... | トークン総使用量で順位付けされたベンダー ✅ |
| Vendor list price · per 1M tokens | Vendor list price · per 1M tokens ❌ |

**ja 统计**：7 keys，5 ✅ + 2 ❌（ja 一致性最高）

### vi.json

| key | 当前 value |
|---|---|
| Vendor-level rule | Vendor-level rule ❌ |
| Vendor | Nhà sản xuất ⚠️（制造商）|
| Vendor deleted successfully | Đã xóa nhà cung cấp thành công ✅ |
| Vendor Name * | Tên nhà cung cấp * ✅ |
| Vendor: | Nhà cung cấp: ✅ |
| Vendors ranked by ... | Nhà cung cấp xếp hạng ... ✅ |
| Vendor list price · per 1M tokens | Vendor list price · per 1M tokens ❌ |

**vi 统计**：7 keys，4 ✅ + 3 ❌

---

## 3 · Commission 概念（全 leak）

### fr / ru / ja / vi（line 109-110，全部相同）

```
"Commission ratio": "Commission ratio",
"Commission ratio must be a number between 0 and 1": "Commission ratio must be a number between 0 and 1"
```

**zh 翻译**：「分成比例」/ 「分成比例必须是 0 到 1 之间的小数」

**4 语言建议改写**：
- fr：Ratio de commission / Le ratio de commission doit être un nombre entre 0 et 1
- ru：Доля комиссии / Доля комиссии должна быть числом от 0 до 1
- ja：コミッション比率 / コミッション比率は 0 から 1 の間の数値である必要があります
- vi：Tỷ lệ hoa hồng / Tỷ lệ hoa hồng phải là số từ 0 đến 1

---

## 4 · Dealer 概念

### 4 语言共同 pattern（line 54, 1590, 5717, 5753, 5755, 5766, 5790）

| key | fr | ru | ja | vi |
|---|---|---|---|---|
| Dealer price | Dealer price ❌ | Dealer price ❌ | Dealer price ❌ | Dealer price ❌ |
| Dealer Billing | Facturation revendeur ✅ | Счёт дилера ✅ | 販売店の請求 ✅ | Hoá đơn đại lý ✅ |
| Dealer status removed | Dealer status removed ❌ | Dealer status removed ❌ | Dealer status removed ❌ | Dealer status removed ❌ |
| Dealer settings saved | Dealer settings saved ❌ | Dealer settings saved ❌ | Dealer settings saved ❌ | Dealer settings saved ❌ |
| Dealer Settings | Dealer Settings ❌ | Dealer Settings ❌ | Dealer Settings ❌ | Dealer Settings ❌ |
| Dealer | Dealer ❌ | Dealer ❌ | Dealer ❌ | Dealer ❌ |
| Dealer wholesale price | Dealer wholesale price ❌ | Dealer wholesale price ❌ | Dealer wholesale price ❌ | Dealer wholesale price ❌ |

**每语言 6 处 leak + 1 处已翻译 = 28 处全语总计**

注意：`Dealer customer invitation message`（line 444）**4 语言都翻译**成完整多行邀请信——✅ 已翻译

---

## 5 · Discount Plan 概念（全 leak）

### 4 语言共同（line 129, 5743）

```
"Discount Plans": "Discount Plans",
"Discount plan": "Discount plan"
```

**zh 翻译**：「客户折扣方案」/ 「折扣方案」

**建议改写**：
- fr：Plans de remise / Plan de remise
- ru：Планы скидок / План скидок
- ja：割引プラン / 割引プラン
- vi：Các gói giảm giá / Gói giảm giá

---

## 6 · Wallet 概念（一致）

4 语言统一：
- fr：`Portefeuille` / `Portefeuille en priorité` / `Gestion du portefeuille` / `Portefeuille uniquement`
- ru：`Кошелёк` / `Кошелёк в приоритете` / `Управление кошельком` / `Только кошелёк`
- ja：`ウォレット` / `ウォレット優先` / `ウォレット管理` / `ウォレットのみ`
- vi：`Ví` / `Ưu tiên ví` / `Quản lý ví` / `Chỉ dùng ví`

✅ **无需改动**

---

## 7 · Quota 概念

4 语言基本翻译：
- fr：`Quota`（借用英文）/ 部分译 `Quota ajusté avec succès`
- ru：`Квота` / `Квота успешно изменена`
- ja：`クォータ` / `クォータの調整に成功しました`
- vi：`Hạn ngạch` / `Hạn mức`（混用：`Quota` 译 Hạn ngạch，`Quota ({{currency}})` 译 Hạn mức）

**leak key**（line 52，4 lang 全 leak）：
- `"Quota issued": "Quota issued"`

**vi 额外语序异常**（line 3992）：
- `"Quota given to users who invite others": "Limit for users inviting others"`（英文漏改+语序不对）

---

## 8 · Invitation 概念（基本一致）

4 语言翻译 OK：
- fr：`Code d'invitation` / `Quota d'invitation` / `Inviteur` / `Récompense de l'invité`
- ru：`Код приглашения` / `Квота приглашения` / `Пригласитель` / `Вознаграждение приглашённого`
- ja：`招待コード` / `招待クォータ` / `招待者` / `招待された方の報酬`
- vi：`Mã mời` / `Hạn mức mời` / `Người mời` / `Phần thưởng người được mời`

✅ **无需改动**

---

## 总结

| 改 / 不改 | 概念 |
|---|---|
| 🔴 必改（30+ 处）| Channel |
| 🔴 必改（10+ 处）| Commission, Discount Plan, Dealer |
| 🟡 改（3-5 处）| Vendor, Quota |
| ✅ 不改 | Wallet, Invitation |

**Phase 4 AI 改写候选范围**：~50-70 keys / 语言 × 4 = 200-280 处