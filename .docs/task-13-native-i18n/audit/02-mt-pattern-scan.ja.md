# 02 · 机翻特征扫描 · JA（task-13 phase 2）

> 创建：2026-09-17
> 数据来源：`web/src/i18n/locales/ja.json` vs `en.json`
> 工具：`web/scripts/scan-mt-patterns.mjs`

## 摘要

| 指标 | 值 |
|---|---|
| 总键数 | 5927 |
| 完全未翻译（与 en 相同）| 594 |
| 至少命中一条机翻特征 | 324 |
| 句式分叉数 | 0 |

## 规则命中统计

| 规则 | 命中数 | 说明 |
|---|---|---|
| `borrowed-english` | 230 | value 保留了英文技术借词（upstream / cache / token 等） |
| `ja-kana-heavy` | 93 | 日文假名比例 ≥ 50%（疑似 Google Translate 输出） |
| `length-too-short` | 1 | value 长度 < en 原文 0.3×（典型：翻译过度精简） |

## 详细命中清单（每规则前 30）

### borrowed-english（230 处，展示前 30）

| key | en | ja |
|---|---|---|
| `API routing rules, all compatible` | API routing rules, all compatible | APIルーティングルールもすべて互換 |
| `. Please fix the JSON before saving.` | . Please fix the JSON before saving. | 。保存する前にJSONを修正してください。 |
| `Access Policy (JSON)` | Access Policy (JSON) | アクセスポリシー (JSON) |
| `Add API` | Add API | API追加 |
| `Add API Shortcut` | Add API Shortcut | API ショートカットを追加 |
| `Add OAuth Provider` | Add OAuth Provider | OAuthプロバイダーを追加 |
| `All API tokens` | All API tokens | すべての API キー |
| `Allow HTTP image requests` | Allow HTTP image requests | HTTP画像リクエストを許可 |
| `Amount options must be a JSON array` | Amount options must be a JSON array | 金額オプションは JSON 配列でなければなりません |
| `API Access` | API Access | API アクセス |
| `API Addresses` | API Addresses | APIアドレス |
| `API Base URL *` | API Base URL * | APIベースURL * |
| `API Endpoints` | API Endpoints | APIエンドポイント |
| `API Info` | API Info | API情報 |
| `API info saved successfully` | API info saved successfully | API情報が正常に保存されました |
| `API key` | API key | APIキー |
| `API Key` | API Key | APIキー |
| `API Key (Production)` | API Key (Production) | APIキー（本番） |
| `API Key (Sandbox)` | API Key (Sandbox) | APIキー（サンドボックス） |
| `API Key *` | API Key * | APIキー * |
| `API Key created successfully` | API Key created successfully | APIキーが正常に作成されました |
| `API Key deleted successfully` | API Key deleted successfully | APIキーが正常に削除されました |
| `API Key disabled successfully` | API Key disabled successfully | APIキーが正常に無効化されました |
| `API Key enabled successfully` | API Key enabled successfully | APIキーが正常に有効化されました |
| `API key from the provider` | API key from the provider | プロバイダからのAPIキー |
| `API key is required` | API key is required | APIキーが必要です |
| `API Key updated successfully` | API Key updated successfully | APIキーが正常に更新されました |
| `API Keys` | API Keys | APIキー |
| `API Private Key` | API Private Key | API 秘密鍵 |
| `API Requests` | API Requests | APIリクエスト |
| ... | （剩余 200 处略）| |

### ja-kana-heavy（93 处，展示前 30）

