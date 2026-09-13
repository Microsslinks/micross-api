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
import { ROLE } from '@/lib/roles'

/** 一个边栏分区的配置：`enabled` 控制整个分区，其余键控制分区内的模块。 */
export type SidebarSectionConfig = {
  enabled: boolean
  [module: string]: boolean
}

/** 用户级 `sidebar_modules`：分区键 → 分区配置。 */
export type SidebarModulesConfig = Record<string, SidebarSectionConfig>

type ManagementAccessModule = {
  key: string
  labelKey: string
  descriptionKey: string
}

export type ManagementAccessSection = {
  key: string
  labelKey: string
  descriptionKey: string
  /** 进入该工作区本身要求的角色；低于该角色的用户看不到这些入口。 */
  requiredRole: number
  modules: ManagementAccessModule[]
}

/**
 * 管理端在「新增/编辑用户」里可以授予的管理工作区。
 *
 * 分区与模块的 key 必须与 `@/hooks/use-sidebar-config` 的 URL_TO_CONFIG_MAP
 * 保持一致，否则这里的开关不会影响任何侧边栏入口。文案 key 与
 * `features/business-settings/site/sections/sidebar-modules-section.tsx` 复用同一批
 * 翻译，保证全局配置与单用户配置的用语一致。
 */
export const MANAGEMENT_ACCESS_SECTIONS: ManagementAccessSection[] = [
  {
    key: 'business',
    labelKey: 'Business Management',
    descriptionKey:
      'Visible to administrators. Upstream supply, pricing, payment, site branding, console content, and access policy.',
    requiredRole: ROLE.ADMIN,
    modules: [
      {
        key: 'channels',
        labelKey: 'Channels',
        descriptionKey: 'Configure upstream providers and routing.',
      },
      {
        key: 'models',
        labelKey: 'Models',
        descriptionKey: 'Manage the model catalog and deployments.',
      },
      {
        key: 'users',
        labelKey: 'User Management',
        descriptionKey: 'User accounts, redemption codes, and subscriptions.',
      },
      {
        key: 'billing',
        labelKey: 'Billing & Pricing',
        descriptionKey:
          'Model pricing, group ratios, quota grants, and payment gateways.',
      },
      {
        key: 'site',
        labelKey: 'Site & Appearance',
        descriptionKey: 'Site name, logo, footer, notices, and navigation.',
      },
      {
        key: 'content',
        labelKey: 'Console Content',
        descriptionKey:
          'Announcements, API addresses, FAQ, status page, and presets.',
      },
      {
        key: 'policies',
        labelKey: 'Access & Limits',
        descriptionKey:
          'Registration policy, request rate limits, and per-user token caps.',
      },
    ],
  },
  {
    key: 'system',
    labelKey: 'System Management',
    descriptionKey:
      'Visible to super administrators only. Routing, authentication, security, and operations.',
    requiredRole: ROLE.SUPER_ADMIN,
    modules: [
      {
        key: 'runtime',
        labelKey: 'Runtime & Access',
        descriptionKey: 'Instance status and runtime diagnostics.',
      },
      {
        key: 'auth',
        labelKey: 'Authentication',
        descriptionKey: 'OAuth providers, passkeys, and bot protection.',
      },
      {
        key: 'security',
        labelKey: 'Security',
        descriptionKey: 'Sensitive word filtering and SSRF protection.',
      },
      {
        key: 'routing',
        labelKey: 'Models & Routing',
        descriptionKey:
          'Global model config, retry policy, and channel affinity.',
      },
      {
        key: 'operations',
        labelKey: 'Operations',
        descriptionKey: 'SMTP, worker proxy, logs, performance, and updates.',
      },
    ],
  },
]

/**
 * 只保留该角色真正能进入的管理工作区。
 *
 * 系统管理要求超级管理员，普通管理员即使被授予也进不去（路由守卫按角色拦截），
 * 因此不给他展示这些开关，避免做出实际不生效的配置。
 */
export function managementSectionsForRole(
  role: number
): ManagementAccessSection[] {
  return MANAGEMENT_ACCESS_SECTIONS.filter(
    (section) => role >= section.requiredRole
  )
}

/**
 * 解析已保存的 `sidebar_modules`。
 *
 * 语义与 `use-sidebar-config` 一致：用户层只能「收窄」，只有显式的 false 才隐藏，
 * 缺省即为可见。返回的配置里本次 UI 管辖的分区会被补全成确定值，其它分区原样保留
 * （用户可能在个人中心收窄过控制台、个人中心等区域，不能被这里的保存覆盖掉）。
 */
export function parseSidebarModules(
  raw: string | null | undefined
): SidebarModulesConfig {
  const stored = parseStoredSections(raw)
  const config: SidebarModulesConfig = { ...stored }

  for (const section of MANAGEMENT_ACCESS_SECTIONS) {
    const current = stored[section.key]
    const normalized: SidebarSectionConfig = {
      enabled: current?.enabled !== false,
    }
    for (const module of section.modules) {
      normalized[module.key] = current?.[module.key] !== false
    }
    config[section.key] = normalized
  }

  return config
}

export function serializeSidebarModules(config: SidebarModulesConfig): string {
  return JSON.stringify(config)
}

/**
 * 局部更新某个分区（分区开关或单个模块开关），返回新对象供表单 `field.onChange` 使用。
 */
export function updateSectionConfig(
  config: SidebarModulesConfig | undefined,
  sectionKey: string,
  patch: Record<string, boolean>
): SidebarModulesConfig {
  const base = config ?? {}
  const stored = base[sectionKey]
  const section: SidebarSectionConfig = stored
    ? { ...stored }
    : { enabled: true }
  Object.entries(patch).forEach(([key, value]) => {
    section[key] = value
  })
  return { ...base, [sectionKey]: section }
}

function parseStoredSections(
  raw: string | null | undefined
): SidebarModulesConfig {
  if (!raw || raw.trim() === '') return {}

  try {
    const parsed = JSON.parse(raw) as Record<string, unknown>
    if (!parsed || typeof parsed !== 'object') return {}

    const sections: SidebarModulesConfig = {}
    Object.entries(parsed).forEach(([sectionKey, sectionValue]) => {
      if (!sectionValue || typeof sectionValue !== 'object') return

      const section: SidebarSectionConfig = { enabled: true }
      Object.entries(sectionValue as Record<string, unknown>).forEach(
        ([moduleKey, moduleValue]) => {
          if (typeof moduleValue !== 'boolean') return
          section[moduleKey] = moduleValue
        }
      )
      sections[sectionKey] = section
    })
    return sections
  } catch {
    return {}
  }
}
