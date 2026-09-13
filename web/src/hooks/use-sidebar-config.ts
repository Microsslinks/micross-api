/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { useMemo } from 'react'

import type { NavGroup, NavItem } from '@/components/layout/types'
import { useStatus } from '@/hooks/use-status'
import { useAuthStore } from '@/stores/auth-store'

type SidebarSectionConfig = {
  enabled: boolean
  [key: string]: boolean
}

type SidebarModulesAdminConfig = Record<string, SidebarSectionConfig>

// User-layer config is shape-identical to admin, but may be null
// to signal "no narrowing" (empty/invalid/legacy users).
type SidebarModulesUserConfig = SidebarModulesAdminConfig | null

/**
 * Default sidebar modules configuration.
 *
 * Sections mirror the sidebar's audience layering:
 *   · `playground` / `console` — general (every signed-in user);
 *   · `personal` — personal workspace;
 *   · `business` — administrator workspace (upstream supply, pricing,
 *     payment, site, content);
 *   · `system` — super administrator workspace (runtime, routing, security,
 *     ops).
 *
 * Keeping the granularity at "one module per sidebar group" lets an
 * administrator be granted billing access without inheriting channel or
 * routing control.
 */
const DEFAULT_SIDEBAR_MODULES: SidebarModulesAdminConfig = {
  playground: {
    enabled: true,
    playground: true,
  },
  console: {
    enabled: true,
    detail: true,
    token: true,
    log: true,
    midjourney: true,
    task: true,
  },
  personal: {
    enabled: true,
    topup: true,
    plans: true,
    earnings: true,
    customers: true,
    billing: true,
    personal: true,
  },
  business: {
    enabled: true,
    channels: true,
    models: true,
    users: true,
    billing: true,
    site: true,
    content: true,
    policies: true,
  },
  system: {
    enabled: true,
    runtime: true,
    auth: true,
    security: true,
    routing: true,
    operations: true,
  },
}

const mergeWithDefaultSidebarModules = (
  config: SidebarModulesAdminConfig
): SidebarModulesAdminConfig => {
  const merged: SidebarModulesAdminConfig = { ...config }

  Object.entries(DEFAULT_SIDEBAR_MODULES).forEach(
    ([sectionKey, defaultSection]) => {
      const existingSection = merged[sectionKey]
      if (!existingSection) {
        merged[sectionKey] = { ...defaultSection }
        return
      }

      merged[sectionKey] = { ...defaultSection, ...existingSection }
      Object.keys(defaultSection).forEach((moduleKey) => {
        if (merged[sectionKey][moduleKey] === undefined) {
          merged[sectionKey][moduleKey] = defaultSection[moduleKey]
        }
      })
    }
  )

  return merged
}

/**
 * Mapping from sidebar URL to its `sidebar_modules` section/module keys.
 *
 * This table is a whitelist: a navigation entry is only rendered when its URL
 * is registered here (see `isModuleEnabled`). Lookups use the longest
 * registered prefix (see `findConfigMapping`), so a group landing page
 * (`/business-settings/billing`) and every section under it
 * (`/business-settings/billing/quota`) resolve to the same module. That is
 * what makes the nested business / system workspaces filterable per module
 * rather than all-or-nothing.
 */
const URL_TO_CONFIG_MAP: Record<string, { section: string; module: string }> = {
  // General
  '/playground': { section: 'playground', module: 'playground' },
  '/dashboard': { section: 'console', module: 'detail' },
  '/dashboard/overview': { section: 'console', module: 'detail' },
  '/dashboard/models': { section: 'console', module: 'detail' },
  '/dashboard/users': { section: 'console', module: 'detail' },
  '/keys': { section: 'console', module: 'token' },
  '/usage-logs': { section: 'console', module: 'log' },
  '/usage-logs/common': { section: 'console', module: 'log' },
  '/usage-logs/drawing': { section: 'console', module: 'midjourney' },
  '/usage-logs/task': { section: 'console', module: 'task' },
  // Personal
  '/wallet': { section: 'personal', module: 'topup' },
  '/plans': { section: 'personal', module: 'plans' },
  '/earnings': { section: 'personal', module: 'earnings' },
  // 「我的客户」是经销商自己的客户名单，跟台账（钱）是两件事，
  // 所以单独占一个开关，管理员可以只关名单、不影响台账。
  '/customers': { section: 'personal', module: 'customers' },
  '/billing': { section: 'personal', module: 'billing' },
  '/profile': { section: 'personal', module: 'personal' },
  // Business management
  '/business-settings': { section: 'business', module: 'billing' },
  '/business-settings/billing': { section: 'business', module: 'billing' },
  '/business-settings/site': { section: 'business', module: 'site' },
  '/business-settings/content': { section: 'business', module: 'content' },
  '/business-settings/policies': { section: 'business', module: 'policies' },
  '/users': { section: 'business', module: 'users' },
  '/redemption-codes': { section: 'business', module: 'users' },
  '/subscriptions': { section: 'business', module: 'users' },
  // Discount plans price a single customer, so they sit with the rest of the
  // pricing surface (`billing`) rather than with user administration.
  // Leaving this route unregistered hides its sidebar entry entirely: the
  // map below is a whitelist, not a labelling aid.
  '/discounts': { section: 'business', module: 'billing' },
  '/channels': { section: 'business', module: 'channels' },
  '/models': { section: 'business', module: 'models' },
  '/models/metadata': { section: 'business', module: 'models' },
  '/models/deployments': { section: 'business', module: 'models' },
  // System management
  '/system-info': { section: 'system', module: 'runtime' },
  '/system-settings': { section: 'system', module: 'routing' },
  '/system-settings/auth': { section: 'system', module: 'auth' },
  '/system-settings/models': { section: 'system', module: 'routing' },
  '/system-settings/security': { section: 'system', module: 'security' },
  '/system-settings/operations': { section: 'system', module: 'operations' },
}

