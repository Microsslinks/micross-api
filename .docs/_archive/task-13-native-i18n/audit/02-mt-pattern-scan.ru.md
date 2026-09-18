# 02 · 机翻特征扫描 · RU（task-13 phase 2）

> 创建：2026-09-17
> 数据来源：`web/src/i18n/locales/ru.json` vs `en.json`
> 工具：`web/scripts/scan-mt-patterns.mjs`

## 摘要

| 指标 | 值 |
|---|---|
| 总键数 | 5927 |
| 完全未翻译（与 en 相同）| 596 |
| 至少命中一条机翻特征 | 284 |
| 句式分叉数 | 0 |

## 规则命中统计

| 规则 | 命中数 | 说明 |
|---|---|---|
| `length-too-long` | 147 | value 长度 > en 原文 1.5×（典型：过度展开） |
| `borrowed-english` | 139 | value 保留了英文技术借词（upstream / cache / token 等） |

## 详细命中清单（每规则前 30）

### length-too-long（147 处，展示前 30）

| key | en | ru |
|---|---|---|
| `Custom Links` | Custom Links | Пользовательские ссылки |
| `30d change` | 30d change | Изменение за 30 дней |
| `Account Info` | Account Info | Информация об аккаунте |
| `active users` | active users | активных пользователей |
| `Add Mapping` | Add Mapping | Добавить сопоставление |
| `Add ratio override` | Add ratio override | Добавить переопределение коэффициента |
| `Add user group` | Add user group | Добавить группу пользователей |
| `Admin area` | Admin area | Область администратора |
| `Admin Only` | Admin Only | Только для администраторов |
| `Advanced Custom` | Advanced Custom | Расширенный пользовательский |
| `Agent ID *` | Agent ID * | Идентификатор агента * |
| `Apply Sync` | Apply Sync | Применить синхронизацию |
| `apps tracked` | apps tracked | приложений отслеживается |
| `Area Chart` | Area Chart | Диаграмма с областями |
| `Auth Style` | Auth Style | Стиль аутентификации |
| `Auto detect (default)` | Auto detect (default) | Автоматическое определение (по умолчанию) |
| `Auto Sync Upstream Models` | Auto Sync Upstream Models | Автоматическая синхронизация моделей провайдера |
| `Average RPM` | Average RPM | Среднее число оборотов в минуту |
| `Average TPM` | Average TPM | Среднее число транзакций в минуту |
| `Bark Push URL` | Bark Push URL | URL для push-уведомлений Bark |
| `Basic Info` | Basic Info | Основная информация |
| `Batch channel test` | Batch channel test | Пакетное тестирование поставщиков |
| `Batch Edit` | Batch Edit | Пакетное редактирование |
| `Batch Edit by Tag` | Batch Edit by Tag | Пакетное редактирование по тегу |
| `Blocked keywords` | Blocked keywords | Заблокированные ключевые слова |
| `Body param` | Body param | Параметр тела запроса |
| `Click to copy` | Click to copy | Нажмите, чтобы скопировать |
| `Click to view image` | Click to view image | Нажмите, чтобы просмотреть изображение |
| `Color preset` | Color preset | Цветовая предустановка |
| `Common Keys` | Common Keys | Часто используемые ключи |
| ... | （剩余 117 处略）| |

### borrowed-english（139 处，展示前 30）

| key | en | ru |
|---|---|---|
| `Access Policy (JSON)` | Access Policy (JSON) | Политика доступа (JSON) |
| `Add API` | Add API | Добавить API |
| `Add API Shortcut` | Add API Shortcut | Добавить ярлык API |
| `Add OAuth Provider` | Add OAuth Provider | Добавить поставщика OAuth |
| `All API tokens` | All API tokens | Все API-ключи |
| `Allow upstream callbacks` | Allow upstream callbacks | Разрешить обратные вызовы upstream |
| `API Access` | API Access | Доступ к API |
| `API Addresses` | API Addresses | Адреса API |
| `API Base URL *` | API Base URL * | Базовый URL API * |
| `API Endpoints` | API Endpoints | Конечные точки API |
| `API Info` | API Info | Информация об API |
| `API key` | API key | Ключ API |
| `API Key` | API Key | Ключ API |
| `API Key (Production)` | API Key (Production) | API-ключ (Продакшн) |
| `API Key (Sandbox)` | API Key (Sandbox) | API-ключ (Песочница) |
| `API Key *` | API Key * | Ключ API * |
| `API key is required` | API key is required | Требуется ключ API |
| `API Keys` | API Keys | Ключи API |
| `API Private Key` | API Private Key | Секретный ключ API |
| `API Requests` | API Requests | Запросы API |
| `API secret` | API secret | Секрет API |
| `API URL` | API URL | URL API |
| `Apply All Upstream Updates` | Apply All Upstream Updates | Применить все обновления из upstream |
| `Base URL` | Base URL | Адрес API |
| `Bind a Pancake store + product` | Bind a Pancake store + product | Привязать магазин и продукт Pancake |
| `Channel Affinity: Upstream Cache Hit` | Channel Affinity: Upstream Cache Hit | Привязка к поставщику: попадание в кэш upstream |
| `Chat configuration JSON` | Chat configuration JSON | JSON конфигурации чата |
| `Confirm Creem Purchase` | Confirm Creem Purchase | Подтвердить покупку Creem |
| `Copy API key` | Copy API key | Скопировать ключ API |
| `Create API Key` | Create API Key | Создать ключ API |
| ... | （剩余 109 处略）| |

