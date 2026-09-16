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
import type {
  CustomerCode,
  CustomerCodeIssuePayload,
} from '@/features/dealer/types'
import type { PermissionCatalog } from '@/lib/admin-permissions'
import { api } from '@/lib/api'
import type { CustomOAuthBinding } from '@/lib/oauth'

import type {
  User,
  GetUsersParams,
  GetUsersResponse,
  SearchUsersParams,
  UserFormData,
  ManageUserAction,
  ManageUserQuotaPayload,
  SetAgentPayload,
  AgentProfile,
  AgentProfilePayload,
  AgentWholesaleQuoteItem,
  ApiResponse,
} from './types'

// ============================================================================
// User Management APIs
// ============================================================================

/**
 * Get paginated users list
 */
export async function getUsers(
  params: GetUsersParams = {}
): Promise<GetUsersResponse> {
  const { p = 1, page_size = 10, sort_by, sort_order } = params
  const res = await api.get('/api/user/', {
    params: {
      p,
      page_size,
      sort_by,
      sort_order,
    },
  })
  return res.data
}

/**
 * Search users by keyword or group
 */
export async function searchUsers(
  params: SearchUsersParams
): Promise<GetUsersResponse> {
  const {
    keyword = '',
    group = '',
    role = '',
    status = '',
    customer_type = '',
    p = 1,
    page_size = 10,
    sort_by,
    sort_order,
  } = params
  const queryParams = new URLSearchParams()
  queryParams.set('keyword', keyword)
  queryParams.set('group', group)
  if (role) queryParams.set('role', role)
  if (status) queryParams.set('status', status)
  if (customer_type) queryParams.set('customer_type', customer_type)
  queryParams.set('p', String(p))
  queryParams.set('page_size', String(page_size))
  if (sort_by) queryParams.set('sort_by', sort_by)
  if (sort_order) queryParams.set('sort_order', sort_order)
  const res = await api.get(`/api/user/search?${queryParams.toString()}`)
  return res.data
}

/**
 * Get single user by ID
 */
export async function getUser(id: number): Promise<ApiResponse<User>> {
  const res = await api.get(`/api/user/${id}`)
  return res.data
}

/**
 * Create a new user
 */
export async function createUser(
  data: UserFormData
): Promise<ApiResponse<User>> {
  const res = await api.post('/api/user/', data)
  return res.data
}

/**
 * Update an existing user
 */
export async function updateUser(
  data: UserFormData & { id: number }
): Promise<ApiResponse<Partial<User>>> {
  const res = await api.put('/api/user/', data)
  return res.data
}

/**
 * Delete a single user (hard delete)
 */
export async function deleteUser(id: number): Promise<ApiResponse> {
  const res = await api.delete(`/api/user/${id}/`)
  return res.data
}

/**
 * Manage user (promote, demote, enable, disable, delete)
 */
export async function manageUser(
  id: number,
  action: ManageUserAction
): Promise<ApiResponse<Partial<User>>> {
  const res = await api.post('/api/user/manage', { id, action })
  return res.data
}

/**
 * Mark a user as a dealer (a business identity on top of the user, not a role).
 * The platform only sets one markup; each model is priced from its own cost.
 */
export async function setUserAsAgent(
  id: number,
  payload: SetAgentPayload
): Promise<ApiResponse<Partial<User>>> {
  const res = await api.post(`/api/user/${id}/agent`, payload)
  return res.data
}

/**
 * List what this dealer pays per model at the given markup.
 *
 * The dialog calls it whenever the markup changes so the chips follow the number.
 * Models whose upstream lines have no purchase discount recorded are left out.
 */
export async function getAgentWholesaleQuote(
  id: number,
  markupRatio: string
): Promise<
  ApiResponse<{ markup_ratio: string; items: AgentWholesaleQuoteItem[] }>
> {
  const res = await api.get(`/api/user/${id}/agent/wholesale`, {
    params: { markup_ratio: markupRatio },
  })
  return res.data
}

/**
 * Remove dealer status from a user (keeps balance, group and discount plan)
 */
export async function unsetUserAsAgent(
  id: number
): Promise<ApiResponse<Partial<User>>> {
  const res = await api.delete(`/api/user/${id}/agent`)
  return res.data
}

/**
 * Read a dealer's operating profile (markup, retail discount floor, quota issuing
 * switch, remark) so the "Dealer Settings" dialog can be filled in.
 */
