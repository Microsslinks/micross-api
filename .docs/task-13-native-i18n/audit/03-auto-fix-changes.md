# 03 · 程序化修复变更报告（task-13 phase 4）

> 生成：2026-09-16T16:10:28
> 模式：**APPLIED**

| 语言 | 修改 key 数 |
|---|---|
| fr | 201 |
| ru | 225 |
| ja | 222 |
| vi | 225 |
| **合计** | **873** |

## fr（201 处）

| key | 改前 | 改后 | 操作 |
|---|---|---|---|
| `Enter HTML code or a URL (e.g., https://example.com) to embed as iframe` | Saisissez du code HTML ou une URL (ex. https://example.com)  | Saisissez du code HTML ou une URL (ex. https://example. com) | fr-punct-space |
| `A trailing wildcard is supported, e.g. claude-*; the model must be billed to this customer.` | A trailing wildcard is supported, e.g. claude-*; the model m | A trailing wildcard is supported, e. g. claude-*; the model  | fr-punct-space |
| `Why this customer is special, e.g. the ticket number` | Why this customer is special, e.g. the ticket number | Why this customer is special, e. g. the ticket number | fr-punct-space |
| `(Override all channels' groups)` | (Remplacer les groupes de tous les canaux) | (Remplacer les groupes de tous les fournisseurs amont) | term:canaux -> fournisseurs amont |
| `(Override all channels' models)` | (Remplacer les modèles de tous les canaux) | (Remplacer les modèles de tous les fournisseurs amont) | term:canaux -> fournisseurs amont |
| `[{"ChatGPT":"https://chat.openai.com"},{"Lobe Chat":"https://chat-preview.lobehub.com/?settings={...}"}]` | [{"ChatGPT":"https://chat.openai.com"},{"Lobe Chat":"https:/ | [{"ChatGPT":"https://chat. openai. com"},{"Lobe Chat":"https | fr-punct-space |
| `{{count}} channel(s) deleted` | {{count}} canal(canaux) supprimé(s) | {{count}} canal(fournisseurs amont) supprimé(s) | term:canaux -> fournisseurs amont |
| `{{count}} channel(s) disabled` | {{count}} canal(canaux) désactivé(s) | {{count}} canal(fournisseurs amont) désactivé(s) | term:canaux -> fournisseurs amont |
| `{{count}} channel(s) enabled` | {{count}} canal(canaux) activé(s) | {{count}} canal(fournisseurs amont) activé(s) | term:canaux -> fournisseurs amont |
| `{{count}} channel(s) failed to disable` | {{count}} canal(canaux) n'ont pas pu être désactivé(s) | {{count}} canal(fournisseurs amont) n'ont pas pu être désact | term:canaux -> fournisseurs amont |
| `{{count}} channel(s) failed to enable` | {{count}} canal(canaux) n'ont pas pu être activé(s) | {{count}} canal(fournisseurs amont) n'ont pas pu être activé | term:canaux -> fournisseurs amont |
| `{{count}} disabled channel(s) deleted` | {{count}} canal(canaux) désactivé(s) supprimé(s) | {{count}} canal(fournisseurs amont) désactivé(s) supprimé(s) | term:canaux -> fournisseurs amont |
| `Actively check all channels` | Vérifier activement tous les canaux | Vérifier activement tous les fournisseurs amont | term:canaux -> fournisseurs amont |
| `Actively check auto-disable-enabled channels` | Vérifier activement les canaux avec désactivation automatiqu | Vérifier activement les fournisseurs amont avec désactivatio | term:canaux -> fournisseurs amont |
| `Add your API keys, set up channels and configure access permissions` | Ajoutez vos clés API, configurez les canaux et les permissio | Ajoutez vos clés API, configurez les fournisseurs amont et l | term:canaux -> fournisseurs amont |
| `Admin Channel Permissions` | Autorisations des canaux administrateur | Autorisations des fournisseurs amont administrateur | term:canaux -> fournisseurs amont |
| `Applied upstream model changes to {{count}} channels` | Modifications des modèles en amont appliquées à {{count}} ca | Modifications des modèles en amont appliquées à {{count}} fo | term:canaux -> fournisseurs amont |
| `Auto-disable-enabled channels only` | Canaux avec désactivation automatique uniquement | Fournisseurs amont avec désactivation automatique uniquement | term:canaux -> fournisseurs amont |
| `Auto-disable-enabled mode probes non-manually-disabled channels with auto-disable enabled.` | Ce mode sonde uniquement les canaux dont la désactivation au | Ce mode sonde uniquement les fournisseurs amont dont la désa | term:canaux -> fournisseurs amont |
| `Automatically disable channels exceeding this response time` | Désactiver automatiquement les canaux dépassant ce temps de  | Désactiver automatiquement les fournisseurs amont dépassant  | term:canaux -> fournisseurs amont |
| `Automatically disable channels when tests fail` | Désactiver automatiquement les canaux lorsque les tests écho | Désactiver automatiquement les fournisseurs amont lorsque le | term:canaux -> fournisseurs amont |
| `Automatically probe all channels in the background` | Sonder automatiquement tous les canaux en arrière-plan | Sonder automatiquement tous les fournisseurs amont en arrièr | term:canaux -> fournisseurs amont |
| `Available variables: {{provider}}, {{field}}, {{op}}, {{required}}, {{current}}, and paths such as {{current.roles}}.` | Variables disponibles : {{provider}}, {{field}}, {{op}}, {{r | Variables disponibles : {{provider}}, {{field}}, {{op}}, {{r | fr-punct-space |
| `Batch channel test` | Test groupé des canaux | Test groupé des fournisseurs amont | term:canaux -> fournisseurs amont |
| `Batch deleted {{count}} channels` | {{count}} canaux supprimés par lot | {{count}} fournisseurs amont supprimés par lot | term:canaux -> fournisseurs amont |
| `Batch detection complete: {{channels}} channels, {{add}} to add, {{remove}} to remove, {{fails}} failed` | Détection par lots terminée : {{channels}} canaux, {{add}} à | Détection par lots terminée : {{channels}} fournisseurs amon | term:canaux -> fournisseurs amont |
| `Batch edit all channels with this tag. Leave fields empty to keep current values.` | Modifiez par lot tous les canaux avec ce tag. Laissez les ch | Modifiez par lot tous les fournisseurs amont avec ce tag. La | term:canaux -> fournisseurs amont |
| `Batch set tag for {{count}} channels` | Étiquette définie par lot pour {{count}} canaux | Étiquette définie par lot pour {{count}} fournisseurs amont | term:canaux -> fournisseurs amont |
| `Batch upstream model updates applied: {{channels}} channels, {{added}} added, {{removed}} removed, {{fails}} failed` | Mises à jour par lot des modèles en amont appliquées : {{cha | Mises à jour par lot des modèles en amont appliquées : {{cha | term:canaux -> fournisseurs amont |
| `Block email aliases (e.g., user+alias@domain.com)` | Bloquer les alias d'e-mail (par exemple, utilisateur+alias@d | Bloquer les alias d'e-mail (par exemple, utilisateur+alias@d | fr-punct-space |
| `Bound Channels` | Canaux liés | Fournisseurs amont liés | term:canaux -> fournisseurs amont |
| `Bring channels back online after successful checks` | Remettre les canaux en ligne après des vérifications réussie | Remettre les fournisseurs amont en ligne après des vérificat | term:canaux -> fournisseurs amont |
| `Channel {{name}}` | Canal {{name}} | Fournisseur amont {{name}} | term:Channel (capitalized standalone) |
| `Channel {{name}} model {{model}}` | Canal {{name}} modèle {{model}} | Fournisseur amont {{name}} modèle {{model}} | term:Channel (capitalized standalone) |
| `Channel consistency repaired: {{success}} succeeded, {{fails}} failed` | Cohérence des canaux réparée : {{success}} réussie(s), {{fai | Cohérence des fournisseurs amont réparée : {{success}} réuss | term:canaux -> fournisseurs amont |
| `Channel copied successfully` | Canal copié avec succès | Fournisseur amont copié avec succès | term:Channel (capitalized standalone) |
| `Channel created successfully` | Canal créé avec succès | Fournisseur amont créé avec succès | term:Channel (capitalized standalone) |
| `Channel deleted successfully` | Canal supprimé avec succès | Fournisseur amont supprimé avec succès | term:Channel (capitalized standalone) |
| `Channel disabled successfully` | Canal désactivé avec succès | Fournisseur amont désactivé avec succès | term:Channel (capitalized standalone) |
| `Channel enabled successfully` | Canal activé avec succès | Fournisseur amont activé avec succès | term:Channel (capitalized standalone) |
| `Channel health checks` | Contrôles de santé des canaux | Contrôles de santé des fournisseurs amont | term:canaux -> fournisseurs amont |
| `Channel ID` | ID du Canal | ID du Fournisseur amont | term:Channel (capitalized standalone) |
| `Channel Management` | Gestion des canaux | Gestion des fournisseurs amont | term:canaux -> fournisseurs amont |
| `Channel models` | Modèles de canaux | Modèles de fournisseurs amont | term:canaux -> fournisseurs amont |
| `Channel test concurrency` | Parallélisme des tests de canaux | Parallélisme des tests de fournisseurs amont | term:canaux -> fournisseurs amont |
| `Channel test concurrency must be between 1 and 32` | Le parallélisme des tests de canaux doit être compris entre  | Le parallélisme des tests de fournisseurs amont doit être co | term:canaux -> fournisseurs amont |
| `Channel test mode` | Mode de test des canaux | Mode de test des fournisseurs amont | term:canaux -> fournisseurs amont |
| `Channel updated successfully` | Canal mis à jour avec succès | Fournisseur amont mis à jour avec succès | term:Channel (capitalized standalone) |
| `Channel-specific settings (JSON format)` | Paramètres spécifiques aux canaux (format JSON) | Paramètres spécifiques aux fournisseurs amont (format JSON) | term:canaux -> fournisseurs amont |
| `Channel:` | Canal : | Fournisseur amont : | term:Channel (capitalized standalone) |
| ... | （剩余 151 处略）| | |

## ru（225 处）

| key | 改前 | 改后 | 操作 |
|---|---|---|---|
| `(Override all channels' groups)` | (Переопределить группы всех каналов) | (Переопределить группы всех поставщиков) | term:канал* inflected -> поставщик* inflected |
| `(Override all channels' models)` | (Переопределить модели всех каналов) | (Переопределить модели всех поставщиков) | term:канал* inflected -> поставщик* inflected |
| `{{count}} channel(s) deleted` | Удалено {{count}} каналов | Удалено {{count}} поставщиков | term:канал* inflected -> поставщик* inflected |
| `{{count}} channel(s) disabled` | Отключено {{count}} каналов | Отключено {{count}} поставщиков | term:канал* inflected -> поставщик* inflected |
| `{{count}} channel(s) enabled` | Включено {{count}} каналов | Включено {{count}} поставщиков | term:канал* inflected -> поставщик* inflected |
| `{{count}} channel(s) failed to disable` | Не удалось отключить {{count}} каналов | Не удалось отключить {{count}} поставщиков | term:канал* inflected -> поставщик* inflected |
| `{{count}} channel(s) failed to enable` | Не удалось включить {{count}} каналов | Не удалось включить {{count}} поставщиков | term:канал* inflected -> поставщик* inflected |
| `{{count}} disabled channel(s) deleted` | Удалено {{count}} отключённых каналов | Удалено {{count}} отключённых поставщиков | term:канал* inflected -> поставщик* inflected |
| `Actively check all channels` | Активно проверять все каналы | Активно проверять все поставщики | term:канал* inflected -> поставщик* inflected |
| `Actively check auto-disable-enabled channels` | Активно проверять каналы с автоотключением | Активно проверять поставщики с автоотключением | term:канал* inflected -> поставщик* inflected |
| `Add a new channel by providing the necessary information.` | Добавьте новый канал, предоставив необходимую информацию. | Добавьте новый поставщик, предоставив необходимую информацию | term:канал* inflected -> поставщик* inflected |
| `Add your API keys, set up channels and configure access permissions` | Добавьте ваши API-ключи, настройте каналы и права доступа | Добавьте ваши API-ключи, настройте поставщики и права доступ | term:канал* inflected -> поставщик* inflected |
| `Admin Channel Permissions` | Права администратора для каналов | Права администратора для поставщиков | term:канал* inflected -> поставщик* inflected |
| `Append to channel` | Добавить в канал | Добавить в поставщик | term:канал* inflected -> поставщик* inflected |
| `Applied upstream model changes to {{count}} channels` | Изменения вышестоящих моделей применены к {{count}} каналам | Изменения вышестоящих моделей применены к {{count}} поставщи | term:канал* inflected -> поставщик* inflected |
| `Applied upstream model changes to channel (ID: {{id}})` | Изменения вышестоящих моделей применены к каналу (ID: {{id}} | Изменения вышестоящих моделей применены к поставщику (ID: {{ | term:канал* inflected -> поставщик* inflected |
| `Are you sure you want to delete channel "{{name}}"? This action cannot be undone.` | Вы уверены, что хотите удалить канал "{{name}}"? Это действи | Вы уверены, что хотите удалить поставщик "{{name}}"? Это дей | term:канал* inflected -> поставщик* inflected |
| `Auto-disable-enabled channels only` | Только каналы с автовыключением | Только поставщики с автовыключением | term:канал* inflected -> поставщик* inflected |
| `Auto-disable-enabled mode probes non-manually-disabled channels with auto-disable enabled.` | В этом режиме проверяются только каналы с включённым автомат | В этом режиме проверяются только поставщики с включённым авт | term:канал* inflected -> поставщик* inflected |
| `Automatically disable channel on repeated failures` | Автоматически отключать канал при повторных неудачах | Автоматически отключать поставщик при повторных неудачах | term:канал* inflected -> поставщик* inflected |
| `Automatically disable channels exceeding this response time` | Автоматически отключать каналы, превышающие это время ответа | Автоматически отключать поставщики, превышающие это время от | term:канал* inflected -> поставщик* inflected |
| `Automatically disable channels when tests fail` | Автоматически отключать каналы при сбое тестов | Автоматически отключать поставщики при сбое тестов | term:канал* inflected -> поставщик* inflected |
| `Automatically probe all channels in the background` | Автоматически проверять все каналы в фоновом режиме | Автоматически проверять все поставщики в фоновом режиме | term:канал* inflected -> поставщик* inflected |
| `Base URL is required for this channel type` | Для этого типа канала требуется Base URL | Для этого типа поставщика требуется Base URL | term:канал* inflected -> поставщик* inflected |
| `Batch channel test` | Пакетное тестирование каналов | Пакетное тестирование поставщиков | term:канал* inflected -> поставщик* inflected |
| `Batch deleted {{count}} channels` | Пакетно удалено каналов: {{count}} | Пакетно удалено поставщиков: {{count}} | term:канал* inflected -> поставщик* inflected |
| `Batch detection complete: {{channels}} channels, {{add}} to add, {{remove}} to remove, {{fails}} failed` | Пакетное обнаружение завершено: {{channels}} каналов, {{add} | Пакетное обнаружение завершено: {{channels}} поставщиков, {{ | term:канал* inflected -> поставщик* inflected |
| `Batch edit all channels with this tag. Leave fields empty to keep current values.` | Пакетно редактировать все каналы с этим тегом. Оставьте поля | Пакетно редактировать все поставщики с этим тегом. Оставьте  | term:канал* inflected -> поставщик* inflected |
| `Batch set tag for {{count}} channels` | Тег пакетно задан для {{count}} каналов | Тег пакетно задан для {{count}} поставщиков | term:канал* inflected -> поставщик* inflected |
| `Batch upstream model updates applied: {{channels}} channels, {{added}} added, {{removed}} removed, {{fails}} failed` | Пакетное обновление моделей: {{channels}} каналов, {{added}} | Пакетное обновление моделей: {{channels}} поставщиков, {{add | term:канал* inflected -> поставщик* inflected |
| `Bound Channels` | Привязанные каналы | Привязанные поставщики | term:канал* inflected -> поставщик* inflected |
| `Bring channels back online after successful checks` | Вернуть каналы в онлайн после успешных проверок | Вернуть поставщики в онлайн после успешных проверок | term:канал* inflected -> поставщик* inflected |
| `Channel {{name}}` | Канал {{name}} | Поставщик {{name}} | term:канал* inflected -> поставщик* inflected |
| `Channel {{name}} model {{model}}` | Канал {{name}}, модель {{model}} | Поставщик {{name}}, модель {{model}} | term:канал* inflected -> поставщик* inflected |
| `Channel affinity reuses the last successful channel based on keys extracted from the request context or JSON body.` | Привязка к каналу повторно использует последний успешный кан | Привязка к поставщику повторно использует последний успешный | term:канал* inflected -> поставщик* inflected |
| `Channel Affinity: Upstream Cache Hit` | Привязка к каналу: попадание в кэш upstream | Привязка к поставщику: попадание в кэш upstream | term:канал* inflected -> поставщик* inflected |
| `Channel consistency repaired: {{success}} succeeded, {{fails}} failed` | Согласованность каналов восстановлена: успешно {{success}},  | Согласованность поставщиков восстановлена: успешно {{success | term:канал* inflected -> поставщик* inflected |
| `Channel copied successfully` | Канал успешно скопирован | Поставщик успешно скопирован | term:канал* inflected -> поставщик* inflected |
| `Channel created successfully` | Канал успешно создан | Поставщик успешно создан | term:канал* inflected -> поставщик* inflected |
| `Channel deleted successfully` | Канал успешно удалён | Поставщик успешно удалён | term:канал* inflected -> поставщик* inflected |
| `Channel disabled successfully` | Канал успешно отключён | Поставщик успешно отключён | term:канал* inflected -> поставщик* inflected |
| `Channel enabled successfully` | Канал успешно включён | Поставщик успешно включён | term:канал* inflected -> поставщик* inflected |
| `Channel Extra Settings` | Дополнительные настройки канала | Дополнительные настройки поставщика | term:канал* inflected -> поставщик* inflected |
| `Channel health checks` | Проверки состояния каналов | Проверки состояния поставщиков | term:канал* inflected -> поставщик* inflected |
| `Channel ID` | ID канала | ID поставщика | term:канал* inflected -> поставщик* inflected |
| `Channel ID is required` | Требуется ID канала | Требуется ID поставщика | term:канал* inflected -> поставщик* inflected |
| `Channel key` | Ключ канала | Ключ поставщика | term:канал* inflected -> поставщик* inflected |
| `Channel key unlocked` | Ключ канала разблокирован | Ключ поставщика разблокирован | term:канал* inflected -> поставщик* inflected |
| `Channel Management` | Управление каналами | Управление поставщиками | term:канал* inflected -> поставщик* inflected |
| `Channel models` | Модели каналов | Модели поставщиков | term:канал* inflected -> поставщик* inflected |
| ... | （剩余 175 处略）| | |

## ja（222 处）

| key | 改前 | 改后 | 操作 |
|---|---|---|---|
| `(Override all channels' groups)` | （全チャネルのグループを上書き） | （全アップストリームプロバイダーのグループを上書き） | term:チャネル -> アップストリームプロバイダー |
| `(Override all channels' models)` | （全チャネルのモデルを上書き） | （全アップストリームプロバイダーのモデルを上書き） | term:チャネル -> アップストリームプロバイダー |
| `{{count}} channel(s) deleted` | {{count}} 個のチャネルを削除しました | {{count}} 個のアップストリームプロバイダーを削除しました | term:チャネル -> アップストリームプロバイダー |
| `{{count}} channel(s) disabled` | {{count}} 個のチャネルを無効にしました | {{count}} 個のアップストリームプロバイダーを無効にしました | term:チャネル -> アップストリームプロバイダー |
| `{{count}} channel(s) enabled` | {{count}} 個のチャネルを有効にしました | {{count}} 個のアップストリームプロバイダーを有効にしました | term:チャネル -> アップストリームプロバイダー |
| `{{count}} channel(s) failed to disable` | {{count}} 個のチャネルの無効化に失敗しました | {{count}} 個のアップストリームプロバイダーの無効化に失敗しました | term:チャネル -> アップストリームプロバイダー |
| `{{count}} channel(s) failed to enable` | {{count}} 個のチャネルの有効化に失敗しました | {{count}} 個のアップストリームプロバイダーの有効化に失敗しました | term:チャネル -> アップストリームプロバイダー |
| `{{count}} disabled channel(s) deleted` | {{count}} 個の無効チャネルを削除しました | {{count}} 個の無効アップストリームプロバイダーを削除しました | term:チャネル -> アップストリームプロバイダー |
| `Actively check all channels` | すべてのチャネルを定期チェック | すべてのアップストリームプロバイダーを定期チェック | term:チャネル -> アップストリームプロバイダー |
| `Actively check auto-disable-enabled channels` | 自動無効化が有効なチャネルを定期チェック | 自動無効化が有効なアップストリームプロバイダーを定期チェック | term:チャネル -> アップストリームプロバイダー |
| `Add a new channel by providing the necessary information.` | 必要な情報を提供して新しいチャネルを追加。 | 必要な情報を提供して新しいアップストリームプロバイダーを追加。 | term:チャネル -> アップストリームプロバイダー |
| `Add your API keys, set up channels and configure access permissions` | APIキーを追加し、チャネルを設定してアクセス権限を構成します | APIキーを追加し、アップストリームプロバイダーを設定してアクセス権限を構成します | term:チャネル -> アップストリームプロバイダー |
| `Admin Channel Permissions` | 管理者のチャネル権限 | 管理者のアップストリームプロバイダー権限 | term:チャネル -> アップストリームプロバイダー |
| `Append to channel` | チャネルに追加 | アップストリームプロバイダーに追加 | term:チャネル -> アップストリームプロバイダー |
| `Applied upstream model changes to {{count}} channels` | {{count}} 件のチャネルに上流モデルの変更を適用しました | {{count}} 件のアップストリームプロバイダーに上流モデルの変更を適用しました | term:チャネル -> アップストリームプロバイダー |
| `Applied upstream model changes to channel (ID: {{id}})` | チャネル（ID: {{id}}）に上流モデルの変更を適用しました | アップストリームプロバイダー（ID: {{id}}）に上流モデルの変更を適用しました | term:チャネル -> アップストリームプロバイダー |
| `Are you sure you want to delete channel "{{name}}"? This action cannot be undone.` | チャネル "{{name}}" を削除してもよろしいですか？この操作は元に戻せません。 | アップストリームプロバイダー "{{name}}" を削除してもよろしいですか？この操作は元に戻せません。 | term:チャネル -> アップストリームプロバイダー |
| `Auto-disable-enabled channels only` | 自動無効化が有効なチャネルのみ | 自動無効化が有効なアップストリームプロバイダーのみ | term:チャネル -> アップストリームプロバイダー |
| `Auto-disable-enabled mode probes non-manually-disabled channels with auto-disable enabled.` | このモードでは、自動無効化が有効で、手動で無効化されていないチャネルのみを検査します。 | このモードでは、自動無効化が有効で、手動で無効化されていないアップストリームプロバイダーのみを検査します。 | term:チャネル -> アップストリームプロバイダー |
| `Automatically disable channel on repeated failures` | 繰り返しの失敗でチャネルを自動的に無効にする | 繰り返しの失敗でアップストリームプロバイダーを自動的に無効にする | term:チャネル -> アップストリームプロバイダー |
| `Automatically disable channels exceeding this response time` | この応答時間を超えるチャネルを自動的に無効にする | この応答時間を超えるアップストリームプロバイダーを自動的に無効にする | term:チャネル -> アップストリームプロバイダー |
| `Automatically disable channels when tests fail` | テストが失敗したときにチャネルを自動的に無効にする | テストが失敗したときにアップストリームプロバイダーを自動的に無効にする | term:チャネル -> アップストリームプロバイダー |
| `Automatically probe all channels in the background` | バックグラウンドですべてのチャネルを自動的にプローブする | バックグラウンドですべてのアップストリームプロバイダーを自動的にプローブする | term:チャネル -> アップストリームプロバイダー |
| `Base URL is required for this channel type` | このチャネルタイプには Base URL が必要です | このアップストリームプロバイダータイプには Base URL が必要です | term:チャネル -> アップストリームプロバイダー |
| `Batch channel test` | チャネル一括テスト | アップストリームプロバイダー一括テスト | term:チャネル -> アップストリームプロバイダー |
| `Batch deleted {{count}} channels` | {{count}} 件のチャネルを一括削除しました | {{count}} 件のアップストリームプロバイダーを一括削除しました | term:チャネル -> アップストリームプロバイダー |
| `Batch detection complete: {{channels}} channels, {{add}} to add, {{remove}} to remove, {{fails}} failed` | 一括検出完了：{{channels}} チャネル、{{add}} 個追加、{{remove}} 個削除、{{fails} | 一括検出完了：{{channels}} アップストリームプロバイダー、{{add}} 個追加、{{remove}} 個削 | term:チャネル -> アップストリームプロバイダー |
| `Batch edit all channels with this tag. Leave fields empty to keep current values.` | このタグを持つすべてのチャネルを一括編集します。現在の値を維持するには、フィールドを空のままにしてください。 | このタグを持つすべてのアップストリームプロバイダーを一括編集します。現在の値を維持するには、フィールドを空のままにしてく | term:チャネル -> アップストリームプロバイダー |
| `Batch set tag for {{count}} channels` | {{count}} 件のチャネルにタグを一括設定しました | {{count}} 件のアップストリームプロバイダーにタグを一括設定しました | term:チャネル -> アップストリームプロバイダー |
| `Batch upstream model updates applied: {{channels}} channels, {{added}} added, {{removed}} removed, {{fails}} failed` | 一括上流モデル更新を処理しました：{{channels}} チャネル、{{added}} 個追加、{{removed}} | 一括上流モデル更新を処理しました：{{channels}} アップストリームプロバイダー、{{added}} 個追加、{ | term:チャネル -> アップストリームプロバイダー |
| `Bound Channels` | バインドされたチャネル | バインドされたアップストリームプロバイダー | term:チャネル -> アップストリームプロバイダー |
| `Bring channels back online after successful checks` | チェックが成功した後、チャネルをオンラインに戻します | チェックが成功した後、アップストリームプロバイダーをオンラインに戻します | term:チャネル -> アップストリームプロバイダー |
| `Channel {{name}}` | チャネル {{name}} | アップストリームプロバイダー {{name}} | term:チャネル -> アップストリームプロバイダー |
| `Channel {{name}} model {{model}}` | チャネル {{name}} モデル {{model}} | アップストリームプロバイダー {{name}} モデル {{model}} | term:チャネル -> アップストリームプロバイダー |
| `Channel affinity reuses the last successful channel based on keys extracted from the request context or JSON body.` | チャネルアフィニティは、リクエストコンテキストまたは JSON Body から抽出したキーに基づいて、前回成功したチャネ | アップストリームプロバイダーアフィニティは、リクエストコンテキストまたは JSON Body から抽出したキーに基づいて | term:チャネル -> アップストリームプロバイダー |
| `Channel Affinity: Upstream Cache Hit` | チャネルアフィニティ：上流キャッシュヒット | アップストリームプロバイダーアフィニティ：上流キャッシュヒット | term:チャネル -> アップストリームプロバイダー |
| `Channel consistency repaired: {{success}} succeeded, {{fails}} failed` | チャネル整合性を修復しました：成功 {{success}} 件、失敗 {{fails}} 件 | アップストリームプロバイダー整合性を修復しました：成功 {{success}} 件、失敗 {{fails}} 件 | term:チャネル -> アップストリームプロバイダー |
| `Channel copied successfully` | チャネルが正常にコピーされました | アップストリームプロバイダーが正常にコピーされました | term:チャネル -> アップストリームプロバイダー |
| `Channel created successfully` | チャネルが正常に作成されました | アップストリームプロバイダーが正常に作成されました | term:チャネル -> アップストリームプロバイダー |
| `Channel deleted successfully` | チャネルが正常に削除されました | アップストリームプロバイダーが正常に削除されました | term:チャネル -> アップストリームプロバイダー |
| `Channel disabled successfully` | チャネルが正常に無効化されました | アップストリームプロバイダーが正常に無効化されました | term:チャネル -> アップストリームプロバイダー |
| `Channel enabled successfully` | チャネルが正常に有効化されました | アップストリームプロバイダーが正常に有効化されました | term:チャネル -> アップストリームプロバイダー |
| `Channel Extra Settings` | チャネル詳細設定 | アップストリームプロバイダー詳細設定 | term:チャネル -> アップストリームプロバイダー |
| `Channel health checks` | チャネルヘルスチェック | アップストリームプロバイダーヘルスチェック | term:チャネル -> アップストリームプロバイダー |
| `Channel ID` | チャネルID | アップストリームプロバイダーID | term:チャネル -> アップストリームプロバイダー |
| `Channel ID is required` | チャネル ID が必要です | アップストリームプロバイダー ID が必要です | term:チャネル -> アップストリームプロバイダー |
| `Channel key` | チャネルキー | アップストリームプロバイダーキー | term:チャネル -> アップストリームプロバイダー |
| `Channel key unlocked` | チャネルキーが解除されました | アップストリームプロバイダーキーが解除されました | term:チャネル -> アップストリームプロバイダー |
| `Channel Management` | チャネル管理 | アップストリームプロバイダー管理 | term:チャネル -> アップストリームプロバイダー |
| `Channel models` | チャネルモデル | アップストリームプロバイダーモデル | term:チャネル -> アップストリームプロバイダー |
| ... | （剩余 172 处略）| | |

## vi（225 处）

| key | 改前 | 改后 | 操作 |
|---|---|---|---|
| `(Override all channels' groups)` | (Ghi đè các nhóm của tất cả các kênh) | (Ghi đè các nhóm của tất cả các nhà cung cấp) | term:kênh lowercase |
| `(Override all channels' models)` | (Ghi đè các mô hình của tất cả các kênh) | (Ghi đè các mô hình của tất cả các nhà cung cấp) | term:kênh lowercase |
| `{{count}} channel(s) deleted` | Đã xóa {{count}} kênh | Đã xóa {{count}} nhà cung cấp | term:kênh lowercase |
| `{{count}} channel(s) disabled` | Đã tắt {{count}} kênh | Đã tắt {{count}} nhà cung cấp | term:kênh lowercase |
| `{{count}} channel(s) enabled` | Đã bật {{count}} kênh | Đã bật {{count}} nhà cung cấp | term:kênh lowercase |
| `{{count}} channel(s) failed to disable` | {{count}} kênh không thể tắt | {{count}} nhà cung cấp không thể tắt | term:kênh lowercase |
| `{{count}} channel(s) failed to enable` | {{count}} kênh không thể bật | {{count}} nhà cung cấp không thể bật | term:kênh lowercase |
| `{{count}} disabled channel(s) deleted` | Đã xóa {{count}} kênh đã tắt | Đã xóa {{count}} nhà cung cấp đã tắt | term:kênh lowercase |
| `Actively check all channels` | Chủ động kiểm tra tất cả kênh | Chủ động kiểm tra tất cả nhà cung cấp | term:kênh lowercase |
| `Actively check auto-disable-enabled channels` | Chủ động kiểm tra kênh đã bật tự động vô hiệu hóa | Chủ động kiểm tra nhà cung cấp đã bật tự động vô hiệu hóa | term:kênh lowercase |
| `Add a new channel by providing the necessary information.` | Thêm kênh mới bằng cách cung cấp thông tin cần thiết. | Thêm nhà cung cấp mới bằng cách cung cấp thông tin cần thiết | term:kênh lowercase |
| `Add your API keys, set up channels and configure access permissions` | Thêm khóa API, thiết lập kênh và cấu hình quyền truy cập | Thêm khóa API, thiết lập nhà cung cấp và cấu hình quyền truy | term:kênh lowercase |
| `Admin Channel Permissions` | Quyền kênh của quản trị viên | Quyền nhà cung cấp của quản trị viên | term:kênh lowercase |
| `Append to channel` | Nối vào kênh | Nối vào nhà cung cấp | term:kênh lowercase |
| `Applied upstream model changes to {{count}} channels` | Đã áp dụng thay đổi mô hình thượng nguồn cho {{count}} kênh | Đã áp dụng thay đổi mô hình thượng nguồn cho {{count}} nhà c | term:kênh lowercase |
| `Applied upstream model changes to channel (ID: {{id}})` | Đã áp dụng thay đổi mô hình thượng nguồn cho kênh (ID: {{id} | Đã áp dụng thay đổi mô hình thượng nguồn cho nhà cung cấp (I | term:kênh lowercase |
| `Are you sure you want to delete channel "{{name}}"? This action cannot be undone.` | Bạn có chắc muốn xóa kênh "{{name}}" không? Hành động này kh | Bạn có chắc muốn xóa nhà cung cấp "{{name}}" không? Hành độn | term:kênh lowercase |
| `Auto-disable-enabled channels only` | Chỉ kênh đã bật tự động vô hiệu hóa | Chỉ nhà cung cấp đã bật tự động vô hiệu hóa | term:kênh lowercase |
| `Auto-disable-enabled mode probes non-manually-disabled channels with auto-disable enabled.` | Chế độ này chỉ kiểm tra các kênh đã bật tự động vô hiệu hóa  | Chế độ này chỉ kiểm tra các nhà cung cấp đã bật tự động vô h | term:kênh lowercase |
| `Automatically disable channel on repeated failures` | Tự động vô hiệu hóa kênh khi xảy ra lỗi lặp lại | Tự động vô hiệu hóa nhà cung cấp khi xảy ra lỗi lặp lại | term:kênh lowercase |
| `Automatically disable channels exceeding this response time` | Tự động vô hiệu hóa các kênh vượt quá thời gian phản hồi này | Tự động vô hiệu hóa các nhà cung cấp vượt quá thời gian phản | term:kênh lowercase |
| `Automatically disable channels when tests fail` | Tự động vô hiệu hóa các kênh khi kiểm thử thất bại | Tự động vô hiệu hóa các nhà cung cấp khi kiểm thử thất bại | term:kênh lowercase |
| `Automatically probe all channels in the background` | Tự động dò tất cả các kênh trong nền | Tự động dò tất cả các nhà cung cấp trong nền | term:kênh lowercase |
| `Base URL is required for this channel type` | Loại kênh này yêu cầu Base URL | Loại nhà cung cấp này yêu cầu Base URL | term:kênh lowercase |
| `Batch channel test` | Kiểm tra kênh hàng loạt | Kiểm tra nhà cung cấp hàng loạt | term:kênh lowercase |
| `Batch deleted {{count}} channels` | Đã xóa hàng loạt {{count}} kênh | Đã xóa hàng loạt {{count}} nhà cung cấp | term:kênh lowercase |
| `Batch detection complete: {{channels}} channels, {{add}} to add, {{remove}} to remove, {{fails}} failed` | Phát hiện hàng loạt hoàn tất: {{channels}} kênh, {{add}} để  | Phát hiện hàng loạt hoàn tất: {{channels}} nhà cung cấp, {{a | term:kênh lowercase |
| `Batch edit all channels with this tag. Leave fields empty to keep current values.` | Chỉnh sửa hàng loạt tất cả các kênh có gắn thẻ này. Để trống | Chỉnh sửa hàng loạt tất cả các nhà cung cấp có gắn thẻ này.  | term:kênh lowercase |
| `Batch set tag for {{count}} channels` | Đã đặt thẻ hàng loạt cho {{count}} kênh | Đã đặt thẻ hàng loạt cho {{count}} nhà cung cấp | term:kênh lowercase |
| `Batch upstream model updates applied: {{channels}} channels, {{added}} added, {{removed}} removed, {{fails}} failed` | Đã áp dụng cập nhật hàng loạt mô hình upstream: {{channels}} | Đã áp dụng cập nhật hàng loạt mô hình upstream: {{channels}} | term:kênh lowercase |
| `Bound Channels` | Các kênh ràng buộc | Các nhà cung cấp ràng buộc | term:kênh lowercase |
| `Bring channels back online after successful checks` | Khôi phục các kênh trực tuyến sau khi kiểm tra thành công | Khôi phục các nhà cung cấp trực tuyến sau khi kiểm tra thành | term:kênh lowercase |
| `Channel {{name}}` | Kênh {{name}} | Nhà cung cấp {{name}} | term:Kênh standalone |
| `Channel {{name}} model {{model}}` | Kênh {{name}} mô hình {{model}} | Nhà cung cấp {{name}} mô hình {{model}} | term:Kênh standalone |
| `Channel affinity reuses the last successful channel based on keys extracted from the request context or JSON body.` | Ưu tiên kênh sẽ sử dụng lại kênh thành công gần nhất dựa trê | Ưu tiên nhà cung cấp sẽ sử dụng lại nhà cung cấp thành công  | term:kênh lowercase |
| `Channel Affinity: Upstream Cache Hit` | Ưu tiên kênh: Cache hit từ upstream | Ưu tiên nhà cung cấp: Cache hit từ upstream | term:kênh lowercase |
| `Channel consistency repaired: {{success}} succeeded, {{fails}} failed` | Đã sửa tính nhất quán kênh: {{success}} thành công, {{fails} | Đã sửa tính nhất quán nhà cung cấp: {{success}} thành công,  | term:kênh lowercase |
| `Channel copied successfully` | Sao chép kênh thành công | Sao chép nhà cung cấp thành công | term:kênh lowercase |
| `Channel created successfully` | Tạo kênh thành công | Tạo nhà cung cấp thành công | term:kênh lowercase |
| `Channel deleted successfully` | Xóa kênh thành công | Xóa nhà cung cấp thành công | term:kênh lowercase |
| `Channel disabled successfully` | Kênh đã bị vô hiệu hóa thành công | Nhà cung cấp đã bị vô hiệu hóa thành công | term:Kênh standalone |
| `Channel enabled successfully` | Kênh đã được bật thành công | Nhà cung cấp đã được bật thành công | term:Kênh standalone |
| `Channel Extra Settings` | Cài đặt thêm kênh | Cài đặt thêm nhà cung cấp | term:kênh lowercase |
| `Channel health checks` | Kiểm tra tình trạng kênh | Kiểm tra tình trạng nhà cung cấp | term:kênh lowercase |
| `Channel ID` | Mã kênh | Mã nhà cung cấp | term:kênh lowercase |
| `Channel ID is required` | Cần có ID kênh | Cần có ID nhà cung cấp | term:kênh lowercase |
| `Channel key` | Khóa kênh | Khóa nhà cung cấp | term:kênh lowercase |
| `Channel key unlocked` | Khóa kênh đã được mở khóa | Khóa nhà cung cấp đã được mở khóa | term:kênh lowercase |
| `Channel Management` | Quản lý kênh | Quản lý nhà cung cấp | term:kênh lowercase |
| `Channel models` | Mô hình kênh | Mô hình nhà cung cấp | term:kênh lowercase |
| ... | （剩余 175 处略）| | |

