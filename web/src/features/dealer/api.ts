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
 */
export async function getSelfCustomerCodes(
  page = 1,
  pageSize = 20
): Promise<
  ApiResponse<{
    items: CustomerCode[]
    total: number
    page: number
    page_size: number
  }>
> {
  const res = await api.get('/api/user/self/agent/codes', {
    params: { page, page_size: pageSize },
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
