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
  ApiResponse,
  AuditCustomerPricingParams,
  CustomerAuditResult,
  CustomerPriceQueryResult,
  DiscountBinding,
  DiscountBindingListParams,
  DiscountBindingPayload,
  DiscountCustomer,
  DiscountModelList,
  DiscountModelListParams,
  DiscountModelListPayload,
  DiscountPage,
  DiscountPlan,
  DiscountPlanDetail,
  DiscountPlanListParams,
  DiscountPlanPayload,
  DiscountRouting,
  DiscountRoutingPayload,
  DiscountRule,
  DiscountRulePayload,
  DiscountSimulateResult,
  DiscountValidateParams,
  DiscountValidateResult,
  QueryCustomerPriceParams,
  SearchCustomersParams,
  SearchCustomersResponse,
  SimulateDiscountParams,
} from './types'

// ============================================================================
// Discount Simulation
// ============================================================================

/**
 * 试算「这个客户 + 这个模型」按几折、会走哪几条线路、每条赚多少。
 * 只读接口：后端不写任何表，也不改路由。
 */
export async function simulateDiscount(
  params: SimulateDiscountParams
): Promise<ApiResponse<DiscountSimulateResult>> {
  const { userId, model, channelId } = params
  const res = await api.get('/api/discount/admin/simulate', {
    params: {
      user_id: userId,
      model,
      channel_id: channelId,
    },
  })
  return res.data
}

/**
 * 客户档案核算：一个客户 + 整份模型清单，逐模型给出折扣、最便宜线路成本、
 * 毛利与结论，并汇总「这份报价能不能签」。只读接口。
 */
export async function auditCustomerPricing(
  params: AuditCustomerPricingParams
): Promise<ApiResponse<CustomerAuditResult>> {
  const { userId, models } = params
  const res = await api.post('/api/discount/admin/customer-audit', {
    user_id: userId,
    models,
  })
  return res.data
}

/**
 * 客户查价：这个客户身上挂了哪几套价、现价按几折、命中的是哪套、其余几套输在哪一层。
 * 只读接口。models 留空时只回价目本。
 */
export async function queryCustomerPrice(
  params: QueryCustomerPriceParams
): Promise<ApiResponse<CustomerPriceQueryResult>> {
  const { userId, models = [] } = params
  const res = await api.post('/api/discount/admin/price-query', {
    user_id: userId,
    models,
  })
  return res.data
}

// ============================================================================
// Model catalog (给「模型清单」零件提供可挑的模型名)
// ============================================================================

/**
 * 拿平台当前真正能提供服务的模型名清单——和模型广场同源
 * （GET /api/channel/models_enabled，取自 abilities 表里启用的模型，
 * 即挂了可用上游渠道的模型）。全量目录（/api/channel/models）不能用：
 * 那是代码里写死的"平台认识的模型"，没渠道也能挑，会选出做不了服务的模型。
 * 只读、不分页；零件拿它当「可挑的选项」，拿不到也不影响手输与粘贴，
 * 所以出错时由调用方降级，这里不做兜底吞错。
 */
export async function getModelNameOptions(): Promise<string[]> {
  const res = await api.get('/api/channel/models_enabled')
  const payload = res.data as ApiResponse<string[]>
  if (!payload?.success || !Array.isArray(payload.data)) {
    return []
  }
  return payload.data
    .map((name) => (typeof name === 'string' ? name.trim() : ''))
    .filter(Boolean)
}

// ============================================================================
// Customers (used to pick who to simulate for)
// ============================================================================

/**
 * 按关键字（用户名）搜索客户，给试算抽屉的下拉用。
 * 只取 id／用户名／显示名／分组四个字段，不依赖用户模块的类型。
 */
export async function searchCustomers(
  params: SearchCustomersParams = {}
): Promise<SearchCustomersResponse> {
  const { keyword = '', p = 1, page_size = 50 } = params
  const res = await api.get('/api/user/search', {
    params: {
      keyword,
      p,
      page_size,
    },
  })
  const payload = res.data as SearchCustomersResponse
  const items = payload?.data?.items
  if (!Array.isArray(items)) {
    return { ...payload, data: payload?.data }
  }
  return {
    ...payload,
    data: payload.data
      ? {
          ...payload.data,
          items: items.map(
            (item): DiscountCustomer => ({
              id: item.id,
              username: item.username,
              display_name: item.display_name,
              group: item.group,
            })
          ),
        }
      : undefined,
  }
}

// ============================================================================
// Plans
// ============================================================================

/** 方案列表；keyword 按方案名与备注模糊匹配。 */
export async function getDiscountPlans(
  params: DiscountPlanListParams = {}
): Promise<ApiResponse<DiscountPage<DiscountPlan>>> {
  const res = await api.get('/api/discount/admin/plans', { params })
  return res.data
}

/** 方案详情：方案本身 + 它的全部规则（一次拿全，省一次请求）。 */
export async function getDiscountPlan(
  planId: number
): Promise<ApiResponse<DiscountPlanDetail>> {
  const res = await api.get(`/api/discount/admin/plans/${planId}`)
  return res.data
}

export async function createDiscountPlan(
  payload: DiscountPlanPayload
): Promise<ApiResponse<DiscountPlan>> {
  const res = await api.post('/api/discount/admin/plans', payload)
  return res.data
}

