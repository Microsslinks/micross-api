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
import type { TFunction } from 'i18next'
import { Activity, Box, ServerCog, Shield } from 'lucide-react'

import { getAuthSectionNavItems } from '@/features/system-settings/auth/section-registry.tsx'
import { getModelsSectionNavItems } from '@/features/system-settings/models/section-registry.tsx'
import { getOperationsSectionNavItems } from '@/features/system-settings/operations/section-registry.tsx'
import { getSecuritySectionNavItems } from '@/features/system-settings/security/section-registry.tsx'
import { ROLE } from '@/lib/roles'

import type { NavGroup, SidebarView } from '../types'

/**
 * Sidebar nav groups for the System Management nested view.
 *
 * Scope rule: everything here decides how the gateway *connects, routes,
 * protects and operates* — i.e. changes that can take the service down.
 * Business-facing configuration (pricing, payment, site branding, console
 * content, access policy) lives in the Business Management view instead,
 * including upstream supply: channels and the model catalog are commercial
 * assets guarded by the `admin` role, so they live there.
 *
 * The former standalone "Operations" group is merged into "Runtime &
 * Operations": Runtime Status stays first as the workspace landing page,
 * then the operations sections follow in registry order (performance →
 * monitoring → logs → email → worker → behavior → updates). Discount
 * policy moved to Business Management → Billing & Pricing.
 */
function getSystemSettingsNavGroups(t: TFunction): NavGroup[] {
  return [
    {
      id: 'system-administration',
      title: t('System Administration'),
      items: [
        {
          title: t('Runtime & Access'),
          icon: ServerCog,
          items: [
            { title: t('Runtime Status'), url: '/system-info', icon: Activity },
            ...getOperationsSectionNavItems(t),
          ],
        },
        {
          title: t('Authentication & Security'),
          icon: Shield,
          items: [...getAuthSectionNavItems(t), ...getSecuritySectionNavItems(t)],
        },
        {
          title: t('Models & Routing'),
          icon: Box,
          items: getModelsSectionNavItems(t),
        },
      ],
    },
  ]
}

/**
 * Nested sidebar view for `/system-settings/*` and the standalone runtime
 * status page (`/system-info`).
 *
 * Activates the Vercel / Cloudflare-style drill-in sidebar:
 * the root navigation is replaced by the system administration
 * groups, with a "Back to Dashboard" affordance in the header.
 *
 * Every entry here is `super_admin` territory, and so is the workspace
 * itself (`requiredRole` below) — an administrator below that threshold
 * keeps the root navigation rather than seeing entries the route guards
 * would reject.
 */
export const SYSTEM_SETTINGS_VIEW: SidebarView = {
  id: 'system-settings',
  pathPattern: /^\/(system-settings|system-info)(\/|$)/,
  requiredRole: ROLE.SUPER_ADMIN,
  parent: {
    to: '/dashboard/overview',
    label: 'Back to Dashboard',
  },
  getNavGroups: getSystemSettingsNavGroups,
}
