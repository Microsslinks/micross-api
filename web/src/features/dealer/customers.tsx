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
import { useCallback, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { EmptyState } from '@/components/empty-state'
import { SectionPageLayout } from '@/components/layout'
import { LoadingState } from '@/components/loading-state'

import { getSelfSellablePlans } from './api'
import { AgentCustomersPanel } from './components/agent-customers-panel'
import type { DealerPlanOption } from './types'

/**
 * 我的客户 —— 只给经销商看。
 *
 * 看的是他名下有哪些人（绑过他的客户号，就归到他名下），以及给某一位改价、发额度。
 * 数据来自 GET /api/user/self/agent/customers（自己看自己，不用管理员权限）；
 * 菜单项由 use-sidebar-data.ts 按 subject_type 决定是否出现，路由上还挡了一道。
 */
export function DealerCustomers() {
  const { t } = useTranslation()
  const [plans, setPlans] = useState<DealerPlanOption[]>([])
  const [plansLoading, setPlansLoading] = useState(true)

  // 货架（能给他客户定的价）先拉一次，改价时就不再有第二次等待：
  // 空货架是正常状态（平台还没给他可卖的方案），不因此拦住名单。
  const fetchPlans = useCallback(async () => {
    try {
      const response = await getSelfSellablePlans()
      setPlans(response.success ? (response.data?.items ?? []) : [])
    } catch {
      setPlans([])
    } finally {
      setPlansLoading(false)
    }
  }, [])

  useEffect(() => {
    void fetchPlans()
  }, [fetchPlans])

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>{t('My Customers')}</SectionPageLayout.Title>
      <SectionPageLayout.Content>
        {plansLoading ? (
          <LoadingState size='lg' />
        ) : (
          <div className='flex w-full flex-col gap-4'>
            <p className='text-muted-foreground text-sm'>
              {t(
                'The customers under your name, and what each of them is priced at.'
              )}
            </p>

            {plans.length === 0 && (
              <EmptyState
                title={t('No discount plan is available to you yet')}
                description={t(
                  'Ask the platform to put discount plans on your shelf before you can price your customers.'
                )}
                size='md'
              />
            )}

            <AgentCustomersPanel plans={plans} />
          </div>
        )}
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