| key | en | ja |
|---|---|---|
| `2. Copy the application token` | 2. Copy the application token | 2. アプリケーショントークンをコピーします |
| `Account created! Please sign in` | Account created! Please sign in | アカウントが作成されました！ログインしてください |
| `Administer user accounts and roles.` | Administer user accounts and roles. | ユーザーアカウントとロールを管理します。 |
| `All upstream data is trusted` | All upstream data is trusted | すべてのアップストリームデータは信頼されています |
| `Allow users to enter promo codes` | Allow users to enter promo codes | ユーザーがプロモーションコードを入力できるようにする |
| `Allow users to log in with password` | Allow users to log in with password | ユーザーがパスワードでログインできるようにする |
| `Are you sure you want to enable all keys?` | Are you sure you want to enable all keys? | すべてのキーを有効にすることをよろしいですか？ |
| `Are you sure you want to sign out? You will need to sign in again to access your account.` | Are you sure you want to sign out? You will need to sign in again to access your | ログアウトしてもよろしいですか？アカウントにアクセスするには再度ログインする必要があります。 |
| `Automatically probe all channels in the background` | Automatically probe all channels in the background | バックグラウンドですべてのチャネルを自動的にプローブする |
| `Background job tracker for queued work.` | Background job tracker for queued work. | キューされた作業のためのバックグラウンドジョブトラッカー。 |
| `Bind an email address to your account.` | Bind an email address to your account. | アカウントにメールアドレスを紐付けます。 |
| `Blacklist (Block listed domains)` | Blacklist (Block listed domains) | ブラックリスト (ブロックされたドメイン) |
| `Channel Affinity` | Channel Affinity | アップストリームプロバイダーアフィニティ |
| `Channel Affinity: Upstream Cache Hit` | Channel Affinity: Upstream Cache Hit | チャネルアフィニティ：上流キャッシュヒット |
| `Character chat, storytelling, persona` | Character chat, storytelling, persona | キャラクター会話・ストーリーテリング・ペルソナ |
| `Choose where to fetch upstream metadata.` | Choose where to fetch upstream metadata. | アップストリームのメタデータをどこからフェッチするかを選択してください。 |
| `Cleared channel affinity cache` | Cleared channel affinity cache | チャネルアフィニティキャッシュをクリアしました |
| `Click any category to drill into its models, apps, and trends` | Click any category to drill into its models, apps, and trends | カテゴリをクリックすると、そのモデル・アプリ・トレンドを掘り下げて確認できます |
| `Concatenate channel system prompt with user&apos;s prompt` | Concatenate channel system prompt with user&apos;s prompt | チャネルのシステムプロンプトをユーザーのプロンプトと連結する |
| `Configure upstream providers and routing.` | Configure upstream providers and routing. | アップストリームプロバイダーとルーティングを設定。 |
| `Confirm cleanup of inactive disk cache?` | Confirm cleanup of inactive disk cache? | 非アクティブなディスクキャッシュをクリーンアップしますか？ |
| `Confirm clearing all channel affinity cache` | Confirm clearing all channel affinity cache | 全チャネルアフィニティキャッシュのクリアを確認 |
| `Copy source field to target field` | Copy source field to target field | ソースフィールドをターゲットフィールドにコピー |
| `Copy the key and paste it here` | Copy the key and paste it here | キーをコピーしてここに貼り付けてください |
| `Create or update system announcements for the dashboard` | Create or update system announcements for the dashboard | ダッシュボードのシステムアナウンスを作成または更新します |
| `Custom database driver detected.` | Custom database driver detected. | カスタムデータベースドライバーが検出されました。 |
| `Default API version for this channel` | Default API version for this channel | このチャネルのデフォルトのAPIバージョン |
| `Default system prompt for this channel` | Default system prompt for this channel | このチャネルのデフォルトのシステムプロンプト |
| `Define endpoint mappings for each provider.` | Define endpoint mappings for each provider. | 各プロバイダーごとにエンドポイントのマッピングを定義してください。 |
| `e.g. This request does not meet access policy` | e.g. This request does not meet access policy | 例：このリクエストはアクセスポリシーを満たしていません |
| ... | （剩余 63 处略）| |

### length-too-short（1 处，展示前 30）

| key | en | ja |
|---|---|---|
| `It seems like the page you're looking for` | It seems like the page you're looking for | お探しのページは |

