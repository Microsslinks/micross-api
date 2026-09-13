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
// ============================================================================
// Enums (mirrors the backend constants; keep the string values in sync)
// ============================================================================

/** 这一单的折扣是从哪一层取到的；对应 `model.DiscountResolvedFrom*`。 */
export const DISCOUNT_SOURCE = {
  MODEL: 'model',
  VENDOR: 'vendor',
  PLAN_BASE: 'plan_base',
  DEFAULT: 'default',
} as const

export type DiscountSource =
  (typeof DISCOUNT_SOURCE)[keyof typeof DISCOUNT_SOURCE]

/** 规则作用范围；对应 `model.DiscountScope*`。 */
export const DISCOUNT_SCOPE = {
  MODEL: 'model',
  VENDOR: 'vendor',
} as const

/** 方案状态；对应 `model.DiscountStatus*`。 */
export const DISCOUNT_PLAN_STATUS = {
  DISABLED: 0,
  ENABLED: 1,
} as const

/** 方案归属；对应 `model.DiscountOwner*`。 */
export const DISCOUNT_OWNER = {
  PLATFORM: 'platform',
  AGENT: 'agent',
} as const

/** 计费模式；对应 `model.DiscountBilling*`。 */
export const DISCOUNT_BILLING_MODE = {
  USAGE: 'usage',
  SUBSCRIPTION: 'subscription',
  FREE: 'free',
} as const

/** 绑定主体；对应 `model.DiscountSubject*`。 */
export const DISCOUNT_SUBJECT = {
  USER: 'user',
  AGENT: 'agent',
} as const

/** 绑定来源；对应 `model.DiscountSource*`。 */
export const DISCOUNT_BINDING_SOURCE = {
  MANUAL: 'manual',
  SUBSCRIPTION: 'subscription',
  CUSTOMER_CODE: 'customer_code',
  MIGRATION: 'migration',
} as const

/**
 * 保存前检查里后端给的原因码；对应 `service.DiscountViolationReason*`。
 * 界面上认识的翻译成中文，没见过的原样显示。
 */
export const DISCOUNT_VIOLATION_REASON = {
  BELOW_MIN_DISCOUNT: 'below_min_discount',
  MIN_ABOVE_BASE: 'min_above_base',
  COST_BREACH: 'cost_breach',
  NO_CHANNEL: 'no_channel',
} as const

/** 客户路由策略的择优口径；对应 `model.RoutingStrategy*`。 */
export const DISCOUNT_ROUTING_STRATEGY = {
  /** 默认：谁毛利高走谁。 */
  MARGIN: 'margin',
  /** 稳定优先：走上游优先级高的线路，不看毛利。 */
  PRIORITY: 'priority',
} as const

export type DiscountRoutingStrategy =
  (typeof DISCOUNT_ROUTING_STRATEGY)[keyof typeof DISCOUNT_ROUTING_STRATEGY]

// ============================================================================
// Simulation result (mirrors service.DiscountSimulateResult)
// ============================================================================

export interface DiscountSimulateUser {
  id: number
  username: string
  group: string
}

export interface DiscountSimulatePlan {
  id: number
  name: string
  status: number
}

export interface DiscountSimulateRule {
  id: number
  scope_type: string
  scope_value: string
  discount: string
  priority: number
}

export interface DiscountSimulateResolution {
  /** 固定 6 位小数字符串，例如 `"0.300000"`。 */
  discount: string
  source: string
  matched_rule: DiscountSimulateRule | null
}

/**
 * 一条候选线路的试算结果。
 * 没录进货折扣时三个值字段都是 `null`——「不知道」不能显示成「赚 0 元」。
 */
export interface DiscountSimulateChannel {
  channel_id: number
  channel_name: string
  cost_ratio: string | null
  gross_margin: string | null
  passes_floor: boolean | null
}

export interface DiscountSimulateResult {
  user: DiscountSimulateUser
  model: string
  vendor: string
  plan: DiscountSimulatePlan | null
  resolution: DiscountSimulateResolution
  /** 只要有一条候选线路没录进货折扣就是 false。 */
  cost_known: boolean
  /** 后台「毛利底线」，固定 6 位小数字符串。 */
  min_margin_ratio: string
  channels: DiscountSimulateChannel[]
  /** 后端原文提示（中文），直接展示给运营看。 */
  warnings: string[]
}

