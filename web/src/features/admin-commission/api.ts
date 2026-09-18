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
  AdminCommissionListFilters,
  AdminCommissionRecordsResponse,
  AdminReverseCommissionResponse,
} from './types'

// ============================================================================
// Admin Commission API (task-20 §20.8)
//
// 端点对应 (router/api-router.go adminCommissionRoute):
//   GET    /api/admin/commission/records
//   POST   /api/admin/commission/records/:id/reverse
//
// 与 user 端的 /api/user/aff/commission/* 路径完全分离 —— admin 权限校验
// 由 adminCommissionRoute.Use(middleware.AdminAuth()) 保证 (router 层)。
// ============================================================================

/**
 * 列出 commission_records（分页 + 过滤）。
 *
 * @param filters - 分页 / inviter / invitee / reversed / breach 等过滤
 *
 * 空 filters 等价于 page=1, page_size=20 的无过滤查询。
 * 后端对 query string 用 strconv.Atoi + strconv.ParseBool 解析：
 *   - filters.page 缺省 1
 *   - filters.page_size 缺省 20，最大 200
 *   - filters.reversed / breach 三态：true / false / undefined（全部）
 */
export async function adminListCommissionRecords(
  filters: AdminCommissionListFilters = {},
): Promise<AdminCommissionRecordsResponse> {
  const res = await api.get('/api/admin/commission/records', {
    params: filters,
  })
  return res.data
}

/**
 * 撤销单条 commission_records。
 *
 * @param recordId  - commission_records.id
 * @param reason    - 撤销原因（必填，传 "" 时后端 service 会兜底）
 *
 * 响应状态码约定：
 *   - 200 + success=true → 撤销成功，data 含更新后的 record
 *   - 200 + success=false + message="该返佣记录已被撤销" → 二次撤销
 *     （controller 不返 HTTP 409 而走 success=false? 看 controller 实现）
 *   - HTTP 409 → 二次撤销（新 controller 路径）
 *   - 200 + success=false + 非空 message → 其他错误（record 不存在等）
 */
export async function adminReverseCommissionRecord(
  recordId: number,
  reason: string,
): Promise<AdminReverseCommissionResponse> {
  const res = await api.post(
    `/api/admin/commission/records/${recordId}/reverse`,
    { reason },
  )
  return res.data
}