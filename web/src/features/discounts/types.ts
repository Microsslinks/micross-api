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
  /** 经销商自己消费，按他的拿货价。 */
  AGENT_WHOLESALE: 'agent_wholesale',
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
  AGENT: 'agent',
  SUBSCRIPTION: 'subscription',
  CUSTOMER_CODE: 'customer_code',
  MIGRATION: 'migration',
} as const

/**
 * 保存前检查里后端给的原因码；对应 `service.DiscountViolationReason*`。
 * 界面上认识的翻译成中文，没见过的原样显示。
 *
 * 注：「规则折扣低于方案最低折扣」(below_min_discount) 已随 rule.Discount 一起废弃——
 * 规则改为纯范围标记后，最低折扣只与基础折扣比较（写入口把关），不再逐条规则比一遍。
 */
export const DISCOUNT_VIOLATION_REASON = {
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
  /**
   * 命中规则按方案基础折扣出价：此字段现在等同于 resolution.discount / entry.base_discount。
   * 留作展示参考；与 rule.Discount（后端已废弃）的概念已不再对应。
   */
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

/**
 * 这位客户身上一条绑定对当前模型的报价与结局。
 * `applied` 为 true 的那条就是这次真正执行的；其余几条的 `reject_reason` 来自后端裁决。
 */
export interface DiscountSimulateCandidate {
  plan_id: number
  plan_name: string
  /** 绑定来源（manual／agent／subscription／customer_code／migration）。 */
  source: string
  /** 落到哪一档：模型级规则 / 厂商级规则 / 方案基础折扣。 */
  specificity: string
  discount: string
  applied: boolean
  rejected: boolean
  reject_reason: string
}

export interface DiscountSimulateResult {
  user: DiscountSimulateUser
  model: string
  vendor: string
  plan: DiscountSimulatePlan | null
  resolution: DiscountSimulateResolution
  /** 该客户挂着的全部生效绑定，含落选者；没挂方案时为空。 */
  candidates: DiscountSimulateCandidate[]
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

// ============================================================================
// Customer pricing audit (mirrors service.CustomerPricingAuditResult)
// ============================================================================

/** 客户核算里每个模型的结论；对应 `service.CustomerAuditVerdict*`。 */
export const CUSTOMER_AUDIT_VERDICT = {
  /** 有过毛利底线的线路，路由能正常挑到赚钱的线。 */
  OK: 'ok',
  /** 已录成本的线路全过不了毛利底线，这单会亏。 */
  LOSS: 'loss',
  /** 有线路但没录进货折扣，成本无从谈起。 */
  UNKNOWN_COST: 'unknown_cost',
  /** 该客户分组下没有可用线路，客户一下单就报错。 */
  NO_CHANNEL: 'no_channel',
} as const

export type CustomerAuditVerdict =
  (typeof CUSTOMER_AUDIT_VERDICT)[keyof typeof CUSTOMER_AUDIT_VERDICT]

/**
 * 客户核算里一个模型的行。
 * 成本与毛利没录进货折扣时是 `null`——「不知道」不能显示成「赚 0 元」。
 */
export interface CustomerAuditModel {
  model: string
  vendor: string
  discount: string
  source: string
  plan: DiscountSimulatePlan | null
  matched_rule: DiscountSimulateRule | null
  channel_count: number
  usable_count: number
  cheapest_cost: string | null
  gross_margin: string | null
  verdict: string
  /** 后端原文说明（中文），直接展示给运营看。 */
  verdict_detail: string
}

export interface CustomerAuditSummary {
  total: number
  ok: number
  loss: number
  unknown_cost: number
  no_channel: number
  /** 能不能签：没有任何会亏、没有线路、成本未知的模型。 */
  signable: boolean
  /** 后端原文一句话结论（中文），直接展示给运营看。 */
  conclusion: string
}

export interface CustomerAuditResult {
  user: DiscountSimulateUser
  min_margin_ratio: string
  models: CustomerAuditModel[]
  summary: CustomerAuditSummary
  /** 后端原文提示（中文），直接展示给运营看。 */
  warnings: string[]
}

export interface AuditCustomerPricingParams {
  userId: number
  models: string[]
}

// ============================================================================
// Customer price check (mirrors service.CustomerPriceQueryResult)
// ============================================================================

/**
 * 价目本里的一条规则：这套价对哪个模型／厂商改价。
 * rule.Discount 已废弃：本字段现在等同于 entry.base_discount（同方案内全部规则同值）。
 * 留作展示参考；做 UI 时建议改读 entry.base_discount，避免按"规则自带折扣"理解。
 */
export interface CustomerPriceBookRule {
  scope_type: string
  scope_value: string
  discount: string
  priority: number
  status: number
}

/**
 * 这个客户身上的一套价（一条绑定 + 它指向的方案与规则）。
 * `in_window` 为 false 时 `window_reason` 写明为什么没生效——价目本连没生效的价
 * 也列出来，否则没法回答「我上周谈的那套价怎么没生效」。
 */
export interface CustomerPriceBookEntry {
  binding_id: number
  plan_id: number
  plan_name: string
  plan_status: number
  source: string
  billing_mode: string
  base_discount: string
  effective_from: number
  effective_to: number
  /** 在生效时间窗内。 */
  in_window: boolean
  /** 不在窗口内时的原因（后端原文，中文）。 */
  window_reason: string
  rules: CustomerPriceBookRule[]
}

/**
 * 一个模型的查价行：核算行的全部字段 + 候选明细。
 * `candidates` 里 `applied` 为 true 的那条就是这次真正执行的价，其余几条写明输在哪一层。
 */
export interface CustomerPriceQueryModel extends CustomerAuditModel {
  candidates: DiscountSimulateCandidate[]
}

export interface CustomerPriceQueryResult {
  user: DiscountSimulateUser
  min_margin_ratio: string
  /** 这个客户身上生效中的全部方案；没挂方案时为空。 */
  price_book: CustomerPriceBookEntry[]
  /** 只填了客户没填模型时为空——价目本本身就是要先看的东西。 */
  models: CustomerPriceQueryModel[]
  summary: CustomerAuditSummary
  /** 后端原文提示（中文），直接展示给运营看。 */
  warnings: string[]
}

export interface QueryCustomerPriceParams {
  userId: number
  /** 留空只查价目本：先看清身上有几套价，再决定问哪个模型。 */
  models?: string[]
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
  /**
   * 经销商给客户发额度时的折算比例（task-09），6 位小数字符串（"0.875000"）。
   * 取值 0~1；1.0 表示保持 1:1 不折算。后端不允许 0 或 > 1。
   */
  topup_conversion_rate: string
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
  /** 经销商发额度折算比例（task-09）；0~1，6 位小数；后端空值兜底 1.000000。 */
  topup_conversion_rate: string
  status: number
  remark: string
}

export interface DiscountRulePayload {
  scope_type: string
  scope_value: string
  /** 规则已改为纯范围标记，命中时按方案基础折扣出价。表单不再接收折扣输入。 */
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
  /** 传 1 只看生效中的绑定；不传（0）返回全部，含已解绑的历史记录 */
  status?: number
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

// ============================================================================
// Model lists (可复用的模型名单；不参与计价)
// ============================================================================

/**
 * 一份起好名字、能反复使用的模型清单（如「企业VIP标准包」），对应 `model.DiscountModelList`。
 * 它只影响「填模型清单格子时能不能一键带入」，不影响任何客户按几折。
 */
export interface DiscountModelList {
  id: number
  name: string
  remark: string
  /** 名单里的模型名，顺序照录入（运营对着客户的单子核对，顺序乱了不好对）。 */
  models: string[]
  created_at: number
  updated_at: number
}

/** 新建 / 更新清单的请求体：名称与名单一起给全（这一版接口不做局部更新）。 */
export interface DiscountModelListPayload {
  name: string
  remark: string
  models: string[]
}

export interface DiscountModelListParams {
  keyword?: string
  page?: number
  page_size?: number
}