export async function getAgentProfile(
  id: number
): Promise<ApiResponse<AgentProfile>> {
  const res = await api.get(`/api/user/${id}/agent/profile`)
  return res.data
}

/**
 * Update a dealer's operating profile. Only the fields sent are changed.
 */
export async function updateAgentProfile(
  id: number,
  payload: AgentProfilePayload
): Promise<ApiResponse<AgentProfile>> {
  const res = await api.put(`/api/user/${id}/agent`, payload)
  return res.data
}

/**
 * List the customer codes this dealer has issued.
 *
 * `onlyUsable` keeps just the codes that can still be handed out.
 */
export async function getAgentCustomerCodes(
  id: number,
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
  const res = await api.get(`/api/user/${id}/agent/codes`, {
    // 页码参数名是 p：后端 common.GetPageQuery 读的是 p / ps / size，
    // 写成 page 会被忽略，翻页时拿回来的还是第一页。
    params: {
      p: page,
      page_size: pageSize,
      ...(onlyUsable ? { usable: 1 } : {}),
    },
  })
  return res.data
}

/**
 * Issue customer codes on behalf of this dealer (an operator doing it for him).
 */
export async function issueAgentCustomerCodes(
  id: number,
  payload: CustomerCodeIssuePayload
): Promise<ApiResponse<{ items: CustomerCode[] }>> {
  const res = await api.post(`/api/user/${id}/agent/codes`, payload)
  return res.data
}

/**
 * Revoke one of this dealer's customer codes.
 */
export async function revokeAgentCustomerCode(
  id: number,
  codeId: number
): Promise<ApiResponse> {
  const res = await api.delete(`/api/user/${id}/agent/codes/${codeId}`)
  return res.data
}

/**
 * Adjust user quota atomically (add/subtract/override)
 */
export async function adjustUserQuota(
  payload: ManageUserQuotaPayload
): Promise<ApiResponse<Partial<User>>> {
  const res = await api.post('/api/user/manage', payload)
  return res.data
}

/**
 * Reset user's Passkey registration
 */
export async function resetUserPasskey(id: number): Promise<ApiResponse> {
  const res = await api.delete(`/api/user/${id}/reset_passkey`)
  return res.data
}

/**
 * Reset user's Two-Factor Authentication setup
 */
export async function resetUserTwoFA(id: number): Promise<ApiResponse> {
  const res = await api.delete(`/api/user/${id}/2fa`)
  return res.data
}

/**
 * task-16：超管重置某用户的邀请人。
 *
 * - inviter_id=0 表示清空；>0 表示新的邀请人
 * - reason 后端硬约束 10-200 字符（前端再挡一道，提交按钮 disabled 直到合法）
 * - 仅超管（role===100）能调；其它角色会被后端守卫直接打回
 */
export async function resetUserInviter(
  id: number,
  payload: { inviter_id: number; reason: string }
): Promise<ApiResponse<Partial<User>>> {
  const res = await api.post(`/api/user/${id}/reset_inviter`, payload)
  return res.data
}

/**
 * Get all available groups
 */
export async function getGroups(): Promise<ApiResponse<string[]>> {
  const res = await api.get('/api/group/')
  return res.data
}

/**
 * Get the permission catalog (resources, actions, and role baselines).
 * Source of truth lives in the backend authz package.
 */
export async function getPermissionCatalog(): Promise<PermissionCatalog> {
  const res = await api.get('/api/authz/catalog')
  return {
    resources: res.data?.data?.resources ?? [],
    roles: res.data?.data?.roles ?? [],
  }
}

// ============================================================================
// Admin Binding Management APIs
// ============================================================================

/**
 * Get user's custom OAuth bindings (admin)
 */
export async function getUserOAuthBindings(
  userId: number
): Promise<ApiResponse<CustomOAuthBinding[]>> {
  const res = await api.get(`/api/user/${userId}/oauth/bindings`)
  return res.data
}

/**
 * Clear a user's built-in binding (admin)
 */
export async function adminClearUserBinding(
  userId: number,
  bindingType: string
): Promise<ApiResponse> {
  const res = await api.delete(`/api/user/${userId}/bindings/${bindingType}`)
  return res.data
}

/**
 * Unbind custom OAuth for a user (admin)
 */
export async function adminUnbindCustomOAuth(
  userId: number,
  providerId: number
): Promise<ApiResponse> {
  const res = await api.delete(
    `/api/user/${userId}/oauth/bindings/${providerId}`
  )
  return res.data
}
