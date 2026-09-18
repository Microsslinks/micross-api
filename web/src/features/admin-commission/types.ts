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

import type { ApiResponse, CommissionRecord } from '@/features/wallet/types'

// ============================================================================
// Admin Commission (task-20 §20.8)
//
// §20.6 加了 commission_records.Reversed / ReversedAt / ReversedBy /
// ReverseReason 字段后，admin UI 需要看到这些字段并提供撤销入口。
// 这里单独定义 AdminCommissionRecord 而不是直接扩展后者 —— user 端
// CommissionRecord 保留窄字段（避免无意中泄露 audit 信息给最终用户）。
// ============================================================================

/**
 * 单条 commission_records 全字段视图（admin）。
 * 与 model/commission.go:CommissionRecord 1:1 对齐，task-20 §20.6 加的
 * 撤销四字段也都在这里。
 */
export interface AdminCommissionRecord extends CommissionRecord {
  reversed: boolean
  reversed_at: number
  reversed_by: number
  reverse_reason: string
}

/**
 * GET /api/admin/commission/records 响应载荷。
 * data.items 用 AdminCommissionRecord 而不是窄 CommissionRecord，
 * 是 admin 列表区别于 user 列表的核心。
 */
export interface AdminCommissionRecordsResponse extends ApiResponse<{
  items: AdminCommissionRecord[]
  page: number
  page_size: number
  total: number
}> {}

/**
 * POST /api/admin/commission/records/:id/reverse 响应载荷。
 * 成功时 data 是更新后的 record（含 Reversed=true 等四字段）。
 * 失败时 HTTP 409 / 200 + success=false，由前端按 status code 判断。
 */
export interface AdminReverseCommissionResponse extends ApiResponse<AdminCommissionRecord> {}

/**
 * 列表过滤参数。与 controller/commission.go adminListCommissionRecordsRequest
 * 字段一一对应：query string 用 snake_case，后端按 c.Query 解析。
 */
export interface AdminCommissionListFilters {
  page?: number
  page_size?: number
  inviter_id?: number
  invitee_id?: number
  reversed?: boolean
  breach?: boolean
}