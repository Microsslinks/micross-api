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

import { SectionPageLayout } from '@/components/layout'

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
 * 下面是已经挂到他名下的人（客户绑完号就出现在这里），以及给谁发额度。
 *
 * 数据来自 /api/user/self/agent/*（自己看自己，不用管理员权限）；
 * 菜单项由 use-sidebar-data.ts 按 subject_type 决定是否出现，路由上还挡了一道。
 */
export function DealerCustomers() {
  const { t } = useTranslation()
  const [plans, setPlans] = useState<DealerPlanOption[]>([])

  // 定价已经全部交给平台管理员（业务口径 2026-09-14），货架不再参与任何操作；
  // 拉它只是为了客户号列表上能把历史号挂过的方案显示成名字而不是编号。
  const fetchPlans = useCallback(async () => {
    try {
      const response = await getSelfSellablePlans()
      setPlans(response.success ? (response.data?.items ?? []) : [])
    } catch {
      setPlans([])
    }
  }, [])

  useEffect(() => {
    void fetchPlans()
  }, [fetchPlans])

  // 签号那块要的三个动作。
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
        <div className='flex w-full flex-col gap-6'>
          <CustomerCodesPanel source={codesSource} />

          <AgentCustomersPanel />
        </div>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
