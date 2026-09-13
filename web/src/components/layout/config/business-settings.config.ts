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
import { type TFunction } from 'i18next'
import {
  Boxes,
  CalendarDays,
  CreditCard,
  Layout,
  Percent,
  Plug,
  Server,
  Settings,
  ShieldAlert,
  TicketPercent,
  Users,
} from 'lucide-react'

import { getBillingSectionNavItems } from '@/features/business-settings/billing/section-registry.tsx'
import { getContentSectionNavItems } from '@/features/business-settings/content/section-registry.tsx'
import { getPoliciesSectionNavItems } from '@/features/business-settings/policies/section-registry.tsx'
import { getSiteSectionNavItems } from '@/features/business-settings/site/section-registry.tsx'
import { ROLE } from '@/lib/roles'

import type { NavGroup, SidebarView } from '../types'

/**
 * Sidebar nav groups for the Business Management nested view.
 *
 * Scope rule: everything here decides *what we sell, to whom, at what
 * price, and how the product looks* — operational decisions that an
 * administrator can own without touching gateway infrastructure.
 *
 * Upstream supply belongs here too: which providers we are connected to
 * (channels) and which of their models we offer are commercial decisions,
 * and both routes are guarded by the `admin` role — not by `super_admin`.
 */
function getBusinessSettingsNavGroups(t: TFunction): NavGroup[] {
  return [
    {
      id: 'business-administration',
      title: t('Business Administration'),
      items: [
        {
          title: t('Upstream Providers'),
          icon: Plug,
          items: [
            { title: t('Channels'), url: '/channels', icon: Server },
            {
              title: t('Models'),
              url: '/models/metadata',
              icon: Boxes,
              activeUrls: ['/models/deployments'],
              configUrls: ['/models/metadata', '/models/deployments'],
            },
          ],
        },
        {
          title: t('User Management'),
          icon: Users,
          items: [
            { title: t('Users'), url: '/users', icon: Users },
            {
              title: t('Redemption Codes'),
              url: '/redemption-codes',
              icon: TicketPercent,
            },
            {
              title: t('Subscriptions'),
              url: '/subscriptions',
              icon: CalendarDays,
            },
          ],
        },
        {
          // "What we charge whom" lives here: group pricing sets the rate for
          // a whole group of users, discount plans set it for one customer or
          // dealer. Keeping both under one group is why this registry-driven
          // group also carries a top-level route (`/discounts`) that predates
          // it — same as the user management group above.
          title: t('Billing & Pricing'),
          icon: CreditCard,
          items: [
            ...getBillingSectionNavItems(t),
            {
              title: t('Discount Plans'),
              url: '/discounts',
              icon: Percent,
            },
          ],
        },
        {
          title: t('Site & Appearance'),
          icon: Settings,
          items: getSiteSectionNavItems(t),
        },
        {
          title: t('Console Content'),
          icon: Layout,
          items: getContentSectionNavItems(t),
        },
        {
          title: t('Access & Limits'),
          icon: ShieldAlert,
          items: getPoliciesSectionNavItems(t),
        },
      ],
    },
  ]
}

/**
 * Nested sidebar view for the Business Management workspace.
 *
 * Peer of {@link SYSTEM_SETTINGS_VIEW}: entering this workspace swaps the
 * root navigation for the business configuration groups. Requires the
 * `admin` role (see `routes/_authenticated/business-settings/route.tsx`).
 *
 * The workspace is not mounted under a single URL prefix: user management
 * and upstream supply predate it and still live on top-level routes
 * (`/users`, `/redemption-codes`, `/subscriptions`, `/discounts`, `/channels`,
 * `/models`).
 * Those paths must be matched here as well, otherwise opening one of them
 * renders its page while the sidebar falls back to the root navigation.
 * Their own route guards keep requiring the `admin` role, which is the same
 * threshold declared below, so widening the match cannot expose the view to
 * non-admins.
 */
export const BUSINESS_SETTINGS_VIEW: SidebarView = {
  id: 'business-settings',
  pathPattern:
    /^\/(business-settings|users|redemption-codes|subscriptions|discounts|channels|models)(\/|$)/,
  requiredRole: ROLE.ADMIN,
  parent: {
    to: '/dashboard/overview',
    label: 'Back to Dashboard',
  },
  getNavGroups: getBusinessSettingsNavGroups,
}
