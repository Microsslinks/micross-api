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
import type { TFunction } from 'i18next'

import {
  DISCOUNT_BILLING_MODE,
  DISCOUNT_BINDING_SOURCE,
  DISCOUNT_OWNER,
  DISCOUNT_PLAN_STATUS,
  DISCOUNT_ROUTING_STRATEGY,
  DISCOUNT_SCOPE,
  DISCOUNT_SOURCE,
  DISCOUNT_SUBJECT,
  DISCOUNT_VIOLATION_REASON,
  type DiscountSource,
} from './types'

// ============================================================================
// Labels (i18n keys; use t(labelKey) in components)
// ============================================================================

/** 折扣来源 → i18n 键 */
export const DISCOUNT_SOURCE_LABEL_KEYS: Record<string, string> = {
  [DISCOUNT_SOURCE.MODEL]: 'Model-level rule',
  [DISCOUNT_SOURCE.VENDOR]: 'Vendor-level rule',
  [DISCOUNT_SOURCE.PLAN_BASE]: 'Plan base discount',
  [DISCOUNT_SOURCE.DEFAULT]: 'Official price',
}

/** 方案状态 → i18n 键 */
export const DISCOUNT_PLAN_STATUS_LABEL_KEYS: Record<number, string> = {
  [DISCOUNT_PLAN_STATUS.ENABLED]: 'Enabled',
  [DISCOUNT_PLAN_STATUS.DISABLED]: 'Disabled',
}

/**
 * 折扣来源的展示文案。
 * 后端将来新增来源时原样显示标识，不吞掉信息，也不假装认识它。
 */
export function getDiscountSourceLabel(t: TFunction, source: string): string {
  const labelKey = DISCOUNT_SOURCE_LABEL_KEYS[source as DiscountSource]
  return labelKey ? t(labelKey) : source
}

/** 方案状态的展示文案；未绑定方案时由调用方给出自己的文案。 */
export function getDiscountPlanStatusLabel(
  t: TFunction,
  status: number
): string {
  const labelKey = DISCOUNT_PLAN_STATUS_LABEL_KEYS[status]
  return labelKey ? t(labelKey) : String(status)
}

/** 规则作用范围 → i18n 键；取值与 `model.DiscountScope*` 一致。 */
export const DISCOUNT_RULE_SCOPE_LABEL_KEYS: Record<string, string> = {
  model: 'Model',
  vendor: 'Vendor',
}

/** 规则作用范围的展示文案；遇见没见过的取值就原样显示。 */
export function getDiscountRuleScopeLabel(
  t: TFunction,
  scopeType: string
): string {
  const labelKey = DISCOUNT_RULE_SCOPE_LABEL_KEYS[scopeType]
  return labelKey ? t(labelKey) : scopeType
}

// ============================================================================
// Limits
// ============================================================================

export const DISCOUNT_SIMULATE_LIMITS = {
  /** 客户下拉一次拉多少个（后端按关键字过滤后再分页）。 */
  CUSTOMER_PAGE_SIZE: 50,
  /** 模型名长度上界，避免把超长字符串发给后端。 */
  MODEL_NAME_MAX_LENGTH: 128,
} as const

// ============================================================================
// Error Messages (i18n keys; use t(ERROR_MESSAGES.xxx) when displaying)
// ============================================================================

export const ERROR_MESSAGES = {
  SIMULATE_FAILED: 'Simulation failed',
  CUSTOMERS_LOAD_FAILED: 'Failed to load customers',
  CUSTOMER_REQUIRED: 'Select a customer',
  MODEL_REQUIRED: 'Model name is required',
  MODEL_TOO_LONG: 'Model name is too long',
  PLANS_LOAD_FAILED: 'Failed to load discount plans',
  PLAN_LOAD_FAILED: 'Failed to load the discount plan',
  PLAN_SAVE_FAILED: 'Failed to save the discount plan',
  PLAN_DELETE_FAILED: 'Failed to delete the discount plan',
  PLAN_STATUS_FAILED: 'Failed to change the plan status',
  PLAN_VALIDATE_FAILED: 'Failed to run the pre-save check',
  RULES_LOAD_FAILED: 'Failed to load the rules',
  RULE_SAVE_FAILED: 'Failed to save the rule',
  RULE_DELETE_FAILED: 'Failed to delete the rule',
  BINDINGS_LOAD_FAILED: 'Failed to load the bindings',
  BINDING_SAVE_FAILED: 'Failed to bind the customer',
  BINDING_DELETE_FAILED: 'Failed to unbind the customer',
  ROUTING_LOAD_FAILED: 'Failed to load the routing policy',
  ROUTING_SAVE_FAILED: 'Failed to save the routing policy',
  ROUTING_RESET_FAILED: 'Failed to reset the routing policy',
} as const

