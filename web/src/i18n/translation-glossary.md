# 翻译术语表

> 创建：2026-09-16（task-13 phase 1.3）
> 范围：核心 8 概念 × 7 语言
> source of truth：`web/src/i18n/locales/zh.json`（已人工审校）

## 8 核心概念术语对照

| key（en） | zh | zh-TW | en | fr | ru | ja | vi | 备注 |
|---|---|---|---|---|---|---|---|---|
| Channel / Channels | 上游供应商 | 上游供應商 | Channel(s) | Fournisseur amont | Поставщик | アップストリームプロバイダー | Nhà cung cấp | **新词**；旧词 canal/канал/チャネル/kênh 不再用 |
| 渠道（**废弃**） | 渠道 | 渠道 | channel (deprecated) | — | — | — | — | 改用"上游供应商" |
| Vendor | 厂商 | 廠商 | Vendor | Fabricant / Fournisseur | Производитель / Поставщик | ベンダー | Nhà sản xuất / Nhà cung cấp | fr/ru/vi 自身两种译法混用 |
| Commission ratio | 分成比例 | 分成比例 | Commission ratio | Commission ratio（**未翻译**）| Commission ratio（**未翻译**）| Commission ratio（**未翻译**）| Commission ratio（**未翻译**）| 4 语言 2 key 全 leak |
| Dealer | 经销商 | 經銷商 | Dealer | Revendeur / Dealer（6 处 leak）| Дилер / Dealer（6 处 leak）| 販売店 / Dealer（6 处 leak）| Đại lý / Dealer（6 处 leak）| 每语言 6 处完全未翻译 |
| Discount Plans | 客户折扣方案 | 客戶折扣方案 | Discount Plans | Discount Plans（**未翻译**）| Discount Plans（**未翻译**）| Discount Plans（**未翻译**）| Discount Plans（**未翻译**）| 4 语言 2 key 全 leak |
| Discount plan | 折扣方案 | 折扣方案 | Discount plan | Discount plan（**未翻译**）| Discount plan（**未翻译**）| Discount plan（**未翻译**）| Discount plan（**未翻译**）| 同上 |
| Wallet | 钱包 | 錢包 | Wallet | Portefeuille | Кошелёк | ウォレット | Ví | 4 语言一致 |
| Quota | 余额 | 餘額 | Quota | Quota | Квота | クォータ | Hạn ngạch | fr/ru/ja 借用英文；vi 用"Hạn ngạch"；2 处 leak |
| Invitation Code | 邀请码 | 邀請碼 | Invitation Code | Code d'invitation | Код приглашения | 招待コード | Mã mời | 4 语言一致 |

## 不在术语表但相邻的概念

| key | 说明 |
|---|---|
| Inviter / Invitee / Invites / Invitation Quota | 4 语言都已翻译，不在术语表问题清单 |
| Affiliate / Referral | **不作为独立 key 存在**（以 Invitation 系列为准）|
| Promoter / Patron | 不存在（不是本平台的概念）|

## 旧词 → 新词迁移说明（master-plan §四 业务口径）

| 旧词（不推荐）| 新词（推荐）| 旧词出现位置 |
|---|---|---|
| 渠道 / canal / канал / チャネル / kênh | 上游供应商 / Fournisseur amont / Поставщик / アップストリームプロバイダー / Nhà cung cấp | fr/ru/ja/vi 每文件 30 个 key（除 `Channel` / `Channels` 2 个 key 外）|
| 制造商 / Fabricant / Производитель / ベンダー / Nhà sản xuất | 厂商 / Fournisseur / Поставщик（部分）/ ベンダー / Nhà cung cấp | fr.json:5434 单独用 Fabricant；其他键用 Fournisseur |

## 维护说明

- 新增 key 时若涉及 8 概念之一，按本表填值
- Phase 6 改写时严格按本表（不一致的旧词应替换）
- 母语者最终签字（Phase 7）时可修订本表；修订记录加在 commit message