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
import { z } from 'zod'

import type { AdminPermissionMatrix } from '@/lib/admin-permissions'

// ============================================================================
// User Schema & Types
// ============================================================================

/** User status: 1 = enabled, 2 = disabled, 3+ = other states */
export const userStatusSchema = z.number()
export type UserStatus = z.infer<typeof userStatusSchema>

/** User role: 1 = common user, 10 = admin, 100 = root */
export const userRoleSchema = z.number()
export type UserRole = z.infer<typeof userRoleSchema>

export const userSchema = z.object({
  id: z.number(),
  username: z.string(),
  display_name: z.string(),
  password: z.string().optional(),
  github_id: z.string().optional(),
  oidc_id: z.string().optional(),
  wechat_id: z.string().optional(),
  telegram_id: z.string().optional(),
  email: z.string().optional(),
  quota: z.number(),
  used_quota: z.number(),
  request_count: z.number(),
  group: z.string(),
  aff_code: z.string().optional(),
  aff_count: z.number().optional(),
  aff_quota: z.number().optional(),
  aff_history_quota: z.number().optional(),
  inviter_id: z.number().optional(),
  linux_do_id: z.string().optional(),
  status: userStatusSchema,
  role: userRoleSchema,
  created_at: z.number().optional(),
  updated_at: z.number().optional(),
  last_login_at: z.number().optional(),
  DeletedAt: z.any().nullable().optional(),
  remark: z.string().optional(),
  /** 用户主体类型：individual = 普通客户，agent = 经销商 */
  subject_type: z.string().optional(),
  /** 绑定的折扣方案 id；大于 0 表示这个客户有专属价 */
  discount_plan_id: z.number().optional(),
  admin_permissions: z
    .record(z.string(), z.record(z.string(), z.boolean()))
    .optional(),
  /** 用户级侧边栏可见性覆盖层（JSON 字符串），由管理端或用户本人在个人中心维护 */
  sidebar_modules: z.string().optional(),
})
export type User = z.infer<typeof userSchema>

export const userListSchema = z.array(userSchema)

// ============================================================================
// API Request/Response Types
// ============================================================================

/** Generic API response */
export interface ApiResponse<T = unknown> {
  success: boolean
  message?: string
  data?: T
}

export type UserSortBy =
  | 'id'
  | 'username'
  | 'quota'
  | 'group'
  | 'created_at'
  | 'last_login_at'

export type UserSortOrder = 'asc' | 'desc'

export interface GetUsersParams {
  p?: number
  page_size?: number
  sort_by?: UserSortBy
  sort_order?: UserSortOrder
}

export interface GetUsersResponse {
  success: boolean
  message?: string
  data?: {
    items: User[]
    total: number
    page: number
    page_size: number
  }
}

export interface SearchUsersParams {
  keyword?: string
  group?: string
  role?: string
  status?: string
  customer_type?: string
  p?: number
  page_size?: number
  sort_by?: UserSortBy
  sort_order?: UserSortOrder
}

export interface UserFormData {
  username: string
  display_name: string
  password?: string
  role?: number // Only used when creating user
  quota?: number // Only used when updating user
  group?: string // Only used when updating user
  remark?: string // Only used when updating user
  admin_permissions?: AdminPermissionMatrix
  /** 序列化后的用户级侧边栏覆盖层；省略时后端不改动该字段 */
  sidebar_modules?: string
}

export type ManageUserAction =
  | 'promote'
  | 'demote'
  | 'enable'
  | 'disable'
  | 'delete'
  | 'add_quota'

export type QuotaAdjustMode = 'add' | 'subtract' | 'override'

export interface ManageUserQuotaPayload {
  id: number
  action: 'add_quota'
  mode: QuotaAdjustMode
  value: number
}

/**
 * 设为经销商的入参。
 *
 * 平台只定一档毛利：每个模型的拿货价由系统按「该模型最便宜一条线路的成本 × 加价率」
 * 各自算出来，所以这里没有逐模型的数字可填。
 */
export interface SetAgentPayload {
  /** 平台加价率，如 "1.1" 表示在自采成本之上加 10%；下限 "1.05" */
  markup_ratio: string
  remark?: string
}

/**
 * 经销商的经营档案：平台给他定的那几个经营参数。
 *
 * 设成经销商时只填加价率，这份档案是之后调整用的——加价率、零售折扣下限、
 * 能不能给下属客户发额度、备注。改这些不影响他的角色、余额和折扣方案。
 */
export interface AgentProfile {
  user_id: number
  /** 平台加价率，是乘数不是百分数："1.1" 表示加价 10% */
  markup_ratio: string
  /** 零售折扣下限，"0" 表示平台不限制他的报价 */
  min_discount: string
  /** 1 = 允许给下属客户发额度，0 = 停发 */
  issue_quota_enabled: number
  remark: string
}

/** 改档案的入参：哪个字段不传就是这次不动它。 */
export interface AgentProfilePayload {
  markup_ratio?: string
  min_discount?: string
  issue_quota_enabled?: number
  remark?: string
}

/** 经销商拿货价目表里的一行：这个模型他按几折拿货，拿去给客户报价的依据。 */
export interface AgentWholesaleQuoteItem {
  model_name: string
  /** 他的拿货折扣 = 成本 × 加价率 */
  discount: string
  /** 这个折扣是从哪条线路的成本算出来的 */
  cost_ratio: string
  /** 那条线路的名字 */
  channel_name: string
}

// ============================================================================
// Dialog Types
// ============================================================================

export type UsersDialogType = 'create' | 'update' | 'delete'