// ============================================================================
// Owner / Billing / Binding / Violation labels
// ============================================================================

/** 方案归属 → i18n 键 */
export const DISCOUNT_OWNER_LABEL_KEYS: Record<string, string> = {
  [DISCOUNT_OWNER.PLATFORM]: 'Platform',
  [DISCOUNT_OWNER.AGENT]: 'Agent',
}

/** 计费模式 → i18n 键；注意 "Free" 已被磁盘空间占用，这里另起一个键。 */
export const DISCOUNT_BILLING_MODE_LABEL_KEYS: Record<string, string> = {
  [DISCOUNT_BILLING_MODE.USAGE]: 'Per usage',
  [DISCOUNT_BILLING_MODE.SUBSCRIPTION]: 'Subscription',
  [DISCOUNT_BILLING_MODE.FREE]: 'Free of charge',
}

/** 绑定来源 → i18n 键 */
export const DISCOUNT_BINDING_SOURCE_LABEL_KEYS: Record<string, string> = {
  [DISCOUNT_BINDING_SOURCE.MANUAL]: 'Manual',
  [DISCOUNT_BINDING_SOURCE.AGENT]: 'Dealer price',
  [DISCOUNT_BINDING_SOURCE.SUBSCRIPTION]: 'Subscription',
  [DISCOUNT_BINDING_SOURCE.CUSTOMER_CODE]: 'Customer code',
  [DISCOUNT_BINDING_SOURCE.MIGRATION]: 'Migration',
}

/** 绑定主体 → i18n 键 */
export const DISCOUNT_SUBJECT_LABEL_KEYS: Record<string, string> = {
  [DISCOUNT_SUBJECT.USER]: 'User',
  [DISCOUNT_SUBJECT.AGENT]: 'Agent',
}

/** 保存前检查的原因码 → i18n 键 */
export const DISCOUNT_VIOLATION_REASON_LABEL_KEYS: Record<string, string> = {
  [DISCOUNT_VIOLATION_REASON.BELOW_MIN_DISCOUNT]:
    'Discount is below the plan minimum',
  [DISCOUNT_VIOLATION_REASON.MIN_ABOVE_BASE]:
    'Plan minimum is above its base discount',
  [DISCOUNT_VIOLATION_REASON.COST_BREACH]: 'Cost exceeds the profit floor',
  [DISCOUNT_VIOLATION_REASON.NO_CHANNEL]: 'No route can serve this rule',
}

/**
 * 查不到文案就原样显示标识：宁可让运营看见一个没翻译的英文码，
 * 也不能假装认识它、显示成别的意思。
 */
function lookupLabel(
  t: TFunction,
  labelKeys: Record<string, string>,
  value: string
): string {
  const labelKey = labelKeys[value]
  return labelKey ? t(labelKey) : value
}

export function getDiscountOwnerLabel(t: TFunction, ownerType: string): string {
  return lookupLabel(t, DISCOUNT_OWNER_LABEL_KEYS, ownerType)
}

export function getDiscountBillingModeLabel(
  t: TFunction,
  billingMode: string
): string {
  return lookupLabel(t, DISCOUNT_BILLING_MODE_LABEL_KEYS, billingMode)
}

export function getDiscountBindingSourceLabel(
  t: TFunction,
  source: string
): string {
  return lookupLabel(t, DISCOUNT_BINDING_SOURCE_LABEL_KEYS, source)
}