// ============================================================================
// Customers (minimal shape needed to pick one for a simulation)
// ============================================================================

export interface DiscountCustomer {
  id: number
  username: string
  display_name: string
  group: string
}

// ============================================================================
// API Request/Response Types
// ============================================================================

/** Generic API response */
export interface ApiResponse<T = unknown> {
  success: boolean
  message?: string
  data?: T
}

export interface SimulateDiscountParams {
  userId: number
  model: string
  channelId?: number
}

export interface SearchCustomersParams {
  keyword?: string
  p?: number
  page_size?: number
}

/** 客户搜索结果：只看接口里我们真正会用的那几个字段，避免依赖用户模块类型。 */
export interface SearchCustomersResponse {
  success: boolean
  message?: string
  data?: {
    items: DiscountCustomer[]
    total: number
    page: number
    page_size: number
  }
}

// ============================================================================
// Plans / Rules / Bindings (mirrors the model layer)
// ============================================================================

/** 分页返回的形状（后端 `common.PageInfo`）。 */
export interface DiscountPage<T> {
  items: T[]
  total: number
  page: number
  page_size: number
}

export interface DiscountPlan {
  id: number
  name: string
  owner_type: string
  owner_id: number
  /** 固定 6 位小数字符串，例如 `"0.300000"`。 */
  base_discount: string
  min_discount: string
  billing_mode: string
  commission_ratio: string
  status: number
  remark: string
  created_at: number
  updated_at: number
}

export interface DiscountRule {
  id: number
  plan_id: number
  scope_type: string
  scope_value: string
  discount: string
  priority: number
  status: number
  created_at: number
  updated_at: number
}

export interface DiscountBinding {
  id: number
  subject_type: string
  subject_id: number
  plan_id: number
  effective_from: number
  effective_to: number
  source: string
  status: number
  created_at: number
  updated_at: number
}

/** 方案详情接口的返回：方案本身 + 它的全部规则。 */
export interface DiscountPlanDetail {
  plan: DiscountPlan
  rules: DiscountRule[]
}

export interface DiscountPlanPayload {
  name: string
  owner_type: string
  owner_id: number
  base_discount: string
  min_discount: string
  billing_mode: string
  commission_ratio: string
  status: number
  remark: string
}

export interface DiscountRulePayload {
  scope_type: string
  scope_value: string
  discount: string
  priority: number
  status: number
}

export interface DiscountBindingPayload {
  subject_type: string
  subject_id: number
  plan_id: number
  effective_from: number
  effective_to: number
  source: string
}

export interface DiscountPlanListParams {
  page?: number
  page_size?: number
  keyword?: string
  owner_type?: string
  status?: number
}

export interface DiscountBindingListParams {
  page?: number
  page_size?: number
  subject_type?: string
  subject_id?: number
  plan_id?: number
}

export interface DiscountValidateParams {
  userId?: number
  channelId?: number
}

// ============================================================================
// Pre-save validation (mirrors service.DiscountValidateResult)
// ============================================================================

export interface DiscountValidateViolation {
  scope_type: string
  scope_value: string
  discount: string
  reason: string
  /** 后端原文说明；界面上不认识原因码时至少还能看这句。 */
  detail: string
  available_channels: string[]
}

export interface DiscountValidateResult {
  passed: boolean
  min_margin_ratio: string
  violations: DiscountValidateViolation[]
  warnings: string[]
}

// ============================================================================
// Customer routing policy (mirrors service.DiscountRoutingView)
// ============================================================================

/**
 * 一个客户的路由策略。没单独配过的客户也会拿到一份：configured 为 false、
 * routing_strategy 是默认的 margin、allow_cost_breach 是 false。
 */
export interface DiscountRouting {
  user_id: number
  username: string
  routing_strategy: string
  /** 是否允许这个客户走会亏本的线路。 */
  allow_cost_breach: boolean
  remark: string
  /** 最后改这一行的管理员 id；没配过时为 0。 */
  updated_by: number
  updated_at: number
  /** 这个客户有没有单独的配置行。 */
  configured: boolean
}

export interface DiscountRoutingPayload {
  user_id: number
  routing_strategy: string
  allow_cost_breach: boolean
  remark: string
}
