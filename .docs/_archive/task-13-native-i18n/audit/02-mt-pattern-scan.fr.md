# 02 · 机翻特征扫描 · FR（task-13 phase 2）

> 创建：2026-09-17
> 数据来源：`web/src/i18n/locales/fr.json` vs `en.json`
> 工具：`web/scripts/scan-mt-patterns.mjs`

## 摘要

| 指标 | 值 |
|---|---|
| 总键数 | 5927 |
| 完全未翻译（与 en 相同）| 665 |
| 至少命中一条机翻特征 | 591 |
| 句式分叉数 | 0 |

## 规则命中统计

| 规则 | 命中数 | 说明 |
|---|---|---|
| `length-too-long` | 275 | value 长度 > en 原文 1.5×（典型：过度展开） |
| `borrowed-english` | 184 | value 保留了英文技术借词（upstream / cache / token 等） |
| `fr-punct-no-space` | 155 | 法语标点前/后缺空格（MT 典型问题） |

## 详细命中清单（每规则前 30）

### length-too-long（275 处，展示前 30）

| key | en | fr |
|---|---|---|
| `Account Info` | Account Info | Informations du compte |
| `Active apps` | Active apps | Applications actives |
| `Add auto group` | Add auto group | Ajouter un groupe automatique |
| `Add chat preset` | Add chat preset | Ajouter un préréglage de chat |
| `Add group rules` | Add group rules | Ajouter des règles de groupe |
| `Add method` | Add method | Ajouter une méthode |
| `Add model pricing` | Add model pricing | Ajouter une tarification de modèle |
| `Add Models` | Add Models | Ajouter des modèles |
| `Add param/header` | Add param/header | Ajouter un param / un en-tête |
| `Add Provider` | Add Provider | Ajouter un fournisseur |
| `Add rule group` | Add rule group | Ajouter un groupe de règles |
| `Add rules for a user group` | Add rules for a user group | Ajouter des règles pour un groupe d’utilisateurs |
| `Add tags...` | Add tags... | Ajouter des étiquettes... |
| `Add time rule group` | Add time rule group | Ajouter un groupe de règles temporelles |
| `Add user group` | Add user group | Ajouter un groupe d'utilisateurs |
| `Admin area` | Admin area | Espace administrateur |
| `Admin Channel Permissions` | Admin Channel Permissions | Autorisations des fournisseurs amont administrateur |
| `Admin notes (only visible to admins)` | Admin notes (only visible to admins) | Notes d'administration (visibles uniquement par les administrateurs) |
| `Admin Only` | Admin Only | Administrateur uniquement |
| `All Sync Status` | All Sync Status | Tous les statuts de synchronisation |
| `Allow Retry` | Allow Retry | Autoriser la relance |
| `Allow users to check in daily for random quota rewards` | Allow users to check in daily for random quota rewards | Permettre aux utilisateurs de se connecter quotidiennement pour des récompenses  |
| `Allow users to enter promo codes` | Allow users to enter promo codes | Autoriser les utilisateurs à saisir des codes promotionnels |
| `API Endpoints` | API Endpoints | Points de terminaison API |
| `API usage records` | API usage records | Historique d'utilisation de l'API |
| `Apply reset` | Apply reset | Appliquer la réinitialisation |
| `Apply Sync` | Apply Sync | Appliquer la synchronisation |
| `Applying...` | Applying... | Application en cours... |
| `Ask anything` | Ask anything | Demandez n'importe quoi |
| `Async task polling` | Async task polling | Interrogation des tâches asynchrones |
| ... | （剩余 245 处略）| |

### borrowed-english（184 处，展示前 30）

