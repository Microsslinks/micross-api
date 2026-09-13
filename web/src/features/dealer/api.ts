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
import { api } from '@/lib/api'

import type {
  AgentCustomer,
  AgentLedger,
  ApiResponse,
  CustomerCode,
  CustomerCodeIssuePayload,
  DealerPlanList,
} from './types'

/**
 * 取当前登录用户的经销商自助台账。
 *
 * 谁都能调这个地址，是不是经销商由后端判定：非经销商返回 `success: false`
 * （HTTP 仍为 200），所以调用方必须看 `success`，不能只看有没有报错。
 */
export async function getAgentLedger(): Promise<ApiResponse<AgentLedger>> {
  const res = await api.get('/api/user/self/agent/ledger')
  return res.data
}

/**
 * 经销商看自己签出去的客户号，最新的在前。
 *
 * `onlyUsable` 为真时只要还能发出去的号：他打开这一页想先看的是"手上还剩哪些号"，
 * 而不是一屏旧号。用完的、作废的、过期的都还在库里，切到全部就能翻到。
 */
export async function getSelfCustomerCodes(
  page = 1,
  pageSize = 20,
  onlyUsable = false
): Promise<
  ApiResponse<{
    items: CustomerCode[]
    total: number
    page: number
    page_size: number
  }>
> {
  const res = await api.get('/api/user/self/agent/codes', {
    // 页码参数名是 p：后端 common.GetPageQuery 读的是 p / ps / size，
    // 写成 page 会被忽略，翻到第二页时拿回来的还是第一页。
    params: {
      p: page,
      page_size: pageSize,
      ...(onlyUsable ? { usable: 1 } : {}),
    },
  })
  return res.data
}

/**
 * 经销商自己签一批客户号发给客户。
 */
export async function issueSelfCustomerCodes(
  payload: CustomerCodeIssuePayload
): Promise<ApiResponse<{ items: CustomerCode[] }>> {
  const res = await api.post('/api/user/self/agent/codes', payload)
  return res.data
}

/**
 * 作废自己的一个客户号（不删行，留着谁签过、被用了几次的痕迹）。
 */
export async function revokeSelfCustomerCode(
  codeId: number
): Promise<ApiResponse> {
  const res = await api.delete(`/api/user/self/agent/codes/${codeId}`)
  return res.data
}

/**
 * 他自己货架上的方案 + 平台给他定的零售折扣下限。
 * 低于下限的方案不会出现在这份清单里，所以界面不用再自己挡一道。
 */
export async function getSelfSellablePlans(): Promise<
  ApiResponse<DealerPlanList>
> {
  const res = await api.get('/api/user/self/agent/plans')
  return res.data
}

/**
 * 经销商看自己名下的客户（谁绑了他的号，就归到他名下）。
 */
export async function getSelfAgentCustomers(
  page = 1,
  pageSize = 20
): Promise<
  ApiResponse<{
    items: AgentCustomer[]
    total: number
    page: number
    page_size: number
  }>
> {
  const res = await api.get('/api/user/self/agent/customers', {
    params: { p: page, page_size: pageSize },
  })
  return res.data
}

/**
 * 给一位下属客户定价；传 0 撤掉自己定的价，让他回到客户号或平台给的价。
 * 成功后返回那位客户最新的台账行。
 */
export async function setSelfAgentCustomerDiscount(
  customerId: number,
  planId: number
): Promise<ApiResponse<AgentCustomer>> {
  const res = await api.put(
    `/api/user/self/agent/customers/${customerId}/discount`,
    { plan_id: planId }
  )
  return res.data
}

/**
 * 给一位下属客户发额度：钱从经销商自己的余额里转过去。
 * 成功后返回那位客户最新的台账行。
 */
export async function issueSelfAgentCustomerQuota(
  customerId: number,
  quota: number
): Promise<ApiResponse<AgentCustomer>> {
  const res = await api.post(
    `/api/user/self/agent/customers/${customerId}/quota`,
    { quota }
  )
  return res.data
}