/**
 * Resolve the config mapping for a URL using the longest matching prefix.
 *
 * Prefixes only match on path boundaries, so `/users` never captures
 * `/users-archive`. Returns `undefined` when no mapping applies, which the
 * caller treats as "hidden" — an unmapped route is not part of any navigation
 * surface until it is registered here.
 */
function findConfigMapping(
  url: string
): { section: string; module: string } | undefined {
  let matched: { section: string; module: string } | undefined
  let matchedLength = -1

  Object.entries(URL_TO_CONFIG_MAP).forEach(([prefix, mapping]) => {
    if (prefix.length <= matchedLength) return
    if (url === prefix || url.startsWith(`${prefix}/`)) {
      matched = mapping
      matchedLength = prefix.length
    }
  })

  return matched
}

/**
 * Parse backend SidebarModulesAdmin configuration
 */
function parseSidebarConfig(
  value: string | null | undefined
): SidebarModulesAdminConfig {
  // If empty string, null, or undefined, use default config
  if (!value || value.trim() === '') {
    return DEFAULT_SIDEBAR_MODULES
  }

  try {
    const parsed = JSON.parse(value) as SidebarModulesAdminConfig
    return mergeWithDefaultSidebarModules(parsed)
  } catch {
    // eslint-disable-next-line no-console
    console.error('Failed to parse sidebar modules configuration')
    return DEFAULT_SIDEBAR_MODULES
  }
}

/**
 * Parse user-level sidebar_modules. Returns null when the value is empty,
 * invalid, or otherwise unusable — the caller treats null as "do not narrow",
 * so legacy users with an empty sidebar_modules field keep the full admin view.
 */
function parseUserSidebarConfig(
  value: string | null | undefined
): SidebarModulesUserConfig {
  if (!value || value.trim() === '') {
    return null
  }
  try {
    const parsed = JSON.parse(value) as SidebarModulesAdminConfig
    if (!parsed || typeof parsed !== 'object') return null
    return parsed
  } catch {
    return null
  }
}

/**
 * Check if a module is enabled. A URL that is not registered in
 * `URL_TO_CONFIG_MAP` is never enabled: the map is the whitelist that decides
 * which routes may appear in navigation at all. Admin config is then the
 * first (authoritative) layer: if admin disables a section/module it is
 * always hidden. User config is a second narrower layer: it can only further
 * hide what admin allowed. A null user config means "do not narrow"
 * (legacy/empty users).
 */
function isModuleEnabled(
  url: string,
  adminConfig: SidebarModulesAdminConfig,
  userConfig: SidebarModulesUserConfig
): boolean {
  const mapping = findConfigMapping(url)
  if (!mapping) {
    // Whitelist: an unregistered URL is hidden. Registering the route in
    // URL_TO_CONFIG_MAP is what grants it a navigation entry, which in turn
    // guarantees every visible entry is one an administrator can disable and
    // a user can narrow through sidebar_modules.
    return false
  }

  const { section, module } = mapping
  const adminSection = adminConfig[section]
  const adminAllowed = Boolean(
    adminSection && adminSection.enabled && adminSection[module] === true
  )
  if (!adminAllowed) return false

  if (!userConfig) return true

  const userSection = userConfig[section]
  if (!userSection) return true
  if (userSection.enabled === false) return false
  return userSection[module] !== false
}

