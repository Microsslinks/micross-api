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
import {
  Activity,
  Briefcase,
  Crown,
  FileText,
  FlaskConical,
  Key,
  LayoutDashboard,
  ListTodo,
  Receipt,
  Settings,
  TrendingUp,
  User,
  Wallet,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'

import type { NavItem, SidebarData } from '@/components/layout/types'
import { CUSTOMER_TYPE } from '@/features/users/constants'
import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'

/**
 * Root navigation groups for the application sidebar.
 *
 * Audience layering:
 *   · General — every signed-in user: the console entry points;
 *   · Personal — the signed-in user's own money and account, split by the
 *     direction money moves: wallet (balance / top-up / redemption),
 *     plans (what you buy), earnings (referral commission), the dealer
 *     ledger (only for `subject_type = 'agent'`), profile;
 *   · Business Management — administrators (`requiredRole: ROLE.ADMIN`);
 *   · System Management — super administrators only.
 *
 * The two management entries are drill-in workspaces: clicking either one
 * swaps the sidebar to the matching nested view (see
 * `layout/config/business-settings.config.ts` and `system-settings.config.ts`).
 */
export function useSidebarData(): SidebarData {
  const { t } = useTranslation()

  // 经销商是叠在用户身上的业务身份（users.subject_type = 'agent'），不是权限等级：
  // 他自己那几个客户侧菜单照旧，只是多出「台账」这一项，普通客户看不到它。
  const isDealer =
    useAuthStore((state) => state.auth.user?.subject_type) ===
    CUSTOMER_TYPE.AGENT

  const dealerBillingItems: NavItem[] = isDealer
    ? [
        {
          title: t('Dealer Billing'),
          url: '/billing',
          icon: Receipt,
        },
      ]
    : []

  return {
    navGroups: [
      {
        id: 'general',
        title: t('General'),
        items: [
          {
            title: t('Overview'),
            url: '/dashboard/overview',
            icon: Activity,
          },
          {
            title: t('Dashboard'),
            url: '/dashboard/models',
            icon: LayoutDashboard,
          },
          {
            title: t('API Keys'),
            url: '/keys',
            icon: Key,
          },
          {
            title: t('Usage Logs'),
            url: '/usage-logs/common',
            icon: FileText,
          },
          {
            title: t('Task Logs'),
            url: '/usage-logs/task',
            activeUrls: ['/usage-logs/drawing'],
            configUrls: ['/usage-logs/drawing', '/usage-logs/task'],
            icon: ListTodo,
          },
          {
            title: t('Test Model'),
            url: '/playground',
            icon: FlaskConical,
          },
        ],
      },
      {
        id: 'personal',
        title: t('Personal'),
        items: [
          {
            title: t('Wallet'),
            url: '/wallet',
            icon: Wallet,
          },
          {
            title: t('Plan'),
            url: '/plans',
            icon: Crown,
          },
          {
            title: t('Earnings'),
            url: '/earnings',
            icon: TrendingUp,
          },
          ...dealerBillingItems,
          {
            title: t('Profile'),
            url: '/profile',
            icon: User,
          },
        ],
      },
      {
        id: 'administration',
        title: t('Administration'),
        items: [
          {
            title: t('Business Management'),
            url: '/channels',
            configUrls: ['/business-settings'],
            icon: Briefcase,
            requiredRole: ROLE.ADMIN,
          },
          {
            title: t('System Management'),
            url: '/system-info',
            configUrls: ['/system-settings'],
            icon: Settings,
            requiredRole: ROLE.SUPER_ADMIN,
          },
        ],
      },
    ],
  }
}
