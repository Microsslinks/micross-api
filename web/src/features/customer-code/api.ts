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

import type { ApiResponse, CustomerBinding } from './types'

/**
 * 我此刻的归属经销商与生效折扣。
 */
export async function getMyAgentBinding(): Promise<
  ApiResponse<CustomerBinding>
> {
  const res = await api.get('/api/user/self/agent/binding')
  return res.data
}

/**
 * 用经销商给的客户号绑定自己：归属到那位经销商，并按号上的折扣方案计价。
 * 拒绝的理由由后端给（号不存在/已作废/已过期/次数用完/已归属别人），直接透出给用户看。
 */
export async function bindCustomerCode(
  code: string
): Promise<ApiResponse<CustomerBinding>> {
  const res = await api.post('/api/user/self/agent/bind', { code })
  return res.data
}