export async function updateDiscountPlan(
  planId: number,
  payload: DiscountPlanPayload
): Promise<ApiResponse<DiscountPlan>> {
  const res = await api.put(`/api/discount/admin/plans/${planId}`, payload)
  return res.data
}

/** 只改启用／停用，不动折扣与规则。 */
export async function updateDiscountPlanStatus(
  planId: number,
  status: number
): Promise<ApiResponse<DiscountPlan>> {
  const res = await api.patch(`/api/discount/admin/plans/${planId}/status`, {
    status,
  })
  return res.data
}

export async function deleteDiscountPlan(
  planId: number
): Promise<ApiResponse<null>> {
  const res = await api.delete(`/api/discount/admin/plans/${planId}`)
  return res.data
}

/**
 * 保存前检查：这套折扣会不会亏。
 * 只读接口——后端不写表、也不阻断保存，界面负责把话说清楚。
 */
export async function validateDiscountPlan(
  planId: number,
  params: DiscountValidateParams = {}
): Promise<ApiResponse<DiscountValidateResult>> {
  const body: { user_id?: number; channel_id?: number } = {}
  if (params.userId) body.user_id = params.userId
  if (params.channelId) body.channel_id = params.channelId
  const res = await api.post(
    `/api/discount/admin/plans/${planId}/validate`,
    body
  )
  return res.data
}

// ============================================================================
// Rules
// ============================================================================

export async function getDiscountRules(
  planId: number
): Promise<ApiResponse<DiscountRule[]>> {
  const res = await api.get(`/api/discount/admin/plans/${planId}/rules`)
  return res.data
}

export async function createDiscountRule(
  planId: number,
  payload: DiscountRulePayload
): Promise<ApiResponse<DiscountRule>> {
  const res = await api.post(
    `/api/discount/admin/plans/${planId}/rules`,
    payload
  )
  return res.data
}

export async function updateDiscountRule(
  ruleId: number,
  payload: DiscountRulePayload
): Promise<ApiResponse<DiscountRule>> {
  const res = await api.put(`/api/discount/admin/rules/${ruleId}`, payload)
  return res.data
}

export async function deleteDiscountRule(
  ruleId: number
): Promise<ApiResponse<null>> {
  const res = await api.delete(`/api/discount/admin/rules/${ruleId}`)
  return res.data
}

// ============================================================================
// Model lists (可复用的模型名单，给「模型清单」零件一键带入用)
// ============================================================================

/** 清单列表；keyword 按清单名与备注模糊匹配。 */
export async function getDiscountModelLists(
  params: DiscountModelListParams = {}
): Promise<ApiResponse<DiscountPage<DiscountModelList>>> {
  const res = await api.get('/api/discount/admin/model-lists', { params })
  return res.data
}

export async function createDiscountModelList(
  payload: DiscountModelListPayload
): Promise<ApiResponse<DiscountModelList>> {
  const res = await api.post('/api/discount/admin/model-lists', payload)
  return res.data
}

export async function updateDiscountModelList(
  listId: number,
  payload: DiscountModelListPayload
): Promise<ApiResponse<DiscountModelList>> {
  const res = await api.put(
    `/api/discount/admin/model-lists/${listId}`,
    payload
  )
  return res.data
}

/** 删掉一份清单：只影响「以后还能不能一键带入」，已填进格子的模型名不受影响。 */
export async function deleteDiscountModelList(
  listId: number
): Promise<ApiResponse<null>> {
  const res = await api.delete(`/api/discount/admin/model-lists/${listId}`)
  return res.data
}

// ============================================================================
// Bindings
// ============================================================================

export async function getDiscountBindings(
  params: DiscountBindingListParams = {}
): Promise<ApiResponse<DiscountPage<DiscountBinding>>> {
  const res = await api.get('/api/discount/admin/bindings', { params })
  return res.data
}

export async function createDiscountBinding(
  payload: DiscountBindingPayload
): Promise<ApiResponse<DiscountBinding>> {
  const res = await api.post('/api/discount/admin/bindings', payload)
  return res.data
}

export async function deleteDiscountBinding(
  bindingId: number
): Promise<ApiResponse<null>> {
  const res = await api.delete(`/api/discount/admin/bindings/${bindingId}`)
  return res.data
}

// ============================================================================
// Customer routing policy（择优策略 + 允许走亏损线路的开关）
// ============================================================================

/**
 * 读一个客户当前生效的路由策略。没单独配过的客户也会返回一份：configured 为 false、
 * routing_strategy 是默认的 margin——「没配过」是常态，不是错误。
 */
export async function getDiscountRouting(
  userId: number
): Promise<ApiResponse<DiscountRouting>> {
  const res = await api.get('/api/discount/admin/routing', {
    params: { user_id: userId },
  })
  return res.data
}

/** 写入（或更新）一个客户的路由策略。 */
export async function saveDiscountRouting(
  payload: DiscountRoutingPayload
): Promise<ApiResponse<null>> {
  const res = await api.put('/api/discount/admin/routing', payload)
  return res.data
}

/** 删掉这个客户的例外配置，回到默认口径（毛利优先、不允许走亏损线路）。 */
export async function resetDiscountRouting(
  userId: number
): Promise<ApiResponse<null>> {
  const res = await api.delete('/api/discount/admin/routing', {
    params: { user_id: userId },
  })
  return res.data
}