| key | en | fr |
|---|---|---|
| `/your/endpoint` | /your/endpoint | /votre/endpoint |
| `Access Policy (JSON)` | Access Policy (JSON) | Politique d'accès (JSON) |
| `Add API` | Add API | Ajouter une API |
| `Add OAuth Provider` | Add OAuth Provider | Ajouter un fournisseur OAuth |
| `All API tokens` | All API tokens | Tous les jetons API |
| `API Access` | API Access | Accès API |
| `API Addresses` | API Addresses | Adresses API |
| `API Info` | API Info | Infos API |
| `API key` | API key | Clé API |
| `API Key` | API Key | Clé API |
| `API Key (Production)` | API Key (Production) | Clé API (Production) |
| `API Key (Sandbox)` | API Key (Sandbox) | Clé API (Sandbox) |
| `API Key *` | API Key * | Clé API * |
| `API Keys` | API Keys | Clés API |
| `API Private Key` | API Private Key | Clé privée de l'API |
| `API Requests` | API Requests | Requêtes API |
| `API secret` | API secret | Secret API |
| `API token management` | API token management | Gestion des tokens API |
| `API URL` | API URL | URL de l'API |
| `Apply All Upstream Updates` | Apply All Upstream Updates | Appliquer toutes les mises à jour upstream |
| `Billable input tokens` | Billable input tokens | Tokens d’entrée facturables |
| `Billable output tokens` | Billable output tokens | Tokens de sortie facturables |
| `Bind a Pancake store + product` | Bind a Pancake store + product | Associer une boutique et un produit Pancake |
| `Cache create (1h) price` | Cache create (1h) price | Prix de création du cache (1 h) |
| `Cache create price` | Cache create price | Prix de création du cache |
| `Cache Creation` | Cache Creation | Création cache |
| `Cache Creation (1h)` | Cache Creation (1h) | Création cache (1h) |
| `Cache Creation (5m)` | Cache Creation (5m) | Création cache (5m) |
| `Cache Directory` | Cache Directory | Répertoire de cache |
| `Cache Directory Info` | Cache Directory Info | Infos du répertoire de cache |
| ... | （剩余 154 处略）| |

### fr-punct-no-space（155 处，展示前 30）

| key | en | fr |
|---|---|---|
| `JSON array of extra links, e.g. [{"label":"Forum","url":"https://..."}]` | JSON array of extra links, e.g. [{"label":"Forum","url":"https://..."}] | Tableau JSON de liens supplémentaires, ex. [{"label":"Forum","url":"https://..." |
| `Enter HTML code or a URL (e.g., https://example.com) to embed as iframe` | Enter HTML code or a URL (e.g., https://example.com) to embed as iframe | Saisissez du code HTML ou une URL (ex. https://example. com) à intégrer en ifram |
| `Welcome to our site...` | Welcome to our site... | Bienvenue sur notre site... |
| `[{"ChatGPT":"https://chat.openai.com"},{"Lobe Chat":"https://chat-preview.lobehub.com/?settings={...}"}]` | [{"ChatGPT":"https://chat.openai.com"},{"Lobe Chat":"https://chat-preview.lobehu | [{"ChatGPT":"https://chat. openai. com"},{"Lobe Chat":"https://chat-preview. lob |
| `Add from available models...` | Add from available models... | Ajouter à partir des modèles disponibles... |
| `Add tags...` | Add tags... | Ajouter des étiquettes... |
| `Applying...` | Applying... | Application en cours... |
| `Batch testing models...` | Batch testing models... | Test des modèles par lots... |
| `Binding...` | Binding... | Liaison en cours... |
| `Calculating...` | Calculating... | Calcul en cours... |
| `Changing...` | Changing... | Modification en cours... |
| `Checking name...` | Checking name... | Vérification du nom... |
| `Checking updates...` | Checking updates... | Vérification des mises à jour... |
| `Cleaning...` | Cleaning... | Nettoyage en cours... |
| `Comma-separated model names, e.g., gpt-4,gpt-3.5-turbo` | Comma-separated model names, e.g., gpt-4,gpt-3.5-turbo | Noms de modèles séparés par des virgules, p. ex., gpt-4,gpt-3.5-turbo |
| `Copying...` | Copying... | Copie... |
| `Creating...` | Creating... | Création... |
| `Deleting...` | Deleting... | Suppression... |
| `Describe this model...` | Describe this model... | Décrire ce modèle... |
| `Describe this vendor...` | Describe this vendor... | Décrire ce fournisseur... |
| `Disabling...` | Disabling... | Désactivation en cours... |
| `Discovering...` | Discovering... | Découverte en cours... |
| `e.g., 0.95` | e.g., 0.95 | par ex., 0.95 |
| `e.g., 100` | e.g., 100 | par ex., 100 |
| `e.g., 123456` | e.g., 123456 | par ex., 123456 |
| `e.g., 2025-04-01-preview` | e.g., 2025-04-01-preview | par ex., 2025-04-01-preview |
| `e.g., 50` | e.g., 50 | par ex., 50 |
| `e.g., 500000` | e.g., 500000 | p. ex., 500000 |
| `e.g., 7342866812345` | e.g., 7342866812345 | par ex., 7342866812345 |
| `e.g., 8 means 8 local currency per USD` | e.g., 8 means 8 local currency per USD | par ex., 8 signifie 8 unités de monnaie locale par USD |
| ... | （剩余 125 处略）| |