export function getDiscountSubjectLabel(
  t: TFunction,
  subjectType: string
): string {
  return lookupLabel(t, DISCOUNT_SUBJECT_LABEL_KEYS, subjectType)
}

export function getDiscountViolationReasonLabel(
  t: TFunction,
  reason: string
): string {
  return lookupLabel(t, DISCOUNT_VIOLATION_REASON_LABEL_KEYS, reason)
}

// ============================================================================
// Filter values (shared by the route search schema and the toolbar)
// ============================================================================

export const DISCOUNT_STATUS_FILTER_VALUES = ['0', '1'] as const
export const DISCOUNT_OWNER_FILTER_VALUES = ['platform', 'agent'] as const

/** 状态下拉项：筛选器与表单共用。 */
export function getDiscountStatusOptions(t: TFunction) {
  return [
    { label: t('Enabled'), value: String(DISCOUNT_PLAN_STATUS.ENABLED) },
    { label: t('Disabled'), value: String(DISCOUNT_PLAN_STATUS.DISABLED) },
  ]
}

/** 归属下拉项：筛选器与表单共用。 */
export function getDiscountOwnerOptions(t: TFunction) {
  return [
    { label: t('Platform'), value: DISCOUNT_OWNER.PLATFORM },
    { label: t('Agent'), value: DISCOUNT_OWNER.AGENT },
  ]
}

/**
 * 计费模式下拉项。
 * 文案统一走 `getDiscountBillingModeLabel`，免得下拉和表格各写一份再走散。
 */
export function getDiscountBillingModeOptions(t: TFunction) {
  return [
    DISCOUNT_BILLING_MODE.USAGE,
    DISCOUNT_BILLING_MODE.SUBSCRIPTION,
    DISCOUNT_BILLING_MODE.FREE,
  ].map((value) => ({ label: getDiscountBillingModeLabel(t, value), value }))
}

/** 规则范围下拉项。 */
export function getDiscountScopeOptions(t: TFunction) {
  return [
    { label: t('Model'), value: DISCOUNT_SCOPE.MODEL },
    { label: t('Vendor'), value: DISCOUNT_SCOPE.VENDOR },
  ]
}

// ============================================================================
// Limits
// ============================================================================

export const DISCOUNT_PLAN_LIMITS = {
  /** 一个方案的规则一次全拉，规则数量本就有限。 */
  RULE_PAGE_SIZE: 100,
  /** 绑定列表每页条数。 */
  BINDING_PAGE_SIZE: 50,
  /** 绑定客户时给用户下拉取的条数。 */
  CUSTOMER_PAGE_SIZE: 50,
}

// ============================================================================
// Customer routing policy (择优策略 / 允许走亏损线路)
// ============================================================================

/** 择优口径 → i18n 键 */
export const DISCOUNT_ROUTING_STRATEGY_LABEL_KEYS: Record<string, string> = {
  [DISCOUNT_ROUTING_STRATEGY.MARGIN]: 'Highest margin first',
  [DISCOUNT_ROUTING_STRATEGY.PRIORITY]: 'Stable route first',
}

/** 择优口径的展示文案；遇见没见过的取值就原样显示，不假装认识它。 */
export function getDiscountRoutingStrategyLabel(
  t: TFunction,
  strategy: string
): string {
  return lookupLabel(t, DISCOUNT_ROUTING_STRATEGY_LABEL_KEYS, strategy)
}

/** 择优策略的两个选项；平铺成单选，与方案表单里的归属／状态一致。 */
export function getDiscountRoutingStrategyOptions(t: TFunction) {
  return [
    DISCOUNT_ROUTING_STRATEGY.MARGIN,
    DISCOUNT_ROUTING_STRATEGY.PRIORITY,
  ].map((value) => ({
    label: getDiscountRoutingStrategyLabel(t, value),
    value,
  }))
}

export const DISCOUNT_ROUTING_LIMITS = {
  /** 备注长度上界，与后端列的 varchar(255) 对齐。 */
  REMARK_MAX_LENGTH: 255,
} as const