/**
 * Check if a navigation item should be visible
 */
function isNavItemVisible(
  item: NavItem,
  adminConfig: SidebarModulesAdminConfig,
  userConfig: SidebarModulesUserConfig
): boolean {
  // Handle dynamic chat presets type — also runs the admin × user AND gate
  if ('type' in item && item.type === 'chat-presets') {
    const adminChat = adminConfig.chat
    const adminAllowed = Boolean(adminChat?.enabled && adminChat.chat === true)
    if (!adminAllowed) return false
    if (!userConfig) return true
    const userChat = userConfig.chat
    if (!userChat) return true
    if (userChat.enabled === false) return false
    return userChat.chat !== false
  }

  // Handle direct link type
  if ('url' in item && item.url) {
    const configUrls = item.configUrls ?? [item.url]
    return configUrls.some((url) =>
      isModuleEnabled(url as string, adminConfig, userConfig)
    )
  }

  // Handle collapsible type (with sub-items)
  if ('items' in item && item.items) {
    // If has sub-items, show this collapsible item if at least one sub-item is visible
    return item.items.some((subItem) =>
      isModuleEnabled(subItem.url as string, adminConfig, userConfig)
    )
  }

  return true
}

/**
 * Filter navigation items
 */
function filterNavItems(
  items: NavItem[],
  adminConfig: SidebarModulesAdminConfig,
  userConfig: SidebarModulesUserConfig
): NavItem[] {
  return items
    .map((item) => {
      // If collapsible item, also filter its sub-items
      if ('items' in item && item.items) {
        const filteredSubItems = item.items.filter((subItem) =>
          isModuleEnabled(subItem.url as string, adminConfig, userConfig)
        )

        return {
          ...item,
          items: filteredSubItems,
        }
      }
      return item
    })
    .filter((item) => isNavItemVisible(item, adminConfig, userConfig))
}

/**
 * Filter sidebar navigation groups by admin × user sidebar_modules config.
 *
 * Two layers, AND-combined:
 *   1. Admin (status.SidebarModulesAdmin) — authoritative, falls back to
 *      DEFAULT_SIDEBAR_MODULES when empty/invalid. Disabling here hides the
 *      item for everyone regardless of user preference.
 *   2. User (auth.user.sidebar_modules) — narrower overlay, null sentinel
 *      means "don't narrow". A section/module is only hidden if the user
 *      explicitly set it to false; undefined fields default to visible so
 *      legacy users with empty sidebar_modules keep the full admin view.
 *      The overlay is also skipped entirely when the backend tells us the
 *      user cannot configure sidebar_settings (e.g. root accounts), so a
 *      stale historical value cannot lock them out of entries they have no
 *      UI to restore.
 */
export function useSidebarConfig(navGroups: NavGroup[]): NavGroup[] {
  const { status } = useStatus()
  const { auth } = useAuthStore()

  const adminConfig = useMemo(
    () =>
      parseSidebarConfig(
        status?.SidebarModulesAdmin as string | null | undefined
      ),
    [status?.SidebarModulesAdmin]
  )

  const userConfig = useMemo(() => {
    // If the backend marks the user as unable to configure the sidebar
    // (e.g. root accounts), skip the user overlay entirely — a stale
    // historical sidebar_modules value from a previous role would otherwise
    // hide admin entries for someone who has no in-product UI to restore
    // them.
    if (auth?.user?.permissions?.sidebar_settings === false) {
      return null
    }
    return parseUserSidebarConfig(auth?.user?.sidebar_modules)
  }, [auth?.user?.permissions?.sidebar_settings, auth?.user?.sidebar_modules])

  const filteredNavGroups = useMemo(
    () =>
      navGroups
        .map((group) => ({
          ...group,
          items: filterNavItems(group.items, adminConfig, userConfig),
        }))
        .filter((group) => group.items.length > 0), // Only show navigation groups with visible items
    [navGroups, adminConfig, userConfig]
  )

  return filteredNavGroups
}

/**
 * Check whether a single route is visible under the current sidebar_modules
 * config. Used by entries living outside the sidebar (e.g. the profile
 * dropdown's wallet link) so they honour the same "wallet display" toggle.
 */
export function useIsSidebarModuleVisible(url: string): boolean {
  const { status } = useStatus()
  const { auth } = useAuthStore()

  const adminConfig = parseSidebarConfig(
    status?.SidebarModulesAdmin as string | null | undefined
  )
  const userConfig =
    auth?.user?.permissions?.sidebar_settings === false
      ? null
      : parseUserSidebarConfig(auth?.user?.sidebar_modules)

  return isModuleEnabled(url, adminConfig, userConfig)
}
