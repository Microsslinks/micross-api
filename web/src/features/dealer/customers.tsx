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
import { useCallback, useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { EmptyState } from '@/components/empty-state'
import { SectionPageLayout } from '@/components/layout'
import { LoadingState } from '@/components/loading-state'

import {
  getSelfCustomerCodes,
  getSelfSellablePlans,
  issueSelfCustomerCodes,
  revokeSelfCustomerCode,
} from './api'
import { AgentCustomersPanel } from './components/agent-customers-panel'
import {
  CustomerCodesPanel,
  type CustomerCodesSource,
} from './components/customer-codes-panel'
import type { DealerPlanOption } from './types'

/**
 * 我的客户 —— 只给经销商看。
 *
 * 分上下两块，顺序就是这门生意的顺序：上面是手上的客户号（先把号给客户），
 * 下面是已经挂到他名下的人（客户绑完号就出现在这里），以及给谁改价、发额度。
 *
 * 数据来自 /api/user/self/agent/*（自己看自己，不用管理员权限）；
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

  // 签号那块要的三个动作。货架变了它也跟着变，所以签号弹窗里的方案永远是最新的。
  const codesSource = useMemo<CustomerCodesSource>(
    () => ({
      list: async (page, onlyUsable) => {
        const response = await getSelfCustomerCodes(page, 20, onlyUsable)
        return {
          items: response.success ? (response.data?.items ?? []) : [],
          total: response.success ? (response.data?.total ?? 0) : 0,
        }
      },
      create: async (payload) => {
        const response = await issueSelfCustomerCodes(payload)
        return {
          success: response.success,
          message: response.message,
          items: response.data?.items,
        }
      },
      revoke: async (codeId) => {
        const response = await revokeSelfCustomerCode(codeId)
        return { success: response.success, message: response.message }
      },
      plans,
    }),
    [plans]
  )

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>{t('My Customers')}</SectionPageLayout.Title>
      <SectionPageLayout.Content>
        {plansLoading ? (
          <LoadingState size='lg' />
        ) : (
          <div className='flex w-full flex-col gap-6'>
            {plans.length === 0 && (
              <EmptyState
                title={t('No discount plan is available to you yet')}
                description={t(
                  'Ask the platform to put discount plans on your shelf before you can price your customers.'
                )}
                size='md'
              />
            )}

            <CustomerCodesPanel source={codesSource} />

            <AgentCustomersPanel plans={plans} />
          </div>
        )}
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
