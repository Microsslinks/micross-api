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

import type { AgentLedger, ApiResponse } from './types'

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
